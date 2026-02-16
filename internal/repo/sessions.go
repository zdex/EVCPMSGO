package repo

import (
	"context"
	"errors"
	"time"

	"cpms/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionsRepo struct{ db *pgxpool.Pool }

func NewSessionsRepo(db *pgxpool.Pool) *SessionsRepo { return &SessionsRepo{db: db} }

func (r *SessionsRepo) Start(ctx context.Context, s models.Session) (string, error) {
	row := r.db.QueryRow(ctx, `
		insert into sessions (charge_point_id, connector_id, transaction_id, id_tag, started_at, meter_start_wh)
		values ($1,$2,$3,$4,$5,$6)
		returning session_id
	`, s.ChargePointId, s.ConnectorId, s.TransactionId, s.IdTag, s.StartedAt, s.MeterStartWh)

	var id string
	if err := row.Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *SessionsRepo) FindByTx(ctx context.Context, cp string, tx int) (*models.Session, error) {
	row := r.db.QueryRow(ctx, `
		select session_id, charge_point_id, connector_id, transaction_id, coalesce(id_tag,''), started_at, ended_at, meter_start_wh, meter_stop_wh, reason
		from sessions
		where charge_point_id=$1 and transaction_id=$2
		order by started_at desc
		limit 1
	`, cp, tx)

	var s models.Session
	if err := row.Scan(&s.SessionId, &s.ChargePointId, &s.ConnectorId, &s.TransactionId, &s.IdTag, &s.StartedAt, &s.EndedAt, &s.MeterStartWh, &s.MeterStopWh, &s.Reason); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *SessionsRepo) End(ctx context.Context, sessionId string, endedAt time.Time, meterStop *int64, reason *string) error {
	_, err := r.db.Exec(ctx, `
		update sessions set ended_at=$2, meter_stop_wh=coalesce($3, meter_stop_wh), reason=coalesce($4, reason), updated_at=now()
		where session_id=$1
	`, sessionId, endedAt, meterStop, reason)
	return err
}

func (r *SessionsRepo) InsertMeterSample(ctx context.Context, sample models.MeterSample) error {
	_, err := r.db.Exec(ctx, `
		insert into meter_samples (session_id, charge_point_id, transaction_id, ts, samples_json)
		values ($1,$2,$3,$4,$5)
	`, sample.SessionId, sample.ChargePointId, sample.TransactionId, sample.Ts, sample.SamplesJSON)
	return err
}

func (r *SessionsRepo) GetByID(ctx context.Context, id string) (*models.Session, error) {
	row := r.db.QueryRow(ctx, `
		select session_id, charge_point_id, connector_id, transaction_id, coalesce(id_tag,''), started_at, ended_at, meter_start_wh, meter_stop_wh, reason
		from sessions where session_id=$1
	`, id)

	var s models.Session
	if err := row.Scan(&s.SessionId, &s.ChargePointId, &s.ConnectorId, &s.TransactionId, &s.IdTag, &s.StartedAt, &s.EndedAt, &s.MeterStartWh, &s.MeterStopWh, &s.Reason); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *SessionsRepo) ListByCharger(ctx context.Context, cp string, limit int) ([]models.Session, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		select session_id, charge_point_id, connector_id, transaction_id, coalesce(id_tag,''), started_at, ended_at, meter_start_wh, meter_stop_wh, reason
		from sessions where charge_point_id=$1
		order by started_at desc
		limit $2
	`, cp, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Session
	for rows.Next() {
		var s models.Session
		if err := rows.Scan(&s.SessionId, &s.ChargePointId, &s.ConnectorId, &s.TransactionId, &s.IdTag, &s.StartedAt, &s.EndedAt, &s.MeterStartWh, &s.MeterStopWh, &s.Reason); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
