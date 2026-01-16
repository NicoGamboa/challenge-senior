package handlers

import (
	"context"

	"challenge/internal/payment"
	"challenge/internal/wallet"
	"challenge/kit/broker"
	"challenge/kit/external_payment_gateway"

	"github.com/stretchr/testify/mock"
)

type BusMock struct {
	mock.Mock
	BusContract
}

func (m *BusMock) Publish(ctx context.Context, evt broker.Event) []error {
	args := m.Called(ctx, evt)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]error)
}

type GatewayMock struct {
	mock.Mock
	external_payment_gateway.Gateway
}

func (m *GatewayMock) Charge(ctx context.Context, paymentID string, amount int64) (string, error) {
	args := m.Called(ctx, paymentID, amount)
	return args.String(0), args.Error(1)
}

type PaymentServiceMock struct {
	mock.Mock
	payment.ServiceContract
}

func (m *PaymentServiceMock) Initialize(ctx context.Context, req payment.CreateRequest) (*payment.Payment, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.Payment), args.Error(1)
}

func (m *PaymentServiceMock) MarkPending(ctx context.Context, paymentID string) error {
	args := m.Called(ctx, paymentID)
	return args.Error(0)
}

func (m *PaymentServiceMock) MarkRejected(ctx context.Context, paymentID, reason string) error {
	args := m.Called(ctx, paymentID, reason)
	return args.Error(0)
}

func (m *PaymentServiceMock) MarkSucceeded(ctx context.Context, paymentID, gatewayID string) error {
	args := m.Called(ctx, paymentID, gatewayID)
	return args.Error(0)
}

func (m *PaymentServiceMock) MarkFailed(ctx context.Context, paymentID, reason string) error {
	args := m.Called(ctx, paymentID, reason)
	return args.Error(0)
}

func (m *PaymentServiceMock) Get(ctx context.Context, paymentID string) (*payment.Payment, error) {
	args := m.Called(ctx, paymentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*payment.Payment), args.Error(1)
}

type WalletServiceMock struct {
	mock.Mock
	wallet.ServiceContract
}

func (m *WalletServiceMock) Credit(ctx context.Context, userID string, amount int64) error {
	args := m.Called(ctx, userID, amount)
	return args.Error(0)
}

func (m *WalletServiceMock) Debit(ctx context.Context, userID string, amount int64) error {
	args := m.Called(ctx, userID, amount)
	return args.Error(0)
}

func (m *WalletServiceMock) Refund(ctx context.Context, userID string, amount int64) error {
	args := m.Called(ctx, userID, amount)
	return args.Error(0)
}

func (m *WalletServiceMock) Balance(ctx context.Context, userID string) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

type AuditorMock struct {
	mock.Mock
	AuditorContract
}

func (m *AuditorMock) Record(ctx context.Context, eventName string, fields map[string]any) {
	m.Called(ctx, eventName, fields)
}

type NotifierMock struct {
	mock.Mock
	NotifierContract
}

func (m *NotifierMock) Notify(ctx context.Context, userID string, msg string) {
	m.Called(ctx, userID, msg)
}

type MetricsMock struct {
	mock.Mock
	MetricsContract
}

func (m *MetricsMock) PaymentsCreatedAdd(n int64)  { m.Called(n) }
func (m *MetricsMock) PaymentsSucceededAdd(n int64) { m.Called(n) }
func (m *MetricsMock) PaymentsFailedAdd(n int64)    { m.Called(n) }
func (m *MetricsMock) WalletDebitsAdd(n int64)      { m.Called(n) }
func (m *MetricsMock) WalletRefundsAdd(n int64)     { m.Called(n) }
