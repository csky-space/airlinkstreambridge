package main

import (
	"AirlinkStreamBridge/receivers"
	"AirlinkStreamBridge/senders"
	"AirlinkStreamBridge/telemetry"
	"log"
)

// import "AirlinkStreamBridge/receivers"
func main() {

	sender, err := senders.NewUDPSender("astra.csky.space", 10000)
	if err != nil {
		log.Println("error: sender not created!")
		return
	}
	receiver, err := receivers.FromUDPSender(sender)
	if err != nil {
		log.Println("error: receiver not created!")
		return
	}
	astra, err := telemetry.NewAstra("s.zamaro@csky.space", "HM9-qc5-Ddq-mgy", "Air-link", receiver, sender)

	astra.StartTelemetry("7A161", "HM9-qc5-Ddq-mgy")

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
