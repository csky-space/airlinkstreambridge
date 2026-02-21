package proxy

type IReceiver interface {
	IProxy

	SetOnData(func(data []byte) error)
	GetDevice() any
}
