package payment

import (
	"challenge/kit/db"
	"context"
	"errors"
	"log"

	"challenge/kit/observability"
)

type Service struct {
	bus        PublisherContract
	store      StoreContract
	repository RepositoryContract
	metrics    *observability.Metrics
}

func NewService(bus PublisherContract, store StoreContract, repo RepositoryContract, metrics *observability.Metrics) *Service {
	return &Service{
		bus:        bus,
		store:      store,
		repository: repo,
		metrics:    metrics,
	}
}

func (s *Service) Initialize(ctx context.Context, req CreateRequest) (*Payment, error) {
	if err := ValidateCreateRequest(req); err != nil {
		log.Printf("layer=service component=payment method=Initialize payment_id=%s user_id=%s amount=%d err=%v", req.PaymentID, req.UserID, req.Amount, err)
		return nil, errors.Join(db.ErrInvalid, err)
	}

	p := &Payment{ID: req.PaymentID, UserID: req.UserID, Amount: req.Amount, Service: req.Service, Status: StatusInitialized}
	if err := s.repository.Save(ctx, p); err != nil {
		log.Printf("layer=service component=payment method=Initialize payment_id=%s user_id=%s err=%v", req.PaymentID, req.UserID, err)
		return nil, err
	}

	return p, nil
}

func (s *Service) MarkPending(ctx context.Context, paymentID string) error {
	p, err := s.repository.Get(ctx, paymentID)
	if err != nil {
		log.Printf("layer=service component=payment method=MarkPending payment_id=%s err=%v", paymentID, err)
		return err
	}
	p.Status = StatusPending
	_ = s.repository.Save(ctx, p)

	evt := ToPaymentPendingEvent(paymentID, p.UserID)
	if s.store != nil {
		_ = s.store.Append(ctx, paymentID, evt)
	}
	if s.bus != nil {
		s.bus.Publish(ctx, evt)
	}
	return nil
}

func (s *Service) MarkRejected(ctx context.Context, paymentID, reason string) error {
	p, err := s.repository.Get(ctx, paymentID)
	if err != nil {
		log.Printf("layer=service component=payment method=MarkRejected payment_id=%s err=%v", paymentID, err)
		return err
	}
	p.Status = StatusRejected
	p.Reason = reason
	_ = s.repository.Save(ctx, p)

	evt := ToPaymentRejectedEvent(paymentID, p.UserID, reason)
	if s.store != nil {
		_ = s.store.Append(ctx, paymentID, evt)
	}
	if s.bus != nil {
		s.bus.Publish(ctx, evt)
	}
	return nil
}

func (s *Service) MarkSucceeded(ctx context.Context, paymentID, gatewayID string) error {
	p, err := s.repository.Get(ctx, paymentID)
	if err != nil {
		log.Printf("layer=service component=payment method=MarkSucceeded payment_id=%s err=%v", paymentID, err)
		return err
	}
	p.Status = StatusSucceeded
	p.GatewayID = gatewayID
	_ = s.repository.Save(ctx, p)

	evt := ToPaymentSucceededEvent(paymentID, p.UserID, gatewayID)
	if s.store != nil {
		_ = s.store.Append(ctx, paymentID, evt)
	}
	if s.bus != nil {
		s.bus.Publish(ctx, evt)
	}
	if s.metrics != nil {
		s.metrics.PaymentsSucceeded.Add(1)
	}
	return nil
}

func (s *Service) MarkFailed(ctx context.Context, paymentID, reason string) error {
	p, err := s.repository.Get(ctx, paymentID)
	if err != nil {
		log.Printf("layer=service component=payment method=MarkFailed payment_id=%s err=%v", paymentID, err)
		return err
	}
	p.Status = StatusFailed
	p.Reason = reason
	_ = s.repository.Save(ctx, p)

	failed := ToPaymentFailedEvent(paymentID, p.UserID, reason)
	if s.store != nil {
		_ = s.store.Append(ctx, paymentID, failed)
	}
	if s.bus != nil {
		s.bus.Publish(ctx, failed)
	}
	if s.metrics != nil {
		s.metrics.PaymentsFailed.Add(1)
	}
	return nil
}

func (s *Service) Get(ctx context.Context, paymentID string) (*Payment, error) {
	p, err := s.repository.Get(ctx, paymentID)
	if err != nil {
		log.Printf("layer=service component=payment method=Get payment_id=%s err=%v", paymentID, err)
		return nil, err
	}
	return p, nil
}
