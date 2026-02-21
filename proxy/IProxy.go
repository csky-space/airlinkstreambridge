package proxy

type IProxy interface {
	GetName() string
	Activate() error
	Deactivate() error
}
