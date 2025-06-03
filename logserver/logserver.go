package logserver

import (
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
