// Package worker runs periodic maintenance (the "worker" container of §53).
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/subscriptions"
)

// Tick runs every job once. Each job is idempotent.
func Tick(ctx context.Context, db *pgxpool.Pool) {
	jobs := []struct {
		name string
		sql  string
	}{
		// Paid period over without a renewal webhook → past_due (grace), then expired.
		{"subscriptions.past_due", `UPDATE subscriptions SET status = 'past_due', updated_at = now()
			WHERE status IN ('active','trial') AND current_period_end <= now()`},
		{"subscriptions.expire", `UPDATE subscriptions SET status = 'expired', updated_at = now()
			WHERE (status = 'cancelled' AND current_period_end <= now())
			   OR (status = 'past_due' AND current_period_end + make_interval(secs => $1) <= now())`},
		{"checkouts.expire", `UPDATE payment_checkouts SET status = 'expired' WHERE status = 'pending' AND created_at < now() - interval '24 hours'`},
		{"sessions.purge", `DELETE FROM sessions WHERE expires_at < now()`},
		{"assessments.abandon", `UPDATE assessments SET status = 'abandoned' WHERE status = 'in_progress' AND started_at < now() - interval '1 day'`},
	}
	for _, j := range jobs {
		var args []any
		if j.name == "subscriptions.expire" {
			args = append(args, subscriptions.PastDueGrace.Seconds())
		}
		tag, err := db.Exec(ctx, j.sql, args...)
		if err != nil {
			slog.Error("worker job failed", "job", j.name, "err", err)
			continue
		}
		if n := tag.RowsAffected(); n > 0 {
			slog.Info("worker job", "job", j.name, "rows", n)
		}
	}
}

func Run(ctx context.Context, db *pgxpool.Pool, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		Tick(ctx, db)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
