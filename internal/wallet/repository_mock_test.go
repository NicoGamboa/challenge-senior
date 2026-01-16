package wallet

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type RepositoryMock struct {
	mock.Mock
	RepositoryContract
}

func (m *RepositoryMock) GetBalance(ctx context.Context, userID string) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *RepositoryMock) SetBalance(ctx context.Context, userID string, amount int64) error {
	args := m.Called(ctx, userID, amount)
	return args.Error(0)
}

func (m *RepositoryMock) DebitIfSufficientFunds(ctx context.Context, userID string, amount int64) error {
	args := m.Called(ctx, userID, amount)
	return args.Error(0)
}
