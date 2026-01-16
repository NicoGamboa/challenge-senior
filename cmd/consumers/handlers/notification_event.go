package handlers

import (
	"context"
	"fmt"

	"challenge/internal/events"
	"challenge/kit/broker"
)

type NotifierContract interface {
	Notify(ctx context.Context, userID string, msg string)
}

type NotificationEvent struct {
	n NotifierContract
}

func NewNotificationEvent(n NotifierContract) *NotificationEvent {
	return &NotificationEvent{n: n}
}

func (h *NotificationEvent) HandlePaymentCompleted(ctx context.Context, evt broker.Event) error {
	if h.n == nil {
		return nil
	}
	e, ok := evt.(events.PaymentSucceeded)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", evt)
	}
	h.n.Notify(ctx, e.UserID, "payment completed")
	return nil
}

func (h *NotificationEvent) HandlePaymentFailed(ctx context.Context, evt broker.Event) error {
	if h.n == nil {
		return nil
	}
	e, ok := evt.(events.PaymentFailed)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", evt)
	}
	h.n.Notify(ctx, e.UserID, "payment failed")
	return nil
}
