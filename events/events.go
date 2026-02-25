package events

import "sync"

type EventBroadcaster struct {
	subscribers map[chan struct{}]struct{}
	mu          sync.Mutex
}

func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{
		subscribers: make(map[chan struct{}]struct{}),
	}
}

func (eb *EventBroadcaster) Subscribe() chan struct{} {
	ch := make(chan struct{}, 1)

	eb.mu.Lock()
	eb.subscribers[ch] = struct{}{}
	eb.mu.Unlock()

	return ch
}

func (eb *EventBroadcaster) Unsubscribe(ch chan struct{}) {
	eb.mu.Lock()
	if _, found := eb.subscribers[ch]; found {
		delete(eb.subscribers, ch)
		close(ch)
	}
	eb.mu.Unlock()
}

func (eb *EventBroadcaster) Fire() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for ch := range eb.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
