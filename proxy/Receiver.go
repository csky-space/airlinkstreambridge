package proxy

import "AirlinkStreamBridge/object"

type Receiver struct {
	name   string
	device any

	onData map[string]func(data []byte) error
}

func (r *Receiver) SubscribeOnData(sub object.IObject, onData func(data []byte) error) {

	r.onData[sub.GetName()] = onData
}

func (r *Receiver) UnsubscribeOnData(sub object.IObject) {
	delete(r.onData, sub.GetName())
}

func (r *Receiver) UnsubscribeAll() {
	for key, _ := range r.onData {
		delete(r.onData, key)
	}
}
