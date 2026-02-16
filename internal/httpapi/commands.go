package httpapi

import (
    "context"
    "encoding/json"
    "net/http"
    "time"

    "cpms/internal/models"
)

type createCommandReq struct {
    Type           string          `json:"type"`
    ChargePointId  string          `json:"chargePointId"`
    IdempotencyKey string          `json:"idempotencyKey"`
    Payload        json.RawMessage `json:"payload"`
}

func (s *Server) CreateAndSendCommand(w http.ResponseWriter, r *http.Request) {
    var req createCommandReq
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid json", http.StatusBadRequest)
        return
    }
    if req.Type == "" || req.ChargePointId == "" || req.IdempotencyKey == "" {
        http.Error(w, "missing type/chargePointId/idempotencyKey", http.StatusBadRequest)
        return
    }
    if len(req.Payload) == 0 {
        req.Payload = json.RawMessage(`{}`)
    }

    existing, err := s.Commands.GetByIdempotency(r.Context(), req.IdempotencyKey)
    if err != nil {
        http.Error(w, "db error", http.StatusInternalServerError)
        return
    }
    if existing != nil {
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(map[string]any{
            "commandId": existing.CommandId,
            "status":    existing.Status,
            "response":  json.RawMessage(existing.ResponseJSON),
            "error":     existing.Error,
        })
        return
    }

    gwBody, _ := json.Marshal(map[string]any{
        "type":           req.Type,
        "chargePointId":  req.ChargePointId,
        "idempotencyKey": req.IdempotencyKey,
        "payload":        json.RawMessage(req.Payload),
    })

    cmdId, err := s.Commands.Create(r.Context(), models.Command{
        ChargePointId:  req.ChargePointId,
        Type:           req.Type,
        IdempotencyKey: req.IdempotencyKey,
        PayloadJSON:    gwBody,
        Status:         "Queued",
    })
    if err != nil {
        http.Error(w, "db error", http.StatusInternalServerError)
        return
    }

    ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
    defer cancel()

    _ = s.Commands.MarkSent(r.Context(), cmdId)
    status, respBody, err := s.Gateway.SendCommand(ctx, gwBody)
    if err != nil {
        _ = s.Commands.MarkFailed(r.Context(), cmdId, err.Error())
        http.Error(w, "gateway error: "+err.Error(), http.StatusBadGateway)
        return
    }
    if status < 200 || status >= 300 {
        _ = s.Commands.MarkFailed(r.Context(), cmdId, string(respBody))
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadGateway)
        _ = json.NewEncoder(w).Encode(map[string]any{
            "commandId":     cmdId,
            "status":        "Failed",
            "gatewayStatus": status,
            "gatewayBody":   json.RawMessage(respBody),
        })
        return
    }

    _ = s.Commands.MarkAcked(r.Context(), cmdId, respBody)

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]any{
        "commandId":       cmdId,
        "status":          "Acked",
        "gatewayResponse": json.RawMessage(respBody),
    })
}
