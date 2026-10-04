// Package pipeline is the admin control surface for the Python content pipeline
// (M11). The UI enqueues *commands* into `content_staging.pipeline_jobs`; the
// Python worker daemon claims and runs them. This is the single control-table
// exception to "Go is read-only on content_staging" (roadmap D2/D5): Go may
// write pipeline_jobs only — never a pipeline data table. Source enable/disable
// is expressed as a job, so Go does not write source_registry either.
package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/audit"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/httpx"
)

type Handler struct{ DB *pgxpool.Pool }

// Job kinds the worker will execute; mirrors the DB CHECK constraint.
var jobKinds = map[string]bool{
	"crawl": true, "run_pipeline": true, "set_source_status": true, "watch": true,
}

var sourceStatuses = map[string]bool{
	"discovered": true, "verified": true, "approved": true, "active": true,
	"paused": true, "retired": true,
}

var crawlFrequencies = map[string]bool{
	"daily": true, "weekly": true, "monthly": true, "manual": true,
}

// readyJobs reports whether the pipeline worker has created its queue table.
func (h Handler) readyJobs(ctx context.Context) bool {
	return h.regclass(ctx, "content_staging.pipeline_jobs")
}

// readySources reports whether the pipeline has created the source registry.
func (h Handler) readySources(ctx context.Context) bool {
	return h.regclass(ctx, "content_staging.source_registry")
}

func (h Handler) regclass(ctx context.Context, name string) bool {
	var ready bool
	err := h.DB.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, name).Scan(&ready)
	return err == nil && ready
}

type Source struct {
	Code           string     `json:"code"`
	Name           string     `json:"name"`
	Status         string     `json:"status"`
	Enabled        bool       `json:"enabled"`
	CrawlFrequency string     `json:"crawl_frequency"`
	BaseURL        string     `json:"base_url"`
	LastCrawledAt  *time.Time `json:"last_crawled_at"`
	PendingJobs    int        `json:"pending_jobs"`
}

type Job struct {
	ID          int64           `json:"id"`
	Kind        string          `json:"kind"`
	Params      json.RawMessage `json:"params"`
	Status      string          `json:"status"`
	Progress    json.RawMessage `json:"progress"`
	Error       string          `json:"error"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	RequestedBy *uuid.UUID      `json:"requested_by"`
	CreatedAt   time.Time       `json:"created_at"`
	StartedAt   *time.Time      `json:"started_at"`
	FinishedAt  *time.Time      `json:"finished_at"`
}

type jobRow struct {
	ID          int64
	Kind        string
	Params      []byte
	Status      string
	Progress    []byte
	Error       string
	Attempts    int
	MaxAttempts int
	RequestedBy *uuid.UUID
	CreatedAt   time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
}

func (r jobRow) toJob() Job {
	return Job{
		ID: r.ID, Kind: r.Kind, Params: json.RawMessage(r.Params), Status: r.Status,
		Progress: json.RawMessage(r.Progress), Error: r.Error, Attempts: r.Attempts,
		MaxAttempts: r.MaxAttempts, RequestedBy: r.RequestedBy, CreatedAt: r.CreatedAt,
		StartedAt: r.StartedAt, FinishedAt: r.FinishedAt,
	}
}

const jobColumns = `id, kind, params, status, progress, error, attempts, max_attempts,
	requested_by, created_at, started_at, finished_at`

// ListSources returns the source registry with the last crawl time and whether a
// run is already pending for that source.
func (h Handler) ListSources(w http.ResponseWriter, r *http.Request) {
	if !h.readySources(r.Context()) || !h.readyJobs(r.Context()) {
		httpx.JSON(w, http.StatusOK, map[string]any{"sources": []Source{}, "available": false})
		return
	}
	rows, err := h.DB.Query(r.Context(), `
		SELECT s.code, s.name, s.status, s.enabled, s.crawl_frequency, s.base_url, s.last_crawled_at,
		       (SELECT count(*)::int FROM content_staging.pipeline_jobs j
		         WHERE j.status IN ('queued','running')
		           AND (j.params->'sources' IS NULL OR j.params->'sources' @> to_jsonb(s.code))) AS pending
		FROM content_staging.source_registry s
		ORDER BY s.priority, s.code`)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Source])
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sources": out, "available": true})
}

// SetSourceStatus enqueues a governance change (status/enabled/crawl_frequency).
func (h Handler) SetSourceStatus(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	var in struct {
		Status         *string `json:"status"`
		Enabled        *bool   `json:"enabled"`
		CrawlFrequency *string `json:"crawl_frequency"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	fields := map[string]string{}
	if in.Status != nil && !sourceStatuses[*in.Status] {
		fields["status"] = "Status tidak dikenal."
	}
	if in.CrawlFrequency != nil && !crawlFrequencies[*in.CrawlFrequency] {
		fields["crawl_frequency"] = "Frekuensi tidak dikenal."
	}
	if len(fields) > 0 {
		httpx.Invalid(w, fields)
		return
	}
	params := map[string]any{"code": code}
	if in.Status != nil {
		params["status"] = *in.Status
	}
	if in.Enabled != nil {
		params["enabled"] = *in.Enabled
	}
	if in.CrawlFrequency != nil {
		params["crawl_frequency"] = *in.CrawlFrequency
	}
	h.enqueue(w, r, "set_source_status", params, http.StatusAccepted)
}

// EnqueueJob accepts {kind, params} and queues a pipeline command.
func (h Handler) EnqueueJob(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Kind   string          `json:"kind"`
		Params json.RawMessage `json:"params"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if !jobKinds[in.Kind] {
		httpx.Invalid(w, map[string]string{"kind": "Jenis job tidak dikenal."})
		return
	}
	params := map[string]any{}
	if len(in.Params) > 0 {
		if err := json.Unmarshal(in.Params, &params); err != nil {
			httpx.Invalid(w, map[string]string{"params": "params harus berupa objek JSON."})
			return
		}
	}
	h.enqueue(w, r, in.Kind, params, http.StatusCreated)
}

func (h Handler) enqueue(w http.ResponseWriter, r *http.Request, kind string, params map[string]any, status int) {
	if !h.readyJobs(r.Context()) {
		httpx.Error(w, http.StatusConflict, "pipeline_data_unavailable", "Pipeline belum siap.")
		return
	}
	body, err := json.Marshal(params)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	u := auth.MustUser(r.Context())
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(ctx)
	var row jobRow
	err = tx.QueryRow(ctx, `INSERT INTO content_staging.pipeline_jobs (kind, params, requested_by)
		VALUES ($1, $2::jsonb, $3) RETURNING `+jobColumns, kind, body, u.ID).
		Scan(&row.ID, &row.Kind, &row.Params, &row.Status, &row.Progress, &row.Error, &row.Attempts,
			&row.MaxAttempts, &row.RequestedBy, &row.CreatedAt, &row.StartedAt, &row.FinishedAt)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	audit.Log(ctx, tx, u.ID, "pipeline.enqueue", "pipeline_job", fmt.Sprintf("%d", row.ID),
		map[string]any{"kind": kind})
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, status, map[string]any{"job": row.toJob()})
}

// ListJobs returns the recent job queue, newest first.
func (h Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	if !h.readyJobs(r.Context()) {
		httpx.JSON(w, http.StatusOK, map[string]any{"jobs": []Job{}, "available": false})
		return
	}
	q := r.URL.Query()
	rows, err := h.DB.Query(r.Context(), `SELECT `+jobColumns+` FROM content_staging.pipeline_jobs
		WHERE ($1 = '' OR status = $1) AND ($2 = '' OR kind = $2)
		ORDER BY created_at DESC, id DESC LIMIT 100`, q.Get("status"), q.Get("kind"))
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	raw, err := pgx.CollectRows(rows, pgx.RowToStructByPos[jobRow])
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	jobs := make([]Job, 0, len(raw))
	for _, row := range raw {
		jobs = append(jobs, row.toJob())
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"jobs": jobs, "available": true})
}

// GetJob returns one job with its per-stage progress.
func (h Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	if !h.readyJobs(r.Context()) {
		httpx.NotFound(w)
		return
	}
	id, ok := httpx.Int64Param(w, r, "id")
	if !ok {
		return
	}
	var row jobRow
	err := h.DB.QueryRow(r.Context(), `SELECT `+jobColumns+` FROM content_staging.pipeline_jobs WHERE id = $1`, id).
		Scan(&row.ID, &row.Kind, &row.Params, &row.Status, &row.Progress, &row.Error, &row.Attempts,
			&row.MaxAttempts, &row.RequestedBy, &row.CreatedAt, &row.StartedAt, &row.FinishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"job": row.toJob()})
}

// CancelJob marks a queued or running job as cancelled; the worker stops it
// between stages.
func (h Handler) CancelJob(w http.ResponseWriter, r *http.Request) {
	if !h.readyJobs(r.Context()) {
		httpx.NotFound(w)
		return
	}
	id, ok := httpx.Int64Param(w, r, "id")
	if !ok {
		return
	}
	h.mutate(w, r, `UPDATE content_staging.pipeline_jobs
		SET status = 'cancelled', finished_at = now(), updated_at = now()
		WHERE id = $1 AND status IN ('queued','running') RETURNING `+jobColumns,
		[]any{id}, "pipeline.cancel", "Job tidak bisa dibatalkan.")
}

// RetryJob re-queues a failed or cancelled job when it still has attempts left.
func (h Handler) RetryJob(w http.ResponseWriter, r *http.Request) {
	if !h.readyJobs(r.Context()) {
		httpx.NotFound(w)
		return
	}
	id, ok := httpx.Int64Param(w, r, "id")
	if !ok {
		return
	}
	h.mutate(w, r, `UPDATE content_staging.pipeline_jobs
		SET status = 'queued', error = '', started_at = NULL, finished_at = NULL, updated_at = now()
		WHERE id = $1 AND status IN ('failed','cancelled') AND attempts < max_attempts RETURNING `+jobColumns,
		[]any{id}, "pipeline.retry", "Job tidak bisa diulang (sudah habis percobaan atau sedang berjalan).")
}

func (h Handler) mutate(w http.ResponseWriter, r *http.Request, sql string, args []any, action, conflictMsg string) {
	u := auth.MustUser(r.Context())
	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(ctx)
	var row jobRow
	err = tx.QueryRow(ctx, sql, args...).
		Scan(&row.ID, &row.Kind, &row.Params, &row.Status, &row.Progress, &row.Error, &row.Attempts,
			&row.MaxAttempts, &row.RequestedBy, &row.CreatedAt, &row.StartedAt, &row.FinishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusConflict, "not_mutable", conflictMsg)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	audit.Log(ctx, tx, u.ID, action, "pipeline_job", fmt.Sprintf("%d", row.ID), nil)
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"job": row.toJob()})
}
