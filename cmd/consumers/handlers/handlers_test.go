package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	"challenge/internal/events"
	"challenge/internal/payment"
	"challenge/kit/broker"
	"challenge/kit/db"
	"challenge/kit/external_payment_gateway"
	"challenge/kit/observability"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPaymentEvent_HandleChargeRequested(t *testing.T) {
	ctx := context.Background()
	logger := observability.NewLogger()

	var tests = []struct {
		name        string
		evt         broker.Event
		handler     func() *PaymentEvent
		expectedErr error
		assert      func(t *testing.T, h *PaymentEvent)
	}{
		{
			name: "unexpected event type",
			evt:  events.PaymentCreated{},
			handler: func() *PaymentEvent {
				bus := new(BusMock)
				gw := new(GatewayMock)
				return NewPaymentEvent(logger, bus, gw, nil)
			},
			expectedErr: errors.New("unexpected"),
		},
		{
			name: "success publishes charge_succeeded",
			evt:  events.PaymentChargeRequested{PaymentID: "p1", UserID: "u1", Amount: 1, Attempt: 1, At: time.Now().UTC()},
			handler: func() *PaymentEvent {
				bus := new(BusMock)
				gw := new(GatewayMock)

				gw.On("Charge", mock.Anything, "p1", int64(1)).Return("gw_p1", nil)
				bus.On("Publish", ctx, mock.MatchedBy(func(e broker.Event) bool {
					evt, ok := e.(events.PaymentChargeSucceeded)
					return ok && evt.PaymentID == "p1" && evt.UserID == "u1" && evt.GatewayID == "gw_p1"
				})).Return([]error(nil))

				return NewPaymentEvent(logger, bus, gw, nil)
			},
			expectedErr: nil,
			assert: func(t *testing.T, h *PaymentEvent) {
				_ = h
			},
		},
		{
			name: "client error publishes charge_failed non retryable",
			evt:  events.PaymentChargeRequested{PaymentID: "p1", UserID: "u1", Amount: 11, Attempt: 5, At: time.Now().UTC()},
			handler: func() *PaymentEvent {
				bus := new(BusMock)
				gw := new(GatewayMock)

				gw.On("Charge", mock.Anything, "p1", int64(11)).Return("", external_payment_gateway.ErrClient)
				bus.On("Publish", ctx, mock.MatchedBy(func(e broker.Event) bool {
					evt, ok := e.(events.PaymentChargeFailed)
					return ok && evt.PaymentID == "p1" && evt.UserID == "u1" && evt.Retryable == false && evt.ErrorCode == "4xx"
				})).Return([]error(nil))

				return NewPaymentEvent(logger, bus, gw, nil)
			},
			expectedErr: nil,
		},
		{
			name: "timeout with attempt=5 sends recovery.requested",
			evt:  events.PaymentChargeRequested{PaymentID: "p1", UserID: "u1", Amount: 5, Attempt: 5, At: time.Now().UTC()},
			handler: func() *PaymentEvent {
				bus := new(BusMock)
				gw := new(GatewayMock)

				gw.On("Charge", mock.Anything, "p1", int64(5)).Return("", external_payment_gateway.ErrTimeout)
				bus.On("Publish", ctx, mock.MatchedBy(func(e broker.Event) bool {
					evt, ok := e.(events.RecoveryRequested)
					return ok && evt.PaymentID == "p1" && evt.UserID == "u1" && evt.Action == "payment.charge" && evt.Attempts == 5 && evt.ErrorCode == "408"
				})).Return([]error(nil))

				return NewPaymentEvent(logger, bus, gw, nil)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.handler()
			err := h.HandleChargeRequested(ctx, tt.evt)
			if tt.expectedErr != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.assert != nil {
				tt.assert(t, h)
			}
		})
	}
}

func TestPaymentFlowEvent_HandleWalletDebited(t *testing.T) {
	ctx := context.Background()
	logger := observability.NewLogger()

	var tests = []struct {
		name        string
		evt         broker.Event
		handler     func() *PaymentFlowEvent
		expectedErr error
	}{
		{
			name:        "unexpected event type",
			evt:         events.PaymentCreated{},
			handler:     func() *PaymentFlowEvent { return NewPaymentFlowEvent(logger, new(BusMock), new(PaymentServiceMock)) },
			expectedErr: errors.New("unexpected"),
		},
		{
			name: "success publishes charge_requested",
			evt:  events.WalletDebited{PaymentID: "p1", UserID: "u1", Amount: 10, At: time.Now().UTC()},
			handler: func() *PaymentFlowEvent {
				bus := new(BusMock)
				ps := new(PaymentServiceMock)
				ps.On("MarkPending", ctx, "p1").Return(nil)
				ps.On("Get", ctx, "p1").Return(&payment.Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet"}, nil)
				bus.On("Publish", ctx, mock.MatchedBy(func(e broker.Event) bool {
					evt, ok := e.(events.PaymentChargeRequested)
					return ok && evt.PaymentID == "p1" && evt.UserID == "u1" && evt.Attempt == 1 && evt.Amount == 10
				})).Return([]error(nil))
				return NewPaymentFlowEvent(logger, bus, ps)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.handler()
			err := h.HandleWalletDebited(ctx, tt.evt)
			if tt.expectedErr != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPaymentResultEvent_HandleChargeFailed(t *testing.T) {
	ctx := context.Background()
	logger := observability.NewLogger()

	var tests = []struct {
		name        string
		evt         broker.Event
		handler     func() *PaymentResultEvent
		expectedErr error
	}{
		{
			name:        "unexpected event type",
			evt:         events.PaymentCreated{},
			handler:     func() *PaymentResultEvent { return NewPaymentResultEvent(logger, new(BusMock), new(PaymentServiceMock)) },
			expectedErr: errors.New("unexpected"),
		},
		{
			name: "success publishes refund_requested",
			evt:  events.PaymentChargeFailed{PaymentID: "p1", UserID: "u1", Reason: "timeout", Retryable: false, ErrorCode: "408", At: time.Now().UTC()},
			handler: func() *PaymentResultEvent {
				bus := new(BusMock)
				ps := new(PaymentServiceMock)
				ps.On("MarkFailed", ctx, "p1", "timeout").Return(nil)
				ps.On("Get", ctx, "p1").Return(&payment.Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet"}, nil)
				bus.On("Publish", ctx, mock.MatchedBy(func(e broker.Event) bool {
					evt, ok := e.(events.WalletRefundRequested)
					return ok && evt.PaymentID == "p1" && evt.UserID == "u1" && evt.Amount == 10
				})).Return([]error(nil))
				return NewPaymentResultEvent(logger, bus, ps)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.handler()
			err := h.HandleChargeFailed(ctx, tt.evt)
			if tt.expectedErr != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWalletEvent_HandleWalletDebitRequested(t *testing.T) {
	ctx := context.Background()
	logger := observability.NewLogger()

	var tests = []struct {
		name        string
		evt         broker.Event
		handler     func() *WalletEvent
		expectedErr error
	}{
		{
			name:        "unexpected event type",
			evt:         events.PaymentCreated{},
			handler:     func() *WalletEvent { return NewWalletEvent(logger, new(BusMock), new(WalletServiceMock)) },
			expectedErr: errors.New("unexpected"),
		},
		{
			name: "db internal on first attempt publishes recovery",
			evt:  events.WalletDebitRequested{PaymentID: "p1", UserID: "u1", Amount: 10, Attempt: 1, At: time.Now().UTC()},
			handler: func() *WalletEvent {
				bus := new(BusMock)
				ws := new(WalletServiceMock)
				ws.On("Debit", ctx, "u1", int64(10)).Return(db.ErrInternal)
				bus.On("Publish", ctx, mock.Anything).Return([]error(nil))
				return NewWalletEvent(logger, bus, ws)
			},
			expectedErr: nil,
		},
		{
			name: "success publishes wallet.debited",
			evt:  events.WalletDebitRequested{PaymentID: "p1", UserID: "u1", Amount: 10, Attempt: 2, At: time.Now().UTC()},
			handler: func() *WalletEvent {
				bus := new(BusMock)
				ws := new(WalletServiceMock)
				ws.On("Debit", ctx, "u1", int64(10)).Return(nil)
				bus.On("Publish", ctx, mock.MatchedBy(func(e broker.Event) bool {
					evt, ok := e.(events.WalletDebited)
					return ok && evt.PaymentID == "p1" && evt.UserID == "u1" && evt.Amount == 10
				})).Return([]error(nil))
				return NewWalletEvent(logger, bus, ws)
			},
			expectedErr: nil,
		},
		{
			name: "non-internal error publishes wallet.debit_rejected",
			evt:  events.WalletDebitRequested{PaymentID: "p1", UserID: "u1", Amount: 10, Attempt: 1, At: time.Now().UTC()},
			handler: func() *WalletEvent {
				bus := new(BusMock)
				ws := new(WalletServiceMock)
				ws.On("Debit", ctx, "u1", int64(10)).Return(errors.New("insufficient funds"))
				bus.On("Publish", ctx, mock.MatchedBy(func(e broker.Event) bool {
					evt, ok := e.(events.WalletDebitRejected)
					return ok && evt.PaymentID == "p1" && evt.UserID == "u1" && evt.Reason == "insufficient funds"
				})).Return([]error(nil))
				return NewWalletEvent(logger, bus, ws)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.handler()
			err := h.HandleWalletDebitRequested(ctx, tt.evt)
			if tt.expectedErr != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestRecoveryEvent_HandleRecoveryRequested(t *testing.T) {
	ctx := context.Background()
	logger := observability.NewLogger()

	var tests = []struct {
		name        string
		evt         broker.Event
		handler     func() *RecoveryEvent
		expectedErr error
	}{
		{
			name:        "unexpected event type",
			evt:         events.PaymentCreated{},
			handler:     func() *RecoveryEvent { return NewRecoveryEvent(logger, new(BusMock), new(PaymentServiceMock), time.Second, func(ctx context.Context, d time.Duration) error { return nil }) },
			expectedErr: errors.New("unexpected"),
		},
		{
			name: "unknown action",
			evt:  events.RecoveryRequested{PaymentID: "p1", UserID: "u1", Action: "unknown", Attempts: 1, At: time.Now().UTC()},
			handler: func() *RecoveryEvent {
				bus := new(BusMock)
				ps := new(PaymentServiceMock)
				ps.On("Get", ctx, "p1").Return(&payment.Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet"}, nil)
				return NewRecoveryEvent(logger, bus, ps, 1*time.Millisecond, func(ctx context.Context, d time.Duration) error { return nil })
			},
			expectedErr: errors.New("unknown"),
		},
		{
			name: "payment.charge republishes charge_requested",
			evt:  events.RecoveryRequested{PaymentID: "p1", UserID: "u1", Action: "payment.charge", Attempts: 5, At: time.Now().UTC()},
			handler: func() *RecoveryEvent {
				bus := new(BusMock)
				ps := new(PaymentServiceMock)
				ps.On("Get", ctx, "p1").Return(&payment.Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet"}, nil)
				bus.On("Publish", ctx, mock.MatchedBy(func(e broker.Event) bool {
					evt, ok := e.(events.PaymentChargeRequested)
					return ok && evt.PaymentID == "p1" && evt.Attempt == 6
				})).Return([]error(nil))
				return NewRecoveryEvent(logger, bus, ps, 1*time.Millisecond, func(ctx context.Context, d time.Duration) error { return nil })
			},
			expectedErr: nil,
		},
		{
			name: "wallet.debit republishes wallet.debit_requested",
			evt:  events.RecoveryRequested{PaymentID: "p1", UserID: "u1", Action: "wallet.debit", Attempts: 1, At: time.Now().UTC()},
			handler: func() *RecoveryEvent {
				bus := new(BusMock)
				ps := new(PaymentServiceMock)
				ps.On("Get", ctx, "p1").Return(&payment.Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet"}, nil)
				bus.On("Publish", ctx, mock.MatchedBy(func(e broker.Event) bool {
					evt, ok := e.(events.WalletDebitRequested)
					return ok && evt.PaymentID == "p1" && evt.UserID == "u1" && evt.Amount == 10 && evt.Attempt == 2
				})).Return([]error(nil))
				return NewRecoveryEvent(logger, bus, ps, 1*time.Millisecond, func(ctx context.Context, d time.Duration) error { return nil })
			},
			expectedErr: nil,
		},
		{
			name: "sleep error propagates",
			evt:  events.RecoveryRequested{PaymentID: "p1", UserID: "u1", Action: "payment.charge", Attempts: 1, At: time.Now().UTC()},
			handler: func() *RecoveryEvent {
				ps := new(PaymentServiceMock)
				return NewRecoveryEvent(logger, new(BusMock), ps, 1*time.Millisecond, func(ctx context.Context, d time.Duration) error {
					return context.Canceled
				})
			},
			expectedErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.handler()
			err := h.HandleRecoveryRequested(ctx, tt.evt)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAuditEvent_HandleAny(t *testing.T) {
	ctx := context.Background()

	var tests = []struct {
		name    string
		evt     broker.Event
		handler func() *AuditEvent
	}{
		{
			name: "nil auditor does nothing",
			evt:  events.PaymentCreated{PaymentID: "p1", UserID: "u1", Amount: 10, At: time.Now().UTC()},
			handler: func() *AuditEvent {
				return NewAuditEvent(nil)
			},
		},
		{
			name: "records event",
			evt:  events.PaymentCreated{PaymentID: "p1", UserID: "u1", Amount: 10, At: time.Now().UTC()},
			handler: func() *AuditEvent {
				a := new(AuditorMock)
				a.On("Record", mock.Anything, "payment.created", mock.Anything).Return()
				return NewAuditEvent(a)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.handler()
			require.NoError(t, h.HandleAny(ctx, tt.evt))
		})
	}
}

func TestMetricsEvent_HandleAny(t *testing.T) {
	ctx := context.Background()

	var tests = []struct {
		name    string
		evt     broker.Event
		handler func() *MetricsEvent
	}{
		{
			name: "nil metrics",
			evt:  events.PaymentCreated{},
			handler: func() *MetricsEvent {
				return NewMetricsEvent(nil)
			},
		},
		{
			name: "payment created increments",
			evt:  events.PaymentCreated{},
			handler: func() *MetricsEvent {
				m := new(MetricsMock)
				m.On("PaymentsCreatedAdd", int64(1)).Return()
				return NewMetricsEvent(m)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.handler()
			require.NoError(t, h.HandleAny(ctx, tt.evt))
		})
	}
}

func TestPaymentResultEvent_HandleChargeSucceeded(t *testing.T) {
	ctx := context.Background()
	logger := observability.NewLogger()

	var tests = []struct {
		name        string
		evt         broker.Event
		handler     func() *PaymentResultEvent
		expectedErr error
	}{
		{
			name:        "unexpected event type",
			evt:         events.PaymentCreated{},
			handler:     func() *PaymentResultEvent { return NewPaymentResultEvent(logger, new(BusMock), new(PaymentServiceMock)) },
			expectedErr: errors.New("unexpected"),
		},
		{
			name: "success calls MarkSucceeded",
			evt:  events.PaymentChargeSucceeded{PaymentID: "p1", UserID: "u1", GatewayID: "g1", At: time.Now().UTC()},
			handler: func() *PaymentResultEvent {
				bus := new(BusMock)
				ps := new(PaymentServiceMock)
				ps.On("MarkSucceeded", ctx, "p1", "g1").Return(nil)
				return NewPaymentResultEvent(logger, bus, ps)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.handler()
			err := h.HandleChargeSucceeded(ctx, tt.evt)
			if tt.expectedErr != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestPaymentFlowEvent_HandleWalletDebitRejected(t *testing.T) {
	ctx := context.Background()
	logger := observability.NewLogger()

	var tests = []struct {
		name        string
		evt         broker.Event
		handler     func() *PaymentFlowEvent
		expectedErr error
	}{
		{
			name:        "unexpected event type",
			evt:         events.PaymentCreated{},
			handler:     func() *PaymentFlowEvent { return NewPaymentFlowEvent(logger, new(BusMock), new(PaymentServiceMock)) },
			expectedErr: errors.New("unexpected"),
		},
		{
			name: "calls MarkRejected",
			evt:  events.WalletDebitRejected{PaymentID: "p1", UserID: "u1", Reason: "insufficient", At: time.Now().UTC()},
			handler: func() *PaymentFlowEvent {
				ps := new(PaymentServiceMock)
				ps.On("MarkRejected", ctx, "p1", "insufficient").Return(nil)
				return NewPaymentFlowEvent(logger, new(BusMock), ps)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.handler()
			err := h.HandleWalletDebitRejected(ctx, tt.evt)
			if tt.expectedErr != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWalletEvent_HandlePaymentInitialized(t *testing.T) {
	ctx := context.Background()
	logger := observability.NewLogger()

	var tests = []struct {
		name        string
		evt         broker.Event
		handler     func() *WalletEvent
		expectedErr error
	}{
		{
			name:        "unexpected event type",
			evt:         events.PaymentCreated{},
			handler:     func() *WalletEvent { return NewWalletEvent(logger, new(BusMock), new(WalletServiceMock)) },
			expectedErr: errors.New("unexpected"),
		},
		{
			name: "publishes wallet.debit_requested",
			evt:  events.PaymentInitialized{PaymentID: "p1", UserID: "u1", Amount: 10, Service: "internet", At: time.Now().UTC()},
			handler: func() *WalletEvent {
				bus := new(BusMock)
				bus.On("Publish", ctx, mock.MatchedBy(func(e broker.Event) bool {
					evt, ok := e.(events.WalletDebitRequested)
					return ok && evt.PaymentID == "p1" && evt.UserID == "u1" && evt.Amount == 10 && evt.Attempt == 1
				})).Return([]error(nil))
				return NewWalletEvent(logger, bus, new(WalletServiceMock))
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.handler()
			err := h.HandlePaymentInitialized(ctx, tt.evt)
			if tt.expectedErr != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWalletEvent_HandleWalletRefundRequested(t *testing.T) {
	ctx := context.Background()
	logger := observability.NewLogger()

	var tests = []struct {
		name        string
		evt         broker.Event
		handler     func() *WalletEvent
		expectedErr error
	}{
		{
			name:        "unexpected event type",
			evt:         events.PaymentCreated{},
			handler:     func() *WalletEvent { return NewWalletEvent(logger, new(BusMock), new(WalletServiceMock)) },
			expectedErr: errors.New("unexpected"),
		},
		{
			name: "refund error propagates",
			evt:  events.WalletRefundRequested{PaymentID: "p1", UserID: "u1", Amount: 10, At: time.Now().UTC()},
			handler: func() *WalletEvent {
				ws := new(WalletServiceMock)
				ws.On("Refund", ctx, "u1", int64(10)).Return(db.ErrInternal)
				return NewWalletEvent(logger, new(BusMock), ws)
			},
			expectedErr: db.ErrInternal,
		},
		{
			name: "success publishes wallet.refunded",
			evt:  events.WalletRefundRequested{PaymentID: "p1", UserID: "u1", Amount: 10, At: time.Now().UTC()},
			handler: func() *WalletEvent {
				bus := new(BusMock)
				ws := new(WalletServiceMock)
				ws.On("Refund", ctx, "u1", int64(10)).Return(nil)
				bus.On("Publish", ctx, mock.MatchedBy(func(e broker.Event) bool {
					evt, ok := e.(events.WalletRefunded)
					return ok && evt.PaymentID == "p1" && evt.UserID == "u1" && evt.Amount == 10
				})).Return([]error(nil))
				return NewWalletEvent(logger, bus, ws)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.handler()
			err := h.HandleWalletRefundRequested(ctx, tt.evt)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestWalletEvent_HandleWalletDebitedAndRefunded(t *testing.T) {
	ctx := context.Background()
	logger := observability.NewLogger()
	h := NewWalletEvent(logger, new(BusMock), new(WalletServiceMock))
	require.NoError(t, h.HandleWalletDebited(ctx, events.WalletDebited{PaymentID: "p1", UserID: "u1", Amount: 10, At: time.Now().UTC()}))
	require.NoError(t, h.HandleWalletRefunded(ctx, events.WalletRefunded{PaymentID: "p1", UserID: "u1", Amount: 10, At: time.Now().UTC()}))
}

func TestNotificationEvent_HandlePaymentFailed(t *testing.T) {
	ctx := context.Background()

	var tests = []struct {
		name        string
		evt         broker.Event
		handler     func() *NotificationEvent
		expectedErr error
	}{
		{
			name:        "nil notifier",
			evt:         events.PaymentFailed{PaymentID: "p1", UserID: "u1", Reason: "x", At: time.Now().UTC()},
			handler:     func() *NotificationEvent { return NewNotificationEvent(nil) },
			expectedErr: nil,
		},
		{
			name: "notifies",
			evt:  events.PaymentFailed{PaymentID: "p1", UserID: "u1", Reason: "x", At: time.Now().UTC()},
			handler: func() *NotificationEvent {
				n := new(NotifierMock)
				n.On("Notify", mock.Anything, "u1", "payment failed").Return()
				return NewNotificationEvent(n)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.handler()
			err := h.HandlePaymentFailed(ctx, tt.evt)
			if tt.expectedErr != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNotificationEvent_HandlePaymentCompleted(t *testing.T) {
	ctx := context.Background()

	var tests = []struct {
		name        string
		evt         broker.Event
		handler     func() *NotificationEvent
		expectedErr error
	}{
		{
			name:        "nil notifier",
			evt:         events.PaymentSucceeded{PaymentID: "p1", UserID: "u1", GatewayID: "g1", At: time.Now().UTC()},
			handler:     func() *NotificationEvent { return NewNotificationEvent(nil) },
			expectedErr: nil,
		},
		{
			name: "notifies",
			evt:  events.PaymentSucceeded{PaymentID: "p1", UserID: "u1", GatewayID: "g1", At: time.Now().UTC()},
			handler: func() *NotificationEvent {
				n := new(NotifierMock)
				n.On("Notify", mock.Anything, "u1", "payment completed").Return()
				return NewNotificationEvent(n)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := tt.handler()
			err := h.HandlePaymentCompleted(ctx, tt.evt)
			if tt.expectedErr != nil {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
