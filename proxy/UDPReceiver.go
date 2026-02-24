package proxy

import (
	"AirlinkStreamBridge/events"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
)

type UDPReceiver struct {
	Receiver
	socket     *net.UDPConn
	udpAddr    string
	udpNetAddr *net.UDPAddr
	setAddrMut sync.Mutex
	sendMut    sync.Mutex
	name       string

	shouldReconnectEvent *events.EventBroadcaster
}

func NewUDPReceiver(name string, address string) (*UDPReceiver, error) {
	receiver := &UDPReceiver{name: name, shouldReconnectEvent: events.NewEventBroadcaster(), udpAddr: address}
	receiver.onData = make(map[string]func(data []byte) error)
	//err := receiver.SetupUDP(address, port)
	//if err != nil {
	//	return nil, err
	//}j
	return receiver, nil
}

func (sender *UDPReceiver) Activate() error {
	r, err := regexp.Compile(`((?:[a-zA-Z0-9.-]+|\[[0-9a-fA-F:]+\])):(\d+)`)
	if err != nil {
		return err
	}
	match := r.FindStringSubmatch(sender.udpAddr)

	if match != nil {
		return sender.SetupUDP(match[0])
	} else {
		return nil
	}
}

func (sender *UDPReceiver) Deactivate() error {
	if sender.socket != nil {
		return sender.socket.Close()
	}
	return nil
}

func (receiver *UDPReceiver) SetupUDP(addressPort string) error {
	log.Printf("setup udp with %s\n", addressPort)
	if receiver.socket != nil {
		receiver.socket.Close()
	}

	receiver.udpAddr = addressPort

	var err error
	//wr.udpSender, err = net.ListenPacket("udp", ":0")
	//if err != nil {
	//	log.Fatalf("ListenPacket error: %v", err)
	//}

	receiver.udpNetAddr, err = net.ResolveUDPAddr("udp", receiver.udpAddr)
	if err != nil {
		return err
	}

	receiver.socket, err = net.DialUDP("udp", nil, receiver.udpNetAddr)
	if err != nil {
		//panic(err)
		return err
	}
	receiver.socket.SetReadBuffer(1 << 20)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("close udp on terminating!")
		receiver.socket.Close()
	}()

	go func() {
		buf := make([]byte, 1400)
		for {
			n, remoteAddr, err := receiver.socket.ReadFromUDP(buf)
			if err != nil {
				log.Printf("ReadFromUDP error: %v", err)
				continue
			}
			log.Printf("Received %d bytes from %s", n, remoteAddr)
			if len(receiver.onData) > 0 {
				for _, onData := range receiver.onData {
					if onData != nil {
						err = onData(buf[:n])
						if err != nil {
							log.Printf("onData error: %v", err)
						}
					}
				}

			}
		}
	}()

	log.Printf("end of setup udp with %s\n", receiver.udpAddr)
	return nil
}

func (receiver *UDPReceiver) GetDevice() any {
	return receiver.socket
}

func (receiver *UDPReceiver) GetName() string {
	return receiver.name
}
