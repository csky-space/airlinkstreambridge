package receivers

import "sync"

type EventBroadcaster struct {
	subscribers []chan struct{}
	mu          sync.Mutex
}

func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{}
}

func (eb *EventBroadcaster) Subscribe() <-chan struct{} {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	ch := make(chan struct{}, 1)
	eb.subscribers = append(eb.subscribers, ch)
	return ch
}

func (eb *EventBroadcaster) Fire() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	for _, ch := range eb.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
