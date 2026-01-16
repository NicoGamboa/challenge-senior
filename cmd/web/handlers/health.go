package handlers

import "net/http"

type Health struct{}

func NewHealth() *Health { return &Health{} }

func (h *Health) Handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
