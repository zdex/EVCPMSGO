package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EventsRepo struct{ db *pgxpool.Pool }

func NewEventsRepo(db *pgxpool.Pool) *EventsRepo { return &EventsRepo{db: db} }

func (r *EventsRepo) InsertRaw(ctx context.Context, chargePointId, eventType string, ts time.Time, payload []byte) error {
	_, err := r.db.Exec(ctx, `
		insert into gateway_events (charge_point_id, event_type, ts, payload)
		values ($1,$2,$3,$4)
	`, chargePointId, eventType, ts, payload)
	return err
}
