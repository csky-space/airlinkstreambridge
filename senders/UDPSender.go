package senders

import (
	"AirlinkStreamBridge/events"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type UDPSender struct {
	socket     *net.UDPConn
	udpAddr    string
	udpNetAddr *net.UDPAddr
	setAddrMut sync.Mutex
	sendMut    sync.Mutex

	shouldReconnectEvent *events.EventBroadcaster
}

func (sender *UDPSender) SetupUDP(address string, port int) error {
	log.Printf("setup udp with %s:%d\n", address, port)
	if sender.socket != nil {
		sender.socket.Close()
	}

	sender.udpAddr = address + ":" + strconv.Itoa(port)

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

	log.Printf("end of setup udp with %s:%d\n", address, port)
	return nil
}

func (sender *UDPSender) Send(data []byte) error {
	sender.sendMut.Lock()
	defer sender.sendMut.Unlock()
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
			sender.shouldReconnectEvent.Fire()
		}
	}()
	if sender.socket != nil {
		_, err := sender.socket.Write(data)
		if err != nil {
			sender.shouldReconnectEvent.Fire()
			log.Printf("raw %v didn't write with error: %s", data, err)
			return err
		}
	} else {
		log.Println("udp socket is nil")
	}

	return nil
}

func (sender *UDPSender) Close() {
	sender.setAddrMut.Lock()
	defer sender.setAddrMut.Unlock()
	if sender.socket != nil {
		sender.socket.Close()
	}
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

			sender.setAddrMut.Lock()
			log.Println("Reconnecting UDP...")

			if sender.socket != nil {
				sender.socket.Close()
			}

			var err error
			sender.socket, err = net.DialUDP("udp", nil, sender.udpNetAddr)
			sender.setAddrMut.Unlock()

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

func NewUDPSender(address string, port int) (ISender, error) {
	if port == 0 {
		port = 9050
	}
	if address == "" {
		address = "127.0.0.1"
	}
	sender := &UDPSender{shouldReconnectEvent: events.NewEventBroadcaster()}
	err := sender.SetupUDP(address, port)
	go sender.udpWatchdog()
	return sender, err
}

func (sender *UDPSender) Relaunch() {
	sender.shouldReconnectEvent.Fire()
	//sender.SetupUDP(sender.udpAddr, sender.udpNetAddr.Port)
}

func (sender *UDPSender) SetSocket(conn *net.UDPConn) {
	sender.setAddrMut.Lock()
	defer sender.setAddrMut.Unlock()
	if sender.socket != nil {
		_ = sender.socket.Close()
	}
	sender.socket = conn
}
