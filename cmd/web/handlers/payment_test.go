package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"challenge/cmd/web/validator"
	"challenge/internal/events"
	"challenge/internal/health"
	"challenge/internal/payment"
	"challenge/internal/readmodels"
	"challenge/kit/broker"
	"challenge/kit/db"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type paymentBusMock struct{ mock.Mock }

func (m *paymentBusMock) Publish(ctx context.Context, evt broker.Event) []error {
	args := m.Called(ctx, evt)
	if v := args.Get(0); v != nil {
		return v.([]error)
	}
	return nil
}

type paymentStoreMock struct{ mock.Mock }

func (m *paymentStoreMock) Append(ctx context.Context, aggregateID string, evt broker.Event) error {
	args := m.Called(ctx, aggregateID, evt)
	return args.Error(0)
}

type paymentServiceMock struct{ mock.Mock }

func (m *paymentServiceMock) Initialize(ctx context.Context, req payment.CreateRequest) (*payment.Payment, error) {
	args := m.Called(ctx, req)
	p, _ := args.Get(0).(*payment.Payment)
	return p, args.Error(1)
}

func (m *paymentServiceMock) Get(ctx context.Context, paymentID string) (*payment.Payment, error) {
	args := m.Called(ctx, paymentID)
	p, _ := args.Get(0).(*payment.Payment)
	return p, args.Error(1)
}

type paymentHealthMock struct{ mock.Mock }

func (m *paymentHealthMock) Check(ctx context.Context) health.Result {
	args := m.Called(ctx)
	return args.Get(0).(health.Result)
}

type paymentReadModelMock struct{ mock.Mock }

func (m *paymentReadModelMock) GetPayment(paymentID string) (readmodels.PaymentView, bool) {
	args := m.Called(paymentID)
	v, _ := args.Get(0).(readmodels.PaymentView)
	return v, args.Bool(1)
}

func TestPayment_Create(t *testing.T) {
	mkReq := func(t *testing.T, body any) *http.Request {
		t.Helper()
		b, err := json.Marshal(body)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	var tests = []struct {
		name       string
		req        func(t *testing.T) *http.Request
		handler    func() *Payment
		assertResp func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "invalid json",
			req: func(t *testing.T) *http.Request {
				_ = t
				return httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader([]byte("{")))
			},
			handler: func() *Payment {
				return NewPayment(validator.NewJSON(), new(paymentBusMock), new(paymentStoreMock), new(paymentServiceMock), nil, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
		{
			name: "health down returns 503",
			req: func(t *testing.T) *http.Request {
				return mkReq(t, createPaymentReq{PaymentID: "p1", UserID: "u1", Amount: 10, Service: "internet"})
			},
			handler: func() *Payment {
				hm := new(paymentHealthMock)
				hm.On("Check", mock.Anything).Return(health.Result{OK: false, Checks: map[string]string{"db": "down"}})
				return NewPayment(validator.NewJSON(), new(paymentBusMock), new(paymentStoreMock), new(paymentServiceMock), hm, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusServiceUnavailable, rr.Code)
				var got map[string]any
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
				require.Equal(t, "down", got["status"])
			},
		},
		{
			name: "initialize invalid returns 400",
			req: func(t *testing.T) *http.Request {
				return mkReq(t, createPaymentReq{PaymentID: "", UserID: "u1", Amount: 10, Service: "internet"})
			},
			handler: func() *Payment {
				ps := new(paymentServiceMock)
				ps.On("Initialize", mock.Anything, mock.Anything).Return((*payment.Payment)(nil), db.ErrInvalid)
				return NewPayment(validator.NewJSON(), new(paymentBusMock), new(paymentStoreMock), ps, nil, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
		{
			name: "initialize internal returns 500",
			req: func(t *testing.T) *http.Request {
				return mkReq(t, createPaymentReq{PaymentID: "p1", UserID: "u1", Amount: 10, Service: "internet"})
			},
			handler: func() *Payment {
				ps := new(paymentServiceMock)
				ps.On("Initialize", mock.Anything, mock.Anything).Return((*payment.Payment)(nil), db.ErrInternal)
				return NewPayment(validator.NewJSON(), new(paymentBusMock), new(paymentStoreMock), ps, nil, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
			},
		},
		{
			name: "success returns 202 and publishes/appends events",
			req: func(t *testing.T) *http.Request {
				return mkReq(t, createPaymentReq{PaymentID: "p1", UserID: "u1", Amount: 10, Service: "internet"})
			},
			handler: func() *Payment {
				bus := new(paymentBusMock)
				store := new(paymentStoreMock)
				ps := new(paymentServiceMock)
				createdMatcher := mock.MatchedBy(func(e broker.Event) bool {
					ce, ok := e.(events.PaymentCreated)
					return ok && ce.PaymentID == "p1" && ce.UserID == "u1" && ce.Amount == 10 && ce.Service == "internet"
				})
				initializedMatcher := mock.MatchedBy(func(e broker.Event) bool {
					ie, ok := e.(events.PaymentInitialized)
					return ok && ie.PaymentID == "p1" && ie.UserID == "u1" && ie.Amount == 10 && ie.Service == "internet"
				})
				ps.On("Initialize", mock.Anything, mock.Anything).Return(&payment.Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet", Status: payment.StatusInitialized}, nil)
				store.On("Append", mock.Anything, "p1", createdMatcher).Return(nil)
				store.On("Append", mock.Anything, "p1", initializedMatcher).Return(nil)
				bus.On("Publish", mock.Anything, createdMatcher).Return([]error(nil))
				bus.On("Publish", mock.Anything, initializedMatcher).Return([]error(nil))
				return NewPayment(validator.NewJSON(), bus, store, ps, nil, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusAccepted, rr.Code)
				var got map[string]any
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
				require.Equal(t, "p1", got["payment_id"])
				require.NotEmpty(t, got["status"])
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rr := httptest.NewRecorder()
			h := tt.handler()
			h.Create(rr, tt.req(t))
			tt.assertResp(t, rr)
		})
	}
}

func TestPayment_Get(t *testing.T) {
	ctx := context.Background()

	var tests = []struct {
		name       string
		url        string
		handler    func() *Payment
		assertResp func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "missing payment_id",
			url:  "/payments/",
			handler: func() *Payment {
				return NewPayment(validator.NewJSON(), new(paymentBusMock), new(paymentStoreMock), new(paymentServiceMock), nil, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
		{
			name: "read model hit",
			url:  "/payments/p1",
			handler: func() *Payment {
				rm := new(paymentReadModelMock)
				rm.On("GetPayment", "p1").Return(readmodels.PaymentView{PaymentID: "p1", UserID: "u1", Amount: 10, Service: "internet", Status: payment.StatusSucceeded, Reason: "", GatewayID: "gw1"}, true)
				return NewPayment(validator.NewJSON(), new(paymentBusMock), new(paymentStoreMock), new(paymentServiceMock), nil, rm)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				var got map[string]any
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
				require.Equal(t, "p1", got["payment_id"])
				require.Equal(t, "u1", got["user_id"])
				require.Equal(t, "internet", got["service"])
			},
		},
		{
			name: "service not found",
			url:  "/payments/p1",
			handler: func() *Payment {
				ps := new(paymentServiceMock)
				ps.On("Get", mock.Anything, "p1").Return((*payment.Payment)(nil), db.ErrNotFound)
				return NewPayment(validator.NewJSON(), new(paymentBusMock), new(paymentStoreMock), ps, nil, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rr.Code)
			},
		},
		{
			name: "service internal error",
			url:  "/payments/p1",
			handler: func() *Payment {
				ps := new(paymentServiceMock)
				ps.On("Get", mock.Anything, "p1").Return((*payment.Payment)(nil), db.ErrInternal)
				return NewPayment(validator.NewJSON(), new(paymentBusMock), new(paymentStoreMock), ps, nil, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
			},
		},
		{
			name: "service success",
			url:  "/payments/p1",
			handler: func() *Payment {
				ps := new(paymentServiceMock)
				ps.On("Get", mock.Anything, "p1").Return(&payment.Payment{ID: "p1", UserID: "u1", Amount: 10, Service: "internet", Status: payment.StatusPending, Reason: "", GatewayID: ""}, nil)
				return NewPayment(validator.NewJSON(), new(paymentBusMock), new(paymentStoreMock), ps, nil, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				var got map[string]any
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
				require.Equal(t, "p1", got["payment_id"])
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req = req.WithContext(ctx)
			h := tt.handler()
			h.Get(rr, req)
			tt.assertResp(t, rr)
		})
	}
}

var _ = errors.New
