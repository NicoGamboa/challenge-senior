package payment

import (
	"time"

	"challenge/internal/events"
)

func ToCreateRequest(paymentID, userID string, amount int64, service string) CreateRequest {
	return CreateRequest{PaymentID: paymentID, UserID: userID, Amount: amount, Service: service}
}

func ToPaymentPendingEvent(paymentID, userID string) events.PaymentPending {
	return events.PaymentPending{PaymentID: paymentID, UserID: userID, At: time.Now().UTC()}
}

func ToPaymentRejectedEvent(paymentID, userID, reason string) events.PaymentRejected {
	return events.PaymentRejected{PaymentID: paymentID, UserID: userID, Reason: reason, At: time.Now().UTC()}
}

func ToPaymentSucceededEvent(paymentID, userID, gatewayID string) events.PaymentSucceeded {
	return events.PaymentSucceeded{PaymentID: paymentID, UserID: userID, GatewayID: gatewayID, At: time.Now().UTC()}
}

func ToPaymentFailedEvent(paymentID, userID, reason string) events.PaymentFailed {
	return events.PaymentFailed{PaymentID: paymentID, UserID: userID, Reason: reason, At: time.Now().UTC()}
}
