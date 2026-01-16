package metrics

import (
	"testing"

	"challenge/kit/observability"
	"github.com/stretchr/testify/require"
)

func TestService_Snapshot(t *testing.T) {
	var tests = []struct {
		name     string
		svc      func() *Service
		expected map[string]int64
	}{
		{
			name: "nil metrics",
			svc: func() *Service {
				return NewService(nil)
			},
			expected: map[string]int64{},
		},
		{
			name: "returns current counters",
			svc: func() *Service {
				m := observability.NewMetrics()
				m.PaymentsCreated.Add(1)
				m.PaymentsSucceeded.Add(2)
				m.PaymentsFailed.Add(3)
				m.WalletDebits.Add(4)
				m.WalletRefunds.Add(5)
				return NewService(m)
			},
			expected: map[string]int64{
				"payments_created":   1,
				"payments_succeeded": 2,
				"payments_failed":    3,
				"wallet_debits":      4,
				"wallet_refunds":     5,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.svc()
			require.Equal(t, tt.expected, svc.Snapshot())
		})
	}
}
