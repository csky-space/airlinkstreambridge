package senders

import (
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
)

type UDPSender struct {
	socket     *net.UDPConn
	udpAddr    string
	udpNetAddr *net.UDPAddr
	setAddrMut sync.Mutex
	sendMut    sync.Mutex
}

func (sender *UDPSender) SetupUDP(address string, port int) error {
	log.Printf("setup udp with %s:%d\n", address, port)
	sender.setAddrMut.Lock()
	defer sender.setAddrMut.Unlock()
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
	_, err := sender.socket.Write(data)
	if err != nil {
		log.Printf("raw %v didn't write with error: %s", data, err)
		return err
	}
	return nil
}

func (sender *UDPSender) Close() {
	sender.socket.Close()
}

func NewUDPSender(address string, port int) (ISender, error) {
	if port == 0 {
		port = 9050
	}
	if address == "" {
		address = "127.0.0.1"
	}
	sender := &UDPSender{}
	err := sender.SetupUDP(address, port)
	return sender, err
}
