package proxy

type ISender interface {
	IProxy

	Send(data []byte) error
	Close()
	Relaunch()
	SetDevice(device any)
}
