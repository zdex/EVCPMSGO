package httpapi

import (
	"encoding/json"
	"net/http"

	"cpms/internal/repo"

	"github.com/go-chi/chi/v5"
)

type createSiteReq struct {
	Name string `json:"name"`
}

func (s *Server) CreateSite(w http.ResponseWriter, r *http.Request) {
	var req createSiteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "invalid json/name", http.StatusBadRequest)
		return
	}
	id, err := s.Sites.Create(r.Context(), req.Name)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"siteId": id, "name": req.Name})
}

type createTariffReq struct {
	PricePerKwh float64 `json:"pricePerKwh"`
	Currency    string  `json:"currency"`
}

func (s *Server) UpsertActiveTariff(w http.ResponseWriter, r *http.Request) {
	siteId := chi.URLParam(r, "siteId")
	var req createTariffReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PricePerKwh <= 0 {
		http.Error(w, "invalid json/pricePerKwh", http.StatusBadRequest)
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	id, err := s.Tariffs.UpsertActiveForSite(r.Context(), siteId, req.PricePerKwh, req.Currency)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"tariffId": id, "siteId": siteId, "pricePerKwh": req.PricePerKwh, "currency": req.Currency, "isActive": true})
}

// compile-time check to ensure we used repo import (avoid unused if file changes)
var _ = repo.NewSitesRepo
