package changes

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/audit"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/httpx"
)

// curriculum_change_reviews.status values.
const (
	StatusPending      = "pending"
	StatusAcknowledged = "acknowledged"
	StatusDismissed    = "dismissed"
	StatusDeferred     = "deferred"
)

// ErrChangeNotFound is returned when the referenced event is not in staging.
var ErrChangeNotFound = errors.New("curriculum change event not found")

// ValidStatus reports whether s is a reviewer-settable review status.
func ValidStatus(s string) bool {
	switch s {
	case StatusAcknowledged, StatusDismissed, StatusDeferred:
		return true
	default:
		return false
	}
}

// ValidateDecision rejects an unknown status. `pending` is the implicit
// undecided state, so a reviewer never sets it explicitly.
func ValidateDecision(status string) error {
	if !ValidStatus(status) {
		return fmt.Errorf("unknown review status %q", status)
	}
	return nil
}

// DecisionInput is a reviewer's outcome for a detected change.
type DecisionInput struct {
	EventID int64
	Status  string
	Notes   string
	UserID  uuid.UUID
}

// RecordDecision upserts the canonical review for a staging change event. It
// copies the event's document/hash/counts by value (roadmap D4) so the ledger
// stays meaningful even after staging is pruned. It never touches canonical
// curriculum content.
func RecordDecision(ctx context.Context, pool *pgxpool.Pool, in DecisionInput) (Review, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Review{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	var (
		sourceDocumentID  uuid.UUID
		documentCode      string
		observedHash      string
		cpAdded           int
		cpChanged         int
		cpRemoved         int
		affectedQuestions int
	)
	if err := tx.QueryRow(ctx, `
		SELECT source_document_id, source_document_code, observed_hash,
		       cp_added, cp_changed, cp_removed,
		       COALESCE((SELECT COALESCE(sum(i.affected_questions), 0)::int
		                 FROM content_staging.curriculum_change_impacts i
		                 WHERE i.change_event_id = e.id), 0)
		FROM content_staging.curriculum_change_events e
		WHERE e.id = $1`, in.EventID).
		Scan(&sourceDocumentID, &documentCode, &observedHash,
			&cpAdded, &cpChanged, &cpRemoved, &affectedQuestions); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Review{}, ErrChangeNotFound
		}
		return Review{}, err
	}

	var reviewID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO curriculum_change_reviews
			(change_event_id, source_document_id, document_code, observed_hash,
			 cp_added, cp_changed, cp_removed, affected_questions,
			 status, decided_by, decided_at, notes, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), $11, now())
		ON CONFLICT (change_event_id) DO UPDATE SET
			source_document_id = EXCLUDED.source_document_id,
			document_code = EXCLUDED.document_code,
			observed_hash = EXCLUDED.observed_hash,
			cp_added = EXCLUDED.cp_added,
			cp_changed = EXCLUDED.cp_changed,
			cp_removed = EXCLUDED.cp_removed,
			affected_questions = EXCLUDED.affected_questions,
			status = EXCLUDED.status,
			decided_by = EXCLUDED.decided_by,
			decided_at = now(),
			notes = EXCLUDED.notes,
			updated_at = now()
		RETURNING id`,
		in.EventID, sourceDocumentID, documentCode, observedHash,
		cpAdded, cpChanged, cpRemoved, affectedQuestions,
		in.Status, in.UserID, in.Notes).Scan(&reviewID); err != nil {
		return Review{}, err
	}

	audit.Log(ctx, tx, in.UserID, "curriculum_change.review", "curriculum_change", fmt.Sprint(in.EventID),
		map[string]any{"status": in.Status, "document_code": documentCode, "observed_hash": observedHash})

	if err := tx.Commit(ctx); err != nil {
		return Review{}, err
	}

	rev := Review{ID: reviewID, EventID: in.EventID, Status: in.Status, Notes: in.Notes}
	if err := pool.QueryRow(ctx, `
		SELECT u.display_name, r.decided_at
		FROM curriculum_change_reviews r LEFT JOIN users u ON u.id = r.decided_by
		WHERE r.id = $1`, reviewID).Scan(&rev.Reviewer, &rev.DecidedAt); err != nil {
		return Review{}, err
	}
	return rev, nil
}

func (h Handler) DecideChange(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.Int64Param(w, r, "id")
	if !ok {
		return
	}
	var in struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if err := ValidateDecision(in.Status); err != nil {
		httpx.Invalid(w, map[string]string{"status": err.Error()})
		return
	}
	if !h.stagingReady(r.Context()) {
		httpx.Error(w, http.StatusConflict, "pipeline_data_unavailable", "Data pipeline belum tersedia.")
		return
	}
	u := auth.MustUser(r.Context())
	rev, err := RecordDecision(r.Context(), h.DB, DecisionInput{
		EventID: id, Status: in.Status, Notes: in.Notes, UserID: u.ID,
	})
	if errors.Is(err, ErrChangeNotFound) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"review": rev})
}
