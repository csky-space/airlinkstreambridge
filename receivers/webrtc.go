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
	"sync"
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

	onRTP           func(data []byte) error
	ws              *Webrtc_websocket
	lastRemoteSdp   string
	pc              *webrtc.PeerConnection
	iceConfigurator *iceconfigurator.ICEConfigurator
	s               webrtc.SettingEngine

	lastPacketTime      time.Time
	peerConnectTimeout  *time.Timer
	videoTrackTimeout   *time.Timer
	reconnectMutex      sync.Mutex
	transmitEnableMutex sync.Mutex

	//PeerClosed            chan struct{}
	//PeerConnected         chan struct{}
	//PeerFailed            chan struct{}
	//PeerDisconnected      chan struct{}
	//WebrtcReceiverCreated chan struct{}
	PeerClosed            *EventBroadcaster
	PeerConnected         *EventBroadcaster
	PeerFailed            *EventBroadcaster
	PeerDisconnected      *EventBroadcaster
	WebrtcReceiverCreated *EventBroadcaster

	channelsMutex sync.Mutex

	isReconnecting    bool
	transmitEnabled   bool
	videoIsRunning    bool
	iceTrickleEnabled bool
}

func (wr *WebrtcReceiver) SetOnRTP(onRTP func(data []byte) error) {
	wr.onRTP = onRTP
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

//func (wr *WebrtcReceiver) resetChannel(ch *chan struct{}) {
//	wr.channelsMutex.Lock()
//	defer wr.channelsMutex.Unlock()
//	if *ch != nil {
//		close(*ch)
//	}
//
//	*ch = make(chan struct{})
//}

func (wr *WebrtcReceiver) VideoIsRunning() bool {
	mut := sync.Mutex{}
	mut.Lock()
	defer mut.Unlock()
	return wr.videoIsRunning
}

func (wr *WebrtcReceiver) IsConnected() bool {
	return wr.pc.ConnectionState() == webrtc.PeerConnectionStateConnected
}

func (wr *WebrtcReceiver) SetTransmitEnabled(enabled bool) {
	wr.transmitEnableMutex.Lock()
	defer wr.transmitEnableMutex.Unlock()

	wr.transmitEnabled = enabled
}

func (wr *WebrtcReceiver) registerDefaultCodecs() error {
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
		return err
	}

	if err := wr.mediaEngine.RegisterCodec(opusCodec, webrtc.RTPCodecTypeAudio); err != nil {
		return err
	}
	return nil
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

func (wr *WebrtcReceiver) setupCodecs(codecs []JSONCodec) error {
	log.Println("setupCodecs")
	if codecs == nil {
		err := wr.registerDefaultCodecs()
		if err != nil {
			return err
		}
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
				return err
			}
		}
	}
	return nil
}

func (wr *WebrtcReceiver) SetupCodecs(codecs []JSONCodec) error {
	err := wr.setupCodecs(codecs)
	if err != nil {
		return err
	}
	return nil
}

func (wr *WebrtcReceiver) Configure(hostUrl string, login string, password string) error {
	log.Println("configuring")
	var err error
	wr.iceConfigurator, err = iceconfigurator.NewICEConfigurator(hostUrl, login, password)
	if err != nil {
		log.Printf("error on ice configuration: %v", err)
		return err
	}
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
	return nil
	//s.SetFireOnTrackBeforeFirstRTP(true)
}

func NewWebrtcReceiver() *WebrtcReceiver {
	wr := &WebrtcReceiver{
		PeerClosed:            NewEventBroadcaster(),
		PeerConnected:         NewEventBroadcaster(),
		PeerFailed:            NewEventBroadcaster(),
		PeerDisconnected:      NewEventBroadcaster(),
		WebrtcReceiverCreated: NewEventBroadcaster(),
		peerConnectTimeout:    time.NewTimer(1000000000 * time.Second), videoTrackTimeout: time.NewTimer(1000000000 * time.Second), iceTrickleEnabled: false,
	}

	return wr
}

func (wr *WebrtcReceiver) Open() error {
	log.Println("open")
	interceptorRegistry := &interceptor.Registry{}

	intervalPliFactory, err := intervalpli.NewReceiverInterceptor(intervalpli.GeneratorInterval(time.Millisecond * 1000))
	if err != nil {
		log.Printf("error on creating pli: %v", err)
		return err
	}
	log.Println("pli")
	interceptorRegistry.Add(intervalPliFactory)

	if err := webrtc.RegisterDefaultInterceptors(wr.mediaEngine, interceptorRegistry); err != nil {
		log.Fatal(err)
		return err
	}
	log.Println("webrtc api")
	wr.api = webrtc.NewAPI(
		webrtc.WithSettingEngine(wr.s),
		webrtc.WithMediaEngine(wr.mediaEngine),
		webrtc.WithInterceptorRegistry(interceptorRegistry),
	)
	log.Println("create peer")
	err = wr.createPeerConnection()
	if err != nil {
		log.Fatalf("failed on creating peer with error: %v", err)
		return err
	}
	wr.ws, err = NewWebrtcWebsocket(wr.iceConfigurator.WsURL + "?name=GS" + wr.iceConfigurator.Login + "&partnerName=" + wr.iceConfigurator.Login)
	if err != nil {
		return err
	}
	log.Println("events")
	wr.ws.SetOnOffer(wr.onOffer)
	wr.ws.SetOnRemoteCandidate(wr.ws.onRemoteCandidate)
	wr.ws.SetOnPing(func() {
		wr.ws.startSignalling()
	})
	log.Println("peer connection going")
	go wr.peerConnectionWatchdog()
	return nil
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
		return err
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
	peerClosed := wr.PeerClosed.Subscribe()
	for {
		select {
		case <-peerClosed:
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
	log.Println("peerConnectionWatchdog")
	if wr.isReconnecting {
		return
	}
	wr.isReconnecting = true
	wr.WebrtcReceiverCreated.Fire()

	connected := wr.PeerConnected.Subscribe()
	disconnected := wr.PeerDisconnected.Subscribe()
	failed := wr.PeerFailed.Subscribe()
	//closed := wr.PeerClosed.Subscribe()
	for {
		select {
		case <-connected:
			log.Println("PeerConnection established")
			//wr.reconnectMutex.Lock()
			wr.isReconnecting = false
			//wr.reconnectMutex.Unlock()
			wr.peerConnectTimeout.Stop()
		case <-failed:
			wr.videoTrackTimeout.Stop()
			log.Println("Peer connection failed, retrying")
			err := wr.ReLaunchPeer()
			if err != nil {
				log.Fatalf("Failed relaunch peer with error: %v", err)
			}
		case <-disconnected:
			wr.videoTrackTimeout.Stop()
			log.Println("Peer connection disconnected, retrying")
			err := wr.ReLaunchPeer()
			if err != nil {
				log.Fatalf("Failed relaunch peer with error: %v", err)
			}
		case <-wr.peerConnectTimeout.C:
			log.Println("Connection timeout, retrying")
			err := wr.ReLaunchPeer()
			if err != nil {
				log.Fatalf("Failed relaunch peer with error: %v", err)
			}
		case <-wr.videoTrackTimeout.C:
			log.Println("Video timeout, retrying")
			err := wr.ReLaunchPeer()
			if err != nil {
				log.Fatalf("Failed relaunch peer with error: %v", err)
			}
		default:
			time.Sleep(time.Millisecond * 100)
		}
		//
	}
}

func (wr *WebrtcReceiver) onTrack(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	if track.Kind() != webrtc.RTPCodecTypeVideo {
		return
	}
	wr.videoTrackTimeout.Reset(time.Second * 5)
	//wr.updateLastPacketTime()

	//go wr.checkPacketTimeout()
	go func() {
		for {
			select {
			case <-wr.PeerClosed.Subscribe():
				wr.videoIsRunning = false
				return
			default:
				pkt, _, err := track.ReadRTP()
				if err != nil {
					wr.videoIsRunning = false
					log.Printf("ReadRTP error: %v", err)
					return
				}

				//wr.updateLastPacketTime()

				raw, err := pkt.Marshal()
				if err != nil {
					wr.videoIsRunning = false
					if errors.Is(err, io.EOF) {
						return
					}
					log.Println("track marshal error:", err)
					continue
				}
				wr.videoTrackTimeout.Reset(time.Second * 5)
				if wr.transmitEnabled {
					wr.videoIsRunning = true

					err = wr.onRTP(raw)
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

func (wr *WebrtcReceiver) createPeerConnection() error {
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
		return err
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
			wr.PeerConnected.Fire()
		case webrtc.PeerConnectionStateClosed:
			wr.PeerClosed.Fire()
		case webrtc.PeerConnectionStateFailed:
			wr.PeerFailed.Fire()
		case webrtc.PeerConnectionStateDisconnected:
			wr.PeerDisconnected.Fire()
		}
	})

	_, _err := wr.pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	})
	if _err != nil {
		log.Printf("Failed to add transceiver: %v", err)
		return err
	}

	wr.pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Track received: %s (SSRC: %d)", track.Codec().MimeType, track.SSRC())
		wr.onTrack(track, receiver)
	})

	return nil
}

func (wr *WebrtcReceiver) CreateDefaultPipeline(hostUrl string, login string, password string) error {
	log.Println("create default pipeline")
	err := wr.Configure(hostUrl, login, password)

	if err != nil {
		log.Printf("Error on configuring: %v", err)
		return err
	}

	err = wr.SetupCodecs(nil)
	if err != nil {
		return err
	}
	wr.Open()
	return nil
}

func NewDefaultWebrtcReceiver(hostUrl string, login string, password string) (*WebrtcReceiver, error) {
	log.Println("new webrtc default")
	wr := NewWebrtcReceiver()

	err := wr.CreateDefaultPipeline(hostUrl, login, password)
	if err != nil {
		return nil, err
	}
	wr.SetTransmitEnabled(true)
	return wr, nil
}

func (wr *WebrtcReceiver) Close() {
	go wr.PeerClose()
	go wr.wsClose()
}

func (wr *WebrtcReceiver) LaunchPeer() error {
	log.Println("LaunchPeer")
	wr.peerConnectTimeout = time.NewTimer(20 * time.Second)
	err := wr.ws.ping()
	if err != nil {
		return err
	}
	return nil
}

func (wr *WebrtcReceiver) ReLaunchPeer() error {
	if (wr.pc != nil) && (wr.pc.ConnectionState() == webrtc.PeerConnectionStateConnecting) && (wr.pc.ConnectionState() == webrtc.PeerConnectionStateConnected) {
		wr.PeerClose()
	}
	err := wr.createPeerConnection()
	if err != nil {
		return err
	}
	err = wr.LaunchPeer()
	if err != nil {
		log.Fatalf("failed on launch peer with error: %v", err)
		return err
	}
	return nil
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
	<-wr.PeerClosed.Subscribe()
	wr.videoTrackTimeout.Stop()
	log.Println("peer closed")
}

func (wr *WebrtcReceiver) WsIsOpen() bool {
	return wr.ws.IsOpen()
}
