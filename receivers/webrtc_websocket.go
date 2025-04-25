package receivers

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net"
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

	ClosedExpectedly   *EventBroadcaster
	ClosedUnexpectedly *EventBroadcaster
	Pinged             *EventBroadcaster
	Requested          *EventBroadcaster
	Offered            *EventBroadcaster
	Candidate          *EventBroadcaster

	onPing            func()
	onOffer           func(sdp string) error
	onRemoteCandidate func(candidate string)

	isConnected bool

	customResolver *net.Resolver
	customDialer   websocket.Dialer
}

func NewWebrtcWebsocket(wsUrl string) (*Webrtc_websocket, error) {
	ws := &Webrtc_websocket{wsUrl: wsUrl,
		wsConn:     nil,
		readMutex:  sync.Mutex{},
		writeMutex: sync.Mutex{},
		onPing:     func() {}, onOffer: func(sdp string) error { return nil }, onRemoteCandidate: func(candidate string) {},
		ClosedExpectedly:   NewEventBroadcaster(),
		ClosedUnexpectedly: NewEventBroadcaster(),
		Pinged:             NewEventBroadcaster(),
		Requested:          NewEventBroadcaster(),
		Offered:            NewEventBroadcaster(),
		Candidate:          NewEventBroadcaster(),
	}

	ws.customResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := &net.Dialer{Timeout: time.Second * 2}
			return d.DialContext(ctx, network, "8.8.8.8:53")
		},
	}

	ws.customDialer = websocket.Dialer{
		NetDialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
			Resolver:  ws.customResolver,
		}).DialContext,
	}
	err := ws.establishWs()
	if err != nil {
		return nil, err
	}
	go ws.connectionWatchdog()
	return ws, nil
}

func (ws *Webrtc_websocket) establishWs() error {
	if ws.wsConn != nil {
		ws.wsConn.Close()
		<-ws.ClosedExpectedly.Subscribe()
	}
	err := ws.open()
	if err != nil {
		return err
	}
	ws.wsConn.SetCloseHandler(func(code int, text string) error {
		ws.isConnected = false
		switch code {
		case websocket.CloseNormalClosure:
			ws.ClosedExpectedly.Fire()
		default:
			ws.ClosedUnexpectedly.Fire()
		}

		return nil
	})
	return nil
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
			ws.wsConn.Close()
			return
			//ws.ClosedUnexpectedly.Fire()
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
			ws.Pinged.Fire()
			ws.onPing()
		case "offer":
			ws.Offered.Fire()
			ws.onOffer(data["sdp"].(string))
		case "candidate":
			ws.Candidate.Fire()
			ws.onRemoteCandidate(data["candidate"].(string))
		default:
			log.Println("Error: Unknown message type:", msgType)
		}
	}
}

func (ws *Webrtc_websocket) ping() error {
	//wr.websocketMut.Lock()
	//defer wr.websocketMut.Unlock()
	message := Ping{ID: id, Type: "ping"}
	messageJSON, _ := json.Marshal(message)
	err := ws.WriteMessage(messageJSON)
	if err != nil {
		return err
	}
	log.Println(string(messageJSON))
	return nil
}

func (ws *Webrtc_websocket) request() error {
	message := Request{ID: id, Type: "request"}
	messageJSON, _ := json.Marshal(message)
	err := ws.WriteMessage(messageJSON)
	if err != nil {
		return err
	}
	log.Println(messageJSON)
	return nil
}

func (ws *Webrtc_websocket) IsOpen() bool {
	return ws.isConnected
}

func (ws *Webrtc_websocket) Close() {
	if ws.wsConn != nil {
		ws.wsConn.Close()
		<-ws.ClosedExpectedly.Subscribe()
	}
}

func (ws *Webrtc_websocket) WriteMessage(message []byte) error {
	ws.writeMutex.Lock()
	defer ws.writeMutex.Unlock()
	err := ws.wsConn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		log.Printf("Websocket message write problem: %v", err)
		return err
	}
	return nil
}

func (ws *Webrtc_websocket) startSignalling() error {
	return ws.request()
}

func (ws *Webrtc_websocket) connectionWatchdog() {
	unexpectedlySubscriber := ws.ClosedUnexpectedly.Subscribe()
	expectedlySubscriber := ws.ClosedExpectedly.Subscribe()
	for {
		select {
		case <-unexpectedlySubscriber:
			err := ws.establishWs()
			if err != nil {
				log.Printf("Establishing websoket connection failed with error: %v", err)
			}
		case <-expectedlySubscriber:
			err := ws.open()
			if err != nil {
				log.Printf("websocket open error: %v", err)
			}
		}
	}
}

func (ws *Webrtc_websocket) open() error {
	wsConn, _, err := ws.customDialer.Dial(ws.wsUrl, nil)
	if err != nil {
		return err
	}
	if wsConn == nil {
		log.Println("wsConn is nil")
	}
	ws.isConnected = true
	ws.wsConn = wsConn
	go ws.readMessages()
	return nil
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
		case <-ws.Pinged.Subscribe():
			ws.onPing = onPingOriginal
			return
		}
	}
}
