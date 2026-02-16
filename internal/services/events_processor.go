package services

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"cpms/internal/models"
	"cpms/internal/repo"
)

type EventsProcessor struct {
	Events   *repo.EventsRepo
	Chargers *repo.ChargersRepo
	State    *repo.StateRepo
	Sessions *repo.SessionsRepo
	MaxSkew  time.Duration
}

func NewEventsProcessor(e *repo.EventsRepo, c *repo.ChargersRepo, st *repo.StateRepo, s *repo.SessionsRepo, maxSkew time.Duration) *EventsProcessor {
	return &EventsProcessor{Events: e, Chargers: c, State: st, Sessions: s, MaxSkew: maxSkew}
}

type baseEvent struct {
	Type string `json:"type"`
}

func (p *EventsProcessor) Ingest(ctx context.Context, raw []byte) (string, error) {
	var b baseEvent
	if err := json.Unmarshal(raw, &b); err != nil {
		return "", err
	}
	if b.Type == "" {
		return "", errors.New("missing type")
	}

	var envelope map[string]any
	_ = json.Unmarshal(raw, &envelope)

	cp, _ := envelope["chargePointId"].(string)
	tsStr, _ := envelope["ts"].(string)
	if cp == "" {
		return "", errors.New("missing chargePointId")
	}

	ts := time.Now().UTC()
	if tsStr != "" {
		if t, err := time.Parse(time.RFC3339, tsStr); err == nil {
			ts = t.UTC()
		}
	}
	if p.MaxSkew > 0 {
		now := time.Now().UTC()
		if ts.Before(now.Add(-p.MaxSkew)) || ts.After(now.Add(p.MaxSkew)) {
			ts = now
		}
	}

	if err := p.Events.InsertRaw(ctx, cp, b.Type, ts, raw); err != nil {
		return b.Type, err
	}

	switch b.Type {
	case "ChargerBooted":
		vendor, _ := envelope["vendor"].(string)
		model, _ := envelope["model"].(string)
		ocpp, _ := envelope["ocppVersion"].(string)

		existing, err := p.Chargers.Get(ctx, cp)
		if err != nil {
			return b.Type, err
		}
		if existing != nil {
			_ = p.Chargers.TouchLastSeen(ctx, cp, ts)
		} else {
			_ = p.Chargers.Upsert(ctx, models.Charger{
				ChargePointId: cp,
				SecretHash:    "",
				IsActive:      false,
				Vendor:        vendor,
				Model:         model,
				OcppVersion:   ocpp,
			})
		}

	case "ChargerHeartbeat":
		_ = p.State.TouchHeartbeat(ctx, cp, ts)

	case "ConnectorStatusChanged":
		connId := intFromAny(envelope["connectorId"])
		status, _ := envelope["status"].(string)
		errCode, _ := envelope["errorCode"].(string)
		_ = p.State.UpsertConnector(ctx, models.ConnectorState{
			ChargePointId: cp,
			ConnectorId:   connId,
			Status:        status,
			ErrorCode:     errCode,
			UpdatedAt:     ts,
		})
		_ = p.Chargers.TouchLastSeen(ctx, cp, ts)

	case "TransactionStarted":
		connId := intFromAny(envelope["connectorId"])
		txId := intFromAny(envelope["transactionId"])
		idTag, _ := envelope["idTag"].(string)

		var ms *int64
		if v, ok := envelope["meterStartWh"]; ok {
			x := int64FromAny(v)
			ms = &x
		}
		session := models.Session{
			ChargePointId: cp,
			ConnectorId:   connId,
			TransactionId: txId,
			IdTag:         idTag,
			StartedAt:     ts,
			MeterStartWh:  ms,
		}
		_, _ = p.Sessions.Start(ctx, session)
		_ = p.Chargers.TouchLastSeen(ctx, cp, ts)

	case "MeterSample":
		txId := intFromAny(envelope["transactionId"])
		sess, err := p.Sessions.FindByTx(ctx, cp, txId)
		if err != nil {
			return b.Type, err
		}
		if sess == nil {
			return b.Type, nil
		}
		_ = p.Sessions.InsertMeterSample(ctx, models.MeterSample{
			SessionId:     sess.SessionId,
			ChargePointId: cp,
			TransactionId: txId,
			Ts:            ts,
			SamplesJSON:   raw,
		})
		_ = p.Chargers.TouchLastSeen(ctx, cp, ts)

	case "TransactionEnded":
		txId := intFromAny(envelope["transactionId"])
		sess, err := p.Sessions.FindByTx(ctx, cp, txId)
		if err != nil {
			return b.Type, err
		}
		if sess == nil {
			return b.Type, nil
		}
		var stop *int64
		if v, ok := envelope["meterStopWh"]; ok {
			x := int64FromAny(v)
			stop = &x
		}
		var reason *string
		if v, ok := envelope["reason"].(string); ok {
			reason = &v
		}
		// Store end markers (meter_stop may be missing)
		_ = p.Sessions.End(ctx, sess.SessionId, ts, stop, reason)
		// Finalize with fallback (StopTransaction -> last register -> sum interval -> Missing)
		_ = p.Sessions.FinalizeWithFallback(ctx, sess.SessionId)
		_ = p.Chargers.TouchLastSeen(ctx, cp, ts)
		_ = p.Sessions.End(ctx, sess.SessionId, ts, stop, reason)
		_ = p.Chargers.TouchLastSeen(ctx, cp, ts)
	}

	return b.Type, nil
}

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	default:
		return 0
	}
}

func int64FromAny(v any) int64 {
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int64:
		return t
	case int:
		return int64(t)
	default:
		return 0
	}
}
