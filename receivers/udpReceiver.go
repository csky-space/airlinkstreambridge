package receivers

import (
	"AirlinkStreamBridge/events"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
)

type udpReceiver struct {
	socket     *net.UDPConn
	udpAddr    string
	udpNetAddr *net.UDPAddr
	setAddrMut sync.Mutex
	sendMut    sync.Mutex

	shouldReconnectEvent *events.EventBroadcaster

	onData func(data []byte) error
}

func NewUDPReceiver() (*udpReceiver, error) {
	receiver := &udpReceiver{}

	return receiver, nil
}

func (receiver *udpReceiver) SetupUDP(address string, port int) error {
	log.Printf("setup udp with %s:%d\n", address, port)
	if receiver.socket != nil {
		receiver.socket.Close()
	}

	receiver.udpAddr = address + ":" + strconv.Itoa(port)

	var err error
	//wr.udpSender, err = net.ListenPacket("udp", ":0")
	//if err != nil {
	//	log.Fatalf("ListenPacket error: %v", err)
	//}

	receiver.udpNetAddr, err = net.ResolveUDPAddr("udp", receiver.udpAddr)
	if err != nil {
		return err
	}

	receiver.socket, err = net.ListenUDP("udp", receiver.udpNetAddr)
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
			if receiver.onData != nil {
				err = receiver.onData(buf[:n])
				if err != nil {
					log.Printf("onData error: %v", err)
				}
			}
		}
	}()

	log.Printf("end of setup udp with %s:%d\n", address, port)
	return nil
}

func (rec *udpReceiver) SetOnData(onData func(data []byte) error) {
	rec.onData = onData
}
