package main

import (
	httpserver "AirlinkStreamBridge/http_server"
	"AirlinkStreamBridge/logserver"
	"log"
)

// import "AirlinkStreamBridge/receivers"
func main() {
	bwriter := logserver.NewBroadcastWriter()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	log.SetOutput(bwriter)

	go bwriter.StartTCPServer()

	server := httpserver.NewHttpServer()
	server.Loop()
}
