package logserver

import (
	"io"
	"log"
	"net"
	"os"
	"sync"
)

type BroadcastWriter struct {
	mu      sync.Mutex
	clients map[net.Conn]struct{}
}

func NewBroadcastWriter() *BroadcastWriter {
	return &BroadcastWriter{
		clients: make(map[net.Conn]struct{}),
	}
}

func (b *BroadcastWriter) AddClient(conn net.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.clients[conn] = struct{}{}
}

func (b *BroadcastWriter) RemoveClient(conn net.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, conn)
	conn.Close()
}

func (b *BroadcastWriter) Write(p []byte) (n int, err error) {
	n, err = os.Stdout.Write(p)
	if err != nil {
		return n, err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for conn := range b.clients {
		if _, e := conn.Write(p); e != nil {
			conn.Close()
			delete(b.clients, conn)
		}
	}
	return n, nil
}

func (bw *BroadcastWriter) StartTCPServer() {
	address := "0.0.0.0:9000"
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Unable to start log server: %v\n", err)
	}
	log.Printf("TCP-server run at %s\n", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error on accept: %v\n", err)
			continue
		}
		log.Printf("Connected client: %s\n", conn.RemoteAddr())

		bw.AddClient(conn)

		go bw.handleClient(conn)
	}
}

func (bw *BroadcastWriter) handleClient(conn net.Conn) {
	defer func() {
		log.Printf("Client disconnected: %s\n", conn.RemoteAddr())
		bw.RemoveClient(conn)
	}()

	buf := make([]byte, 1)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Read error %s: %v\n", conn.RemoteAddr(), err)
			}
			return
		}
	}
}
