package db

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
)

type MockClient struct {
	mu sync.Mutex

	wallets map[string]int64
	payments map[string]map[string]any

	walletsPersistPath string
}

type MockOption func(*MockClient) error

func NewMockClient(opts ...MockOption) (*MockClient, error) {
	c := &MockClient{
		wallets:  make(map[string]int64),
		payments: make(map[string]map[string]any),
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func WithWalletsJSONFile(path string) MockOption {
	return func(c *MockClient) error {
		b, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return errors.Join(ErrInternal, err)
		}
		if len(b) == 0 {
			return nil
		}
		var m map[string]int64
		if err := json.Unmarshal(b, &m); err != nil {
			return errors.Join(ErrInternal, err)
		}
		c.wallets = m
		return nil
	}
}

func WithWalletsJSONPersistence(path string) MockOption {
	return func(c *MockClient) error {
		c.walletsPersistPath = path
		return nil
	}
}

func (c *MockClient) persistWalletsLocked() error {
	if c.walletsPersistPath == "" {
		return nil
	}
	dir := filepath.Dir(c.walletsPersistPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("layer=client component=db method=persistWalletsLocked path=%s err=%v", c.walletsPersistPath, err)
		return errors.Join(ErrInternal, err)
	}

	b, err := json.MarshalIndent(c.wallets, "", "  ")
	if err != nil {
		log.Printf("layer=client component=db method=persistWalletsLocked path=%s err=%v", c.walletsPersistPath, err)
		return errors.Join(ErrInternal, err)
	}
	b = append(b, '\n')

	tmp := c.walletsPersistPath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		log.Printf("layer=client component=db method=persistWalletsLocked path=%s err=%v", c.walletsPersistPath, err)
		return errors.Join(ErrInternal, err)
	}
	if err := os.Rename(tmp, c.walletsPersistPath); err != nil {
		log.Printf("layer=client component=db method=persistWalletsLocked path=%s err=%v", c.walletsPersistPath, err)
		return errors.Join(ErrInternal, err)
	}
	return nil
}

type mockRow struct {
	vals []any
	err  error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.vals) {
		return errors.Join(ErrInternal, errors.New("scan arg mismatch"))
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			v, _ := r.vals[i].(string)
			*d = v
		case *int64:
			v, _ := r.vals[i].(int64)
			*d = v
		default:
			dv := reflect.ValueOf(dest[i])
			if dv.Kind() != reflect.Ptr || dv.IsNil() {
				return errors.Join(ErrInternal, errors.New("unsupported scan type"))
			}
			ev := dv.Elem()
			switch ev.Kind() {
			case reflect.String:
				if s, ok := r.vals[i].(string); ok {
					ev.SetString(s)
					continue
				}
			case reflect.Int64:
				if n, ok := r.vals[i].(int64); ok {
					ev.SetInt(n)
					continue
				}
			}
			return errors.Join(ErrInternal, errors.New("unsupported scan type"))
		}
	}
	return nil
}

func (c *MockClient) Exec(ctx context.Context, query string, args ...any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	toString := func(v any) (string, bool) {
		if v == nil {
			return "", false
		}
		s, ok := v.(string)
		if ok {
			return s, true
		}
		rv := reflect.ValueOf(v)
		if rv.IsValid() && rv.Kind() == reflect.String {
			return rv.String(), true
		}
		return "", false
	}

	switch query {
	case "INSERT INTO wallets (user_id, balance) VALUES (?, ?) ON DUPLICATE KEY UPDATE balance = ?":
		if len(args) != 3 {
			return errors.Join(ErrInternal, errors.New("invalid args"))
		}
		userID, _ := toString(args[0])
		bal, _ := args[1].(int64)
		c.wallets[userID] = bal
		return c.persistWalletsLocked()
	case "UPDATE wallets SET balance = balance - ? WHERE user_id = ? AND balance >= ?":
		if len(args) != 3 {
			return errors.Join(ErrInternal, errors.New("invalid args"))
		}
		amount, _ := args[0].(int64)
		userID, _ := toString(args[1])
		min, _ := args[2].(int64)
		cur := c.wallets[userID]
		if cur < min {
			return ErrConflict
		}
		c.wallets[userID] = cur - amount
		return c.persistWalletsLocked()
	case "INSERT INTO payments (payment_id, user_id, amount, service, status, reason, gateway_id) VALUES (?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE user_id=?, amount=?, service=?, status=?, reason=?, gateway_id=?":
		if len(args) != 13 {
			return errors.Join(ErrInternal, errors.New("invalid args"))
		}
		paymentID, _ := toString(args[0])
		userID, _ := toString(args[1])
		service, _ := toString(args[3])
		status, _ := toString(args[4])
		reason, _ := toString(args[5])
		gatewayID, _ := toString(args[6])
		c.payments[paymentID] = map[string]any{
			"payment_id": paymentID,
			"user_id":    userID,
			"amount":     args[2].(int64),
			"service":    service,
			"status":     status,
			"reason":     reason,
			"gateway_id": gatewayID,
		}
		return nil
	default:
		log.Printf("layer=client component=db method=Exec err=unsupported query query=%q", query)
		return errors.Join(ErrInternal, errors.New("unsupported query"))
	}
}

func (c *MockClient) QueryRow(ctx context.Context, query string, args ...any) (Row, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch query {
	case "SELECT balance FROM wallets WHERE user_id = ?":
		if len(args) != 1 {
			return &mockRow{err: errors.Join(ErrInternal, errors.New("invalid args"))}, nil
		}
		userID, _ := args[0].(string)
		bal, ok := c.wallets[userID]
		if !ok {
			return &mockRow{err: ErrNotFound}, nil
		}
		return &mockRow{vals: []any{bal}}, nil
	case "SELECT payment_id, user_id, amount, service, status, reason, gateway_id FROM payments WHERE payment_id = ?":
		if len(args) != 1 {
			return &mockRow{err: errors.Join(ErrInternal, errors.New("invalid args"))}, nil
		}
		paymentID, _ := args[0].(string)
		row, ok := c.payments[paymentID]
		if !ok {
			return &mockRow{err: ErrNotFound}, nil
		}
		return &mockRow{vals: []any{
			row["payment_id"].(string),
			row["user_id"].(string),
			row["amount"].(int64),
			row["service"].(string),
			row["status"].(string),
			row["reason"].(string),
			row["gateway_id"].(string),
		}}, nil
	default:
		log.Printf("layer=client component=db method=QueryRow err=unsupported query query=%q", query)
		return &mockRow{err: errors.Join(ErrInternal, errors.New("unsupported query"))}, nil
	}
}
