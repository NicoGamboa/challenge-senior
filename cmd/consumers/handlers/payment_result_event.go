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

type PaymentResultEvent struct {
	logger  *observability.Logger
	bus     BusContract
	payment payment.ServiceContract
}

func NewPaymentResultEvent(logger *observability.Logger, bus BusContract, paymentSvc payment.ServiceContract) *PaymentResultEvent {
	return &PaymentResultEvent{logger: logger, bus: bus, payment: paymentSvc}
}

func (h *PaymentResultEvent) HandleChargeSucceeded(ctx context.Context, evt broker.Event) error {
	e, ok := evt.(events.PaymentChargeSucceeded)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", evt)
	}
	return h.payment.MarkSucceeded(ctx, e.PaymentID, e.GatewayID)
}

func (h *PaymentResultEvent) HandleChargeFailed(ctx context.Context, evt broker.Event) error {
	e, ok := evt.(events.PaymentChargeFailed)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", evt)
	}
	if err := h.payment.MarkFailed(ctx, e.PaymentID, e.Reason); err != nil {
		return err
	}
	p, err := h.payment.Get(ctx, e.PaymentID)
	if err != nil {
		return err
	}
	h.bus.Publish(ctx, events.WalletRefundRequested{PaymentID: p.ID, UserID: p.UserID, Amount: p.Amount, At: time.Now().UTC()})
	return nil
}
