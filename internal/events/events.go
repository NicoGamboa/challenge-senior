package events

import "time"

type PaymentInitialized struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	Service   string    `json:"service"`
	At        time.Time `json:"at"`
}

func (PaymentInitialized) Name() string { return "payment.initialized" }

type PaymentCreated struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	Service   string    `json:"service"`
	At        time.Time `json:"at"`
}

func (PaymentCreated) Name() string { return "payment.created" }

type PaymentRejected struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason"`
	At        time.Time `json:"at"`
}

func (PaymentRejected) Name() string { return "payment.rejected" }

type WalletDebited struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	At        time.Time `json:"at"`
}

func (WalletDebited) Name() string { return "wallet.debited" }

type WalletCredited struct {
	UserID string    `json:"user_id"`
	Amount int64     `json:"amount"`
	At     time.Time `json:"at"`
}

func (WalletCredited) Name() string { return "wallet.credited" }

type WalletDebitRejected struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason"`
	At        time.Time `json:"at"`
}

func (WalletDebitRejected) Name() string { return "wallet.debit_rejected" }

type WalletDebitRequested struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	Attempt   int       `json:"attempt"`
	At        time.Time `json:"at"`
}

func (WalletDebitRequested) Name() string { return "wallet.debit_requested" }

type WalletRefundRequested struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	At        time.Time `json:"at"`
}

func (WalletRefundRequested) Name() string { return "wallet.refund_requested" }

type PaymentPending struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	At        time.Time `json:"at"`
}

func (PaymentPending) Name() string { return "payment.pending" }

type PaymentChargeRequested struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	Service   string    `json:"service"`
	Attempt   int       `json:"attempt"`
	At        time.Time `json:"at"`
}

func (PaymentChargeRequested) Name() string { return "payment.charge_requested" }

type PaymentChargeSucceeded struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	GatewayID string    `json:"gateway_id"`
	At        time.Time `json:"at"`
}

func (PaymentChargeSucceeded) Name() string { return "payment.charge_succeeded" }

type PaymentChargeFailed struct {
	PaymentID  string    `json:"payment_id"`
	UserID     string    `json:"user_id"`
	Reason     string    `json:"reason"`
	Retryable  bool      `json:"retryable"`
	ErrorCode  string    `json:"error_code"`
	At         time.Time `json:"at"`
}

func (PaymentChargeFailed) Name() string { return "payment.charge_failed" }

type RecoveryRequested struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	Reason    string    `json:"reason"`
	ErrorCode string    `json:"error_code"`
	Attempts  int       `json:"attempts"`
	At        time.Time `json:"at"`
}

func (RecoveryRequested) Name() string { return "recovery.requested" }

type PaymentSubmitted struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	Service   string    `json:"service"`
	Attempt   int       `json:"attempt"`
	At        time.Time `json:"at"`
}

func (PaymentSubmitted) Name() string { return "payment.submitted" }

type PaymentSucceeded struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	GatewayID string    `json:"gateway_id"`
	At        time.Time `json:"at"`
}

func (PaymentSucceeded) Name() string { return "payment.completed" }

type PaymentFailed struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason"`
	At        time.Time `json:"at"`
}

func (PaymentFailed) Name() string { return "payment.failed" }

type WalletRefunded struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	At        time.Time `json:"at"`
}

func (WalletRefunded) Name() string { return "wallet.refunded" }

type PaymentDLQ struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason"`
	At        time.Time `json:"at"`
}

func (PaymentDLQ) Name() string { return "payment.dlq" }
