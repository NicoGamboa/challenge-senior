package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"challenge/cmd/web/validator"
	"challenge/internal/events"
	"challenge/internal/health"
	"challenge/internal/payment"
	"challenge/internal/readmodels"
	"challenge/kit/broker"
	"challenge/kit/db"
)

type PaymentBusContract interface {
	Publish(ctx context.Context, evt broker.Event) []error
}

type PaymentStoreContract interface {
	Append(ctx context.Context, aggregateID string, evt broker.Event) error
}

type PaymentServiceContract interface {
	Initialize(ctx context.Context, req payment.CreateRequest) (*payment.Payment, error)
	Get(ctx context.Context, paymentID string) (*payment.Payment, error)
}

type PaymentHealthContract interface {
	Check(ctx context.Context) health.Result
}

type PaymentReadModelContract interface {
	GetPayment(paymentID string) (readmodels.PaymentView, bool)
}

type Payment struct {
	json    *validator.JSON
	bus     PaymentBusContract
	store   PaymentStoreContract
	payment PaymentServiceContract
	health  PaymentHealthContract
	rm      PaymentReadModelContract
}

func NewPayment(jsonV *validator.JSON, bus PaymentBusContract, store PaymentStoreContract, paymentSvc PaymentServiceContract, healthSvc PaymentHealthContract, rm PaymentReadModelContract) *Payment {
	return &Payment{json: jsonV, bus: bus, store: store, payment: paymentSvc, health: healthSvc, rm: rm}
}

type createPaymentReq struct {
	PaymentID string `json:"payment_id"`
	UserID    string `json:"user_id"`
	Amount    int64  `json:"amount"`
	Service   string `json:"service"`
}

func (h *Payment) Create(w http.ResponseWriter, r *http.Request) {
	var req createPaymentReq
	if err := h.json.Decode(w, r, &req); err != nil {
		log.Printf("layer=handler component=payment method=Create err=%v", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if h.health != nil {
		res := h.health.Check(r.Context())
		if !res.OK {
			log.Printf("layer=handler component=payment method=Create err=service_unavailable checks=%v", res.Checks)
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "down", "checks": res.Checks})
			return
		}
	}

	domainReq := payment.ToCreateRequest(req.PaymentID, req.UserID, req.Amount, req.Service)
	p, err := h.payment.Initialize(r.Context(), domainReq)
	if err != nil {
		log.Printf("layer=handler component=payment method=Create payment_id=%s user_id=%s err=%v", req.PaymentID, req.UserID, err)
		if db.IsInvalid(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()

	createdEvt := events.PaymentCreated{PaymentID: p.ID, UserID: p.UserID, Amount: p.Amount, Service: p.Service, At: now}
	if err := h.store.Append(r.Context(), p.ID, createdEvt); err != nil {
		log.Printf("layer=handler component=payment method=Create payment_id=%s err=%v", p.ID, err)
	}
	h.bus.Publish(r.Context(), createdEvt)

	initializedEvt := events.PaymentInitialized{PaymentID: p.ID, UserID: p.UserID, Amount: p.Amount, Service: p.Service, At: now}
	if err := h.store.Append(r.Context(), p.ID, initializedEvt); err != nil {
		log.Printf("layer=handler component=payment method=Create payment_id=%s err=%v", p.ID, err)
	}
	h.bus.Publish(r.Context(), initializedEvt)

	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(map[string]any{"payment_id": p.ID, "status": p.Status}); err != nil {
		log.Printf("layer=handler component=payment method=Create payment_id=%s err=%v", p.ID, err)
	}
}

func (h *Payment) Get(w http.ResponseWriter, r *http.Request) {
	paymentID := strings.TrimPrefix(r.URL.Path, "/payments/")
	if paymentID == "" {
		log.Printf("layer=handler component=payment method=Get err=missing payment_id")
		http.Error(w, "missing payment_id", http.StatusBadRequest)
		return
	}
	if h.rm != nil {
		if v, ok := h.rm.GetPayment(paymentID); ok {
			if err := json.NewEncoder(w).Encode(map[string]any{
				"payment_id": v.PaymentID,
				"user_id":    v.UserID,
				"amount":     v.Amount,
				"service":    v.Service,
				"status":     v.Status,
				"reason":     v.Reason,
				"gateway_id": v.GatewayID,
			}); err != nil {
				log.Printf("layer=handler component=payment method=Get payment_id=%s err=%v", paymentID, err)
			}
			return
		}
	}

	p, err := h.payment.Get(r.Context(), paymentID)
	if err != nil {
		log.Printf("layer=handler component=payment method=Get payment_id=%s err=%v", paymentID, err)
		if db.IsNotFound(err) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(map[string]any{
		"payment_id": p.ID,
		"user_id":    p.UserID,
		"amount":     p.Amount,
		"service":    p.Service,
		"status":     p.Status,
		"reason":     p.Reason,
		"gateway_id": p.GatewayID,
	}); err != nil {
		log.Printf("layer=handler component=payment method=Get payment_id=%s err=%v", paymentID, err)
	}
}
