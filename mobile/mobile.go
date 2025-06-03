package mobile

import (
	httpserver "AirlinkStreamBridge/http_server"
	"AirlinkStreamBridge/logserver"
	"log"
)

type ServerController struct {
	server *httpserver.Http_server
}

func NewServerController() *ServerController {
	return &ServerController{}
}

func (sc *ServerController) Start() {
	bwriter := logserver.NewBroadcastWriter()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(bwriter)

	go bwriter.StartTCPServer()

	sc.Close()
	sc.server = httpserver.NewHttpServer()
	go sc.server.Loop()
}

func (sc *ServerController) Close() {
	if sc.server != nil {
		sc.server.CloseHandle(nil, nil)
		sc.server.CloseApp()
	}
}
