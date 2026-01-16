package payment

import (
	"context"
	"log"
	"sync"

	"challenge/kit/db"
)

type Repository interface {
	Save(ctx context.Context, p *Payment) error
	Get(ctx context.Context, paymentID string) (*Payment, error)
}

type SQLRepository struct {
	db db.Client
}

func NewSQLRepository(dbClient db.Client) *SQLRepository {
	return &SQLRepository{db: dbClient}
}

const (
	qPaymentUpsert = "INSERT INTO payments (payment_id, user_id, amount, service, status, reason, gateway_id) VALUES (?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE user_id=?, amount=?, service=?, status=?, reason=?, gateway_id=?"
	qPaymentGet    = "SELECT payment_id, user_id, amount, service, status, reason, gateway_id FROM payments WHERE payment_id = ?"
)

func (r *SQLRepository) Save(ctx context.Context, p *Payment) error {
	if err := r.db.Exec(
		ctx,
		qPaymentUpsert,
		p.ID,
		p.UserID,
		p.Amount,
		p.Service,
		p.Status,
		p.Reason,
		p.GatewayID,
		p.UserID,
		p.Amount,
		p.Service,
		p.Status,
		p.Reason,
		p.GatewayID,
	); err != nil {
		log.Printf("layer=repo component=payment repo=SQLRepository method=Save payment_id=%s user_id=%s err=%v", p.ID, p.UserID, err)
		return err
	}
	return nil
}

func (r *SQLRepository) Get(ctx context.Context, paymentID string) (*Payment, error) {
	row, err := r.db.QueryRow(ctx, qPaymentGet, paymentID)
	if err != nil {
		log.Printf("layer=repo component=payment repo=SQLRepository method=Get payment_id=%s err=%v", paymentID, err)
		return nil, err
	}
	var p Payment
	if err := row.Scan(&p.ID, &p.UserID, &p.Amount, &p.Service, &p.Status, &p.Reason, &p.GatewayID); err != nil {
		log.Printf("layer=repo component=payment repo=SQLRepository method=Get payment_id=%s err=%v", paymentID, err)
		return nil, err
	}
	return &p, nil
}

type InMemoryRepository struct {
	mu   sync.Mutex
	data map[string]*Payment
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{data: make(map[string]*Payment)}
}

func (r *InMemoryRepository) Save(ctx context.Context, p *Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cpy := *p
	r.data[p.ID] = &cpy
	return nil
}

func (r *InMemoryRepository) Get(ctx context.Context, paymentID string) (*Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.data[paymentID]
	if !ok {
		log.Printf("layer=repo component=payment repo=InMemoryRepository method=Get payment_id=%s err=%v", paymentID, db.ErrNotFound)
		return nil, db.ErrNotFound
	}
	cpy := *p
	return &cpy, nil
}
