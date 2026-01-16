package handlers

import (
	"context"
	"fmt"
	"time"

	"challenge/internal/events"
	"challenge/internal/wallet"
	"challenge/kit/broker"
	"challenge/kit/db"
	"challenge/kit/observability"
)

type WalletEvent struct {
	logger *observability.Logger
	bus    BusContract
	wallet wallet.ServiceContract
}

func NewWalletEvent(logger *observability.Logger, bus BusContract, walletSvc wallet.ServiceContract) *WalletEvent {
	return &WalletEvent{logger: logger, bus: bus, wallet: walletSvc}
}

func (h *WalletEvent) HandlePaymentInitialized(ctx context.Context, evt broker.Event) error {
	e, ok := evt.(events.PaymentInitialized)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", evt)
	}

	h.bus.Publish(ctx, events.WalletDebitRequested{PaymentID: e.PaymentID, UserID: e.UserID, Amount: e.Amount, Attempt: 1, At: time.Now().UTC()})
	return nil
}

func (h *WalletEvent) HandleWalletDebitRequested(ctx context.Context, evt broker.Event) error {
	e, ok := evt.(events.WalletDebitRequested)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", evt)
	}

	if err := h.wallet.Debit(ctx, e.UserID, e.Amount); err != nil {
		if db.IsInternal(err) && e.Attempt == 1 {
			h.bus.Publish(ctx, events.RecoveryRequested{PaymentID: e.PaymentID, UserID: e.UserID, Action: "wallet.debit", Reason: err.Error(), ErrorCode: "db_internal", Attempts: e.Attempt, At: time.Now().UTC()})
			return nil
		}
		h.bus.Publish(ctx, events.WalletDebitRejected{PaymentID: e.PaymentID, UserID: e.UserID, Reason: err.Error(), At: time.Now().UTC()})
		return nil
	}

	h.bus.Publish(ctx, events.WalletDebited{PaymentID: e.PaymentID, UserID: e.UserID, Amount: e.Amount, At: time.Now().UTC()})
	return nil
}

func (h *WalletEvent) HandleWalletRefundRequested(ctx context.Context, evt broker.Event) error {
	e, ok := evt.(events.WalletRefundRequested)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", evt)
	}
	if err := h.wallet.Refund(ctx, e.UserID, e.Amount); err != nil {
		return err
	}
	h.bus.Publish(ctx, events.WalletRefunded{PaymentID: e.PaymentID, UserID: e.UserID, Amount: e.Amount, At: time.Now().UTC()})
	return nil
}

func (h *WalletEvent) HandleWalletDebited(ctx context.Context, evt broker.Event) error {
	e, ok := evt.(events.WalletDebited)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", evt)
	}
	if h.logger != nil {
		h.logger.Info("wallet debited", "payment_id", e.PaymentID, "user_id", e.UserID, "amount", e.Amount)
	}
	return nil
}

func (h *WalletEvent) HandleWalletRefunded(ctx context.Context, evt broker.Event) error {
	e, ok := evt.(events.WalletRefunded)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", evt)
	}
	if h.logger != nil {
		h.logger.Info("wallet refunded", "payment_id", e.PaymentID, "user_id", e.UserID, "amount", e.Amount)
	}
	return nil
}
