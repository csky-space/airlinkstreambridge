package receivers

import (
	"bytes"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

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

type Webrtc_websocket struct {
	readMutex  sync.Mutex
	writeMutex sync.Mutex
	wsConn     *websocket.Conn
	wsUrl      string

	ClosedExpectedly   chan struct{}
	ClosedUnexpectedly chan struct{}
	Pinged             chan struct{}
	Requested          chan struct{}
	Offered            chan struct{}
	Candidate          chan struct{}

	onPing            func()
	onOffer           func(sdp string) error
	onRemoteCandidate func(candidate string)

	isConnected bool
}

func NewWebrtcWebsocket(wsUrl string) *Webrtc_websocket {
	ws := &Webrtc_websocket{wsUrl: wsUrl,
		wsConn:     nil,
		readMutex:  sync.Mutex{},
		writeMutex: sync.Mutex{},
		onPing:     func() {}, onOffer: func(sdp string) error { return nil }, onRemoteCandidate: func(candidate string) {},
		ClosedExpectedly:   make(chan struct{}, 1),
		ClosedUnexpectedly: make(chan struct{}, 1),
		Pinged:             make(chan struct{}, 1),
		Requested:          make(chan struct{}, 1),
		Offered:            make(chan struct{}, 1),
		Candidate:          make(chan struct{}, 1),
	}
	ws.establishWs()
	go ws.connectionWatchdog()
	return ws
}

func (ws *Webrtc_websocket) establishWs() {
	if ws.wsConn != nil {
		ws.wsConn.Close()
		<-ws.ClosedExpectedly
	}
	ws.open()
	ws.wsConn.SetCloseHandler(func(code int, text string) error {
		ws.isConnected = false
		switch code {
		case websocket.CloseNormalClosure:
			select {
			case ws.ClosedExpectedly <- struct{}{}:
			default:
			}
		default:
			select {
			case ws.ClosedUnexpectedly <- struct{}{}:
			default:
			}
		}

		return nil
	})
}

func (ws *Webrtc_websocket) readMessages() {
	ws.wsConn.SetPingHandler(func(msg string) error {
		err := ws.wsConn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(time.Second))
		return err
	})
	for {
		_, message, err := ws.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			select {
			case ws.ClosedUnexpectedly <- struct{}{}:
			default:
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
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
			select {
			case ws.Pinged <- struct{}{}:
			default:
			}
			ws.onPing()
		case "offer":
			select {
			case ws.Offered <- struct{}{}:
			default:
			}
			ws.onOffer(data["sdp"].(string))
		case "candidate":

			select {
			case ws.Candidate <- struct{}{}:
			default:
			}
			ws.onRemoteCandidate(data["candidate"].(string))
		default:
			log.Println("Error: Unknown message type:", msgType)
		}
	}
}

func (ws *Webrtc_websocket) ping() {
	//wr.websocketMut.Lock()
	//defer wr.websocketMut.Unlock()
	message := Ping{ID: id, Type: "ping"}
	messageJSON, _ := json.Marshal(message)
	ws.WriteMessage(messageJSON)
	log.Println(string(messageJSON))
}

func (ws *Webrtc_websocket) request() {
	message := Request{ID: id, Type: "request"}
	messageJSON, _ := json.Marshal(message)
	ws.WriteMessage(messageJSON)
	log.Println(messageJSON)
}

func (ws *Webrtc_websocket) IsOpen() bool {
	return ws.isConnected
}

func (ws *Webrtc_websocket) Close() {
	if ws.wsConn != nil {
		ws.wsConn.Close()
		<-ws.ClosedExpectedly
	}
}

func (ws *Webrtc_websocket) WriteMessage(message []byte) {
	ws.writeMutex.Lock()
	defer ws.writeMutex.Unlock()
	err := ws.wsConn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		log.Printf("Websocket message write problem: %v", err)
		return
	}
}

func (ws *Webrtc_websocket) startSignalling() {
	ws.request()
}

func (ws *Webrtc_websocket) connectionWatchdog() {
	for {
		select {
		case <-ws.ClosedUnexpectedly:
			ws.establishWs()
		case <-ws.ClosedExpectedly:
			ws.open()
		}
	}
}

func (ws *Webrtc_websocket) open() {
	wsConn, _, err := websocket.DefaultDialer.Dial(ws.wsUrl, nil)
	if err == nil {
		ws.isConnected = true
		ws.wsConn = wsConn
		go ws.readMessages()
		return
	}
}

func (ws *Webrtc_websocket) SetOnPing(onPing func()) {
	ws.onPing = onPing
}

func (ws *Webrtc_websocket) SetOnOffer(onOffer func(sdp string) error) {
	ws.onOffer = onOffer
}

func (ws *Webrtc_websocket) SetOnRemoteCandidate(onRemoteCandidate func(candidate string)) {
	ws.onRemoteCandidate = onRemoteCandidate
}

func (ws *Webrtc_websocket) SingleShotOnPing(onPing func()) {
	onPingOriginal := ws.onPing
	ws.onPing = onPing
	timeout := time.After(3 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
		case <-timeout:
			ws.onPing = onPingOriginal
			return
		case <-ws.Pinged:
			ws.onPing = onPingOriginal
			return
		}
	}
}
