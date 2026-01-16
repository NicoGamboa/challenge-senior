package readmodels

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"challenge/internal/events"
	"challenge/internal/payment"
	"challenge/kit/broker"
	"challenge/kit/db"
)

type PaymentView struct {
	PaymentID string
	UserID    string
	Amount    int64
	Service   string
	Status    payment.Status
	Reason    string
	GatewayID string
	UpdatedAt time.Time
}

type WalletView struct {
	UserID    string
	Balance   int64
	UpdatedAt time.Time
}

type Projector struct {
	mu       sync.RWMutex
	payments map[string]PaymentView
	wallets  map[string]WalletView
}

func NewProjector() *Projector {
	return &Projector{
		payments: make(map[string]PaymentView),
		wallets:  make(map[string]WalletView),
	}
}

func (p *Projector) Replay(ctx context.Context, store *db.Store) error {
	for _, rec := range store.All(ctx) {
		if err := p.ApplyRecord(ctx, rec); err != nil {
			return err
		}
	}
	return nil
}

func (p *Projector) Apply(ctx context.Context, evt broker.Event) error {
	switch e := evt.(type) {
	case events.PaymentCreated:
		p.applyPaymentCreated(e)
	case events.PaymentInitialized:
		p.applyPaymentInitialized(e)
	case events.PaymentPending:
		p.applyPaymentPending(e)
	case events.PaymentRejected:
		p.applyPaymentRejected(e)
	case events.PaymentSucceeded:
		p.applyPaymentSucceeded(e)
	case events.PaymentFailed:
		p.applyPaymentFailed(e)
	case events.WalletCredited:
		p.applyWalletCredited(e)
	case events.WalletDebited:
		p.applyWalletDebited(e)
	case events.WalletRefunded:
		p.applyWalletRefunded(e)
	default:
		return nil
	}
	return nil
}

func (p *Projector) ApplyRecord(ctx context.Context, rec db.Record) error {
	switch rec.EventName {
	case (events.PaymentCreated{}).Name():
		var e events.PaymentCreated
		if err := json.Unmarshal(rec.Payload, &e); err != nil {
			return errors.Join(db.ErrInternal, err)
		}
		p.applyPaymentCreated(e)
	case (events.PaymentInitialized{}).Name():
		var e events.PaymentInitialized
		if err := json.Unmarshal(rec.Payload, &e); err != nil {
			return errors.Join(db.ErrInternal, err)
		}
		p.applyPaymentInitialized(e)
	case (events.PaymentPending{}).Name():
		var e events.PaymentPending
		if err := json.Unmarshal(rec.Payload, &e); err != nil {
			return errors.Join(db.ErrInternal, err)
		}
		p.applyPaymentPending(e)
	case (events.PaymentRejected{}).Name():
		var e events.PaymentRejected
		if err := json.Unmarshal(rec.Payload, &e); err != nil {
			return errors.Join(db.ErrInternal, err)
		}
		p.applyPaymentRejected(e)
	case (events.PaymentSucceeded{}).Name():
		var e events.PaymentSucceeded
		if err := json.Unmarshal(rec.Payload, &e); err != nil {
			return errors.Join(db.ErrInternal, err)
		}
		p.applyPaymentSucceeded(e)
	case (events.PaymentFailed{}).Name():
		var e events.PaymentFailed
		if err := json.Unmarshal(rec.Payload, &e); err != nil {
			return errors.Join(db.ErrInternal, err)
		}
		p.applyPaymentFailed(e)
	case (events.WalletCredited{}).Name():
		var e events.WalletCredited
		if err := json.Unmarshal(rec.Payload, &e); err != nil {
			return errors.Join(db.ErrInternal, err)
		}
		p.applyWalletCredited(e)
	case (events.WalletDebited{}).Name():
		var e events.WalletDebited
		if err := json.Unmarshal(rec.Payload, &e); err != nil {
			return errors.Join(db.ErrInternal, err)
		}
		p.applyWalletDebited(e)
	case (events.WalletRefunded{}).Name():
		var e events.WalletRefunded
		if err := json.Unmarshal(rec.Payload, &e); err != nil {
			return errors.Join(db.ErrInternal, err)
		}
		p.applyWalletRefunded(e)
	default:
		return nil
	}
	return nil
}

func (p *Projector) GetPayment(paymentID string) (PaymentView, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	v, ok := p.payments[paymentID]
	return v, ok
}

func (p *Projector) GetWallet(userID string) (WalletView, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	v, ok := p.wallets[userID]
	return v, ok
}

func (p *Projector) applyPaymentCreated(e events.PaymentCreated) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cur := p.payments[e.PaymentID]
	cur.PaymentID = e.PaymentID
	cur.UserID = e.UserID
	cur.Amount = e.Amount
	cur.Service = e.Service
	if cur.Status == "" {
		cur.Status = payment.StatusInitialized
	}
	cur.UpdatedAt = e.At
	p.payments[e.PaymentID] = cur
}

func (p *Projector) applyPaymentInitialized(e events.PaymentInitialized) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cur := p.payments[e.PaymentID]
	cur.PaymentID = e.PaymentID
	cur.UserID = e.UserID
	cur.Amount = e.Amount
	cur.Service = e.Service
	cur.Status = payment.StatusInitialized
	cur.UpdatedAt = e.At
	p.payments[e.PaymentID] = cur
}

func (p *Projector) applyPaymentPending(e events.PaymentPending) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cur := p.payments[e.PaymentID]
	cur.PaymentID = e.PaymentID
	cur.UserID = e.UserID
	cur.Status = payment.StatusPending
	cur.UpdatedAt = e.At
	p.payments[e.PaymentID] = cur
}

func (p *Projector) applyPaymentRejected(e events.PaymentRejected) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cur := p.payments[e.PaymentID]
	cur.PaymentID = e.PaymentID
	cur.UserID = e.UserID
	cur.Status = payment.StatusRejected
	cur.Reason = e.Reason
	cur.UpdatedAt = e.At
	p.payments[e.PaymentID] = cur
}

func (p *Projector) applyPaymentSucceeded(e events.PaymentSucceeded) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cur := p.payments[e.PaymentID]
	cur.PaymentID = e.PaymentID
	cur.UserID = e.UserID
	cur.Status = payment.StatusSucceeded
	cur.GatewayID = e.GatewayID
	cur.UpdatedAt = e.At
	p.payments[e.PaymentID] = cur
}

func (p *Projector) applyPaymentFailed(e events.PaymentFailed) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cur := p.payments[e.PaymentID]
	cur.PaymentID = e.PaymentID
	cur.UserID = e.UserID
	cur.Status = payment.StatusFailed
	cur.Reason = e.Reason
	cur.UpdatedAt = e.At
	p.payments[e.PaymentID] = cur
}

func (p *Projector) applyWalletDebited(e events.WalletDebited) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cur := p.wallets[e.UserID]
	cur.UserID = e.UserID
	cur.Balance -= e.Amount
	cur.UpdatedAt = e.At
	p.wallets[e.UserID] = cur
}

func (p *Projector) applyWalletCredited(e events.WalletCredited) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cur := p.wallets[e.UserID]
	cur.UserID = e.UserID
	cur.Balance += e.Amount
	cur.UpdatedAt = e.At
	p.wallets[e.UserID] = cur
}

func (p *Projector) applyWalletRefunded(e events.WalletRefunded) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cur := p.wallets[e.UserID]
	cur.UserID = e.UserID
	cur.Balance += e.Amount
	cur.UpdatedAt = e.At
	p.wallets[e.UserID] = cur
}
