package repo

import (
	"context"
	"errors"

	"cpms/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TariffsRepo struct{ db *pgxpool.Pool }

func NewTariffsRepo(db *pgxpool.Pool) *TariffsRepo { return &TariffsRepo{db: db} }

func (r *TariffsRepo) UpsertActiveForSite(ctx context.Context, siteId string, pricePerKwh float64, currency string) (string, error) {
	// Deactivate previous active tariffs for the site, then insert a new active tariff
	_, err := r.db.Exec(ctx, `update tariffs set is_active=false, updated_at=now() where site_id=$1 and is_active=true`, siteId)
	if err != nil {
		return "", err
	}
	row := r.db.QueryRow(ctx, `
		insert into tariffs (site_id, price_per_kwh, currency, is_active)
		values ($1,$2,$3,true)
		returning tariff_id
	`, siteId, pricePerKwh, currency)
	var id string
	if err := row.Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *TariffsRepo) GetActiveForSite(ctx context.Context, siteId string) (*models.Tariff, error) {
	row := r.db.QueryRow(ctx, `
		select tariff_id, site_id, price_per_kwh::float8, currency, is_active, created_at, updated_at
		from tariffs
		where site_id=$1 and is_active=true
		order by created_at desc
		limit 1
	`, siteId)
	var t models.Tariff
	if err := row.Scan(&t.TariffId, &t.SiteId, &t.PricePerKwh, &t.Currency, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}
