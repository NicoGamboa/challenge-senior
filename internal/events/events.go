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

func (e PaymentInitialized) PartitionKey() string { return e.PaymentID }

type PaymentCreated struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	Service   string    `json:"service"`
	At        time.Time `json:"at"`
}

func (PaymentCreated) Name() string { return "payment.created" }

func (e PaymentCreated) PartitionKey() string { return e.PaymentID }

type PaymentRejected struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason"`
	At        time.Time `json:"at"`
}

func (PaymentRejected) Name() string { return "payment.rejected" }

func (e PaymentRejected) PartitionKey() string { return e.PaymentID }

type WalletDebited struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	At        time.Time `json:"at"`
}

func (WalletDebited) Name() string { return "wallet.debited" }

func (e WalletDebited) PartitionKey() string { return e.PaymentID }

type WalletCredited struct {
	UserID string    `json:"user_id"`
	Amount int64     `json:"amount"`
	At     time.Time `json:"at"`
}

func (WalletCredited) Name() string { return "wallet.credited" }

func (e WalletCredited) PartitionKey() string { return e.UserID }

type WalletDebitRejected struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason"`
	At        time.Time `json:"at"`
}

func (WalletDebitRejected) Name() string { return "wallet.debit_rejected" }

func (e WalletDebitRejected) PartitionKey() string { return e.PaymentID }

type WalletDebitRequested struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	Attempt   int       `json:"attempt"`
	At        time.Time `json:"at"`
}

func (WalletDebitRequested) Name() string { return "wallet.debit_requested" }

func (e WalletDebitRequested) PartitionKey() string { return e.PaymentID }

type WalletRefundRequested struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	At        time.Time `json:"at"`
}

func (WalletRefundRequested) Name() string { return "wallet.refund_requested" }

func (e WalletRefundRequested) PartitionKey() string { return e.PaymentID }

type PaymentPending struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	At        time.Time `json:"at"`
}

func (PaymentPending) Name() string { return "payment.pending" }

func (e PaymentPending) PartitionKey() string { return e.PaymentID }

type PaymentChargeRequested struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	Service   string    `json:"service"`
	Attempt   int       `json:"attempt"`
	At        time.Time `json:"at"`
}

func (PaymentChargeRequested) Name() string { return "payment.charge_requested" }

func (e PaymentChargeRequested) PartitionKey() string { return e.PaymentID }

type PaymentChargeSucceeded struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	GatewayID string    `json:"gateway_id"`
	At        time.Time `json:"at"`
}

func (PaymentChargeSucceeded) Name() string { return "payment.charge_succeeded" }

func (e PaymentChargeSucceeded) PartitionKey() string { return e.PaymentID }

type PaymentChargeFailed struct {
	PaymentID  string    `json:"payment_id"`
	UserID     string    `json:"user_id"`
	Reason     string    `json:"reason"`
	Retryable  bool      `json:"retryable"`
	ErrorCode  string    `json:"error_code"`
	At         time.Time `json:"at"`
}

func (PaymentChargeFailed) Name() string { return "payment.charge_failed" }

func (e PaymentChargeFailed) PartitionKey() string { return e.PaymentID }

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

func (e RecoveryRequested) PartitionKey() string { return e.PaymentID }

type PaymentSubmitted struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	Service   string    `json:"service"`
	Attempt   int       `json:"attempt"`
	At        time.Time `json:"at"`
}

func (PaymentSubmitted) Name() string { return "payment.submitted" }

func (e PaymentSubmitted) PartitionKey() string { return e.PaymentID }

type PaymentSucceeded struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	GatewayID string    `json:"gateway_id"`
	At        time.Time `json:"at"`
}

func (PaymentSucceeded) Name() string { return "payment.completed" }

func (e PaymentSucceeded) PartitionKey() string { return e.PaymentID }

type PaymentFailed struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason"`
	At        time.Time `json:"at"`
}

func (PaymentFailed) Name() string { return "payment.failed" }

func (e PaymentFailed) PartitionKey() string { return e.PaymentID }

type WalletRefunded struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Amount    int64     `json:"amount"`
	At        time.Time `json:"at"`
}

func (WalletRefunded) Name() string { return "wallet.refunded" }

func (e WalletRefunded) PartitionKey() string { return e.PaymentID }

type PaymentDLQ struct {
	PaymentID string    `json:"payment_id"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason"`
	At        time.Time `json:"at"`
}

func (PaymentDLQ) Name() string { return "payment.dlq" }

func (e PaymentDLQ) PartitionKey() string { return e.PaymentID }
