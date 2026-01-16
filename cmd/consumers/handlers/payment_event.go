package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"challenge/internal/events"
	"challenge/internal/recovery"
	"challenge/kit/broker"
	"challenge/kit/external_payment_gateway"
	"challenge/kit/observability"
)

type PaymentEvent struct {
	logger  *observability.Logger
	gateway external_payment_gateway.Gateway
	recovery *recovery.Service
	bus     BusContract
}

func NewPaymentEvent(logger *observability.Logger, bus BusContract, gateway external_payment_gateway.Gateway, recoverySvc *recovery.Service) *PaymentEvent {
	return &PaymentEvent{logger: logger, bus: bus, gateway: gateway, recovery: recoverySvc}
}

func (h *PaymentEvent) HandleChargeRequested(ctx context.Context, evt broker.Event) error {
	e, ok := evt.(events.PaymentChargeRequested)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", evt)
	}

	attempt := e.Attempt
	for {
		callCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		gwID, err := h.gateway.Charge(callCtx, e.PaymentID, e.Amount)
		cancel()
		if err == nil {
			h.logger.Info("gateway charge succeeded", "payment_id", e.PaymentID, "gateway_id", gwID, "attempt", attempt)
			h.bus.Publish(ctx, events.PaymentChargeSucceeded{PaymentID: e.PaymentID, UserID: e.UserID, GatewayID: gwID, At: time.Now().UTC()})
			return nil
		}

		retryable := errors.Is(err, external_payment_gateway.ErrTimeout) || errors.Is(err, external_payment_gateway.ErrServer) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, external_payment_gateway.ErrCircuitOpen)
		clientError := errors.Is(err, external_payment_gateway.ErrClient)

		errorCode := ""
		if errors.Is(err, external_payment_gateway.ErrCircuitOpen) {
			errorCode = "cb_open"
		}
		if errors.Is(err, external_payment_gateway.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
			errorCode = "408"
		}
		if errors.Is(err, external_payment_gateway.ErrServer) {
			errorCode = "5xx"
		}
		if errors.Is(err, external_payment_gateway.ErrClient) {
			errorCode = "4xx"
		}

		if retryable && attempt < 5 {
			backoff := time.Duration(50*attempt) * time.Millisecond
			h.logger.Info("gateway retrying", "payment_id", e.PaymentID, "attempt", attempt, "backoff", backoff.String(), "error_code", errorCode)
			time.Sleep(backoff)
			attempt++
			continue
		}

		reason := err.Error()

		if clientError {
			h.logger.Error("gateway charge failed (client error)", "payment_id", e.PaymentID, "attempt", attempt, "reason", reason)
			h.bus.Publish(ctx, events.PaymentChargeFailed{PaymentID: e.PaymentID, UserID: e.UserID, Reason: reason, Retryable: false, ErrorCode: errorCode, At: time.Now().UTC()})
			return nil
		}

		if retryable {
			if attempt >= 5 {
				if attempt == 5 {
					h.logger.Error("gateway retries exhausted, sending to recovery", "payment_id", e.PaymentID, "attempts", attempt, "reason", reason, "error_code", errorCode)
					req := events.RecoveryRequested{PaymentID: e.PaymentID, UserID: e.UserID, Action: "payment.charge", Reason: reason, ErrorCode: errorCode, Attempts: attempt, At: time.Now().UTC()}
					if h.recovery != nil {
						h.recovery.SendToDLQ(ctx, req.Name(), reason, e)
					}
					h.bus.Publish(ctx, req)
					return nil
				}

				h.logger.Error("gateway retry after recovery failed, failing payment", "payment_id", e.PaymentID, "attempt", attempt, "reason", reason, "error_code", errorCode)
				h.bus.Publish(ctx, events.PaymentChargeFailed{PaymentID: e.PaymentID, UserID: e.UserID, Reason: reason, Retryable: false, ErrorCode: errorCode, At: time.Now().UTC()})
				return nil
			}
		}

		h.logger.Error("gateway charge failed", "payment_id", e.PaymentID, "attempt", attempt, "reason", reason)
		h.bus.Publish(ctx, events.PaymentChargeFailed{PaymentID: e.PaymentID, UserID: e.UserID, Reason: reason, Retryable: false, ErrorCode: errorCode, At: time.Now().UTC()})
		return nil
	}
}
