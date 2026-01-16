package external_payment_gateway

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrTimeout = errors.New("gateway timeout")
var ErrServer = errors.New("gateway 5xx")
var ErrClient = errors.New("gateway 4xx")
var ErrCircuitOpen = errors.New("circuit open")

type Gateway interface {
	Charge(ctx context.Context, paymentID string, amount int64) (string, error)
}

type CircuitBreakerConfig struct {
	FailureThreshold int
	SuccessThreshold int
	OpenTimeout      time.Duration
	IsFailure        func(error) bool
}

type CircuitBreakerGateway struct {
	next Gateway
	cfg  CircuitBreakerConfig

	mu           sync.Mutex
	state        int
	failures     int
	successes    int
	openedAt     time.Time
	halfInFlight bool
}

const (
	cbClosed = iota
	cbOpen
	cbHalfOpen
)

func NewCircuitBreakerGateway(next Gateway, cfg CircuitBreakerConfig) *CircuitBreakerGateway {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 1
	}
	if cfg.OpenTimeout <= 0 {
		cfg.OpenTimeout = 2 * time.Second
	}
	if cfg.IsFailure == nil {
		cfg.IsFailure = func(err error) bool {
			return errors.Is(err, ErrTimeout) || errors.Is(err, ErrServer) || errors.Is(err, context.DeadlineExceeded)
		}
	}
	return &CircuitBreakerGateway{next: next, cfg: cfg, state: cbClosed}
}

func (g *CircuitBreakerGateway) Charge(ctx context.Context, paymentID string, amount int64) (string, error) {
	if err := g.beforeCall(); err != nil {
		return "", err
	}

	gwID, err := g.next.Charge(ctx, paymentID, amount)
	g.afterCall(err)
	return gwID, err
}

func (g *CircuitBreakerGateway) beforeCall() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	switch g.state {
	case cbClosed:
		return nil
	case cbOpen:
		if time.Since(g.openedAt) >= g.cfg.OpenTimeout {
			g.state = cbHalfOpen
			g.successes = 0
			g.halfInFlight = false
		} else {
			return ErrCircuitOpen
		}
		fallthrough
	case cbHalfOpen:
		if g.halfInFlight {
			return ErrCircuitOpen
		}
		g.halfInFlight = true
		return nil
	default:
		return ErrCircuitOpen
	}
}

func (g *CircuitBreakerGateway) afterCall(err error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state == cbHalfOpen {
		g.halfInFlight = false
	}

	if err == nil {
		switch g.state {
		case cbClosed:
			g.failures = 0
		case cbHalfOpen:
			g.successes++
			if g.successes >= g.cfg.SuccessThreshold {
				g.state = cbClosed
				g.failures = 0
				g.successes = 0
			}
		}
		return
	}

	if !g.cfg.IsFailure(err) {
		return
	}

	switch g.state {
	case cbClosed:
		g.failures++
		if g.failures >= g.cfg.FailureThreshold {
			g.state = cbOpen
			g.openedAt = time.Now().UTC()
			g.successes = 0
			g.halfInFlight = false
		}
	case cbHalfOpen:
		g.state = cbOpen
		g.openedAt = time.Now().UTC()
		g.failures = g.cfg.FailureThreshold
		g.successes = 0
		g.halfInFlight = false
	}
}

type FakeGateway struct{}

func NewFakeGateway() *FakeGateway { return &FakeGateway{} }

func (g *FakeGateway) Charge(ctx context.Context, paymentID string, amount int64) (string, error) {
	if amount%5 == 0 {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(300 * time.Millisecond):
			return "", ErrTimeout
		}
	}
	if amount%11 == 0 {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(50 * time.Millisecond):
			return "", ErrClient
		}
	}
	if amount%7 == 0 {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(50 * time.Millisecond):
			return "", ErrServer
		}
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(50 * time.Millisecond):
		return fmt.Sprintf("gw_%s", paymentID), nil
	}
}
