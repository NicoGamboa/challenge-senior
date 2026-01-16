package observability

import "sync/atomic"

type Metrics struct {
	PaymentsCreated   atomic.Int64
	PaymentsSucceeded atomic.Int64
	PaymentsFailed    atomic.Int64
	WalletDebits      atomic.Int64
	WalletRefunds     atomic.Int64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) PaymentsCreatedAdd(n int64) {
	m.PaymentsCreated.Add(n)
}

func (m *Metrics) PaymentsSucceededAdd(n int64) {
	m.PaymentsSucceeded.Add(n)
}

func (m *Metrics) PaymentsFailedAdd(n int64) {
	m.PaymentsFailed.Add(n)
}

func (m *Metrics) WalletDebitsAdd(n int64) {
	m.WalletDebits.Add(n)
}

func (m *Metrics) WalletRefundsAdd(n int64) {
	m.WalletRefunds.Add(n)
}
