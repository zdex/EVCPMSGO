package repo

import (
	"context"
	"time"

	"cpms/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StateRepo struct{ db *pgxpool.Pool }

func NewStateRepo(db *pgxpool.Pool) *StateRepo { return &StateRepo{db: db} }

func (r *StateRepo) UpsertConnector(ctx context.Context, st models.ConnectorState) error {
	_, err := r.db.Exec(ctx, `
		insert into connector_state (charge_point_id, connector_id, status, error_code, updated_at)
		values ($1,$2,$3,$4, now())
		on conflict (charge_point_id, connector_id) do update set
		  status=excluded.status,
		  error_code=excluded.error_code,
		  updated_at=now()
	`, st.ChargePointId, st.ConnectorId, st.Status, st.ErrorCode)
	return err
}

func (r *StateRepo) ListConnectors(ctx context.Context, cp string) ([]models.ConnectorState, error) {
	rows, err := r.db.Query(ctx, `
		select charge_point_id, connector_id, status, error_code, updated_at
		from connector_state where charge_point_id=$1
		order by connector_id asc
	`, cp)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.ConnectorState
	for rows.Next() {
		var s models.ConnectorState
		if err := rows.Scan(&s.ChargePointId, &s.ConnectorId, &s.Status, &s.ErrorCode, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *StateRepo) TouchHeartbeat(ctx context.Context, cp string, t time.Time) error {
	_, err := r.db.Exec(ctx, `update chargers set last_seen_at=$2, updated_at=now() where charge_point_id=$1`, cp, t)
	return err
}
