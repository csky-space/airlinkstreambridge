package UDP

import (
	"AirlinkStreamBridge/events"
	"AirlinkStreamBridge/proxy"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"
)

type UDPSender struct {
	proxy.ISender
	socket     *net.UDPConn
	udpAddr    string
	udpNetAddr *net.UDPAddr
	name       string

	shouldReconnectEvent *events.EventBroadcaster
}

func NewUDPSender(name string, address string) (*UDPSender, error) {
	sender := &UDPSender{name: name, shouldReconnectEvent: events.NewEventBroadcaster(), udpAddr: address}
	//err := sender.SetupUDP(address, port)
	//go sender.udpWatchdog()
	return sender, nil
}

func (sender *UDPSender) udpWatchdog() {
	reconnect := sender.shouldReconnectEvent.Subscribe()
	defer sender.shouldReconnectEvent.Unsubscribe(reconnect)

	for {
		select {
		case <-reconnect:
			log.Println("try reconnect")
			if sender == nil || sender.udpNetAddr == nil {
				log.Println("sender or udpNetAddr is nil, stopping watchdog")
				return
			}

			log.Println("Reconnecting UDP...")

			if sender.socket != nil {
				sender.socket.Close()
			}

			var err error
			sender.socket, err = net.DialUDP("udp", nil, sender.udpNetAddr)

			if err != nil {
				log.Printf("Reconnect failed: %v", err)
				time.Sleep(2 * time.Second)
				sender.shouldReconnectEvent.Fire()
			} else {
				sender.socket.SetWriteBuffer(1 << 20)
				log.Println("Reconnected successfully")
			}

		case <-time.After(30 * time.Second):
			if sender.socket != nil {
			}
		}
	}
}

func (sender *UDPSender) Activate() error {
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

func (sender *UDPSender) SetupUDP(address string) error {
	log.Printf("setup udp with %s\n", address)
	if sender.socket != nil {
		sender.socket.Close()
	}

	sender.udpAddr = address

	var err error
	//wr.udpSender, err = net.ListenPacket("udp", ":0")
	//if err != nil {
	//	log.Fatalf("ListenPacket error: %v", err)
	//}

	sender.udpNetAddr, err = net.ResolveUDPAddr("udp", sender.udpAddr)
	if err != nil {
		return err
	}

	sender.socket, err = net.DialUDP("udp", nil, sender.udpNetAddr)
	if err != nil {
		//panic(err)
		return err
	}
	sender.socket.SetWriteBuffer(1 << 20)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("close udp on terminating!")
		sender.socket.Close()
	}()

	log.Printf("end of setup udp with %s\n", address)
	return nil
}

func (sender *UDPSender) Relaunch() {
	sender.shouldReconnectEvent.Fire()
	//sender.SetupUDP(sender.udpAddr, sender.udpNetAddr.Port)
}

func (sender *UDPSender) SetDevice(device any) {
	switch d := device.(type) {
	case *net.UDPConn:
		sender.socket = d
	}
}

func (sender *UDPSender) GetName() string {
	return sender.name
}

func (sender *UDPSender) Send(data []byte) error {
	if sender.socket == nil {
		return nil
	}
	_, err := sender.socket.Write(data)
	return err
}
