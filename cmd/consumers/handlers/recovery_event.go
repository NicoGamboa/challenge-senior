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

type SleepFunc func(ctx context.Context, d time.Duration) error

func DefaultSleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

type RecoveryEvent struct {
	logger  *observability.Logger
	bus     BusContract
	payment payment.ServiceContract
	delay   time.Duration
	sleep   SleepFunc
}

func NewRecoveryEvent(logger *observability.Logger, bus BusContract, paymentSvc payment.ServiceContract, delay time.Duration, sleep SleepFunc) *RecoveryEvent {
	if sleep == nil {
		sleep = DefaultSleep
	}
	return &RecoveryEvent{logger: logger, bus: bus, payment: paymentSvc, delay: delay, sleep: sleep}
}

func (h *RecoveryEvent) HandleRecoveryRequested(ctx context.Context, evt broker.Event) error {
	e, ok := evt.(events.RecoveryRequested)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", evt)
	}

	if h.logger != nil {
		h.logger.Info("recovery scheduled", "payment_id", e.PaymentID, "delay", h.delay.String(), "action", e.Action, "error_code", e.ErrorCode, "attempts", e.Attempts)
	}

	if err := h.sleep(ctx, h.delay); err != nil {
		return err
	}

	p, err := h.payment.Get(ctx, e.PaymentID)
	if err != nil {
		return err
	}

	switch e.Action {
	case "payment.charge":
		h.bus.Publish(ctx, events.PaymentChargeRequested{PaymentID: p.ID, UserID: p.UserID, Amount: p.Amount, Service: p.Service, Attempt: e.Attempts + 1, At: time.Now().UTC()})
		return nil
	case "wallet.debit":
		h.bus.Publish(ctx, events.WalletDebitRequested{PaymentID: p.ID, UserID: p.UserID, Amount: p.Amount, Attempt: e.Attempts + 1, At: time.Now().UTC()})
		return nil
	default:
		return fmt.Errorf("unknown recovery action: %s", e.Action)
	}
}
