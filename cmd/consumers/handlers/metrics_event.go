package handlers

import (
	"context"

	"challenge/internal/events"
	"challenge/kit/broker"
)

type MetricsContract interface {
	PaymentsCreatedAdd(n int64)
	PaymentsSucceededAdd(n int64)
	PaymentsFailedAdd(n int64)
	WalletDebitsAdd(n int64)
	WalletRefundsAdd(n int64)
}

type MetricsEvent struct {
	m MetricsContract
}

func NewMetricsEvent(m MetricsContract) *MetricsEvent {
	return &MetricsEvent{m: m}
}

func (h *MetricsEvent) HandleAny(ctx context.Context, evt broker.Event) error {
	if h.m == nil {
		return nil
	}

	switch evt.(type) {
	case events.PaymentCreated:
		h.m.PaymentsCreatedAdd(1)
	case events.PaymentSucceeded:
		h.m.PaymentsSucceededAdd(1)
	case events.PaymentFailed:
		h.m.PaymentsFailedAdd(1)
	case events.WalletDebited:
		h.m.WalletDebitsAdd(1)
	case events.WalletRefunded:
		h.m.WalletRefundsAdd(1)
	}
	return nil
}
