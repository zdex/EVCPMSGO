package repo

import (
	"context"
	"errors"

	"cpms/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SitesRepo struct{ db *pgxpool.Pool }

func NewSitesRepo(db *pgxpool.Pool) *SitesRepo { return &SitesRepo{db: db} }

func (r *SitesRepo) Create(ctx context.Context, name string) (string, error) {
	row := r.db.QueryRow(ctx, `insert into sites (name) values ($1) on conflict (name) do update set name=excluded.name returning site_id`, name)
	var id string
	if err := row.Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *SitesRepo) GetByName(ctx context.Context, name string) (*models.Site, error) {
	row := r.db.QueryRow(ctx, `select site_id, name, payout_wallet, created_at from sites where name=$1`, name)
	var s models.Site
	if err := row.Scan(&s.SiteId, &s.Name, &s.PayoutWallet, &s.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *SitesRepo) SetPayoutWallet(ctx context.Context, siteId string, wallet string) error {
	_, err := r.db.Exec(ctx, `update sites set payout_wallet=$2 where site_id=$1`, siteId, wallet)
	return err
}

func (r *SitesRepo) GetPayoutWallet(ctx context.Context, siteId string) (string, error) {
	row := r.db.QueryRow(ctx, `select coalesce(payout_wallet,'') from sites where site_id=$1`, siteId)
	var w string
	if err := row.Scan(&w); err != nil {
		return "", err
	}
	return w, nil
}
