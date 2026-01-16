package payment

import (
	"context"
	"testing"

	"challenge/kit/db"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPaymentSQLRepository_Save(t *testing.T) {
	ctx := context.Background()

	var tests = []struct {
		name        string
		p           *Payment
		repo        func() *SQLRepository
		expectedErr error
	}{
		{
			name: "exec error",
			p:    &Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet", Status: StatusInitialized},
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				c.On(
					"Exec",
					ctx,
					"INSERT INTO payments (payment_id, user_id, amount, service, status, reason, gateway_id) VALUES (?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE user_id=?, amount=?, service=?, status=?, reason=?, gateway_id=?",
					[]any{"p1", "u1", int64(10), "internet", StatusInitialized, "", "", "u1", int64(10), "internet", StatusInitialized, "", ""},
				).Return(db.ErrInternal)
				return NewSQLRepository(c)
			},
			expectedErr: db.ErrInternal,
		},
		{
			name: "success",
			p:    &Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet", Status: StatusInitialized},
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				c.On(
					"Exec",
					ctx,
					"INSERT INTO payments (payment_id, user_id, amount, service, status, reason, gateway_id) VALUES (?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE user_id=?, amount=?, service=?, status=?, reason=?, gateway_id=?",
					[]any{"p1", "u1", int64(10), "internet", StatusInitialized, "", "", "u1", int64(10), "internet", StatusInitialized, "", ""},
				).Return(nil)
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
			err := repo.Save(ctx, tt.p)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPaymentSQLRepository_Get(t *testing.T) {
	ctx := context.Background()

	var tests = []struct {
		name        string
		paymentID   string
		repo        func() *SQLRepository
		expected    *Payment
		expectedErr error
	}{
		{
			name:      "client error",
			paymentID: "p1",
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				c.On("QueryRow", ctx, "SELECT payment_id, user_id, amount, service, status, reason, gateway_id FROM payments WHERE payment_id = ?", []any{"p1"}).Return((db.Row)(nil), db.ErrInternal)
				return NewSQLRepository(c)
			},
			expected:    nil,
			expectedErr: db.ErrInternal,
		},
		{
			name:      "scan not found",
			paymentID: "p1",
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				r := new(db.RowMock)
				r.On("Scan", mock.Anything).Return(db.ErrNotFound)
				c.On("QueryRow", ctx, "SELECT payment_id, user_id, amount, service, status, reason, gateway_id FROM payments WHERE payment_id = ?", []any{"p1"}).Return(r, nil)
				return NewSQLRepository(c)
			},
			expected:    nil,
			expectedErr: db.ErrNotFound,
		},
		{
			name:      "success",
			paymentID: "p1",
			repo: func() *SQLRepository {
				c := new(db.ClientMock)
				r := new(db.RowMock)
				r.On("Scan", mock.Anything).Run(func(args mock.Arguments) {
					d := args.Get(0).([]any)
					*(d[0].(*string)) = "p1"
					*(d[1].(*string)) = "u1"
					*(d[2].(*int64)) = 10
					*(d[3].(*string)) = "internet"
					*(d[4].(*Status)) = StatusInitialized
					*(d[5].(*string)) = ""
					*(d[6].(*string)) = ""
				}).Return(nil)
				c.On("QueryRow", ctx, "SELECT payment_id, user_id, amount, service, status, reason, gateway_id FROM payments WHERE payment_id = ?", []any{"p1"}).Return(r, nil)
				return NewSQLRepository(c)
			},
			expected:    &Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet", Status: StatusInitialized, Reason: "", GatewayID: ""},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := tt.repo()
			p, err := repo.Get(ctx, tt.paymentID)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expected, p)
		})
	}
}
