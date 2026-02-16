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
		select session_id, charge_point_id, connector_id, transaction_id, coalesce(id_tag,''), started_at, ended_at, meter_start_wh, meter_stop_wh, reason, energy_wh, energy_source, is_estimated, finalized_at
		from sessions
		where charge_point_id=$1 and transaction_id=$2
		order by started_at desc
		limit 1
	`, cp, tx)

	var s models.Session
	if err := row.Scan(&s.SessionId, &s.ChargePointId, &s.ConnectorId, &s.TransactionId, &s.IdTag, &s.StartedAt, &s.EndedAt, &s.MeterStartWh, &s.MeterStopWh, &s.Reason, &s.EnergyWh, &s.EnergySource, &s.IsEstimated, &s.FinalizedAt); err != nil {
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
		select session_id, charge_point_id, connector_id, transaction_id, coalesce(id_tag,''), started_at, ended_at, meter_start_wh, meter_stop_wh, reason, energy_wh, energy_source, is_estimated, finalized_at
		from sessions where session_id=$1
	`, id)

	var s models.Session
	if err := row.Scan(&s.SessionId, &s.ChargePointId, &s.ConnectorId, &s.TransactionId, &s.IdTag, &s.StartedAt, &s.EndedAt, &s.MeterStartWh, &s.MeterStopWh, &s.Reason, &s.EnergyWh, &s.EnergySource, &s.IsEstimated, &s.FinalizedAt); err != nil {
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
		select session_id, charge_point_id, connector_id, transaction_id, coalesce(id_tag,''), started_at, ended_at, meter_start_wh, meter_stop_wh, reason, energy_wh, energy_source, is_estimated, finalized_at
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

// FinalizeWithFallback computes energy_wh and marks session finalized.
// Fallback order:
// 1) meterStopWh - meterStartWh (if meterStopWh present)
// 2) last Energy.Active.Import.Register from MeterSample - meterStartWh
// 3) sum of Energy.Active.Import.Interval from MeterSample (Wh per interval)
// If nothing available, energy_wh remains NULL and is_estimated=true.
func (r *SessionsRepo) FinalizeWithFallback(ctx context.Context, sessionId string) error {
	sess, err := r.GetByID(ctx, sessionId)
	if err != nil {
		return err
	}
	if sess == nil {
		return nil
	}
	// Already finalized? keep idempotent
	// We can't see finalized_at from models.Session in MVP struct, so we do a lightweight check.
	var already bool
	_ = r.db.QueryRow(ctx, `select finalized_at is not null from sessions where session_id=$1`, sessionId).Scan(&already)
	if already {
		return nil
	}

	// 1) meterStop - meterStart
	if sess.MeterStartWh != nil && sess.MeterStopWh != nil {
		energy := *sess.MeterStopWh - *sess.MeterStartWh
		if energy < 0 {
			energy = 0
		}
		_, err := r.db.Exec(ctx, `
            update sessions set energy_wh=$2, energy_source='StopTransaction', is_estimated=false, finalized_at=now(), updated_at=now()
            where session_id=$1
        `, sessionId, energy)
		return err
	}

	// 2) last register - meterStart
	if sess.MeterStartWh != nil {
		lastReg, ok, err := r.GetLastEnergyRegisterWh(ctx, sessionId)
		if err != nil {
			return err
		}
		if ok {
			energy := lastReg - *sess.MeterStartWh
			if energy < 0 {
				energy = 0
			}
			// Also fill meter_stop_wh from the best known register
			_, err := r.db.Exec(ctx, `
                update sessions set meter_stop_wh=coalesce(meter_stop_wh,$2), energy_wh=$3, energy_source='MeterValues.Register', is_estimated=false, finalized_at=now(), updated_at=now()
                where session_id=$1
            `, sessionId, lastReg, energy)
			return err
		}
	}

	// 3) sum interval samples (Wh per interval)
	sumInterval, ok, err := r.SumEnergyIntervalWh(ctx, sessionId)
	if err != nil {
		return err
	}
	if ok {
		_, err := r.db.Exec(ctx, `
            update sessions set energy_wh=$2, energy_source='MeterValues.Interval', is_estimated=false, finalized_at=now(), updated_at=now()
            where session_id=$1
        `, sessionId, sumInterval)
		return err
	}

	// Nothing usable; mark finalized but estimated with NULL energy_wh
	_, err = r.db.Exec(ctx, `
        update sessions set energy_source='Missing', is_estimated=true, finalized_at=now(), updated_at=now()
        where session_id=$1
    `, sessionId)
	return err
}

// GetLastEnergyRegisterWh finds the most recent sample with measurand Energy.Active.Import.Register (Wh).
// Returns (value, true, nil) if found.
func (r *SessionsRepo) GetLastEnergyRegisterWh(ctx context.Context, sessionId string) (int64, bool, error) {
	// samples_json contains the full normalized MeterSample event, including "samples":[...].
	// We extract the last matching sample by timestamp, then take the max timestamp row.
	var val int64
	err := r.db.QueryRow(ctx, `
        with candidates as (
          select ms.ts,
                 (s->>'value')::bigint as v
          from meter_samples ms,
               jsonb_array_elements(ms.samples_json->'samples') s
          where ms.session_id=$1
            and s->>'measurand'='Energy.Active.Import.Register'
            and (s->>'unit' is null or s->>'unit'='Wh')
        )
        select v from candidates order by ts desc limit 1
    `, sessionId).Scan(&val)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return val, true, nil
}

// SumEnergyIntervalWh sums measurand Energy.Active.Import.Interval values (Wh per interval).
// Returns (sum, true, nil) if at least one interval sample found.
func (r *SessionsRepo) SumEnergyIntervalWh(ctx context.Context, sessionId string) (int64, bool, error) {
	var sum int64
	err := r.db.QueryRow(ctx, `
        with candidates as (
          select (s->>'value')::bigint as v
          from meter_samples ms,
               jsonb_array_elements(ms.samples_json->'samples') s
          where ms.session_id=$1
            and s->>'measurand'='Energy.Active.Import.Interval'
            and (s->>'unit' is null or s->>'unit'='Wh')
        )
        select coalesce(sum(v),0) from candidates
    `, sessionId).Scan(&sum)
	if err != nil {
		return 0, false, err
	}
	if sum <= 0 {
		return 0, false, nil
	}
	return sum, true, nil
}
