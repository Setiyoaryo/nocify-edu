// Package audit records who changed what (§39 audit logging).
package audit

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/nocify/eduplay/api/internal/db"
)

// Log writes an audit row. actor may be uuid.Nil for system actions.
// Failures are logged, not returned: auditing must never break the user action,
// but callers that need atomicity pass the surrounding transaction as q.
func Log(ctx context.Context, q db.DBTX, actor uuid.UUID, action, entityType, entityID string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	var actorArg any
	if actor != uuid.Nil {
		actorArg = actor
	}
	if _, err := q.Exec(ctx, `INSERT INTO audit_logs (actor_id, action, entity_type, entity_id, data) VALUES ($1,$2,$3,$4,$5)`,
		actorArg, action, entityType, entityID, data); err != nil {
		slog.ErrorContext(ctx, "audit log write failed", "err", err, "action", action)
	}
}
