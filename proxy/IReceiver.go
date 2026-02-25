package proxy

import "AirlinkStreamBridge/object"

type IReceiver interface {
	IProxy

	SubscribeOnData(sub object.IObject, onData func(data []byte) error)
	UnsubscribeOnData(sub object.IObject)
	UnsubscribeAll()
	GetDevice() any
}
