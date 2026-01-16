package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEventNames(t *testing.T) {
	now := time.Now().UTC()

	var tests = []struct {
		name     string
		evt      interface{ Name() string }
		expected string
	}{
		{name: "payment.initialized", evt: PaymentInitialized{At: now}, expected: "payment.initialized"},
		{name: "payment.created", evt: PaymentCreated{At: now}, expected: "payment.created"},
		{name: "payment.rejected", evt: PaymentRejected{At: now}, expected: "payment.rejected"},
		{name: "wallet.debited", evt: WalletDebited{At: now}, expected: "wallet.debited"},
		{name: "wallet.credited", evt: WalletCredited{At: now}, expected: "wallet.credited"},
		{name: "wallet.debit_rejected", evt: WalletDebitRejected{At: now}, expected: "wallet.debit_rejected"},
		{name: "wallet.debit_requested", evt: WalletDebitRequested{At: now}, expected: "wallet.debit_requested"},
		{name: "wallet.refund_requested", evt: WalletRefundRequested{At: now}, expected: "wallet.refund_requested"},
		{name: "payment.pending", evt: PaymentPending{At: now}, expected: "payment.pending"},
		{name: "payment.charge_requested", evt: PaymentChargeRequested{At: now}, expected: "payment.charge_requested"},
		{name: "payment.charge_succeeded", evt: PaymentChargeSucceeded{At: now}, expected: "payment.charge_succeeded"},
		{name: "payment.charge_failed", evt: PaymentChargeFailed{At: now}, expected: "payment.charge_failed"},
		{name: "recovery.requested", evt: RecoveryRequested{At: now}, expected: "recovery.requested"},
		{name: "payment.submitted", evt: PaymentSubmitted{At: now}, expected: "payment.submitted"},
		{name: "payment.completed", evt: PaymentSucceeded{At: now}, expected: "payment.completed"},
		{name: "payment.failed", evt: PaymentFailed{At: now}, expected: "payment.failed"},
		{name: "wallet.refunded", evt: WalletRefunded{At: now}, expected: "wallet.refunded"},
		{name: "payment.dlq", evt: PaymentDLQ{At: now}, expected: "payment.dlq"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, tt.evt.Name())
		})
	}
}
