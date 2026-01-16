package notification

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

func (s *Service) Notify(ctx context.Context, userID string, msg string) {
	if s.logger == nil {
		return
	}
	s.logger.Info("notify", "user_id", userID, "msg", msg)
}
