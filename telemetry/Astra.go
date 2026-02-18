package telemetry

import (
	"AirlinkStreamBridge/receivers"
	"AirlinkStreamBridge/senders"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Enem-20/mavgoink/message"
	"github.com/Enem-20/mavgoink/system"
)

type Astra struct {
	isAuthorized         bool
	sysId                uint8
	compId               uint8
	seq                  uint8
	maxMavlinkPacketSize int
	hosts                map[string]string
	currentHost          string
	receiver             receivers.IReceiver
	sender               senders.ISender
	mavSystem            *system.System
}

type Modem struct {
	Name     string `json:"name"`
	IsOnline bool   `json:"isOnline"`
	IMEI     string `json:"imei"`
	Version  int    `json:"version"`
}

type Response struct {
	Modems []Modem `json:"modems"`
}

func NewAstra(login string, password string, modemType string, receiver receivers.IReceiver, sender senders.ISender) (*Astra, error) {
	astra := &Astra{
		sysId:                uint8(255),
		compId:               uint8(1),
		seq:                  uint8(0),
		maxMavlinkPacketSize: 280,
		hosts: map[string]string{
			"Astra":    "astra.csky.space",
			"Air-link": "air-link.space",
		},
		receiver:  receiver,
		sender:    sender,
		mavSystem: system.NewSystem(system.MAVLINK_VERSION_2, 255, "Default"),
	}

	astra.currentHost = astra.hosts[modemType]

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

func (astra *Astra) SendAuth(modem string, password string) {
	payload := [payloadSize]byte{}
	msg := astra.mavSystem.CreateMessage(1, 52000, byte(payloadSize))

	copy(payload[0:len(modem)], []byte(modem))
	copy(payload[50:50+len(password)], []byte(password))
	msg.PushBytes(payload[:])
	msg.PushByte(13)
	log.Printf("Sending auth packet %#X", msg.GetRawMessage())
	err := astra.sender.Send(msg.GetRawMessage())
	if err != nil {
		log.Printf("Failed to send auth packet: %v", err)
	}
}

func (astra *Astra) StartTelemetry(modem string, password string) {
	var senderToGs *senders.UDPSender
	var err error
	senderToGs, err = senders.NewUDPSender("127.0.0.1", 14550)
	if err != nil {
		log.Println("error: senderToGs not created!")
		return
	}
	astra.receiver.SetOnData(func(data []byte) error {
		msg := message.NewMessage()
		msg.ParseFromBytes(data)
		if msg.Payload.IsFull() {
			if msg.Header.GetMsgID() == 52001 {
				switch msg.Payload.GetByte(0) {
				case 1:
					log.Println("Authorized!")
					astra.isAuthorized = true
				default:
					log.Println("Authorization failed!")
					astra.isAuthorized = false
				}
			} else {
				senderToGs.Send(data)
			}
		}
		return nil
	})

	for !astra.isAuthorized {
		astra.SendAuth(modem, password)
		time.Sleep(5 * time.Second)
	}
	for {
		time.Sleep(10 * time.Millisecond)
	}
}

func (astra *Astra) SetOnTelemetry(onData func(data []byte) error) {
	astra.receiver.SetOnData(onData)
}
