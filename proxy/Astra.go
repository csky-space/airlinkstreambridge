package proxy

import (
	"AirlinkStreamBridge/requests"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/Enem-20/mavgoink/message"
	"github.com/Enem-20/mavgoink/system"
)

const (
	MAVLINK_ASTRA_RESPONSE_MSG_ID             = 52001
	MAVLINK_ASTRA_REQUEST_MSG_CRC_EXTRA       = 13
	MAVLINK_ASTRA_REQUEST_LOGIN_FIELD_SIZE    = 50
	MAVLINK_ASTRA_REQUEST_PASSWORD_FIELD_SIZE = 50
	MAVLINK_ASTRA_REQUEST_PAYLOAD_SIZE        = MAVLINK_ASTRA_REQUEST_LOGIN_FIELD_SIZE + MAVLINK_ASTRA_REQUEST_PASSWORD_FIELD_SIZE
)

type Modem struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	IsOnline bool   `json:"isOnline"`
	IMEI     string `json:"imei"`
	Version  int    `json:"version"`
}

type AstraRegistry struct {
	outputTypes map[string]reflect.Type
	inputTypes  map[string]reflect.Type
}

func NewAstraRegistry() *AstraRegistry {
	registry := &AstraRegistry{
		outputTypes: make(map[string]reflect.Type),
		inputTypes:  make(map[string]reflect.Type),
	}
	registry.RegisterOutputType("UDP", reflect.TypeOf((*UDPSender)(nil)).Elem())
	registry.RegisterInputType("UDP", reflect.TypeOf((*UDPReceiver)(nil)).Elem())
	return registry
}

func (registry *AstraRegistry) RegisterOutputType(name string, typ reflect.Type) {
	registry.outputTypes[name] = typ
}

func (registry *AstraRegistry) RegisterInputType(name string, typ reflect.Type) {
	registry.inputTypes[name] = typ
}

func (registry *AstraRegistry) GetOutputType(name string) (reflect.Type, bool) {
	typ, exists := registry.outputTypes[name]
	return typ, exists
}

func (registry *AstraRegistry) GetInputType(name string) (reflect.Type, bool) {
	typ, exists := registry.inputTypes[name]
	return typ, exists
}

type Astra struct {
	Receiver
	ISender
	Modem
	Login        string
	Password     string
	isAuthorized bool
	sysId        uint8
	compId       uint8
	seq          uint8
	hosts        map[string]string
	currentHost  string
	receiver     IReceiver
	sender       ISender
	mavSystem    *system.System
	name         string
}

type Response struct {
	Modems []Modem `json:"modems"`
}

type CreateAstraRequest struct {
	Login     string                       `json:"login"`
	Password  string                       `json:"password"`
	ModemType string                       `json:"modemType"`
	Receiver  requests.CreateInputRequest  `json:"receiver"`
	Sender    requests.CreateOutputRequest `json:"sender"`
}

func NewAstra(name string, login string, password string, modemType string) (*Astra, error) {
	var hosts map[string]string
	hosts = map[string]string{
		"Astra":    "astra.csky.space",
		"Air-link": "air-link.space",
	}
	currentHost := hosts[modemType]
	sender, err := NewUDPSender("serverSender", currentHost+":10000")
	if err != nil {
		log.Println("error: sender not created!")
		return nil, err
	}

	receiver, err := NewUDPReceiver("serverReceiver", currentHost+":10000")
	if err != nil {
		log.Println("error: receiver not created!")
		return nil, err
	}
	err = receiver.Activate()
	if err != nil {
		log.Println("error: receiver Activate failed! " + err.Error())
		return nil, err
	}

	astra := &Astra{
		sysId:       uint8(255),
		compId:      uint8(1),
		seq:         uint8(0),
		hosts:       hosts,
		currentHost: currentHost,
		receiver:    receiver,
		sender:      sender,
		mavSystem:   system.NewSystem(system.MAVLINK_VERSION_2, 255, "Default"),
		Login:       login,
		Password:    password,
		name:        name,
	}
	astra.onData = make(map[string]func(data []byte) error)
	sender.SetDevice(receiver.GetDevice())

	return astra, nil
}

func (astra *Astra) getModems(login string, password string) ([]Modem, error) {
	data := []byte(`{"login":"` + login + `", "password":"` + password + `"}`)
	resp, err := http.Post("https://"+astra.currentHost+"/api/gs/getModems", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var jsonedBody Response
	body, err := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &jsonedBody)
	if err != nil {
		return nil, err
	}

	return jsonedBody.Modems, nil
}

const payloadSize = 100

func (astra *Astra) SendAuth() {
	payload := [payloadSize]byte{}
	msg := astra.mavSystem.CreateMessage(1, 52000, byte(payloadSize))

	copy(payload[0:len(astra.Login)], []byte(astra.Login))
	copy(payload[MAVLINK_ASTRA_REQUEST_LOGIN_FIELD_SIZE:MAVLINK_ASTRA_REQUEST_LOGIN_FIELD_SIZE+len(astra.Password)], []byte(astra.Password))
	msg.PushBytes(payload[:])
	msg.PushByte(MAVLINK_ASTRA_REQUEST_MSG_CRC_EXTRA)
	log.Printf("Sending auth packet %#X", msg.GetRawMessage())
	err := astra.sender.Send(msg.GetRawMessage())
	if err != nil {
		log.Printf("Failed to send auth packet: %v", err)
	}
}

func (astra *Astra) Activate() error {
	astra.receiver.SubscribeOnData(astra, func(data []byte) error {
		msg := message.NewMessage()
		msg.ParseFromBytes(data)
		if msg.Payload.IsFull() {
			if msg.Header.GetMsgID() == MAVLINK_ASTRA_RESPONSE_MSG_ID {
				switch msg.Payload.GetByte(0) {
				case 1:
					log.Println("Authorized!")
					astra.isAuthorized = true
				default:
					log.Println("Authorization failed!")
					astra.isAuthorized = false
				}
			} else {
				if len(astra.onData) > 0 {
					for _, onData := range astra.onData {
						onData(data)
					}
				}
			}
		}
		return nil
	})

	for !astra.isAuthorized {
		astra.SendAuth()
		time.Sleep(5 * time.Second)
	}
	return nil
}

func (astra *Astra) SetOnTelemetry(onData func(data []byte) error) {
	astra.receiver.SubscribeOnData(astra, onData)
}

func (astra *Astra) GetName() string {
	return astra.name
}

func (astra *Astra) Deactivate() error {
	return nil
}

func (astra *Astra) GetDevice() any {
	return astra.receiver.GetDevice()
}

func (astra *Astra) GetServerSender() ISender {
	return astra.sender
}

func (astra *Astra) Send(data []byte) error {
	if (astra != nil) && (astra.sender != nil) {
		return astra.sender.Send(data)
	}
	return errors.New("Astra or Astra.sender wasn't created")
}

func (astra *Astra) Close() {

}

func (astra *Astra) Relaunch() {

}

func (astra *Astra) SetDevice(device any) {

}
