// Package changes is the reviewer CMS backend for detected curriculum changes
// (§27, §28, §44). It lists the change events the Python watcher recorded in
// `content_staging`, shows a reviewer the per-CP diff and the questions it
// affects, and records the review decision in canonical
// `curriculum_change_reviews`.
//
// Go only ever READS `content_staging`; the decision ledger lives in canonical
// tables (roadmap D2/D3). M7 is detect + review only: nothing here mutates
// `curriculum_cp`, `skills` or `questions`.
package changes

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/httpx"
)

type Handler struct{ DB *pgxpool.Pool }

// stagingReady reports whether the pipeline has created the M7 tables yet. The
// pipeline is a separate service, so the CMS degrades to "no changes" instead
// of a 500 in a fresh environment.
func (h Handler) stagingReady(ctx context.Context) bool {
	var ready bool
	err := h.DB.QueryRow(ctx,
		`SELECT to_regclass('content_staging.curriculum_change_events') IS NOT NULL`).Scan(&ready)
	return err == nil && ready
}

// changeEventsSQL is the reviewer queue: every detected event with its frozen
// impact counts and the canonical review decision (left-joined, so an
// undecided event shows as 'pending').
const changeEventsSQL = `
	SELECT e.id, e.source_document_code AS document_code, e.url, e.curriculum_version_id,
	       e.baseline_hash, e.observed_hash, e.detection, e.document_type,
	       e.cp_added, e.cp_changed, e.cp_removed, e.status AS event_status, e.error,
	       COALESCE((SELECT count(*)::int FROM content_staging.curriculum_change_impacts i
	                 WHERE i.change_event_id = e.id), 0) AS impact_count,
	       COALESCE((SELECT COALESCE(sum(i.affected_questions), 0)::int
	                 FROM content_staging.curriculum_change_impacts i
	                 WHERE i.change_event_id = e.id), 0) AS affected_questions,
	       COALESCE(r.status, 'pending') AS review_status,
	       u.display_name, r.decided_at, e.created_at
	FROM content_staging.curriculum_change_events e
	LEFT JOIN curriculum_change_reviews r ON r.change_event_id = e.id
	LEFT JOIN users u ON u.id = r.decided_by
`

type ChangeSummary struct {
	EventID             int64      `json:"event_id"`
	DocumentCode        string     `json:"document_code"`
	URL                 string     `json:"url"`
	CurriculumVersionID *int       `json:"curriculum_version_id"`
	BaselineHash        string     `json:"baseline_hash"`
	ObservedHash        string     `json:"observed_hash"`
	Detection           string     `json:"detection"`
	DocumentType        string     `json:"document_type"`
	CPAdded             int        `json:"cp_added"`
	CPChanged           int        `json:"cp_changed"`
	CPRemoved           int        `json:"cp_removed"`
	EventStatus         string     `json:"event_status"`
	Error               string     `json:"error"`
	ImpactCount         int        `json:"impact_count"`
	AffectedQuestions   int        `json:"affected_questions"`
	ReviewStatus        string     `json:"review_status"`
	Reviewer            *string    `json:"reviewer"`
	DecidedAt           *time.Time `json:"decided_at"`
	CreatedAt           time.Time  `json:"created_at"`
}

func (h Handler) ListChanges(w http.ResponseWriter, r *http.Request) {
	if !h.stagingReady(r.Context()) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"changes": []ChangeSummary{}, "counts": map[string]int{}, "available": false,
		})
		return
	}
	q := r.URL.Query()
	rows, err := h.DB.Query(r.Context(), `
		SELECT * FROM (`+changeEventsSQL+`) c
		WHERE ($1 = '' OR document_code = $1)
		  AND ($2 = '' OR review_status = $2)
		  AND ($3 = '' OR event_status = $3)
		  AND ($4 = '' OR document_code ILIKE '%' || $4 || '%' OR url ILIKE '%' || $4 || '%')
		ORDER BY created_at DESC, id DESC LIMIT 300`,
		q.Get("document"), q.Get("status"), q.Get("event_status"), q.Get("q"))
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[ChangeSummary])
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	counts := map[string]int{}
	crows, err := h.DB.Query(r.Context(),
		`SELECT review_status, count(*)::int FROM (`+changeEventsSQL+`) c GROUP BY review_status`)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	for crows.Next() {
		var s string
		var n int
		_ = crows.Scan(&s, &n)
		counts[s] = n
	}
	crows.Close()
	httpx.JSON(w, http.StatusOK, map[string]any{"changes": out, "counts": counts, "available": true})
}

// Review is the canonical reviewer decision on a change event.
type Review struct {
	ID        int64      `json:"id"`
	EventID   int64      `json:"event_id"`
	Status    string     `json:"status"`
	Notes     string     `json:"notes"`
	Reviewer  *string    `json:"reviewer"`
	DecidedAt *time.Time `json:"decided_at"`
}

type ImpactQuestion struct {
	QuestionID   uuid.UUID `json:"question_id"`
	QuestionCode string    `json:"question_code"`
	SkillID      int       `json:"skill_id"`
	SkillName    string    `json:"skill_name"`
	Match        string    `json:"match"`
	Status       string    `json:"status"`
}

type Impact struct {
	ID                int64            `json:"id"`
	CPID              *int             `json:"cp_id"`
	CPCode            string           `json:"cp_code"`
	ChangeType        string           `json:"change_type"`
	OldStatement      string           `json:"old_statement"`
	NewStatement      string           `json:"new_statement"`
	StatementHash     string           `json:"statement_hash"`
	AffectedSkills    int              `json:"affected_skills"`
	AffectedQuestions int              `json:"affected_questions"`
	Questions         []ImpactQuestion `json:"questions"`
}

type ChangeDetail struct {
	ChangeSummary
	Impacts []Impact `json:"impacts"`
	Review  *Review  `json:"review"`
}

func (h Handler) GetChange(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.Int64Param(w, r, "id")
	if !ok {
		return
	}
	if !h.stagingReady(r.Context()) {
		httpx.NotFound(w)
		return
	}
	rows, err := h.DB.Query(r.Context(), `SELECT * FROM (`+changeEventsSQL+`) c WHERE id = $1`, id)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	summary, err := pgx.CollectOneRow(rows, pgx.RowToStructByPos[ChangeSummary])
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}

	impacts, err := h.loadImpacts(r.Context(), id)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	review, err := h.loadReview(r.Context(), id)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"change": ChangeDetail{ChangeSummary: summary, Impacts: impacts, Review: review},
	})
}

func (h Handler) loadImpacts(ctx context.Context, eventID int64) ([]Impact, error) {
	rows, err := h.DB.Query(ctx, `
		SELECT id, cp_id, cp_code, change_type, old_statement, new_statement, statement_hash,
		       affected_skills, affected_questions
		FROM content_staging.curriculum_change_impacts
		WHERE change_event_id = $1
		ORDER BY cp_code`, eventID)
	if err != nil {
		return nil, err
	}
	// A dedicated scan struct: Questions is assembled below, not a column.
	type impactRow struct {
		ID                int64
		CPID              *int
		CPCode            string
		ChangeType        string
		OldStatement      string
		NewStatement      string
		StatementHash     string
		AffectedSkills    int
		AffectedQuestions int
	}
	scanned, err := pgx.CollectRows(rows, pgx.RowToStructByPos[impactRow])
	if err != nil {
		return nil, err
	}
	impacts := make([]Impact, 0, len(scanned))
	for _, r := range scanned {
		questions, err := h.loadImpactQuestions(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		impacts = append(impacts, Impact{
			ID: r.ID, CPID: r.CPID, CPCode: r.CPCode, ChangeType: r.ChangeType,
			OldStatement: r.OldStatement, NewStatement: r.NewStatement,
			StatementHash: r.StatementHash, AffectedSkills: r.AffectedSkills,
			AffectedQuestions: r.AffectedQuestions, Questions: questions,
		})
	}
	return impacts, nil
}

func (h Handler) loadImpactQuestions(ctx context.Context, impactID int64) ([]ImpactQuestion, error) {
	rows, err := h.DB.Query(ctx, `
		SELECT q.question_id, q.question_code, q.skill_id, COALESCE(k.name, ''), q.match, q.status
		FROM content_staging.curriculum_change_questions q
		LEFT JOIN skills k ON k.id = q.skill_id
		WHERE q.impact_id = $1
		ORDER BY q.question_code, q.question_id`, impactID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[ImpactQuestion])
}

func (h Handler) loadReview(ctx context.Context, eventID int64) (*Review, error) {
	var rev Review
	var decidedAt *time.Time
	err := h.DB.QueryRow(ctx, `
		SELECT r.id, r.change_event_id, r.status, r.notes, u.display_name, r.decided_at
		FROM curriculum_change_reviews r
		LEFT JOIN users u ON u.id = r.decided_by
		WHERE r.change_event_id = $1`, eventID).
		Scan(&rev.ID, &rev.EventID, &rev.Status, &rev.Notes, &rev.Reviewer, &decidedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rev.DecidedAt = decidedAt
	return &rev, nil
}
