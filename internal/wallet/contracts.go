package wallet

import "context"

// RepositoryContract define wallet repository responsibility.
type RepositoryContract interface {
	GetBalance(ctx context.Context, userID string) (int64, error)
	SetBalance(ctx context.Context, userID string, amount int64) error
	DebitIfSufficientFunds(ctx context.Context, userID string, amount int64) error
}

// ServiceContract define wallet service responsibility.
type ServiceContract interface {
	Credit(ctx context.Context, userID string, amount int64) error
	Debit(ctx context.Context, userID string, amount int64) error
	Refund(ctx context.Context, userID string, amount int64) error
	Balance(ctx context.Context, userID string) (int64, error)
}
