package broker

import (
	"context"
	"log"
	"sync"
)

type Event interface {
	Name() string
}

type Publisher interface {
	Publish(ctx context.Context, evt Event) []error
}

type Handler func(ctx context.Context, evt Event) error

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func New() *Bus {
	return &Bus{handlers: make(map[string][]Handler)}
}

func (b *Bus) Subscribe(eventName string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], h)
}

func (b *Bus) Publish(ctx context.Context, evt Event) []error {
	b.mu.RLock()
	hs := append([]Handler(nil), b.handlers[evt.Name()]...)
	b.mu.RUnlock()

	var errs []error
	for i, h := range hs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("broker handler panic event=%s handler_index=%d panic=%v", evt.Name(), i, r)
					errs = append(errs, context.Canceled)
				}
			}()
			if err := h(ctx, evt); err != nil {
				log.Printf("broker handler error event=%s handler_index=%d error=%v", evt.Name(), i, err)
				errs = append(errs, err)
			}
		}()
	}
	return errs
}
