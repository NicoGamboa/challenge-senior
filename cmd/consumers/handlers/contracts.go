package handlers

import (
	"errors"

	"challenge/kit/broker"
)

var (
	ErrUnexpectedEventType   = errors.New("unexpected")
	ErrUnknownRecoveryAction = errors.New("unknown")
)

// BusContract defines the publish responsibility used by consumers handlers.
type BusContract = broker.Publisher
