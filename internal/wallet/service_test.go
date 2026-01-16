package wallet

import (
	"context"
	"testing"

	"challenge/kit/observability"
	"challenge/kit/db"

	"github.com/stretchr/testify/require"
)

func TestWalletService_Credit(t *testing.T) {
	ctx := context.Background()
	metricsKit := observability.NewMetrics()

	var tests = []struct {
		name        string
		userID      string
		amount      int64
		service     func() ServiceContract
		expectedErr error
	}{
		{
			name:   "invalid request",
			userID: "",
			amount: 10,
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expectedErr: db.ErrInvalid,
		},
		{
			name:   "get balance error",
			userID: "u1",
			amount: 10,
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("GetBalance", ctx, "u1").Return(int64(0), db.ErrInternal)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expectedErr: db.ErrInternal,
		},
		{
			name:   "set balance error",
			userID: "u1",
			amount: 10,
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("GetBalance", ctx, "u1").Return(int64(5), nil)
				repo.On("SetBalance", ctx, "u1", int64(15)).Return(db.ErrInternal)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expectedErr: db.ErrInternal,
		},
		{
			name:   "success",
			userID: "u1",
			amount: 10,
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("GetBalance", ctx, "u1").Return(int64(5), nil)
				repo.On("SetBalance", ctx, "u1", int64(15)).Return(nil)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.service()
			err := svc.Credit(ctx, tt.userID, tt.amount)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWalletService_Debit(t *testing.T) {
	ctx := context.Background()
	metricsKit := observability.NewMetrics()

	var tests = []struct {
		name        string
		userID      string
		amount      int64
		service     func() ServiceContract
		expectedErr error
	}{
		{
			name:   "invalid request",
			userID: "",
			amount: 10,
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expectedErr: db.ErrInvalid,
		},
		{
			name:   "insufficient funds",
			userID: "u1",
			amount: 10,
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("DebitIfSufficientFunds", ctx, "u1", int64(10)).Return(ErrInsufficientFunds)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expectedErr: ErrInsufficientFunds,
		},
		{
			name:   "repo error",
			userID: "u1",
			amount: 10,
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("DebitIfSufficientFunds", ctx, "u1", int64(10)).Return(db.ErrInternal)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expectedErr: db.ErrInternal,
		},
		{
			name:   "success",
			userID: "u1",
			amount: 10,
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("DebitIfSufficientFunds", ctx, "u1", int64(10)).Return(nil)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.service()
			err := svc.Debit(ctx, tt.userID, tt.amount)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWalletService_Refund(t *testing.T) {
	ctx := context.Background()
	metricsKit := observability.NewMetrics()

	var tests = []struct {
		name        string
		userID      string
		amount      int64
		service     func() ServiceContract
		expectedErr error
	}{
		{
			name:   "invalid request",
			userID: "",
			amount: 10,
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expectedErr: db.ErrInvalid,
		},
		{
			name:   "get balance error",
			userID: "u1",
			amount: 10,
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("GetBalance", ctx, "u1").Return(int64(0), db.ErrInternal)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expectedErr: db.ErrInternal,
		},
		{
			name:   "set balance error",
			userID: "u1",
			amount: 10,
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("GetBalance", ctx, "u1").Return(int64(5), nil)
				repo.On("SetBalance", ctx, "u1", int64(15)).Return(db.ErrInternal)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expectedErr: db.ErrInternal,
		},
		{
			name:   "success",
			userID: "u1",
			amount: 10,
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("GetBalance", ctx, "u1").Return(int64(5), nil)
				repo.On("SetBalance", ctx, "u1", int64(15)).Return(nil)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.service()
			err := svc.Refund(ctx, tt.userID, tt.amount)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWalletService_Balance(t *testing.T) {
	ctx := context.Background()
	metricsKit := observability.NewMetrics()

	var tests = []struct {
		name        string
		userID      string
		service     func() ServiceContract
		expected    int64
		expectedErr error
	}{
		{
			name:   "invalid request",
			userID: "",
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expected:    0,
			expectedErr: db.ErrInvalid,
		},
		{
			name:   "repo error",
			userID: "u1",
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("GetBalance", ctx, "u1").Return(int64(0), db.ErrInternal)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expected:    0,
			expectedErr: db.ErrInternal,
		},
		{
			name:   "success",
			userID: "u1",
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("GetBalance", ctx, "u1").Return(int64(7), nil)
				return NewServiceWithRepo(repo, metricsKit)
			},
			expected:    7,
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.service()
			bal, err := svc.Balance(ctx, tt.userID)
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
