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
	"challenge/internal/readmodels"
	"challenge/kit/broker"
	"challenge/kit/db"
)

type WalletBusContract interface {
	Publish(ctx context.Context, evt broker.Event) []error
}

type WalletStoreContract interface {
	Append(ctx context.Context, aggregateID string, evt broker.Event) error
}

type WalletServiceContract interface {
	Credit(ctx context.Context, userID string, amount int64) error
	Balance(ctx context.Context, userID string) (int64, error)
}

type WalletReadModelContract interface {
	GetWallet(userID string) (readmodels.WalletView, bool)
}

type Wallet struct {
	json   *validator.JSON
	bus    WalletBusContract
	store  WalletStoreContract
	wallet WalletServiceContract
	rm     WalletReadModelContract
}

func NewWallet(jsonV *validator.JSON, bus WalletBusContract, store WalletStoreContract, walletSvc WalletServiceContract, rm WalletReadModelContract) *Wallet {
	return &Wallet{json: jsonV, bus: bus, store: store, wallet: walletSvc, rm: rm}
}

type creditReq struct {
	UserID string `json:"user_id"`
	Amount int64  `json:"amount"`
}

func (h *Wallet) Credit(w http.ResponseWriter, r *http.Request) {
	var req creditReq
	if err := h.json.Decode(w, r, &req); err != nil {
		log.Printf("layer=handler component=wallet method=Credit err=%v", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if err := h.wallet.Credit(r.Context(), req.UserID, req.Amount); err != nil {
		log.Printf("layer=handler component=wallet method=Credit user_id=%s amount=%d err=%v", req.UserID, req.Amount, err)
		if db.IsInvalid(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()
	credited := events.WalletCredited{UserID: req.UserID, Amount: req.Amount, At: now}
	if h.store != nil {
		if err := h.store.Append(r.Context(), req.UserID, credited); err != nil {
			log.Printf("layer=handler component=wallet method=Credit user_id=%s err=%v", req.UserID, err)
		}
	}
	if h.bus != nil {
		h.bus.Publish(r.Context(), credited)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Wallet) Balance(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimPrefix(r.URL.Path, "/wallet/")
	if userID == "" {
		log.Printf("layer=handler component=wallet method=Balance err=missing user_id")
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}
	if h.rm != nil {
		if v, ok := h.rm.GetWallet(userID); ok {
			if err := json.NewEncoder(w).Encode(map[string]any{"user_id": userID, "balance": v.Balance}); err != nil {
				log.Printf("layer=handler component=wallet method=Balance user_id=%s err=%v", userID, err)
			}
			return
		}
	}
	bal, err := h.wallet.Balance(r.Context(), userID)
	if err != nil {
		log.Printf("layer=handler component=wallet method=Balance user_id=%s err=%v", userID, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(map[string]any{"user_id": userID, "balance": bal}); err != nil {
		log.Printf("layer=handler component=wallet method=Balance user_id=%s err=%v", userID, err)
	}
}
