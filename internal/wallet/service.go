package wallet

import (
	"context"
	"errors"
	"log"

	"challenge/kit/db"
	"challenge/kit/observability"
)

var ErrInsufficientFunds = errors.New("insufficient funds")

type Service struct {
	repo    Repository
	metrics *observability.Metrics
}

func NewService(metrics *observability.Metrics) *Service {
	return NewServiceWithRepo(NewInMemoryRepository(), metrics)
}

func NewServiceWithRepo(repo Repository, metrics *observability.Metrics) *Service {
	return &Service{repo: repo, metrics: metrics}
}

func (s *Service) Credit(ctx context.Context, userID string, amount int64) error {
	if err := ValidateCreditRequest(ToCreditRequest(userID, amount)); err != nil {
		log.Printf("layer=service component=wallet method=Credit user_id=%s amount=%d err=%v", userID, amount, err)
		return errors.Join(db.ErrInvalid, err)
	}
	current, err := s.repo.GetBalance(ctx, userID)
	if err != nil {
		log.Printf("layer=service component=wallet method=Credit user_id=%s amount=%d err=%v", userID, amount, err)
		return err
	}
	if err := s.repo.SetBalance(ctx, userID, current+amount); err != nil {
		log.Printf("layer=service component=wallet method=Credit user_id=%s amount=%d err=%v", userID, amount, err)
		return err
	}
	return nil
}

func (s *Service) Debit(ctx context.Context, userID string, amount int64) error {
	if userID == "" || amount <= 0 {
		log.Printf("layer=service component=wallet method=Debit user_id=%s amount=%d err=%v", userID, amount, ErrInvalidRequest)
		return errors.Join(db.ErrInvalid, ErrInvalidRequest)
	}
	if err := s.repo.DebitIfSufficientFunds(ctx, userID, amount); err != nil {
		log.Printf("layer=service component=wallet method=Debit user_id=%s amount=%d err=%v", userID, amount, err)
		return err
	}
	if s.metrics != nil {
		s.metrics.WalletDebits.Add(1)
	}
	return nil
}

func (s *Service) Refund(ctx context.Context, userID string, amount int64) error {
	if userID == "" || amount <= 0 {
		log.Printf("layer=service component=wallet method=Refund user_id=%s amount=%d err=%v", userID, amount, ErrInvalidRequest)
		return errors.Join(db.ErrInvalid, ErrInvalidRequest)
	}
	current, err := s.repo.GetBalance(ctx, userID)
	if err != nil {
		log.Printf("layer=service component=wallet method=Refund user_id=%s amount=%d err=%v", userID, amount, err)
		return err
	}
	if err := s.repo.SetBalance(ctx, userID, current+amount); err != nil {
		log.Printf("layer=service component=wallet method=Refund user_id=%s amount=%d err=%v", userID, amount, err)
		return err
	}
	if s.metrics != nil {
		s.metrics.WalletRefunds.Add(1)
	}
	return nil
}

func (s *Service) Balance(ctx context.Context, userID string) (int64, error) {
	if userID == "" {
		log.Printf("layer=service component=wallet method=Balance user_id=%s err=%v", userID, ErrInvalidRequest)
		return 0, errors.Join(db.ErrInvalid, ErrInvalidRequest)
	}
	bal, err := s.repo.GetBalance(ctx, userID)
	if err != nil {
		log.Printf("layer=service component=wallet method=Balance user_id=%s err=%v", userID, err)
		return 0, err
	}
	return bal, nil
}
