package proxy

type Receiver struct {
	name   string
	device any

	onData []func(data []byte) error
}

func (r *Receiver) SubscribeOnData(onData func(data []byte) error) {
	r.onData = append(r.onData, onData)
}

func (r *Receiver) UnsubscribeOnData(onData func(data []byte) error) {
	for i, subscribed := range r.onData {
		if &subscribed == &onData {
			r.onData = append(r.onData[:i], r.onData[i+1:]...)
			return
		}
	}
}
