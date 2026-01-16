package wallet

import (
	"context"
	"testing"

	"challenge/kit/db"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWalletSQLRepository_GetBalance(t *testing.T) {
	ctx := context.Background()

	var tests = []struct {
		name        string
		userID      string
		repo        func() *SQLRepository
		expected    int64
		expectedErr error
	}{
		{
			name:   "client error",
			userID: "u1",
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				c.On("QueryRow", ctx, "SELECT balance FROM wallets WHERE user_id = ?", []any{"u1"}).Return((db.Row)(nil), db.ErrInternal)
				return NewSQLRepository(c)
			},
			expected:    0,
			expectedErr: db.ErrInternal,
		},
		{
			name:   "not found returns 0, nil",
			userID: "u1",
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				r := new(db.RowMock)
				r.On("Scan", mock.Anything).Return(db.ErrNotFound)
				c.On("QueryRow", ctx, "SELECT balance FROM wallets WHERE user_id = ?", []any{"u1"}).Return(r, nil)
				return NewSQLRepository(c)
			},
			expected:    0,
			expectedErr: nil,
		},
		{
			name:   "scan error",
			userID: "u1",
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				r := new(db.RowMock)
				r.On("Scan", mock.Anything).Return(db.ErrInternal)
				c.On("QueryRow", ctx, "SELECT balance FROM wallets WHERE user_id = ?", []any{"u1"}).Return(r, nil)
				return NewSQLRepository(c)
			},
			expected:    0,
			expectedErr: db.ErrInternal,
		},
		{
			name:   "success",
			userID: "u1",
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				r := new(db.RowMock)
				r.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
					d := args.Get(0).([]any)
					*(d[0].(*int64)) = 42
				}).Return(nil)
				c.On("QueryRow", ctx, "SELECT balance FROM wallets WHERE user_id = ?", []any{"u1"}).Return(r, nil)
				return NewSQLRepository(c)
			},
			expected:    42,
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := tt.repo()
			bal, err := repo.GetBalance(ctx, tt.userID)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, bal)
		})
	}
}

func TestWalletSQLRepository_SetBalance(t *testing.T) {
	ctx := context.Background()

	var tests = []struct {
		name        string
		userID      string
		amount      int64
		repo        func() *SQLRepository
		expectedErr error
	}{
		{
			name:   "exec error",
			userID: "u1",
			amount: 10,
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				c.On("Exec", ctx, "INSERT INTO wallets (user_id, balance) VALUES (?, ?) ON DUPLICATE KEY UPDATE balance = ?", []any{"u1", int64(10), int64(10)}).Return(db.ErrInternal)
				return NewSQLRepository(c)
			},
			expectedErr: db.ErrInternal,
		},
		{
			name:   "success",
			userID: "u1",
			amount: 10,
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				c.On("Exec", ctx, "INSERT INTO wallets (user_id, balance) VALUES (?, ?) ON DUPLICATE KEY UPDATE balance = ?", []any{"u1", int64(10), int64(10)}).Return(nil)
				return NewSQLRepository(c)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := tt.repo()
			err := repo.SetBalance(ctx, tt.userID, tt.amount)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWalletSQLRepository_DebitIfSufficientFunds(t *testing.T) {
	ctx := context.Background()

	var tests = []struct {
		name        string
		userID      string
		amount      int64
		repo        func() *SQLRepository
		expectedErr error
	}{
		{
			name:   "conflict maps to insufficient funds",
			userID: "u1",
			amount: 10,
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				c.On("Exec", ctx, "UPDATE wallets SET balance = balance - ? WHERE user_id = ? AND balance >= ?", []any{int64(10), "u1", int64(10)}).Return(db.ErrConflict)
				return NewSQLRepository(c)
			},
			expectedErr: ErrInsufficientFunds,
		},
		{
			name:   "exec error",
			userID: "u1",
			amount: 10,
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				c.On("Exec", ctx, "UPDATE wallets SET balance = balance - ? WHERE user_id = ? AND balance >= ?", []any{int64(10), "u1", int64(10)}).Return(db.ErrInternal)
				return NewSQLRepository(c)
			},
			expectedErr: db.ErrInternal,
		},
		{
			name:   "success",
			userID: "u1",
			amount: 10,
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				c.On("Exec", ctx, "UPDATE wallets SET balance = balance - ? WHERE user_id = ? AND balance >= ?", []any{int64(10), "u1", int64(10)}).Return(nil)
				return NewSQLRepository(c)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := tt.repo()
			err := repo.DebitIfSufficientFunds(ctx, tt.userID, tt.amount)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
