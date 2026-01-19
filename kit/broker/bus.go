package broker

import (
	"context"
	"hash/fnv"
	"log"
	"runtime"
	"sync"
	"time"
)

type Event interface {
	Name() string
}

type Publisher interface {
	Publish(ctx context.Context, evt Event) []error
}

type Handler func(ctx context.Context, evt Event) error

type BusConfig struct {
	ShardCount      int
	BufferPerShard  int
	RetryBackoff    time.Duration
	RetryBackoffMax time.Duration
	MaxAttempts     int
}

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler

	shards []chan delivery
	done   chan struct{}
	wg     sync.WaitGroup
	cfg    BusConfig
}

func New() *Bus {
	cfg := BusConfig{
		ShardCount:      runtime.GOMAXPROCS(0),
		BufferPerShard:  256,
		RetryBackoff:    25 * time.Millisecond,
		RetryBackoffMax: 2 * time.Second,
		MaxAttempts:     0,
	}
	if cfg.ShardCount < 1 {
		cfg.ShardCount = 1
	}
	return NewWithConfig(cfg)
}

func NewWithConfig(cfg BusConfig) *Bus {
	if cfg.ShardCount < 1 {
		cfg.ShardCount = 1
	}
	if cfg.BufferPerShard < 1 {
		cfg.BufferPerShard = 1
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 25 * time.Millisecond
	}
	if cfg.RetryBackoffMax <= 0 {
		cfg.RetryBackoffMax = 2 * time.Second
	}

	b := &Bus{
		handlers: make(map[string][]Handler),
		shards:   make([]chan delivery, cfg.ShardCount),
		done:     make(chan struct{}),
		cfg:      cfg,
	}
	for i := range b.shards {
		b.shards[i] = make(chan delivery, cfg.BufferPerShard)
		b.wg.Add(1)
		go b.worker(i)
	}
	return b
}

func (b *Bus) Subscribe(eventName string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], h)
}

func (b *Bus) Close() {
	select {
	case <-b.done:
		return
	default:
		close(b.done)
	}
	b.wg.Wait()
}

func (b *Bus) Publish(ctx context.Context, evt Event) []error {
	b.mu.RLock()
	hs := append([]Handler(nil), b.handlers[evt.Name()]...)
	b.mu.RUnlock()

	var errs []error
	for i, h := range hs {
		key := partitionKey(evt)
		shard := shardForKey(key, len(b.shards))
		d := delivery{ctx: ctx, evt: evt, handler: h, handlerIndex: i}

		select {
		case <-b.done:
			errs = append(errs, context.Canceled)
		case b.shards[shard] <- d:
			// queued
		default:
			// backpressure: block until it can be enqueued or bus closes
			select {
			case <-b.done:
				errs = append(errs, context.Canceled)
			case b.shards[shard] <- d:
				// queued
			}
		}
	}
	return errs
}

type delivery struct {
	ctx          context.Context
	evt          Event
	handler       Handler
	handlerIndex int
}

func (b *Bus) worker(shard int) {
	defer b.wg.Done()

	for {
		select {
		case <-b.done:
			return
		case d := <-b.shards[shard]:
			b.processDelivery(shard, d)
		}
	}
}

func (b *Bus) processDelivery(shard int, d delivery) {
	attempt := 0
	backoff := b.cfg.RetryBackoff

	for {
		attempt++
		err := b.safeHandle(d.ctx, d.handler, d.evt, d.handlerIndex)
		if err == nil {
			return
		}

		if b.cfg.MaxAttempts > 0 && attempt >= b.cfg.MaxAttempts {
			log.Printf("broker handler max attempts reached shard=%d event=%s handler_index=%d attempts=%d", shard, d.evt.Name(), d.handlerIndex, attempt)
			return
		}

		select {
		case <-b.done:
			return
		case <-time.After(backoff):
			// retry
		}

		backoff *= 2
		if backoff > b.cfg.RetryBackoffMax {
			backoff = b.cfg.RetryBackoffMax
		}
	}
}

func (b *Bus) safeHandle(ctx context.Context, h Handler, evt Event, idx int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("broker handler panic event=%s handler_index=%d panic=%v", evt.Name(), idx, r)
			err = context.Canceled
		}
	}()
	if err := h(ctx, evt); err != nil {
		log.Printf("broker handler error event=%s handler_index=%d error=%v", evt.Name(), idx, err)
		return err
	}
	return nil
}

func partitionKey(evt Event) string {
	type partitioned interface {
		PartitionKey() string
	}
	if p, ok := evt.(partitioned); ok {
		if k := p.PartitionKey(); k != "" {
			return k
		}
	}
	return evt.Name()
}

func shardForKey(key string, shards int) int {
	if shards <= 1 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32() % uint32(shards))
}
