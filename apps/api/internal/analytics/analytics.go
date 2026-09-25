// Package analytics stores first-party business/product events (§50).
// Only opaque ids and non-personal properties are stored.
package analytics

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/db"
	"github.com/nocify/eduplay/api/internal/httpx"
)

// Track records a server-side event. child may be uuid.Nil.
func Track(ctx context.Context, q db.DBTX, name string, user, child uuid.UUID, props map[string]any) {
	if props == nil {
		props = map[string]any{}
	}
	var u, c any
	if user != uuid.Nil {
		u = user
	}
	if child != uuid.Nil {
		c = child
	}
	if _, err := q.Exec(ctx, `INSERT INTO analytics_events (name, user_id, child_id, props) VALUES ($1,$2,$3,$4)`, name, u, c, props); err != nil {
		slog.ErrorContext(ctx, "analytics write failed", "err", err, "event", name)
	}
}

// Client-reported events are allow-listed; everything else is derived server-side
// so it cannot be spoofed (register, challenge_completed, payment_success, ...).
var clientEvents = map[string]bool{
	"landing_visit": true, "pricing_viewed": true, "progress_viewed": true, "subjects_viewed": true,
}

type Handler struct{ DB *pgxpool.Pool }

func (h Handler) Collect(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name  string            `json:"name"`
		Props map[string]string `json:"props"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if !clientEvents[in.Name] || len(in.Props) > 10 {
		httpx.Error(w, http.StatusBadRequest, "unknown_event", "Unknown event.")
		return
	}
	props := map[string]any{}
	for k, v := range in.Props {
		if len(k) <= 40 && len(v) <= 200 {
			props[k] = v
		}
	}
	var user uuid.UUID
	if u, ok := auth.UserFrom(r.Context()); ok {
		user = u.ID
	}
	Track(r.Context(), h.DB, in.Name, user, uuid.Nil, props)
	w.WriteHeader(http.StatusNoContent)
}
