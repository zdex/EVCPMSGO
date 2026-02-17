package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// POST /v1/sessions/{sessionId}/finalize?force=true
func (s *Server) FinalizeSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "sessionId")
	if id == "" {
		http.Error(w, "missing sessionId", http.StatusBadRequest)
		return
	}

	force := r.URL.Query().Get("force") == "true"

	if force {
		if err := s.Sessions.FinalizeWithFallbackForce(r.Context(), id, true); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
	} else {
		if err := s.Sessions.FinalizeWithFallback(r.Context(), id); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
	}

	sess, err := s.Sessions.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if sess == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"session": sess,
		"force":   force,
	})
}
