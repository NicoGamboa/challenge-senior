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
	"challenge/internal/readmodels"
	"challenge/kit/broker"
	"challenge/kit/db"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type walletBusMock struct{ mock.Mock }

func (m *walletBusMock) Publish(ctx context.Context, evt broker.Event) []error {
	args := m.Called(ctx, evt)
	if v := args.Get(0); v != nil {
		return v.([]error)
	}
	return nil
}

type walletStoreMock struct{ mock.Mock }

func (m *walletStoreMock) Append(ctx context.Context, aggregateID string, evt broker.Event) error {
	args := m.Called(ctx, aggregateID, evt)
	return args.Error(0)
}

type walletServiceMock struct{ mock.Mock }

func (m *walletServiceMock) Credit(ctx context.Context, userID string, amount int64) error {
	args := m.Called(ctx, userID, amount)
	return args.Error(0)
}

func (m *walletServiceMock) Balance(ctx context.Context, userID string) (int64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(int64), args.Error(1)
}

type walletReadModelMock struct{ mock.Mock }

func (m *walletReadModelMock) GetWallet(userID string) (readmodels.WalletView, bool) {
	args := m.Called(userID)
	v, _ := args.Get(0).(readmodels.WalletView)
	return v, args.Bool(1)
}

func TestWallet_Credit(t *testing.T) {
	mkReq := func(t *testing.T, body any) *http.Request {
		t.Helper()
		b, err := json.Marshal(body)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/wallet/credit", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	var tests = []struct {
		name       string
		req        func(t *testing.T) *http.Request
		handler    func() *Wallet
		assertResp func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "invalid json",
			req: func(t *testing.T) *http.Request {
				_ = t
				return httptest.NewRequest(http.MethodPost, "/wallet/credit", bytes.NewReader([]byte("{")))
			},
			handler: func() *Wallet {
				return NewWallet(validator.NewJSON(), new(walletBusMock), new(walletStoreMock), new(walletServiceMock), nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
		{
			name: "credit invalid returns 400",
			req: func(t *testing.T) *http.Request {
				return mkReq(t, creditReq{UserID: "", Amount: 10})
			},
			handler: func() *Wallet {
				ws := new(walletServiceMock)
				ws.On("Credit", mock.Anything, "", int64(10)).Return(db.ErrInvalid)
				return NewWallet(validator.NewJSON(), new(walletBusMock), new(walletStoreMock), ws, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
		{
			name: "credit internal returns 500",
			req: func(t *testing.T) *http.Request {
				return mkReq(t, creditReq{UserID: "u1", Amount: 10})
			},
			handler: func() *Wallet {
				ws := new(walletServiceMock)
				ws.On("Credit", mock.Anything, "u1", int64(10)).Return(db.ErrInternal)
				return NewWallet(validator.NewJSON(), new(walletBusMock), new(walletStoreMock), ws, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
			},
		},
		{
			name: "success returns 204 and writes to store/bus",
			req: func(t *testing.T) *http.Request {
				return mkReq(t, creditReq{UserID: "u1", Amount: 10})
			},
			handler: func() *Wallet {
				bus := new(walletBusMock)
				store := new(walletStoreMock)
				ws := new(walletServiceMock)
				ws.On("Credit", mock.Anything, "u1", int64(10)).Return(nil)
				store.On("Append", mock.Anything, "u1", mock.MatchedBy(func(e broker.Event) bool {
					ce, ok := e.(events.WalletCredited)
					return ok && ce.UserID == "u1" && ce.Amount == 10
				})).Return(nil)
				bus.On("Publish", mock.Anything, mock.MatchedBy(func(e broker.Event) bool {
					ce, ok := e.(events.WalletCredited)
					return ok && ce.UserID == "u1" && ce.Amount == 10
				})).Return([]error(nil))
				return NewWallet(validator.NewJSON(), bus, store, ws, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, rr.Code)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rr := httptest.NewRecorder()
			h := tt.handler()
			h.Credit(rr, tt.req(t))
			tt.assertResp(t, rr)
		})
	}
}

func TestWallet_Balance(t *testing.T) {
	ctx := context.Background()

	var tests = []struct {
		name       string
		url        string
		handler    func() *Wallet
		assertResp func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "missing user_id",
			url:  "/wallet/",
			handler: func() *Wallet {
				return NewWallet(validator.NewJSON(), new(walletBusMock), new(walletStoreMock), new(walletServiceMock), nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rr.Code)
			},
		},
		{
			name: "read model hit",
			url:  "/wallet/u1",
			handler: func() *Wallet {
				rm := new(walletReadModelMock)
				rm.On("GetWallet", "u1").Return(readmodels.WalletView{UserID: "u1", Balance: 99}, true)
				return NewWallet(validator.NewJSON(), new(walletBusMock), new(walletStoreMock), new(walletServiceMock), rm)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				var got map[string]any
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
				require.Equal(t, "u1", got["user_id"])
				require.Equal(t, float64(99), got["balance"])
			},
		},
		{
			name: "service error returns 500",
			url:  "/wallet/u1",
			handler: func() *Wallet {
				ws := new(walletServiceMock)
				ws.On("Balance", mock.Anything, "u1").Return(int64(0), errors.New("boom"))
				return NewWallet(validator.NewJSON(), new(walletBusMock), new(walletStoreMock), ws, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rr.Code)
			},
		},
		{
			name: "success",
			url:  "/wallet/u1",
			handler: func() *Wallet {
				ws := new(walletServiceMock)
				ws.On("Balance", mock.Anything, "u1").Return(int64(10), nil)
				return NewWallet(validator.NewJSON(), new(walletBusMock), new(walletStoreMock), ws, nil)
			},
			assertResp: func(t *testing.T, rr *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rr.Code)
				var got map[string]any
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
				require.Equal(t, "u1", got["user_id"])
				require.Equal(t, float64(10), got["balance"])
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
			h.Balance(rr, req)
			tt.assertResp(t, rr)
		})
	}
}
