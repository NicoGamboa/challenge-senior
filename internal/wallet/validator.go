package wallet

import "errors"

var ErrInvalidRequest = errors.New("invalid request")

type CreditRequest struct {
	UserID string
	Amount int64
}

func ValidateCreditRequest(r CreditRequest) error {
	if r.UserID == "" || r.Amount <= 0 {
		return ErrInvalidRequest
	}
	return nil
}

func ValidateSufficientFunds(current, amount int64) error {
	if amount <= 0 {
		return ErrInvalidRequest
	}
	if current < amount {
		return ErrInsufficientFunds
	}
	return nil
}
