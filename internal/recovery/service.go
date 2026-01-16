package recovery

import (
	"context"

	"challenge/kit/observability"
)

type Service struct {
	logger *observability.Logger
}

func NewService(logger *observability.Logger) *Service {
	return &Service{logger: logger}
}

func (s *Service) SendToDLQ(ctx context.Context, topic string, reason string, payload any) {
	if s.logger == nil {
		return
	}
	s.logger.Error("dlq", "topic", topic, "reason", reason, "payload", payload)
}
