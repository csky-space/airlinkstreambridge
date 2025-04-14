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
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	"github.com/pion/logging"
	"github.com/pion/webrtc/v4"
)

var id string = rand.Text()

type Ping struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}
type Request struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}
type Answer struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	SDP  string `json:"sdp"`
}
type Candidate struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Candidate string `json:"candidate"`
}
type WebrtcReceiver struct {
	api             *webrtc.API
	mediaEngine     *webrtc.MediaEngine
	wsConn          *websocket.Conn
	wsUrl           string
	lastRemoteSdp   string
	pc              *webrtc.PeerConnection
	udpSender       *net.UDPConn
	udpAddr         string
	setUDPAddrMut   sync.Mutex
	iceConfigurator *iceconfigurator.ICEConfigurator
	s               webrtc.SettingEngine

	lastPacketTime      time.Time
	reconnectMutex      sync.Mutex
	transmitEnableMutex sync.Mutex
	websocketMut        sync.Mutex
	isReconnecting      bool
	shutdownChan        chan struct{}
	transmitEnabled     bool
	videoIsRunning      bool
	iceTrickleEnabled   bool
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
	wr.setUDPAddrMut.Lock()
	defer wr.setUDPAddrMut.Unlock()
	wr.udpAddr = address + ":" + strconv.Itoa(port)
	serverAddr, err := net.ResolveUDPAddr("udp", wr.udpAddr)
	if err != nil {
		panic(err)
	}
	wr.udpSender, err = net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		panic(err)
	}
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
	wr.wsUrl = wr.iceConfigurator.WsURL + "?name=GS" + login + "&partnerName=" + login
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
	wr := &WebrtcReceiver{shutdownChan: make(chan struct{}), iceTrickleEnabled: false}
	wr.websocketMut = sync.Mutex{}
	//wr.packetReceived = make(chan struct{}, 1)

	return wr
}

func (wr *WebrtcReceiver) Open() {
	log.Println("open")
	interceptorRegistry := &interceptor.Registry{}

	intervalPliFactory, err := intervalpli.NewReceiverInterceptor(intervalpli.GeneratorInterval(time.Millisecond * 3000))
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
	wr.establishWs()
}

func (wr *WebrtcReceiver) establishWs() {
	websocket.DefaultDialer.HandshakeTimeout = 10 * time.Second
	wsConn, _, err := websocket.DefaultDialer.Dial(wr.wsUrl, nil)

	if err != nil {
		log.Println("Websocket connection error:", err)
		return
	}
	wr.wsConn = wsConn
	go wr.readMessages()
}

func (wr *WebrtcReceiver) readMessages() {
	wr.ping()
	for {
		_, message, err := wr.wsConn.ReadMessage()
		if err != nil {
			log.Println("Error during reading a message:", err)
			break
		}
		log.Printf("Received: %s\n", message)
		var data map[string]interface{}
		if err := json.Unmarshal(message, &data); err != nil {
			log.Println("Error during jsonify:", err)
			continue
		}
		msgType, ok := data["type"].(string)
		if !ok {
			_, ok := data["candidate"].(string)
			if ok {
				msgType = "candidate"
			} else {
				msgType = "error"
			}
		}

		switch msgType {
		case "ping":
			wr.request()
		case "offer":
			wr.onOffer(data["sdp"].(string))
		case "candidate":
			wr.onRemoteCandidate(data["candidate"].(string))
		default:
			log.Println("Error: Unknown message type:", msgType)
		}
	}
}

func (wr *WebrtcReceiver) ping() {
	wr.websocketMut.Lock()
	defer wr.websocketMut.Unlock()
	message := Ping{ID: id, Type: "ping"}
	messageJSON, _ := json.Marshal(message)
	wr.wsConn.WriteMessage(websocket.TextMessage, messageJSON)
	log.Println(string(messageJSON))
}

func (wr *WebrtcReceiver) request() {
	wr.websocketMut.Lock()
	defer wr.websocketMut.Unlock()
	message := Request{ID: id, Type: "request"}
	messageJSON, _ := json.Marshal(message)
	wr.wsConn.WriteMessage(websocket.TextMessage, messageJSON)
	log.Println(messageJSON)
}

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
		case <-wr.shutdownChan:
			return
		case <-ticker.C:
			wr.reconnectMutex.Lock()
			elapsed := time.Since(wr.lastPacketTime).Seconds()
			wr.reconnectMutex.Unlock()

			if elapsed > (5 * time.Second).Seconds() {
				log.Println("No RTP packets received for 5 seconds, restarting peer connection")
				wr.relaunchPeer()
				return
			}
		}
	}
}

func (wr *WebrtcReceiver) relaunchPeer() {
	wr.reconnectMutex.Lock()
	defer wr.reconnectMutex.Unlock()

	if wr.isReconnecting {
		return
	}
	wr.isReconnecting = true
	wr.shutdownChan = make(chan struct{})
	if wr.pc != nil {
		wr.pc.Close()
		wr.pc = nil
	}

	go func() {
		defer func() {
			wr.reconnectMutex.Lock()
			wr.isReconnecting = false
			wr.reconnectMutex.Unlock()
		}()

		startTime := time.Now()
		timeout := time.After(10 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		<-wr.shutdownChan

		wr.shutdownChan = make(chan struct{})
		for {
			select {
			case <-ticker.C:
				log.Printf("Ms elapsed: %v", time.Since(startTime).Milliseconds())
				if (wr.pc != nil) && (wr.pc.ConnectionState() == webrtc.PeerConnectionStateConnected) {
					log.Println("PeerConnection reestablished")
				}
			case <-timeout:
				log.Println("Reconnection timeout, retrying PeerConnection")
				if wr.pc != nil {
					wr.pc.Close()
					<-wr.shutdownChan
				}

				wr.shutdownChan = make(chan struct{})
				timeout = time.After(10 * time.Second)
				wr.createPeerConnection()
				wr.ping()
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
			case <-wr.shutdownChan:
				wr.videoIsRunning = false
				return
			default:
				pkt, _, err := track.ReadRTP()
				if err != nil {
					wr.videoIsRunning = false
					log.Printf("ReadRTP error: %v", err)
					wr.relaunchPeer()
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
					//log.Println("write to udp")
					//os.Stdout.Sync()
					//runtime.Gosched()
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
			wr.wsConn.WriteMessage(websocket.TextMessage, candidateJSONData)
		})
	}

	wr.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State changed: %s", state)

	})

	wr.pc.OnICEGatheringStateChange(func(state webrtc.ICEGatheringState) {
		log.Printf("ICE Gathering State changed: %s\n", state)
		if state == webrtc.ICEGatheringStateComplete {
			wr.websocketMut.Lock()
			defer wr.websocketMut.Unlock()
			answerJSON, _ := json.Marshal(Answer{
				ID:   id,
				Type: "answer",
				SDP:  wr.pc.LocalDescription().SDP,
			})
			log.Printf("local desc is: %s", answerJSON)

			if err := wr.wsConn.WriteMessage(websocket.TextMessage, answerJSON); err != nil {
				log.Printf("Failed to send answer: %v", err)
			}
		}
	})
}

func (wr *WebrtcReceiver) createPeerConnection() {
	iceServers := []webrtc.ICEServer{
		{
			URLs: wr.iceConfigurator.StunServers,
		},
		{
			URLs:           []string{wr.iceConfigurator.TurnServers[0].URL, wr.iceConfigurator.TurnServers[1].URL},
			Username:       wr.iceConfigurator.TurnServers[0].Username,
			Credential:     wr.iceConfigurator.TurnServers[0].Credential,
			CredentialType: webrtc.ICECredentialTypePassword,
		},
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
		case webrtc.PeerConnectionStateClosed:
			close(wr.shutdownChan)
		case webrtc.PeerConnectionStateFailed:
			wr.relaunchPeer()
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

func (wr *WebrtcReceiver) WsIsOpen() bool {
	for wr.wsConn == nil {
		return false
	}
	err := wr.wsConn.SetWriteDeadline(time.Now().Add(20 * time.Second))
	if err != nil {
		log.Println("Error on set deadline:", err)
		return false
	}
	wr.websocketMut.Lock()
	defer wr.websocketMut.Unlock()
	err = wr.wsConn.WriteMessage(websocket.PingMessage, []byte{})
	if err != nil {
		log.Println("Error during writing a message:", err)
		return false
	}
	return true
}

func (wr *WebrtcReceiver) CreateDefaultPipeline(hostUrl string, login string, password string) {
	log.Println("create default pipeline")
	wr.Configure(hostUrl, login, password)
	wr.SetupOutputProtocol("UDP", "127.0.0.1", 9050)
	wr.SetupCodecs(nil)
	wr.Open()

}

func NewDefaultWebrtcReceiver(hostUrl string, login string, password string) *WebrtcReceiver {
	log.Println("new webrtc default")
	wr := &WebrtcReceiver{shutdownChan: make(chan struct{}), iceTrickleEnabled: false, websocketMut: sync.Mutex{}}
	wr.CreateDefaultPipeline(hostUrl, login, password)
	wr.SetTransmitEnabled(true)
	return wr
}

func (wr *WebrtcReceiver) Close() {
	if wr.pc != nil {
		wr.pc.Close()
	}
	if wr.wsConn != nil {
		wr.wsConn.Close()
	}
}
