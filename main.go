package main

import (
	"AirlinkStreamBridge/proxy"
	"log"
	"time"
)

// import "AirlinkStreamBridge/receivers"
func main() {
	_proxy := proxy.NewProxy()
	if _proxy == nil {
		log.Println("error: _proxy not created!")
	}
	astra, err := proxy.NewAstra("Astra1", "s.zamaro@csky.space", "HM9-qc5-Ddq-mgy", "Air-link")
	if err != nil {
		panic(err)
	}
	astra.Activate()

	var senderToGs *proxy.UDPSender
	senderToGs, err = proxy.NewUDPSender("senderToGs", "127.0.0.1:14550")
	if err != nil {
		log.Println("error: senderToGs not created!")
		return
	}
	var receiverFromGS *proxy.UDPReceiver
	receiverFromGS, err = proxy.NewUDPReceiver("receiverFromGS", "127.0.0.1:14550")
	if err != nil {
		log.Println("error: receiverFromGS creation failed! " + err.Error())
	}
	receiverFromGS.Activate()
	if err != nil {
		log.Println("error: receiverFromGS Activate failed! " + err.Error())
		return
	}
	senderToGs.SetDevice(receiverFromGS.GetDevice())
	//receiverFromGS.SetOnData(func(data []byte) error {
	//	_proxy.Broadcast(data)
	//	return nil
	//})
	//_proxy.AddOutput(astra)
	_proxy.AddInput(astra)
	_proxy.AddOutput(senderToGs)
	_proxy.AddInput(receiverFromGS)
	_proxy.AddOutput(astra)
	_proxy.Assign(astra.GetName(), senderToGs.GetName())
	_proxy.Assign(receiverFromGS.GetName(), astra.GetName())

	for {
		time.Sleep(10 * time.Millisecond)
	}

	//bwriter := logserver.NewBroadcastWriter()
	//
	//log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	//
	//log.SetOutput(bwriter)
	//
	//go bwriter.StartTCPServer()
	//
	//server := httpserver.NewHttpServer()
	//server.Loop()
}
