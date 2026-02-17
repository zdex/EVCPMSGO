package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) ListSettlements(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	items, err := s.Settlements.List(r.Context(), status, limit)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
}

type markSubmittedReq struct {
	Chain       string  `json:"chain"`
	TxHash      string  `json:"txHash"`
	ExternalRef *string `json:"externalRef"`
}

func (s *Server) MarkSettlementSubmitted(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "settlementId")
	var req markSubmittedReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Chain == "" || req.TxHash == "" {
		http.Error(w, "invalid json/chain/txHash", http.StatusBadRequest)
		return
	}
	if err := s.Settlements.MarkSubmitted(r.Context(), id, req.Chain, req.TxHash, req.ExternalRef); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) MarkSettlementConfirmed(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "settlementId")
	if err := s.Settlements.MarkConfirmed(r.Context(), id); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type markFailedReq struct {
	Error string `json:"error"`
}

func (s *Server) MarkSettlementFailed(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "settlementId")
	var req markFailedReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Error == "" {
		req.Error = "failed"
	}
	if err := s.Settlements.MarkFailed(r.Context(), id, req.Error); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
