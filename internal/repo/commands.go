package repo

import (
	"context"
	"errors"

	"cpms/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommandsRepo struct{ db *pgxpool.Pool }

func NewCommandsRepo(db *pgxpool.Pool) *CommandsRepo { return &CommandsRepo{db: db} }

func (r *CommandsRepo) Create(ctx context.Context, c models.Command) (string, error) {
	row := r.db.QueryRow(ctx, `
        insert into commands (charge_point_id, type, idempotency_key, payload, status)
        values ($1,$2,$3,$4,$5)
        returning command_id
    `, c.ChargePointId, c.Type, c.IdempotencyKey, c.PayloadJSON, c.Status)

	var id string
	if err := row.Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *CommandsRepo) GetByIdempotency(ctx context.Context, idem string) (*models.Command, error) {
	row := r.db.QueryRow(ctx, `
        select command_id, charge_point_id, type, idempotency_key, payload, status, response, error, created_at, updated_at
        from commands where idempotency_key=$1
    `, idem)

	var c models.Command
	if err := row.Scan(&c.CommandId, &c.ChargePointId, &c.Type, &c.IdempotencyKey, &c.PayloadJSON, &c.Status, &c.ResponseJSON, &c.Error, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *CommandsRepo) MarkSent(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `update commands set status='Sent', updated_at=now() where command_id=$1`, id)
	return err
}

func (r *CommandsRepo) MarkAcked(ctx context.Context, id string, response []byte) error {
	_, err := r.db.Exec(ctx, `update commands set status='Acked', response=$2, updated_at=now() where command_id=$1`, id, response)
	return err
}

func (r *CommandsRepo) MarkFailed(ctx context.Context, id string, errMsg string) error {
	_, err := r.db.Exec(ctx, `update commands set status='Failed', error=$2, updated_at=now() where command_id=$1`, id, errMsg)
	return err
}
