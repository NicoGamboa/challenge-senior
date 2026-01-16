package handlers

import (
	"context"
	"fmt"

	"challenge/internal/events"
	"challenge/kit/broker"
)

type AuditorContract interface {
	Record(ctx context.Context, eventName string, fields map[string]any)
}

type AuditEvent struct {
	audit AuditorContract
}

func NewAuditEvent(a AuditorContract) *AuditEvent {
	return &AuditEvent{audit: a}
}

func (h *AuditEvent) HandleAny(ctx context.Context, evt broker.Event) error {
	if h.audit == nil {
		return nil
	}

	fields := map[string]any{"type": fmt.Sprintf("%T", evt)}
	switch e := evt.(type) {
	case events.PaymentCreated:
		fields["payment_id"] = e.PaymentID
		fields["user_id"] = e.UserID
		fields["amount"] = e.Amount
	case events.PaymentInitialized:
		fields["payment_id"] = e.PaymentID
		fields["user_id"] = e.UserID
		fields["amount"] = e.Amount
		fields["service"] = e.Service
	case events.PaymentPending:
		fields["payment_id"] = e.PaymentID
		fields["user_id"] = e.UserID
	case events.PaymentChargeRequested:
		fields["payment_id"] = e.PaymentID
		fields["user_id"] = e.UserID
		fields["attempt"] = e.Attempt
	case events.PaymentChargeSucceeded:
		fields["payment_id"] = e.PaymentID
		fields["user_id"] = e.UserID
		fields["gateway_id"] = e.GatewayID
	case events.PaymentChargeFailed:
		fields["payment_id"] = e.PaymentID
		fields["user_id"] = e.UserID
		fields["reason"] = e.Reason
	case events.RecoveryRequested:
		fields["payment_id"] = e.PaymentID
		fields["user_id"] = e.UserID
		fields["action"] = e.Action
		fields["reason"] = e.Reason
		fields["error_code"] = e.ErrorCode
		fields["attempts"] = e.Attempts
	case events.WalletDebitRequested:
		fields["payment_id"] = e.PaymentID
		fields["user_id"] = e.UserID
		fields["amount"] = e.Amount
		fields["attempt"] = e.Attempt
	case events.WalletDebited:
		fields["payment_id"] = e.PaymentID
		fields["user_id"] = e.UserID
		fields["amount"] = e.Amount
	case events.WalletRefunded:
		fields["payment_id"] = e.PaymentID
		fields["user_id"] = e.UserID
		fields["amount"] = e.Amount
	}

	h.audit.Record(ctx, evt.Name(), fields)
	return nil
}
