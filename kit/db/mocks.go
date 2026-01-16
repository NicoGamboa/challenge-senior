package db

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type ClientMock struct {
	mock.Mock
	Client
}

func (m *ClientMock) Exec(ctx context.Context, query string, args ...any) error {
	ret := m.Called(ctx, query, args)
	return ret.Error(0)
}

func (m *ClientMock) QueryRow(ctx context.Context, query string, args ...any) (Row, error) {
	ret := m.Called(ctx, query, args)
	if ret.Get(0) == nil {
		return nil, ret.Error(1)
	}
	return ret.Get(0).(Row), ret.Error(1)
}

type RowMock struct {
	mock.Mock
	Row
}

func (m *RowMock) Scan(dest ...any) error {
	ret := m.Called(dest)
	return ret.Error(0)
}
