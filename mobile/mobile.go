package mobile

import (
	httpserver "AirlinkStreamBridge/http_server"
	"AirlinkStreamBridge/logserver"
	"io"
	"log"
	"net"
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

	go startTCPServer(bwriter)

	sc.Close()
	sc.server = httpserver.NewHttpServer()
	go sc.server.Loop()
}

func (sc *ServerController) Close() {
	if sc.server != nil {
		sc.server.CloseApp()
	}
}

func startTCPServer(bw *logserver.BroadcastWriter) {
	address := "0.0.0.0:9000"
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Не удалось запустить TCP-сервер: %v\n", err)
	}
	log.Printf("TCP-сервер запущен на %s\n", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Ошибка при принятии соединения: %v\n", err)
			continue
		}
		log.Printf("Клиент подключился: %s\n", conn.RemoteAddr())

		bw.AddClient(conn)

		go handleClient(conn, bw)
	}
}

func handleClient(conn net.Conn, bw *logserver.BroadcastWriter) {
	defer func() {
		log.Printf("Клиент отключился: %s\n", conn.RemoteAddr())
		bw.RemoveClient(conn)
	}()

	buf := make([]byte, 1)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Ошибка чтения от %s: %v\n", conn.RemoteAddr(), err)
			}
			return
		}
	}
}
