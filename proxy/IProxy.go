package proxy

import "AirlinkStreamBridge/object"

type IProxy interface {
	object.IObject
	Activate() error
	Deactivate() error
}
