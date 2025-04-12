package receivers

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"myproject/receivers/iceconfigurator"
	"net"
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
	iceConfigurator *iceconfigurator.ICEConfigurator

	lastPacketTime      time.Time
	reconnectMutex      sync.Mutex
	transmitEnableMutex sync.Mutex
	isReconnecting      bool
	shutdownChan        chan struct{}
	transmitEnabled     bool
}

func (wr *WebrtcReceiver) SetTransmitEnabled(enabled bool) {
	wr.transmitEnableMutex.Lock()
	defer wr.transmitEnableMutex.Unlock()

	wr.transmitEnabled = enabled
}

func NewWebrtcReceiver(hostUrl string, login string, password string) *WebrtcReceiver {
	wr := &WebrtcReceiver{shutdownChan: make(chan struct{})}
	//wr.packetReceived = make(chan struct{}, 1)
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
	//s.SetFireOnTrackBeforeFirstRTP(true)

	h265Codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeH265,
			ClockRate: 90000,
			//SDPFmtpLine: "profile-level-id=42e01f;hevc-profile=1",
			RTCPFeedback: []webrtc.RTCPFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "goog-remb"},
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
				{Type: "goog-remb"},
				//{Type: "ccm", Parameter: "fir"},
			},
		}, PayloadType: 111,
	}

	if err := wr.mediaEngine.RegisterCodec(h265Codec, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}

	if err := wr.mediaEngine.RegisterCodec(opusCodec, webrtc.RTPCodecTypeAudio); err != nil {
		panic(err)
	}

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
		webrtc.WithSettingEngine(s),
		webrtc.WithMediaEngine(wr.mediaEngine),
		webrtc.WithInterceptorRegistry(interceptorRegistry),
	)
	wr.createPeerConnection()
	wr.establishWs()

	return wr
}

func (wr *WebrtcReceiver) establishWs() {
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
			fmt.Println("Error during readiFng a message:", err)
			break
		}
		fmt.Printf("Received: %s\n", message)
		var data map[string]interface{}
		if err := json.Unmarshal(message, &data); err != nil {
			fmt.Println("Error during jsonify:", err)
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
			fmt.Println("Error: Unknown message type:", msgType)
		}
	}
}

func (wr *WebrtcReceiver) ping() {
	message := Ping{ID: id, Type: "ping"}
	messageJSON, _ := json.Marshal(message)
	wr.wsConn.WriteMessage(websocket.TextMessage, messageJSON)
	fmt.Println(string(messageJSON))
}

func (wr *WebrtcReceiver) request() {
	message := Request{ID: id, Type: "request"}
	messageJSON, _ := json.Marshal(message)
	wr.wsConn.WriteMessage(websocket.TextMessage, messageJSON)
	fmt.Println(messageJSON)
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
	fmt.Printf("candidate: %s\n", candidate)
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
			elapsed := time.Since(wr.lastPacketTime)
			wr.reconnectMutex.Unlock()

			if elapsed > 5*time.Second {
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

	if wr.pc != nil {
		wr.pc.Close()
		wr.pc = nil
	}

	wr.shutdownChan = make(chan struct{})

	go func() {
		defer func() {
			wr.reconnectMutex.Lock()
			wr.isReconnecting = false
			wr.reconnectMutex.Unlock()
		}()

		timeout := time.After(10 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if wr.pc != nil {
					log.Println("PeerConnection reestablished")
					return
				}
				wr.createPeerConnection()
			case <-timeout:
				log.Println("Reconnection timeout, retrying PeerConnection")
				wr.createPeerConnection()
				return
			case <-wr.shutdownChan:
				return
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
				return
			default:
				pkt, _, err := track.ReadRTP()
				if err != nil {
					log.Printf("ReadRTP error: %v", err)
					wr.relaunchPeer()
					return
				}

				wr.updateLastPacketTime()

				raw, err := pkt.Marshal()
				if err != nil {
					if errors.Is(err, io.EOF) {
						return
					}
					log.Println("track marshal error:", err)
					continue
				}

				_, _ = wr.udpSender.Write(raw)
			}
		}
	}()
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
	//pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
	//	if candidate == nil {
	//		return
	//	}
	//	candidateInit := candidate.ToJSON()
	//	candidateJSON := Candidate{
	//		ID:        id,
	//		Type:      "candidate",
	//		Candidate: candidateInit.Candidate,
	//	}
	//	candidateJSONData, err := json.Marshal(candidateJSON)
	//	if err != nil {
	//		log.Printf("Candidate serialization problem: %v", err)
	//		return
	//	}
	//	fmt.Printf("Candidate: %s\n", string(candidateJSONData))
	//	wr.wsConn.WriteMessage(websocket.TextMessage, candidateJSONData)
	//})
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE Connection State changed: %s", state)
	})

	pc.OnICEGatheringStateChange(func(state webrtc.ICEGatheringState) {
		fmt.Printf("ICE Gathering State changed: %s\n", state)
		if state == webrtc.ICEGatheringStateComplete {
			answerJSON, _ := json.Marshal(Answer{
				ID:   id,
				Type: "answer",
				SDP:  pc.LocalDescription().SDP,
			})
			fmt.Printf("local desc is: %s", answerJSON)
			if err := wr.wsConn.WriteMessage(websocket.TextMessage, answerJSON); err != nil {
				log.Printf("Failed to send answer: %v", err)
			}
		}
	})

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		fmt.Printf("dc: %s", dc.Label())
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("dc message: %s", msg.Data)
		})
	})
	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:9070")
	if err != nil {
		panic(err)
	}
	wr.udpSender, err = net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		panic(err)
	}

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Printf("PeerConnection State changed: %s", state)
		if state == webrtc.PeerConnectionStateFailed {

		}
	})

	_, _err := pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	})
	if _err != nil {
		log.Printf("Failed to add transceiver: %v", err)
	}

	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Track received: %s (SSRC: %d)", track.Codec().MimeType, track.SSRC())
		wr.onTrack(track, receiver)
	})
}
func (wr *WebrtcReceiver) WsIsOpen() bool {
	for wr.wsConn == nil {
		time.Sleep(time.Millisecond * 100)
	}
	err := wr.wsConn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		fmt.Println("Error on set deadline:", err)
		return false
	}
	err = wr.wsConn.WriteMessage(websocket.PingMessage, []byte{})
	if err != nil {
		fmt.Println("Error during writing a message:", err)
		return false
	}
	return true
}

func (wr *WebrtcReceiver) Close() {
	close(wr.shutdownChan)
	if wr.pc != nil {
		wr.pc.Close()
	}
	if wr.wsConn != nil {
		wr.wsConn.Close()
	}
}
