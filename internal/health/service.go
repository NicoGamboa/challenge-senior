package health

import (
	"context"
	"sync"
	"time"
)

type CheckFunc func(ctx context.Context) error

type Service struct {
	mu sync.Mutex

	checks map[string]CheckFunc
	ttl    time.Duration

	nextCheckAt time.Time
	lastResult  Result
}

type Result struct {
	At     time.Time
	OK     bool
	Checks map[string]string
}

func NewService(ttl time.Duration, checks map[string]CheckFunc) *Service {
	return &Service{ttl: ttl, checks: checks, lastResult: Result{Checks: map[string]string{}}}
}

func (s *Service) Check(ctx context.Context) Result {
	s.mu.Lock()
	if time.Now().Before(s.nextCheckAt) {
		res := s.lastResult
		s.mu.Unlock()
		return res
	}
	s.mu.Unlock()

	res := Result{At: time.Now().UTC(), OK: true, Checks: make(map[string]string, len(s.checks))}
	for name, fn := range s.checks {
		if fn == nil {
			res.OK = false
			res.Checks[name] = "invalid check"
			continue
		}
		if err := fn(ctx); err != nil {
			res.OK = false
			res.Checks[name] = err.Error()
			continue
		}
		res.Checks[name] = "ok"
	}

	s.mu.Lock()
	s.lastResult = res
	s.nextCheckAt = time.Now().Add(s.ttl)
	s.mu.Unlock()

	return res
}
