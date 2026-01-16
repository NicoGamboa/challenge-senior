package payment

import "errors"

var (
	ErrInvalidPayment = errors.New("invalid payment")
)

type CreateRequest struct {
	PaymentID string
	UserID    string
	Amount    int64
	Service   string
}

func ValidateCreateRequest(r CreateRequest) error {
	if r.PaymentID == "" || r.UserID == "" || r.Amount <= 0 || r.Service == "" {
		return ErrInvalidPayment
	}
	return nil
}
