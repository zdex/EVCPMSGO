package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"cpms/internal/config"
	"cpms/internal/gatewayclient"
	"cpms/internal/repo"
	"cpms/internal/security"
	"cpms/internal/services"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	Cfg       config.Config
	Chargers  *repo.ChargersRepo
	State     *repo.StateRepo
	Sessions  *repo.SessionsRepo
	Commands  *repo.CommandsRepo
	Gateway   *gatewayclient.Client
	Processor *services.EventsProcessor
}

func NewServer(cfg config.Config, chargers *repo.ChargersRepo, state *repo.StateRepo, sessions *repo.SessionsRepo, commands *repo.CommandsRepo, gw *gatewayclient.Client, processor *services.EventsProcessor) *Server {
	return &Server{Cfg: cfg, Chargers: chargers, State: state, Sessions: sessions, Commands: commands, Gateway: gw, Processor: processor}
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	r.Route("/v1/gateway", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler { return RequireBearer(s.Cfg.GatewayAPIKey, next) })
		r.Post("/chargers/{chargePointId}/auth", s.AuthCharger)
		r.Post("/events", s.IngestEvent)
	})

	r.Get("/v1/chargers/{chargePointId}", s.GetCharger)
	r.Get("/v1/chargers/{chargePointId}/connectors", s.ListConnectors)
	r.Get("/v1/chargers/{chargePointId}/sessions", s.ListSessionsByCharger)
	r.Get("/v1/sessions/{sessionId}", s.GetSession)

	r.Post("/v1/commands", s.CreateAndSendCommand)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	return r
}

type authReq struct {
	PresentedSecret string `json:"presentedSecret"`
	RemoteAddr      string `json:"remoteAddr,omitempty"`
	CertFingerprint string `json:"certFingerprint,omitempty"`
}

type authResp struct {
	Allowed     bool   `json:"allowed"`
	OcppVersion string `json:"ocppVersion,omitempty"`
}

func (s *Server) AuthCharger(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "chargePointId")

	var req authReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	ch, err := s.Chargers.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if ch == nil || !ch.IsActive || ch.SecretHash == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(authResp{Allowed: false})
		return
	}

	presentedHash := security.HashSecretSHA256(req.PresentedSecret)
	ok := security.ConstantTimeEqualHex(ch.SecretHash, presentedHash)

	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(authResp{Allowed: false})
		return
	}

	_ = s.Chargers.TouchLastSeen(r.Context(), id, time.Now().UTC())
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(authResp{Allowed: true, OcppVersion: ch.OcppVersion})
}

func (s *Server) IngestEvent(w http.ResponseWriter, r *http.Request) {
	raw, err := readAll(r, 2<<20)
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	evtType, err := s.Processor.Ingest(r.Context(), raw)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true, "type": evtType})
}

func (s *Server) GetCharger(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "chargePointId")
	ch, err := s.Chargers.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if ch == nil {
		http.NotFound(w, r)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"chargePointId": ch.ChargePointId,
		"isActive":      ch.IsActive,
		"vendor":        ch.Vendor,
		"model":         ch.Model,
		"ocppVersion":   ch.OcppVersion,
		"lastSeenAt":    ch.LastSeenAt,
		"createdAt":     ch.CreatedAt,
		"updatedAt":     ch.UpdatedAt,
	})
}

func (s *Server) ListConnectors(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "chargePointId")
	items, err := s.State.ListConnectors(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}

func (s *Server) GetSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "sessionId")
	sess, err := s.Sessions.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if sess == nil {
		http.NotFound(w, r)
		return
	}
	_ = json.NewEncoder(w).Encode(sess)
}

func (s *Server) ListSessionsByCharger(w http.ResponseWriter, r *http.Request) {
	cp := chi.URLParam(r, "chargePointId")
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	items, err := s.Sessions.ListByCharger(r.Context(), cp, limit)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(items)
}
