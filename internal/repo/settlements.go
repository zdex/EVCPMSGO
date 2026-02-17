package repo

import (
	"context"
	"errors"

	"cpms/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SettlementsRepo struct{ db *pgxpool.Pool }

func NewSettlementsRepo(db *pgxpool.Pool) *SettlementsRepo { return &SettlementsRepo{db: db} }

func (r *SettlementsRepo) CreateForSession(ctx context.Context, sessionId string, siteId string, amount float64, currency string) (string, error) {
	row := r.db.QueryRow(ctx, `
		insert into settlements (session_id, site_id, amount, currency, status)
		values ($1,$2,$3,$4,'Pending')
		on conflict (session_id) do update set updated_at=now()
		returning settlement_id
	`, sessionId, siteId, amount, currency)
	var id string
	if err := row.Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *SettlementsRepo) Get(ctx context.Context, settlementId string) (*models.Settlement, error) {
	row := r.db.QueryRow(ctx, `
		select settlement_id, session_id, site_id, amount::float8, currency, status, chain, tx_hash, external_ref, error, created_at, updated_at
		from settlements where settlement_id=$1
	`, settlementId)
	var s models.Settlement
	if err := row.Scan(&s.SettlementId, &s.SessionId, &s.SiteId, &s.Amount, &s.Currency, &s.Status, &s.Chain, &s.TxHash, &s.ExternalRef, &s.Error, &s.CreatedAt, &s.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *SettlementsRepo) List(ctx context.Context, status string, limit int) ([]models.Settlement, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows pgx.Rows
	var err error
	if status == "" {
		rows, err = r.db.Query(ctx, `
			select settlement_id, session_id, site_id, amount::float8, currency, status, chain, tx_hash, external_ref, error, created_at, updated_at
			from settlements order by created_at desc limit $1
		`, limit)
	} else {
		rows, err = r.db.Query(ctx, `
			select settlement_id, session_id, site_id, amount::float8, currency, status, chain, tx_hash, external_ref, error, created_at, updated_at
			from settlements where status=$1 order by created_at asc limit $2
		`, status, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.Settlement, 0, limit)
	for rows.Next() {
		var s models.Settlement
		if err := rows.Scan(&s.SettlementId, &s.SessionId, &s.SiteId, &s.Amount, &s.Currency, &s.Status, &s.Chain, &s.TxHash, &s.ExternalRef, &s.Error, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *SettlementsRepo) MarkSubmitted(ctx context.Context, settlementId string, chain string, txHash string, externalRef *string) error {
	_, err := r.db.Exec(ctx, `
		update settlements set status='Submitted', chain=$2, tx_hash=$3, external_ref=$4, updated_at=now(), error=null
		where settlement_id=$1
	`, settlementId, chain, txHash, externalRef)
	return err
}

func (r *SettlementsRepo) MarkConfirmed(ctx context.Context, settlementId string) error {
	_, err := r.db.Exec(ctx, `
		update settlements set status='Confirmed', updated_at=now()
		where settlement_id=$1
	`, settlementId)
	return err
}

func (r *SettlementsRepo) MarkFailed(ctx context.Context, settlementId string, errMsg string) error {
	_, err := r.db.Exec(ctx, `
		update settlements set status='Failed', error=$2, updated_at=now()
		where settlement_id=$1
	`, settlementId, errMsg)
	return err
}
