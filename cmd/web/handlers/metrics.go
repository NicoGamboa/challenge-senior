package handlers

import (
	"encoding/json"
	"net/http"

	"challenge/internal/metrics"
)

type Metrics struct {
	svc *metrics.Service
}

func NewMetrics(svc *metrics.Service) *Metrics {
	return &Metrics{svc: svc}
}

func (h *Metrics) Handler(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(h.svc.Snapshot())
}
