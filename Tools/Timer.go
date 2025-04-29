package tools

import (
	"sync"
	"time"
)

type ElapsedTimer struct {
	startTime time.Time
	ticker    *time.Ticker
	quit      chan struct{}
	mu        sync.Mutex
	stopped   bool
}

func NewElapsedTimer(interval time.Duration) *ElapsedTimer {
	return &ElapsedTimer{
		ticker: time.NewTicker(interval),
		quit:   make(chan struct{}),
	}
}

func (et *ElapsedTimer) Start() {
	et.mu.Lock()
	defer et.mu.Unlock()

	if !et.startTime.IsZero() {
		return
	}

	et.startTime = time.Now()
}

func (et *ElapsedTimer) Elapsed() time.Duration {
	et.mu.Lock()
	defer et.mu.Unlock()

	if et.startTime.IsZero() {
		return 0
	}
	return time.Since(et.startTime)
}

func (et *ElapsedTimer) Stop() time.Duration {
	et.mu.Lock()
	defer et.mu.Unlock()

	if et.stopped {
		return time.Since(et.startTime)
	}
	et.stopped = true
	close(et.quit)
	return time.Since(et.startTime)
}
