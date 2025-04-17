package receivers

import (
	"AirlinkStreamBridge/receivers/iceconfigurator"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	"github.com/pion/logging"
	"github.com/pion/webrtc/v4"
)

var id string = rand.Text()

type WebrtcReceiver struct {
	api         *webrtc.API
	mediaEngine *webrtc.MediaEngine

	ws              *Webrtc_websocket
	lastRemoteSdp   string
	pc              *webrtc.PeerConnection
	udpSender       *net.UDPConn
	udpAddr         string
	setUDPAddrMut   sync.Mutex
	iceConfigurator *iceconfigurator.ICEConfigurator
	s               webrtc.SettingEngine
	udpNetAddr      *net.UDPAddr

	lastPacketTime      time.Time
	peerConnectTimeout  *time.Timer
	videoTrackTimeout   *time.Timer
	reconnectMutex      sync.Mutex
	transmitEnableMutex sync.Mutex

	peerDisconnected chan struct{}
	peerConnected    chan struct{}
	peerFailed       chan struct{}

	channelsMutex sync.Mutex

	isReconnecting    bool
	transmitEnabled   bool
	videoIsRunning    bool
	iceTrickleEnabled bool
}

type JSONRtcpFeedback struct {
	Type      string `json:"type"`
	Parameter string `json:"parameter"`
}

type JSONCodec struct {
	Mime        string             `json:"mime"`
	ClockRate   int                `json:"clockRate"`
	PayloadType int                `json:"payloadType"`
	Feedbacks   []JSONRtcpFeedback `json:"feedbacks"`
}

func (wr *WebrtcReceiver) resetChannel(ch *chan struct{}) {
	wr.channelsMutex.Lock()
	defer wr.channelsMutex.Unlock()
	if *ch != nil {
		close(*ch)
	}

	*ch = make(chan struct{})
}

func (wr *WebrtcReceiver) VideoIsRunning() bool {
	mut := sync.Mutex{}
	mut.Lock()
	defer mut.Unlock()
	return wr.videoIsRunning
}

func (wr *WebrtcReceiver) IsConnected() bool {
	return wr.pc.ConnectionState() == webrtc.PeerConnectionStateConnected
}

func (wr *WebrtcReceiver) SetupUDP(address string, port int) {
	log.Printf("setup udp with %s:%d\n", address, port)
	wr.setUDPAddrMut.Lock()
	defer wr.setUDPAddrMut.Unlock()
	if wr.udpSender != nil {
		wr.udpSender.Close()
	}

	wr.udpAddr = address + ":" + strconv.Itoa(port)
	var err error
	//wr.udpSender, err = net.ListenPacket("udp", ":0")
	//if err != nil {
	//	log.Fatalf("ListenPacket error: %v", err)
	//}

	wr.udpNetAddr, err = net.ResolveUDPAddr("udp", wr.udpAddr)
	if err != nil {
		panic(err)
	}

	wr.udpSender, err = net.DialUDP("udp", nil, wr.udpNetAddr)
	if err != nil {
		panic(err)
	}
	wr.udpSender.SetWriteBuffer(1 << 20)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Println("Received signal:", sig)
		wr.Close()
		os.Exit(0)
	}()
	log.Printf("end of setup udp with %s:%d\n", address, port)
}

func (wr *WebrtcReceiver) SetTransmitEnabled(enabled bool) {
	wr.transmitEnableMutex.Lock()
	defer wr.transmitEnableMutex.Unlock()

	wr.transmitEnabled = enabled
}

func (wr *WebrtcReceiver) registerDefaultCodecs() {
	h265Codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeH265,
			ClockRate: 90000,
			//SDPFmtpLine: "profile-level-id=42e01f;hevc-profile=1",
			RTCPFeedback: []webrtc.RTCPFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "transport-cc"},
				{Type: "ccm", Parameter: "fir"},
			},
		}, PayloadType: 96,
	}

	opusCodec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeOpus,
			RTCPFeedback: []webrtc.RTCPFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "transport-cc"},
				{Type: "ccm", Parameter: "fir"},
			},
		}, PayloadType: 111,
	}

	if err := wr.mediaEngine.RegisterCodec(h265Codec, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}

	if err := wr.mediaEngine.RegisterCodec(opusCodec, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}
}

func collectFeedback(feedbacks []JSONRtcpFeedback) []webrtc.RTCPFeedback {
	result := []webrtc.RTCPFeedback{}
	for feedbackNumber := 0; feedbackNumber < len(feedbacks); feedbackNumber++ {
		feedback := feedbacks[feedbackNumber]
		result = append(result, webrtc.RTCPFeedback{
			Type:      feedback.Type,
			Parameter: feedback.Parameter,
		})
	}

	return result
}

func (wr *WebrtcReceiver) setupCodecs(codecs []JSONCodec) {
	log.Println("setupCodecs")
	if codecs == nil {
		wr.registerDefaultCodecs()
	} else {
		for codecNumber := 0; codecNumber < len(codecs); codecNumber++ {
			rtcpFeedback := collectFeedback(codecs[codecNumber].Feedbacks)
			codec := webrtc.RTPCodecParameters{

				RTPCodecCapability: webrtc.RTPCodecCapability{
					MimeType:     codecs[codecNumber].Mime,
					RTCPFeedback: rtcpFeedback,
				},
				PayloadType: webrtc.PayloadType(codecs[codecNumber].PayloadType),
			}
			if err := wr.mediaEngine.RegisterCodec(codec, webrtc.RTPCodecTypeVideo); err != nil {
				panic(err)
			}
		}
	}
}

func (wr *WebrtcReceiver) SetupCodecs(codecs []JSONCodec) {
	wr.setupCodecs(codecs)
}

func (wr *WebrtcReceiver) SetupOutputProtocol(protocol string, address string, port int) {
	log.Println("output protocol")
	if protocol == "" {
		protocol = "UDP"
	}
	if address == "" {
		address = "127.0.0.1"
	}
	if port == 0 {
		port = 9050
	}
	log.Printf("Output set as:\nprotocol:%s\naddress:%s\nport:%d", protocol, address, port)
	switch protocol {
	case "UDP":
		wr.SetupUDP(address, port)
	default:
		log.Printf("Unsupported output protocol %s", protocol)
	}
}

func (wr *WebrtcReceiver) Configure(hostUrl string, login string, password string) {
	log.Println("configuring")

	wr.iceConfigurator = iceconfigurator.NewICEConfigurator(hostUrl, login, password)
	wr.lastRemoteSdp = ""

	wr.mediaEngine = &webrtc.MediaEngine{}
	loggingFactory := logging.NewDefaultLoggerFactory()
	loggingFactory.DefaultLogLevel.Set(logging.LogLevelTrace)
	s := webrtc.SettingEngine{
		//LoggerFactory: loggingFactory,
	}
	s.SetICETimeouts(5*time.Second, 30*time.Second, 5*time.Second)
	s.SetIPFilter(func(ip net.IP) bool {
		return ip.To4() != nil
	})
	wr.s = s
	//s.SetFireOnTrackBeforeFirstRTP(true)
}

func NewWebrtcReceiver() *WebrtcReceiver {
	wr := &WebrtcReceiver{peerDisconnected: make(chan struct{}), iceTrickleEnabled: false, peerConnected: make(chan struct{}),
		peerConnectTimeout: time.NewTimer(1000000000 * time.Second), videoTrackTimeout: time.NewTimer(1000000000 * time.Second)}

	return wr
}

func (wr *WebrtcReceiver) Open() {
	log.Println("open")
	interceptorRegistry := &interceptor.Registry{}

	intervalPliFactory, err := intervalpli.NewReceiverInterceptor(intervalpli.GeneratorInterval(time.Millisecond * 1000))
	if err != nil {
		panic(err)
	}

	interceptorRegistry.Add(intervalPliFactory)

	if err := webrtc.RegisterDefaultInterceptors(wr.mediaEngine, interceptorRegistry); err != nil {
		log.Fatal(err)
	}

	wr.api = webrtc.NewAPI(
		webrtc.WithSettingEngine(wr.s),
		webrtc.WithMediaEngine(wr.mediaEngine),
		webrtc.WithInterceptorRegistry(interceptorRegistry),
	)
	wr.createPeerConnection()
	wr.ws = NewWebrtcWebsocket(wr.iceConfigurator.WsURL + "?name=GS" + wr.iceConfigurator.Login + "&partnerName=" + wr.iceConfigurator.Login)

	wr.ws.SetOnOffer(wr.onOffer)
	wr.ws.SetOnRemoteCandidate(wr.ws.onRemoteCandidate)
	wr.ws.SetOnPing(func() {
		wr.ws.startSignalling()
	})

	go wr.peerConnectionWatchdog()
}

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

func (wr *WebrtcReceiver) onOffer(sdp string) error {
	if err := wr.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  sdp,
	}); err != nil {
		log.Printf("Failed to set remote description: %v", err)
		return err
	}

	answer, err := wr.pc.CreateAnswer(nil)
	if err != nil {
		log.Printf("Failed to create answer: %v", err)
		return err
	}

	if err = wr.pc.SetLocalDescription(answer); err != nil {
		log.Printf("Failed to set local description: %v", err)
		return err
	}

	return nil
}

func (wr *WebrtcReceiver) onRemoteCandidate(candidate string) error {
	log.Printf("candidate: %s\n", candidate)
	err := wr.pc.AddICECandidate(webrtc.ICECandidateInit{
		Candidate: candidate,
	})
	if err != nil {
		log.Printf("Failed to add remote ICE candidate: %v", err)
	} else {
		log.Printf("Added remote ICE candidate: %s", candidate)
	}

	return nil
}

func isPrivateIP(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed.IsPrivate() || parsed.IsLoopback()
}

func (wr *WebrtcReceiver) updateLastPacketTime() {
	wr.reconnectMutex.Lock()
	defer wr.reconnectMutex.Unlock()
	wr.lastPacketTime = time.Now()
}

func (wr *WebrtcReceiver) checkPacketTimeout() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wr.peerDisconnected:
			return
		case <-ticker.C:
			wr.reconnectMutex.Lock()
			elapsed := time.Since(wr.lastPacketTime).Seconds()
			wr.reconnectMutex.Unlock()

			if elapsed > (5 * time.Second).Seconds() {
				wr.videoTrackTimeout.Reset(5 * time.Second)
				return
			}
		}
	}
}

func (wr *WebrtcReceiver) peerConnectionWatchdog() {
	wr.reconnectMutex.Lock()
	defer wr.reconnectMutex.Unlock()

	if wr.isReconnecting {
		return
	}
	wr.isReconnecting = true

	go func() {
		//wr.SetupUDP("127.0.0.1", )
		defer func() {
			wr.reconnectMutex.Lock()
			wr.isReconnecting = false
			wr.reconnectMutex.Unlock()
		}()

		for {
			select {
			case <-wr.peerConnected:
				log.Println("PeerConnection established")
			case <-wr.peerFailed:
				log.Println("Peer connection failed, retrying PeerConnection")
				wr.ReLaunchPeer()
			case <-wr.peerConnectTimeout.C:
				log.Println("Connection timeout, retrying PeerConnection")
				wr.ReLaunchPeer()
			case <-wr.videoTrackTimeout.C:
				log.Println("Video timeout, retrying PeerConnection")
				wr.ReLaunchPeer()
			}
		}
	}()
}

func (wr *WebrtcReceiver) onTrack(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	if track.Kind() != webrtc.RTPCodecTypeVideo {
		return
	}

	wr.updateLastPacketTime()

	go wr.checkPacketTimeout()

	go func() {
		for {
			select {
			case <-wr.peerDisconnected:
				wr.videoIsRunning = false
				return
			default:
				pkt, _, err := track.ReadRTP()
				if err != nil {
					wr.videoIsRunning = false
					log.Printf("ReadRTP error: %v", err)
					return
				}

				wr.updateLastPacketTime()

				raw, err := pkt.Marshal()
				if err != nil {
					wr.videoIsRunning = false
					if errors.Is(err, io.EOF) {
						return
					}
					log.Println("track marshal error:", err)
					continue
				}
				if wr.transmitEnabled {
					wr.videoIsRunning = true

					_, err = wr.udpSender.Write(raw)
					if err != nil {
						wr.videoIsRunning = false
						log.Printf("raw %v didn't write with error: %s", raw, err)
					}
				} else {
					wr.videoIsRunning = false
				}

			}
		}
	}()
}

func (wr *WebrtcReceiver) iceSetup() {
	if wr.iceTrickleEnabled {
		wr.pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
			if candidate == nil {
				return
			}
			candidateInit := candidate.ToJSON()
			candidateJSON := Candidate{
				ID:        id,
				Type:      "candidate",
				Candidate: candidateInit.Candidate,
			}
			candidateJSONData, err := json.Marshal(candidateJSON)
			if err != nil {
				log.Printf("Candidate serialization problem: %v", err)
				return
			}
			fmt.Printf("Candidate: %s\n", string(candidateJSONData))
			wr.ws.WriteMessage(candidateJSONData)
		})
	}

	wr.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State changed: %s", state)

	})

	wr.pc.OnICEGatheringStateChange(func(state webrtc.ICEGatheringState) {
		log.Printf("ICE Gathering State changed: %s\n", state)
		if state == webrtc.ICEGatheringStateComplete {
			answerJSON, _ := json.Marshal(Answer{
				ID:   id,
				Type: "answer",
				SDP:  wr.pc.LocalDescription().SDP,
			})
			log.Printf("local desc is: %s", answerJSON)

			wr.ws.WriteMessage(answerJSON)
		}
	})
}

func (wr *WebrtcReceiver) createPeerConnection() {
	iceServers := []webrtc.ICEServer{}
	for i := 0; i < len(wr.iceConfigurator.StunServers); i++ {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs: []string{wr.iceConfigurator.StunServers[i]},
		})
	}
	for i := 0; i < len(wr.iceConfigurator.TurnServers); i++ {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs:           []string{wr.iceConfigurator.TurnServers[i].URL},
			Username:       wr.iceConfigurator.TurnServers[i].Username,
			Credential:     wr.iceConfigurator.TurnServers[i].Credential,
			CredentialType: webrtc.ICECredentialTypePassword,
		})
	}

	pc, err := wr.api.NewPeerConnection(webrtc.Configuration{
		ICETransportPolicy: webrtc.ICETransportPolicyAll,
		ICEServers:         iceServers,
	})
	if err != nil {
		log.Printf("Failed to create PeerConnection: %v", err)
		return
	}

	wr.pc = pc

	wr.iceSetup()

	wr.pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		log.Printf("dc: %s", dc.Label())
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Printf("dc message: %s", msg.Data)
		})
	})

	wr.pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("PeerConnection State changed: %s", state)
		switch state {
		case webrtc.PeerConnectionStateConnected:
			wr.resetChannel(&wr.peerConnected)
			wr.peerConnectTimeout.Stop()
		case webrtc.PeerConnectionStateClosed:
			wr.resetChannel(&wr.peerDisconnected)
		case webrtc.PeerConnectionStateFailed:
			wr.resetChannel(&wr.peerFailed)
		}
	})

	_, _err := wr.pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	})
	if _err != nil {
		log.Printf("Failed to add transceiver: %v", err)
	}

	wr.pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Track received: %s (SSRC: %d)", track.Codec().MimeType, track.SSRC())
		wr.onTrack(track, receiver)
	})
}

func (wr *WebrtcReceiver) CreateDefaultPipeline(hostUrl string, login string, password string, UDPPort int) {
	log.Println("create default pipeline")
	wr.Configure(hostUrl, login, password)

	wr.SetupOutputProtocol("UDP", "127.0.0.1", UDPPort)
	wr.SetupCodecs(nil)
	wr.Open()

}

func NewDefaultWebrtcReceiver(hostUrl string, login string, password string, port int) *WebrtcReceiver {
	log.Println("new webrtc default")
	wr := NewWebrtcReceiver()
	if port == 0 {
		port = 9050
	}
	wr.CreateDefaultPipeline(hostUrl, login, password, port)
	wr.SetTransmitEnabled(true)
	return wr
}

func (wr *WebrtcReceiver) Close() {
	go wr.PeerClose()
	go wr.wsClose()
	if wr.udpSender != nil {
		go wr.udpSender.Close()
	}

}

func (wr *WebrtcReceiver) LaunchPeer() {
	log.Println("LaunchPeer")
	wr.peerConnectTimeout = time.NewTimer(20 * time.Second)
	wr.ws.ping()
}

func (wr *WebrtcReceiver) ReLaunchPeer() {
	if (wr.pc != nil) && (wr.pc.ConnectionState() != webrtc.PeerConnectionStateClosed) {
		wr.PeerClose()
	}
	wr.createPeerConnection()
	wr.LaunchPeer()
}

func (wr *WebrtcReceiver) wsClose() {
	if wr.ws != nil {
		wr.ws.Close()
	}
}

func (wr *WebrtcReceiver) PeerClose() {
	log.Println("PeerClose")
	if wr.pc != nil {
		wr.pc.Close()
	}
	<-wr.peerDisconnected
	wr.videoTrackTimeout.Stop()
	log.Println("peer closed")
}

func (wr *WebrtcReceiver) WsIsOpen() bool {
	return wr.ws.IsOpen()
}
