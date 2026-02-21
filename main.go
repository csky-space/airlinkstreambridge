package main

import (
	httpserver "AirlinkStreamBridge/http_server"
	"AirlinkStreamBridge/logserver"
	"log"
)

// import "AirlinkStreamBridge/receivers"
func main() {
	//_proxy := proxy.NewProxy()
	//if _proxy == nil {
	//	log.Println("error: _proxy not created!")
	//}
	//astra, err := receivers.NewAstra("s.zamaro@csky.space", "HM9-qc5-Ddq-mgy", "Air-link")
	//if err != nil {
	//	panic(err)
	//}
	//astra.Activate()
	//
	//var senderToGs *UDP.UDPSender
	//senderToGs, err = UDP.NewUDPSender("senderToGs", "127.0.0.1:14550")
	//if err != nil {
	//	log.Println("error: senderToGs not created!")
	//	return
	//}
	//var receiverFromGS *UDP.UDPReceiver
	//receiverFromGS, err = UDP.NewUDPReceiver("receiverFromGS", "127.0.0.1:14550")
	//if err != nil {
	//	log.Println("error: receiverFromGS creation failed! " + err.Error())
	//}
	//receiverFromGS.Activate()
	//if err != nil {
	//	log.Println("error: receiverFromGS Activate failed! " + err.Error())
	//	return
	//}
	//senderToGs.SetDevice(receiverFromGS.GetDevice())
	////receiverFromGS.SetOnData(func(data []byte) error {
	////	_proxy.Broadcast(data)
	////	return nil
	////})
	////_proxy.AddOutput(astra)
	//_proxy.Assign(astra, senderToGs)
	//_proxy.Assign(receiverFromGS, astra)
	//
	//for {
	//	time.Sleep(10 * time.Millisecond)
	//}
	bwriter := logserver.NewBroadcastWriter()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	log.SetOutput(bwriter)

	go bwriter.StartTCPServer()

	server := httpserver.NewHttpServer()
	server.Loop()
}
