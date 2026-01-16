package readmodels

import (
	"context"
	"testing"
	"time"

	"challenge/internal/events"
	"challenge/internal/payment"
	"challenge/kit/db"
)

func TestProjector_Replay_BuildsPaymentAndWalletViews(t *testing.T) {
	ctx := context.Background()
	store := db.New()

	now := time.Now().UTC()

	created := events.PaymentCreated{PaymentID: "p1", UserID: "u1", Amount: 10, Service: "internet", At: now}
	pending := events.PaymentPending{PaymentID: "p1", UserID: "u1", At: now.Add(time.Second)}
	succeeded := events.PaymentSucceeded{PaymentID: "p1", UserID: "u1", GatewayID: "gw_1", At: now.Add(2 * time.Second)}
	credited := events.WalletCredited{UserID: "u1", Amount: 20, At: now}
	debited := events.WalletDebited{PaymentID: "p1", UserID: "u1", Amount: 10, At: now.Add(time.Second)}

	if err := store.Append(ctx, "p1", created); err != nil {
		t.Fatalf("append created: %v", err)
	}
	if err := store.Append(ctx, "p1", pending); err != nil {
		t.Fatalf("append pending: %v", err)
	}
	if err := store.Append(ctx, "p1", succeeded); err != nil {
		t.Fatalf("append succeeded: %v", err)
	}
	if err := store.Append(ctx, "u1", credited); err != nil {
		t.Fatalf("append credited: %v", err)
	}
	if err := store.Append(ctx, "p1", debited); err != nil {
		t.Fatalf("append debited: %v", err)
	}

	p := NewProjector()
	if err := p.Replay(ctx, store); err != nil {
		t.Fatalf("replay: %v", err)
	}

	pv, ok := p.GetPayment("p1")
	if !ok {
		t.Fatalf("expected payment view")
	}
	if pv.Status != payment.StatusSucceeded {
		t.Fatalf("expected status %q, got %q", payment.StatusSucceeded, pv.Status)
	}
	if pv.GatewayID != "gw_1" {
		t.Fatalf("expected gateway_id gw_1, got %q", pv.GatewayID)
	}

	wv, ok := p.GetWallet("u1")
	if !ok {
		t.Fatalf("expected wallet view")
	}
	if wv.Balance != 10 {
		t.Fatalf("expected balance 10, got %d", wv.Balance)
	}
}
