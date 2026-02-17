package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type setWalletReq struct {
	Wallet string `json:"wallet"`
}

func (s *Server) SetSiteWallet(w http.ResponseWriter, r *http.Request) {
	siteId := chi.URLParam(r, "siteId")
	var req setWalletReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Wallet == "" {
		http.Error(w, "invalid json/wallet", http.StatusBadRequest)
		return
	}
	if err := s.Sites.SetPayoutWallet(r.Context(), siteId, req.Wallet); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
