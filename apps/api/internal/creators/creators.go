// Package creators is the creator / revenue attribution surface (§39).
//
// It records who created content and how it performs — usage, completion and a
// quality score derived from real assessment evidence — so revenue share can be
// built later without another migration. M8 is readiness only: it computes no
// money. The rollup itself lives in `question_attribution` and is rebuilt from
// raw evidence by attribution.go.
//
// Creator identity is decoupled from `users` because `questions.created_by`
// conflates the author with whoever pressed publish, and external authors have
// no login. Pipeline-published content is attributed to the seeded system
// creator by migration 0007.
package creators

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/audit"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/httpx"
)

// Creator kinds and statuses.
const (
	KindSystem   = "system"
	KindStaff    = "staff"
	KindExternal = "external"

	StatusActive   = "active"
	StatusInactive = "inactive"
)

// SystemCreatorID is the fixed id seeded by migration 0007 for
// pipeline-published content.
var SystemCreatorID = uuid.MustParse("00000000-0000-0000-0000-00000000c0de")

var (
	ErrCreatorNotFound  = errors.New("creator not found")
	ErrQuestionNotFound = errors.New("question not found")
)

type Handler struct{ DB *pgxpool.Pool }

type Creator struct {
	ID          uuid.UUID  `json:"id"`
	Code        string     `json:"code"`
	DisplayName string     `json:"display_name"`
	Kind        string     `json:"kind"`
	UserID      *uuid.UUID `json:"user_id"`
	SharePct    float64    `json:"share_pct"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
}

// CreatorSummary is a creator with its aggregate attribution, used by the list
// and the detail header.
type CreatorSummary struct {
	ID              uuid.UUID  `json:"id"`
	Code            string     `json:"code"`
	DisplayName     string     `json:"display_name"`
	Kind            string     `json:"kind"`
	UserID          *uuid.UUID `json:"user_id"`
	SharePct        float64    `json:"share_pct"`
	Status          string     `json:"status"`
	Questions       int        `json:"questions"`
	Published       int        `json:"published"`
	UsageCount      int        `json:"usage_count"`
	CompletionCount int        `json:"completion_count"`
	AvgQuality      float64    `json:"avg_quality"`
	CreatedAt       time.Time  `json:"created_at"`
}

// creatorSummarySelect is the shared base for the list and the detail header;
// callers append a constant WHERE/GROUP BY clause.
const creatorSummarySelect = `
	SELECT c.id, c.code, c.display_name, c.kind, c.user_id, c.share_pct::float8, c.status,
	       count(q.id)::int,
	       count(q.id) FILTER (WHERE q.status = 'published')::int,
	       COALESCE(sum(a.usage_count), 0)::int,
	       COALESCE(sum(a.completion_count), 0)::int,
	       COALESCE(round(avg(a.quality_score) FILTER (WHERE a.usage_count > 0), 2), 0)::float8,
	       c.created_at
	FROM creators c
	LEFT JOIN questions q ON q.creator_id = c.id
	LEFT JOIN question_attribution a ON a.question_id = q.id`

func (h Handler) ListCreators(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, err := h.DB.Query(r.Context(), creatorSummarySelect+`
		WHERE ($1 = '' OR c.status = $1)
		  AND ($2 = '' OR c.display_name ILIKE '%' || $2 || '%' OR c.code ILIKE '%' || $2 || '%')
		GROUP BY c.id
		ORDER BY c.display_name, c.id
		LIMIT 300`, q.Get("status"), q.Get("q"))
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[CreatorSummary])
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	counts := map[string]int{}
	crows, err := h.DB.Query(r.Context(), `SELECT status, count(*)::int FROM creators GROUP BY status`)
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
	httpx.JSON(w, http.StatusOK, map[string]any{"creators": out, "counts": counts})
}

// AttributionRow is one of a creator's questions with its rollup. Questions with
// no evidence yet come back with zero counts (LEFT JOIN).
type AttributionRow struct {
	QuestionID      uuid.UUID  `json:"question_id"`
	Code            string     `json:"code"`
	Type            string     `json:"type"`
	Status          string     `json:"status"`
	SkillID         int        `json:"skill_id"`
	SkillName       string     `json:"skill_name"`
	UsageCount      int        `json:"usage_count"`
	CompletionCount int        `json:"completion_count"`
	CorrectCount    int        `json:"correct_count"`
	FirstTryCount   int        `json:"first_try_count"`
	HintCount       int        `json:"hint_count"`
	QualityScore    float64    `json:"quality_score"`
	PublishedAt     *time.Time `json:"published_at"`
	ComputedAt      *time.Time `json:"computed_at"`
}

func (h Handler) GetCreator(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "creatorID")
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(),
		creatorSummarySelect+" WHERE c.id = $1 GROUP BY c.id", id)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	creator, err := pgx.CollectOneRow(rows, pgx.RowToStructByPos[CreatorSummary])
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	questions, err := h.loadQuestions(r.Context(), id)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"creator": creator, "questions": questions})
}

func (h Handler) loadQuestions(ctx context.Context, creatorID uuid.UUID) ([]AttributionRow, error) {
	rows, err := h.DB.Query(ctx, `
		SELECT q.id, q.code, q.type, q.status, q.skill_id, COALESCE(s.name, ''),
		       COALESCE(a.usage_count, 0), COALESCE(a.completion_count, 0),
		       COALESCE(a.correct_count, 0), COALESCE(a.first_try_count, 0), COALESCE(a.hint_count, 0),
		       COALESCE(a.quality_score, 0)::float8,
		       COALESCE(a.published_at, q.published_at), a.computed_at
		FROM questions q
		LEFT JOIN skills s ON s.id = q.skill_id
		LEFT JOIN question_attribution a ON a.question_id = q.id
		WHERE q.creator_id = $1
		ORDER BY q.updated_at DESC, q.id
		LIMIT 300`, creatorID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[AttributionRow])
}

func validKind(k string) bool {
	switch k {
	case KindSystem, KindStaff, KindExternal:
		return true
	default:
		return false
	}
}

func validStatus(s string) bool {
	switch s {
	case StatusActive, StatusInactive:
		return true
	default:
		return false
	}
}

type createInput struct {
	Code        string     `json:"code"`
	DisplayName string     `json:"display_name"`
	Kind        string     `json:"kind"`
	UserID      *uuid.UUID `json:"user_id"`
	SharePct    *float64   `json:"share_pct"`
}

func (h Handler) CreateCreator(w http.ResponseWriter, r *http.Request) {
	var in createInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if in.DisplayName == "" || len(in.DisplayName) > 200 {
		httpx.Invalid(w, map[string]string{"display_name": "Nama kreator wajib diisi (maks. 200 karakter)."})
		return
	}
	if in.Kind == "" {
		in.Kind = KindStaff
	}
	if !validKind(in.Kind) {
		httpx.Invalid(w, map[string]string{"kind": "Jenis kreator tidak dikenal."})
		return
	}
	share := 0.0
	if in.SharePct != nil {
		share = *in.SharePct
	}
	if share < 0 || share > 100 {
		httpx.Invalid(w, map[string]string{"share_pct": "Porsi bagi hasil harus 0–100."})
		return
	}
	code := strings.TrimSpace(in.Code)
	if code == "" {
		code = "C-" + strings.ToUpper(uuid.NewString()[:8])
	}
	u := auth.MustUser(r.Context())
	var c Creator
	err := h.DB.QueryRow(r.Context(), `
		INSERT INTO creators (code, display_name, kind, user_id, share_pct)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, code, display_name, kind, user_id, share_pct::float8, status, created_at`,
		code, in.DisplayName, in.Kind, in.UserID, share).
		Scan(&c.ID, &c.Code, &c.DisplayName, &c.Kind, &c.UserID, &c.SharePct, &c.Status, &c.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpx.Invalid(w, map[string]string{"code": "Kode kreator sudah dipakai."})
			return
		}
		httpx.Internal(w, r, err)
		return
	}
	audit.Log(r.Context(), h.DB, u.ID, "creator.create", "creator", c.ID.String(),
		map[string]any{"code": c.Code, "kind": c.Kind, "share_pct": c.SharePct})
	httpx.JSON(w, http.StatusCreated, map[string]any{"creator": c})
}

type updateInput struct {
	DisplayName *string  `json:"display_name"`
	Status      *string  `json:"status"`
	SharePct    *float64 `json:"share_pct"`
}

func (h Handler) UpdateCreator(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "creatorID")
	if !ok {
		return
	}
	var in updateInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	if in.DisplayName != nil {
		name := strings.TrimSpace(*in.DisplayName)
		if name == "" || len(name) > 200 {
			httpx.Invalid(w, map[string]string{"display_name": "Nama kreator wajib diisi (maks. 200 karakter)."})
			return
		}
		in.DisplayName = &name
	}
	if in.Status != nil && !validStatus(*in.Status) {
		httpx.Invalid(w, map[string]string{"status": "Status kreator tidak dikenal."})
		return
	}
	if in.SharePct != nil && (*in.SharePct < 0 || *in.SharePct > 100) {
		httpx.Invalid(w, map[string]string{"share_pct": "Porsi bagi hasil harus 0–100."})
		return
	}
	u := auth.MustUser(r.Context())
	var c Creator
	err := h.DB.QueryRow(r.Context(), `
		UPDATE creators SET
			display_name = COALESCE($2, display_name),
			status = COALESCE($3, status),
			share_pct = COALESCE($4, share_pct),
			updated_at = now()
		WHERE id = $1
		RETURNING id, code, display_name, kind, user_id, share_pct::float8, status, created_at`,
		id, in.DisplayName, in.Status, in.SharePct).
		Scan(&c.ID, &c.Code, &c.DisplayName, &c.Kind, &c.UserID, &c.SharePct, &c.Status, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	audit.Log(r.Context(), h.DB, u.ID, "creator.update", "creator", c.ID.String(),
		map[string]any{"status": c.Status, "share_pct": c.SharePct})
	httpx.JSON(w, http.StatusOK, map[string]any{"creator": c})
}

// AssignCreator sets a question's creator and refreshes its rollup so the
// denormalised creator_id follows.
func (h Handler) AssignCreator(w http.ResponseWriter, r *http.Request) {
	questionID, ok := httpx.UUIDParam(w, r, "questionID")
	if !ok {
		return
	}
	var in struct {
		CreatorID uuid.UUID `json:"creator_id"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	u := auth.MustUser(r.Context())
	err := h.setCreator(r.Context(), questionID, &in.CreatorID, u.ID)
	if errors.Is(err, ErrCreatorNotFound) {
		httpx.Invalid(w, map[string]string{"creator_id": "Kreator tidak ditemukan."})
		return
	}
	if errors.Is(err, ErrQuestionNotFound) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// UnassignCreator clears a question's creator (back to unattributed).
func (h Handler) UnassignCreator(w http.ResponseWriter, r *http.Request) {
	questionID, ok := httpx.UUIDParam(w, r, "questionID")
	if !ok {
		return
	}
	u := auth.MustUser(r.Context())
	err := h.setCreator(r.Context(), questionID, nil, u.ID)
	if errors.Is(err, ErrQuestionNotFound) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h Handler) setCreator(ctx context.Context, questionID uuid.UUID, creatorID *uuid.UUID, actor uuid.UUID) error {
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if creatorID != nil {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM creators WHERE id = $1)`, *creatorID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrCreatorNotFound
		}
	}
	tag, err := tx.Exec(ctx, `UPDATE questions SET creator_id = $2, updated_at = now() WHERE id = $1`, questionID, creatorID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrQuestionNotFound
	}
	if err := RecomputeQuestions(ctx, tx, []uuid.UUID{questionID}); err != nil {
		return err
	}
	action := "question.creator_unassign"
	data := map[string]any{}
	if creatorID != nil {
		action = "question.creator_assign"
		data["creator_id"] = creatorID.String()
	}
	audit.Log(ctx, tx, actor, action, "question", questionID.String(), data)
	return tx.Commit(ctx)
}
