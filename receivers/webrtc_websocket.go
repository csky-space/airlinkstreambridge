package receivers

import (
	"AirlinkStreamBridge/events"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
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

	ClosedExpectedly   *events.EventBroadcaster
	ClosedUnexpectedly *events.EventBroadcaster
	Pinged             *events.EventBroadcaster
	Requested          *events.EventBroadcaster
	Offered            *events.EventBroadcaster
	Candidate          *events.EventBroadcaster
	CloseRequester     *events.EventBroadcaster

	onPing            func()
	onOffer           func(sdp string) error
	onRemoteCandidate func(candidate string)

	isConnected bool

	customResolver *net.Resolver
	customDialer   websocket.Dialer
}

func NewWebrtcWebsocket(wsUrl string) (*Webrtc_websocket, error) {
	ws := &Webrtc_websocket{wsUrl: wsUrl,
		isConnected: false,
		wsConn:      nil,
		readMutex:   sync.Mutex{},
		writeMutex:  sync.Mutex{},
		onPing:      func() {}, onOffer: func(sdp string) error { return nil }, onRemoteCandidate: func(candidate string) {},
		ClosedExpectedly:   events.NewEventBroadcaster(),
		ClosedUnexpectedly: events.NewEventBroadcaster(),
		Pinged:             events.NewEventBroadcaster(),
		Requested:          events.NewEventBroadcaster(),
		Offered:            events.NewEventBroadcaster(),
		Candidate:          events.NewEventBroadcaster(),
		CloseRequester:     events.NewEventBroadcaster(),
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
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	err := ws.establishWs()
	if err != nil {
		return nil, err
	}
	go ws.connectionWatchdog()
	return ws, nil
}

func (ws *Webrtc_websocket) establishWs() error {
	log.Println("Enter establish ws")
	if ws.isConnected == true {
		ws.Close()
		log.Println("Wait for close")
	}
	err := ws.open()
	if err != nil {
		log.Println("Failed to open websocket: " + err.Error())
		return err
	}
	ws.wsConn.SetCloseHandler(func(code int, text string) error {
		ws.isConnected = false
		switch code {
		case websocket.CloseAbnormalClosure:
			log.Println("websocket unexpectedly closed! " + text)
			ws.ClosedUnexpectedly.Fire()
		case websocket.CloseInternalServerErr:
			log.Println("websocket closed due to: internal server Err! " + text)
			ws.ClosedUnexpectedly.Fire()
		case websocket.CloseTLSHandshake:
			log.Println("websocket closed due to: TLS handshake error! " + text)
			ws.ClosedUnexpectedly.Fire()
		case websocket.ClosePolicyViolation:
			log.Println("websocket closed due to: policy violation! " + text)
			ws.ClosedUnexpectedly.Fire()
		case websocket.CloseProtocolError:
			log.Println("websocket closed due to: protocol error! " + text)
			ws.ClosedUnexpectedly.Fire()
		case websocket.CloseUnsupportedData:
			log.Println("websocket closed due to: unsupproted data! " + text)
			ws.ClosedUnexpectedly.Fire()
		case websocket.CloseGoingAway:
			log.Println("websocket closed due to: going away! " + text)
			ws.ClosedExpectedly.Fire()
		case websocket.CloseNormalClosure:
			log.Println("websocket expectedly closed! " + text)
			ws.ClosedExpectedly.Fire()
		default:
			log.Println("websocket unexpectedly closed! " + text)
			ws.ClosedUnexpectedly.Fire()
		}

		return nil
	})
	if ws.wsConn == nil {
		log.Println("Failed to establish websocket: wsConn is nil")
		return errors.New("Failed to establish websocket: wsConn is nil")
	}

	log.Println("websocket successfully created!")
	return nil
}

func (ws *Webrtc_websocket) readMessages() {
	log.Println("Start reading websocket!")
	ws.wsConn.SetPingHandler(func(msg string) error {
		err := ws.wsConn.WriteControl(websocket.PongMessage, []byte(msg), time.Now().Add(time.Second))
		return err
	})

	CloseRequestSub := ws.CloseRequester.Subscribe()
	defer ws.CloseRequester.Unsubscribe(CloseRequestSub)
	for {
		select {
		case <-CloseRequestSub:
			log.Println("Websocket readMessages received close request")
			return
		default:
		}
		_, message, err := ws.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			ws.Close()
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
		time.Sleep(10 * time.Millisecond)
	}
}

func (ws *Webrtc_websocket) ping() error {
	//wr.websocketMut.Lock()
	//defer wr.websocketMut.Unlock()
	message := Ping{ID: id, Type: "ping"}
	messageJSON, _ := json.Marshal(message)
	err := ws.WriteMessage(messageJSON)
	if err != nil {
		log.Println("Error on send ping: " + err.Error())
		return err
	}
	log.Println("We sent ping: " + string(messageJSON))
	return nil
}

func (ws *Webrtc_websocket) request() error {
	message := Request{ID: id, Type: "request"}
	messageJSON, _ := json.Marshal(message)
	log.Println("We sent request: " + string(messageJSON))
	err := ws.WriteMessage(messageJSON)
	if err != nil {
		log.Println("Error on send request: " + err.Error())
		return err
	}

	return nil
}

func (ws *Webrtc_websocket) IsOpen() bool {
	return ws.isConnected
}

func (ws *Webrtc_websocket) Close() {
	if ws != nil {
		closeSub := ws.CloseRequester.Subscribe()
		defer ws.CloseRequester.Unsubscribe(closeSub)
		ws.CloseRequester.Fire()
		ws.isConnected = false
		if ws.wsConn != nil {
			ws.wsConn.Close()
		}
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

	defer func() {
		ws.ClosedUnexpectedly.Unsubscribe(unexpectedlySubscriber)
		ws.ClosedExpectedly.Unsubscribe(expectedlySubscriber)
	}()
	for {
		select {
		case <-unexpectedlySubscriber:
			log.Println("unexpectedly close handler")
			//err := ws.establishWs()
			//if err != nil {
			//	log.Printf("Establishing websoket connection failed with error: %v", err)
			//}
		case <-expectedlySubscriber:
			log.Println("expectedly close handler")
			err := ws.open()
			if err != nil {
				log.Printf("websocket open error: %v", err)
			}
		}
	}
}

func (ws *Webrtc_websocket) open() error {
	//requestHeader := http.Header{}
	//requestHeader.Set("User-Agent", "Mozilla/5.0 (Go-Client)")
	//requestHeader.Set("Origin", "https://stage.air-link.pace")
	wsConn, _, err := ws.customDialer.Dial(ws.wsUrl, nil)
	if err != nil {
		log.Println("Failed to Dial: " + err.Error())
		return err
	}
	if wsConn == nil {
		log.Println("Failed to Dial: wsConn is nil")
		return errors.New("wsConn is nil")
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

	go func() {
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
	}()
}
