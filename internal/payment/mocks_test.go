package payment

import (
	"context"

	"challenge/kit/broker"
	"github.com/stretchr/testify/mock"
)

type RepositoryMock struct {
	mock.Mock
	RepositoryContract
}

func (m *RepositoryMock) Save(ctx context.Context, p *Payment) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *RepositoryMock) Get(ctx context.Context, paymentID string) (*Payment, error) {
	args := m.Called(ctx, paymentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Payment), args.Error(1)
}

type PublisherMock struct {
	mock.Mock
	PublisherContract
}

func (m *PublisherMock) Publish(ctx context.Context, evt broker.Event) []error {
	args := m.Called(ctx, evt)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]error)
}

type StoreMock struct {
	mock.Mock
	StoreContract
}

func (m *StoreMock) Append(ctx context.Context, aggregateID string, evt broker.Event) error {
	args := m.Called(ctx, aggregateID, evt)
	return args.Error(0)
}
