package payment

import (
	"context"
	"testing"

	"challenge/kit/observability"
	"challenge/kit/db"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPaymentService_Initialize(t *testing.T) {
	ctx := context.Background()
	metricsKit := observability.NewMetrics()

	var tests = []struct {
		name        string
		req         CreateRequest
		service     func() ServiceContract
		expected    *Payment
		expectedErr error
	}{
		{
			name: "invalid request",
			req:  CreateRequest{PaymentID: "", UserID: "u1", Amount: 10, Service: "internet"},
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				return NewService(nil, nil, repo, metricsKit)
			},
			expected:    nil,
			expectedErr: db.ErrInvalid,
		},
		{
			name: "repo save error",
			req:  CreateRequest{PaymentID: "p1", UserID: "u1", Amount: 10, Service: "internet"},
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("Save", ctx, mock.AnythingOfType("*payment.Payment")).Return(db.ErrInternal)
				return NewService(nil, nil, repo, metricsKit)
			},
			expected:    nil,
			expectedErr: db.ErrInternal,
		},
		{
			name: "success",
			req:  CreateRequest{PaymentID: "p1", UserID: "u1", Amount: 10, Service: "internet"},
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("Save", ctx, mock.AnythingOfType("*payment.Payment")).Return(nil)
				return NewService(nil, nil, repo, metricsKit)
			},
			expected:    &Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet", Status: StatusInitialized},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.service()
			p, err := svc.Initialize(ctx, tt.req)
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

func TestPaymentService_MarkPending(t *testing.T) {
	ctx := context.Background()
	metricsKit := observability.NewMetrics()

	var tests = []struct {
		name        string
		paymentID   string
		service     func() ServiceContract
		expectedErr error
	}{
		{
			name:      "repo get error",
			paymentID: "p1",
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("Get", ctx, "p1").Return((*Payment)(nil), db.ErrNotFound)
				return NewService(nil, nil, repo, metricsKit)
			},
			expectedErr: db.ErrNotFound,
		},
		{
			name:      "success publishes and appends",
			paymentID: "p1",
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				pub := new(PublisherMock)
				st := new(StoreMock)
				p := &Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet", Status: StatusInitialized}
				repo.On("Get", ctx, "p1").Return(p, nil)
				repo.On("Save", ctx, mock.AnythingOfType("*payment.Payment")).Return(nil)
				st.On("Append", ctx, "p1", mock.Anything).Return(nil)
				pub.On("Publish", ctx, mock.Anything).Return([]error(nil))
				return NewService(pub, st, repo, metricsKit)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.service()
			err := svc.MarkPending(ctx, tt.paymentID)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPaymentService_MarkRejected(t *testing.T) {
	ctx := context.Background()
	metricsKit := observability.NewMetrics()

	var tests = []struct {
		name        string
		paymentID   string
		reason      string
		service     func() ServiceContract
		expectedErr error
	}{
		{
			name:      "repo get error",
			paymentID: "p1",
			reason:    "x",
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("Get", ctx, "p1").Return((*Payment)(nil), db.ErrNotFound)
				return NewService(nil, nil, repo, metricsKit)
			},
			expectedErr: db.ErrNotFound,
		},
		{
			name:      "success",
			paymentID: "p1",
			reason:    "no funds",
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				pub := new(PublisherMock)
				st := new(StoreMock)
				p := &Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet", Status: StatusInitialized}
				repo.On("Get", ctx, "p1").Return(p, nil)
				repo.On("Save", ctx, mock.AnythingOfType("*payment.Payment")).Return(nil)
				st.On("Append", ctx, "p1", mock.Anything).Return(nil)
				pub.On("Publish", ctx, mock.Anything).Return([]error(nil))
				return NewService(pub, st, repo, metricsKit)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.service()
			err := svc.MarkRejected(ctx, tt.paymentID, tt.reason)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPaymentService_MarkSucceeded(t *testing.T) {
	ctx := context.Background()
	metricsKit := observability.NewMetrics()

	var tests = []struct {
		name        string
		paymentID   string
		gatewayID   string
		service     func() ServiceContract
		expectedErr error
	}{
		{
			name:      "repo get error",
			paymentID: "p1",
			gatewayID: "g1",
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("Get", ctx, "p1").Return((*Payment)(nil), db.ErrNotFound)
				return NewService(nil, nil, repo, metricsKit)
			},
			expectedErr: db.ErrNotFound,
		},
		{
			name:      "success increments metrics",
			paymentID: "p1",
			gatewayID: "g1",
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				pub := new(PublisherMock)
				st := new(StoreMock)
				p := &Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet", Status: StatusInitialized}
				repo.On("Get", ctx, "p1").Return(p, nil)
				repo.On("Save", ctx, mock.AnythingOfType("*payment.Payment")).Return(nil)
				st.On("Append", ctx, "p1", mock.Anything).Return(nil)
				pub.On("Publish", ctx, mock.Anything).Return([]error(nil))
				return NewService(pub, st, repo, metricsKit)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.service()
			err := svc.MarkSucceeded(ctx, tt.paymentID, tt.gatewayID)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPaymentService_MarkFailed(t *testing.T) {
	ctx := context.Background()
	metricsKit := observability.NewMetrics()

	var tests = []struct {
		name        string
		paymentID   string
		reason      string
		service     func() ServiceContract
		expectedErr error
	}{
		{
			name:      "repo get error",
			paymentID: "p1",
			reason:    "x",
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("Get", ctx, "p1").Return((*Payment)(nil), db.ErrNotFound)
				return NewService(nil, nil, repo, metricsKit)
			},
			expectedErr: db.ErrNotFound,
		},
		{
			name:      "success",
			paymentID: "p1",
			reason:    "timeout",
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				pub := new(PublisherMock)
				st := new(StoreMock)
				p := &Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet", Status: StatusInitialized}
				repo.On("Get", ctx, "p1").Return(p, nil)
				repo.On("Save", ctx, mock.AnythingOfType("*payment.Payment")).Return(nil)
				st.On("Append", ctx, "p1", mock.Anything).Return(nil)
				pub.On("Publish", ctx, mock.Anything).Return([]error(nil))
				return NewService(pub, st, repo, metricsKit)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.service()
			err := svc.MarkFailed(ctx, tt.paymentID, tt.reason)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPaymentService_Get(t *testing.T) {
	ctx := context.Background()
	metricsKit := observability.NewMetrics()

	var tests = []struct {
		name        string
		paymentID   string
		service     func() ServiceContract
		expected    *Payment
		expectedErr error
	}{
		{
			name:      "not found",
			paymentID: "p1",
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				repo.On("Get", ctx, "p1").Return((*Payment)(nil), db.ErrNotFound)
				return NewService(nil, nil, repo, metricsKit)
			},
			expected:    nil,
			expectedErr: db.ErrNotFound,
		},
		{
			name:      "success",
			paymentID: "p1",
			service: func() ServiceContract {
				repo := new(RepositoryMock)
				p := &Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet", Status: StatusInitialized}
				repo.On("Get", ctx, "p1").Return(p, nil)
				return NewService(nil, nil, repo, metricsKit)
			},
			expected:    &Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet", Status: StatusInitialized},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := tt.service()
			p, err := svc.Get(ctx, tt.paymentID)
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
