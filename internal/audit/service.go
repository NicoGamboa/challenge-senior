package audit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"challenge/kit/observability"
)

type Service struct {
	logger *observability.Logger
	fileMu sync.Mutex
	f      *os.File
}

func NewService(logger *observability.Logger) *Service {
	return &Service{logger: logger}
}

func NewServiceWithFile(logger *observability.Logger, path string) (*Service, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		if logger != nil {
			logger.Error("audit error", "layer", "service", "component", "audit", "method", "NewServiceWithFile", "path", path, "error", err.Error())
		}
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		if logger != nil {
			logger.Error("audit error", "layer", "service", "component", "audit", "method", "NewServiceWithFile", "path", path, "error", err.Error())
		}
		return nil, err
	}
	return &Service{logger: logger, f: f}, nil
}

func (s *Service) Close() error {
	s.fileMu.Lock()
	defer s.fileMu.Unlock()
	if s.f == nil {
		return nil
	}
	err := s.f.Close()
	if err != nil && s.logger != nil {
		s.logger.Error("audit error", "layer", "service", "component", "audit", "method", "Close", "error", err.Error())
	}
	s.f = nil
	return err
}

func (s *Service) Record(ctx context.Context, eventName string, fields map[string]any) {
	if s.logger == nil {
		return
	}
	s.logger.Info("audit", "event", eventName, "fields", fields)

	s.fileMu.Lock()
	if s.f != nil {
		line := map[string]any{
			"at":    time.Now().UTC(),
			"event": eventName,
			"fields": fields,
		}
		b, err := json.Marshal(line)
		if err != nil {
			s.logger.Error("audit error", "layer", "service", "component", "audit", "method", "Record", "event", eventName, "error", err.Error())
		} else {
			if _, err := s.f.Write(append(b, '\n')); err != nil {
				s.logger.Error("audit error", "layer", "service", "component", "audit", "method", "Record", "event", eventName, "error", err.Error())
			}
		}
	}
	s.fileMu.Unlock()
}
