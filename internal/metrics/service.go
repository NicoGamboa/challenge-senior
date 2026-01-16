package metrics

import "challenge/kit/observability"

type Service struct {
	m *observability.Metrics
}

func NewService(m *observability.Metrics) *Service {
	return &Service{m: m}
}

func (s *Service) Snapshot() map[string]int64 {
	if s.m == nil {
		return map[string]int64{}
	}
	return map[string]int64{
		"payments_created":   s.m.PaymentsCreated.Load(),
		"payments_succeeded": s.m.PaymentsSucceeded.Load(),
		"payments_failed":    s.m.PaymentsFailed.Load(),
		"wallet_debits":      s.m.WalletDebits.Load(),
		"wallet_refunds":     s.m.WalletRefunds.Load(),
	}
}
