package main

//import "AirlinkStreamBridge/receivers"
import (
	httpserver "AirlinkStreamBridge/http_server"
	"log"
	"os"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	f := os.Stdout
	if f != nil {
		f.Sync()
	}
	server := httpserver.NewHttpServer()
	server.Loop()
	//recv := receivers.NewDefaultWebrtcReceiver("stage.air-link.space", "001D0", "HM9-qc5-Ddq-mgy")
	//for recv.WsIsOpen() {
	//
	//}
}
