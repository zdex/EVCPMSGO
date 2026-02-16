package repo

import (
	"context"
	"errors"
	"time"

	"cpms/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChargersRepo struct{ db *pgxpool.Pool }

func NewChargersRepo(db *pgxpool.Pool) *ChargersRepo { return &ChargersRepo{db: db} }

func (r *ChargersRepo) Upsert(ctx context.Context, c models.Charger) error {
	_, err := r.db.Exec(ctx, `
		insert into chargers (charge_point_id, secret_hash, is_active, vendor, model, ocpp_version)
		values ($1,$2,$3,$4,$5,$6)
		on conflict (charge_point_id) do update set
		  secret_hash=excluded.secret_hash,
		  is_active=excluded.is_active,
		  vendor=excluded.vendor,
		  model=excluded.model,
		  ocpp_version=excluded.ocpp_version,
		  updated_at=now()
	`, c.ChargePointId, c.SecretHash, c.IsActive, c.Vendor, c.Model, c.OcppVersion)
	return err
}

func (r *ChargersRepo) Get(ctx context.Context, id string) (*models.Charger, error) {
	row := r.db.QueryRow(ctx, `
		select charge_point_id, secret_hash, is_active, coalesce(vendor,''), coalesce(model,''), coalesce(ocpp_version,'1.6J'),
		       created_at, updated_at, last_seen_at
		from chargers where charge_point_id=$1
	`, id)

	var c models.Charger
	if err := row.Scan(&c.ChargePointId, &c.SecretHash, &c.IsActive, &c.Vendor, &c.Model, &c.OcppVersion, &c.CreatedAt, &c.UpdatedAt, &c.LastSeenAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *ChargersRepo) TouchLastSeen(ctx context.Context, id string, t time.Time) error {
	_, err := r.db.Exec(ctx, `update chargers set last_seen_at=$2, updated_at=now() where charge_point_id=$1`, id, t)
	return err
}
