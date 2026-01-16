package payment

import (
	"context"

	"challenge/kit/broker"
)

// RepositoryContract define payment repository responsibility.
type RepositoryContract interface {
	Save(ctx context.Context, p *Payment) error
	Get(ctx context.Context, paymentID string) (*Payment, error)
}

// ServiceContract define payment service responsibility.
type ServiceContract interface {
	Initialize(ctx context.Context, req CreateRequest) (*Payment, error)
	MarkPending(ctx context.Context, paymentID string) error
	MarkRejected(ctx context.Context, paymentID, reason string) error
	MarkSucceeded(ctx context.Context, paymentID, gatewayID string) error
	MarkFailed(ctx context.Context, paymentID, reason string) error
	Get(ctx context.Context, paymentID string) (*Payment, error)
}

// PublisherContract define publish responsibility (broker).
type PublisherContract interface {
	Publish(ctx context.Context, evt broker.Event) []error
}

// StoreContract define append responsibility (event store).
type StoreContract interface {
	Append(ctx context.Context, aggregateID string, evt broker.Event) error
}
