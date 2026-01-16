package handlers

import (
	"context"
	"fmt"
	"time"

	"challenge/internal/events"
	"challenge/internal/payment"
	"challenge/kit/broker"
	"challenge/kit/observability"
)

type PaymentFlowEvent struct {
	logger  *observability.Logger
	bus     BusContract
	payment payment.ServiceContract
}

func NewPaymentFlowEvent(logger *observability.Logger, bus BusContract, paymentSvc payment.ServiceContract) *PaymentFlowEvent {
	return &PaymentFlowEvent{logger: logger, bus: bus, payment: paymentSvc}
}

func (h *PaymentFlowEvent) HandleWalletDebited(ctx context.Context, evt broker.Event) error {
	e, ok := evt.(events.WalletDebited)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", evt)
	}
	if err := h.payment.MarkPending(ctx, e.PaymentID); err != nil {
		return err
	}
	p, err := h.payment.Get(ctx, e.PaymentID)
	if err != nil {
		return err
	}

	h.bus.Publish(ctx, events.PaymentChargeRequested{PaymentID: p.ID, UserID: p.UserID, Amount: p.Amount, Service: p.Service, Attempt: 1, At: time.Now().UTC()})
	return nil
}

func (h *PaymentFlowEvent) HandleWalletDebitRejected(ctx context.Context, evt broker.Event) error {
	e, ok := evt.(events.WalletDebitRejected)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", evt)
	}
	return h.payment.MarkRejected(ctx, e.PaymentID, e.Reason)
}
