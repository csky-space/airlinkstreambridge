package senders

type ISender interface {
	Send(data []byte) error
	Close()
}
