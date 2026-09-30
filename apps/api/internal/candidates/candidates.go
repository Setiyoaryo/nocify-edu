package candidates

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

// stagingReady reports whether the pipeline has created its schema yet. The
// pipeline is a separate service; in a fresh environment it may not have run, so
// the CMS degrades to "no candidates" instead of a 500.
func (h Handler) stagingReady(ctx context.Context) bool {
	var ready bool
	err := h.DB.QueryRow(ctx, `SELECT to_regclass('content_staging.question_candidates') IS NOT NULL
		AND to_regclass('content_staging.material_candidates') IS NOT NULL`).Scan(&ready)
	return err == nil && ready
}

// candidatesUnion is the shared read-only view of both candidate tables with
// their publication state. The Go role has SELECT on content_staging only.
const candidatesUnion = `
	SELECT 'material' AS kind, m.id, m.source_id, s.code AS source_code, m.title AS label,
	       m.status, m.confidence::float8,
	       (SELECT count(*)::int FROM jsonb_array_elements(m.qa_issues) e WHERE (e->>'blocking')::boolean) AS blocking_issues,
	       m.skill_id, COALESCE(k.name, '') AS skill_name,
	       m.phase_code, m.grade_code, m.subject_code,
	       p.status AS publication_status, p.decision, m.updated_at
	FROM content_staging.material_candidates m
	JOIN content_staging.source_registry s ON s.id = m.source_id
	LEFT JOIN skills k ON k.id = m.skill_id
	LEFT JOIN candidate_publications p ON p.kind = 'material' AND p.candidate_id = m.id
	UNION ALL
	SELECT 'question', q.id, q.source_id, s.code, q.prompt,
	       q.status, q.confidence::float8,
	       (SELECT count(*)::int FROM jsonb_array_elements(q.qa_issues) e WHERE (e->>'blocking')::boolean),
	       q.skill_id, COALESCE(k.name, ''),
	       q.phase_code, q.grade_code, q.subject_code,
	       p.status, p.decision, q.updated_at
	FROM content_staging.question_candidates q
	JOIN content_staging.source_registry s ON s.id = q.source_id
	LEFT JOIN skills k ON k.id = q.skill_id
	LEFT JOIN candidate_publications p ON p.kind = 'question' AND p.candidate_id = q.id
`

type CandidateSummary struct {
	Kind              string    `json:"kind"`
	ID                int64     `json:"id"`
	SourceID          int       `json:"source_id"`
	SourceCode        string    `json:"source_code"`
	Label             string    `json:"label"`
	Status            string    `json:"status"`
	Confidence        float64   `json:"confidence"`
	BlockingIssues    int       `json:"blocking_issues"`
	SkillID           *int      `json:"skill_id"`
	SkillName         string    `json:"skill_name"`
	PhaseCode         string    `json:"phase_code"`
	GradeCode         string    `json:"grade_code"`
	SubjectCode       string    `json:"subject_code"`
	PublicationStatus *string   `json:"publication_status"`
	Decision          *string   `json:"decision"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (h Handler) ListCandidates(w http.ResponseWriter, r *http.Request) {
	if !h.stagingReady(r.Context()) {
		httpx.JSON(w, http.StatusOK, map[string]any{"candidates": []CandidateSummary{}, "counts": map[string]int{}, "available": false})
		return
	}
	q := r.URL.Query()
	rows, err := h.DB.Query(r.Context(), `
		SELECT kind, id, source_id, source_code, label, status, confidence, blocking_issues,
		       skill_id, skill_name, phase_code, grade_code, subject_code,
		       publication_status, decision, updated_at
		FROM (`+candidatesUnion+`) c
		WHERE ($1 = '' OR kind = $1) AND ($2 = '' OR status = $2) AND ($3 = '' OR source_code = $3)
		  AND ($4 = '' OR label ILIKE '%' || $4 || '%' OR source_code ILIKE '%' || $4 || '%')
		ORDER BY updated_at DESC LIMIT 300`,
		q.Get("kind"), q.Get("status"), q.Get("source"), q.Get("q"))
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[CandidateSummary])
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	counts := map[string]int{}
	crows, err := h.DB.Query(r.Context(), `SELECT status, count(*)::int FROM (`+candidatesUnion+`) c GROUP BY status`)
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
	httpx.JSON(w, http.StatusOK, map[string]any{"candidates": out, "counts": counts, "available": true})
}

// detailRow is the flat shape both detail queries scan into; assembled into
// CandidateDetail for the client.
type detailRow struct {
	ID                  int64
	SourceID            int
	SkillID             *int
	CPID                *int
	PhaseCode           string
	GradeCode           string
	SubjectCode         string
	Template            string
	Type                string
	Difficulty          *int
	LearningObjective   string
	Title               string
	Summary             string
	Prompt              string
	Payload             []byte
	ContentHash         string
	Readability         []byte
	Language            string
	Confidence          float64
	QA                  []byte
	Status              string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	SourceCode          string
	SourceName          string
	SourceBaseURL       string
	DocumentID          int64
	DocumentTitle       string
	CanonicalURL        string
	RetrievedAt         *time.Time
	License             string
	ChunkID             int64
	Heading             string
	ChunkText           string
	PageStart           *int
	PageEnd             *int
	SkillName           string
	SkillCode           string
	CPCode              string
	CPStatement         string
	PhaseName           string
	GradeName           string
	SubjectName         string
	PubStatus           *string
	PubDecision         *string
	PubNotes            *string
	PubChecklist        []byte
	PubReviewer         *string
	PubDecidedAt        *time.Time
	CanonicalQuestionID *uuid.UUID
	PubAttempts         *int
	PubLastError        *string
	PubPublishedAt      *time.Time
}

type CandidateDetail struct {
	Kind              string          `json:"kind"`
	ID                int64           `json:"id"`
	SourceID          int             `json:"source_id"`
	Status            string          `json:"status"`
	Confidence        float64         `json:"confidence"`
	QA                json.RawMessage `json:"qa_issues"`
	ContentHash       string          `json:"content_hash"`
	Language          string          `json:"language"`
	Template          string          `json:"template"`
	Type              string          `json:"type"`
	Difficulty        *int            `json:"difficulty"`
	LearningObjective string          `json:"learning_objective"`
	Title             string          `json:"title"`
	Summary           string          `json:"summary"`
	Prompt            string          `json:"prompt"`
	Payload           json.RawMessage `json:"payload,omitempty"`
	Readability       json.RawMessage `json:"readability,omitempty"`
	Source            SourceRef       `json:"source"`
	Lineage           Lineage         `json:"lineage"`
	Mapping           Mapping         `json:"mapping"`
	Publication       *Publication    `json:"publication"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type SourceRef struct {
	ID      int    `json:"id"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
}

type Lineage struct {
	DocumentID    int64      `json:"document_id"`
	DocumentTitle string     `json:"document_title"`
	CanonicalURL  string     `json:"canonical_url"`
	RetrievedAt   *time.Time `json:"retrieved_at"`
	License       string     `json:"license"`
	ChunkID       int64      `json:"chunk_id"`
	Heading       string     `json:"heading"`
	Text          string     `json:"text"`
	PageStart     *int       `json:"page_start"`
	PageEnd       *int       `json:"page_end"`
}

type Mapping struct {
	SkillID     *int   `json:"skill_id"`
	SkillName   string `json:"skill_name"`
	SkillCode   string `json:"skill_code"`
	CPID        *int   `json:"cp_id"`
	CPCode      string `json:"cp_code"`
	CPStatement string `json:"cp_statement"`
	PhaseCode   string `json:"phase_code"`
	PhaseName   string `json:"phase_name"`
	GradeCode   string `json:"grade_code"`
	GradeName   string `json:"grade_name"`
	SubjectCode string `json:"subject_code"`
	SubjectName string `json:"subject_name"`
}

type Publication struct {
	ID                  int64           `json:"id"`
	Kind                string          `json:"kind"`
	CandidateID         int64           `json:"candidate_id"`
	Decision            string          `json:"decision"`
	Status              string          `json:"status"`
	Notes               string          `json:"notes"`
	Checklist           json.RawMessage `json:"checklist"`
	Reviewer            *string         `json:"reviewer"`
	DecidedAt           time.Time       `json:"decided_at"`
	CanonicalQuestionID *uuid.UUID      `json:"canonical_question_id"`
	Attempts            int             `json:"attempts"`
	LastError           string          `json:"last_error"`
	PublishedAt         *time.Time      `json:"published_at"`
}

const questionDetailSQL = `
	SELECT q.id, q.source_id, q.skill_id, q.cp_id, q.phase_code, q.grade_code, q.subject_code,
	       q.template, q.type, q.difficulty, q.learning_objective, '' AS title, '' AS summary,
	       q.prompt, q.payload, q.content_hash, NULL::jsonb AS readability, d.language,
	       q.confidence::float8, q.qa_issues, q.status, q.created_at, q.updated_at,
	       s.code, s.name, s.base_url,
	       d.id, d.title, d.canonical_url, d.retrieved_at, d.license,
	       c.id, c.heading, c.text, c.page_start, c.page_end,
	       COALESCE(k.name, ''), COALESCE(k.code, ''), COALESCE(cp.code, ''), COALESCE(cp.statement, ''),
	       COALESCE(ph.name, ''), COALESCE(g.name, ''), COALESCE(sub.name, ''),
	       p.status, p.decision, p.notes, p.checklist, u.display_name, p.decided_at,
	       p.canonical_question_id, p.attempts, p.last_error, p.published_at
	FROM content_staging.question_candidates q
	JOIN content_staging.source_documents d ON d.id = q.document_id
	JOIN content_staging.document_chunks c ON c.id = q.chunk_id
	JOIN content_staging.source_registry s ON s.id = q.source_id
	LEFT JOIN skills k ON k.id = q.skill_id
	LEFT JOIN curriculum_cp cp ON cp.id = q.cp_id
	LEFT JOIN phases ph ON ph.code = q.phase_code
	LEFT JOIN grades g ON g.code = q.grade_code
	LEFT JOIN subjects sub ON sub.code = q.subject_code
	LEFT JOIN candidate_publications p ON p.kind = 'question' AND p.candidate_id = q.id
	LEFT JOIN users u ON u.id = p.decided_by
	WHERE q.id = $1`

const materialDetailSQL = `
	SELECT m.id, m.source_id, m.skill_id, m.cp_id, m.phase_code, m.grade_code, m.subject_code,
	       '' AS template, '' AS type, m.difficulty_hint, m.learning_objective, m.title, m.summary,
	       '' AS prompt, NULL::jsonb AS payload, m.content_hash,
	       jsonb_build_object(
	           'age_band', m.age_band, 'reading_level', m.reading_level, 'complexity', m.complexity,
	           'word_count', m.word_count, 'sentence_count', m.sentence_count,
	           'avg_sentence_length', m.avg_sentence_length,
	           'visual_required', m.visual_required, 'audio_required', m.audio_required) AS readability,
	       m.language,
	       m.confidence::float8, m.qa_issues, m.status, m.created_at, m.updated_at,
	       s.code, s.name, s.base_url,
	       d.id, d.title, d.canonical_url, d.retrieved_at, d.license,
	       c.id, c.heading, c.text, c.page_start, c.page_end,
	       COALESCE(k.name, ''), COALESCE(k.code, ''), COALESCE(cp.code, ''), COALESCE(cp.statement, ''),
	       COALESCE(ph.name, ''), COALESCE(g.name, ''), COALESCE(sub.name, ''),
	       p.status, p.decision, p.notes, p.checklist, u.display_name, p.decided_at,
	       p.canonical_question_id, p.attempts, p.last_error, p.published_at
	FROM content_staging.material_candidates m
	JOIN content_staging.source_documents d ON d.id = m.document_id
	JOIN content_staging.document_chunks c ON c.id = m.chunk_id
	JOIN content_staging.source_registry s ON s.id = m.source_id
	LEFT JOIN skills k ON k.id = m.skill_id
	LEFT JOIN curriculum_cp cp ON cp.id = m.cp_id
	LEFT JOIN phases ph ON ph.code = m.phase_code
	LEFT JOIN grades g ON g.code = m.grade_code
	LEFT JOIN subjects sub ON sub.code = m.subject_code
	LEFT JOIN candidate_publications p ON p.kind = 'material' AND p.candidate_id = m.id
	LEFT JOIN users u ON u.id = p.decided_by
	WHERE m.id = $1`

func (d detailRow) toDetail(kind string) CandidateDetail {
	detail := CandidateDetail{
		Kind:              kind,
		ID:                d.ID,
		SourceID:          d.SourceID,
		Status:            d.Status,
		Confidence:        d.Confidence,
		QA:                json.RawMessage(d.QA),
		ContentHash:       d.ContentHash,
		Language:          d.Language,
		Template:          d.Template,
		Type:              d.Type,
		Difficulty:        d.Difficulty,
		LearningObjective: d.LearningObjective,
		Title:             d.Title,
		Summary:           d.Summary,
		Prompt:            d.Prompt,
		Source:            SourceRef{ID: d.SourceID, Code: d.SourceCode, Name: d.SourceName, BaseURL: d.SourceBaseURL},
		Lineage: Lineage{
			DocumentID: d.DocumentID, DocumentTitle: d.DocumentTitle, CanonicalURL: d.CanonicalURL,
			RetrievedAt: d.RetrievedAt, License: d.License, ChunkID: d.ChunkID, Heading: d.Heading,
			Text: d.ChunkText, PageStart: d.PageStart, PageEnd: d.PageEnd,
		},
		Mapping: Mapping{
			SkillID: d.SkillID, SkillName: d.SkillName, SkillCode: d.SkillCode,
			CPID: d.CPID, CPCode: d.CPCode, CPStatement: d.CPStatement,
			PhaseCode: d.PhaseCode, PhaseName: d.PhaseName,
			GradeCode: d.GradeCode, GradeName: d.GradeName,
			SubjectCode: d.SubjectCode, SubjectName: d.SubjectName,
		},
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
	if len(d.Payload) > 0 {
		detail.Payload = json.RawMessage(d.Payload)
	}
	if len(d.Readability) > 0 {
		detail.Readability = json.RawMessage(d.Readability)
	}
	if d.PubStatus != nil {
		pub := Publication{
			Kind: kind, CandidateID: d.ID,
			Decision: deref(d.PubDecision), Status: *d.PubStatus,
			Notes: deref(d.PubNotes), Checklist: json.RawMessage(d.PubChecklist),
			Reviewer: d.PubReviewer, CanonicalQuestionID: d.CanonicalQuestionID,
			Attempts: deref(d.PubAttempts), LastError: deref(d.PubLastError),
			PublishedAt: d.PubPublishedAt,
		}
		if d.PubDecidedAt != nil {
			pub.DecidedAt = *d.PubDecidedAt
		}
		detail.Publication = &pub
	}
	return detail
}

func deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

func (h Handler) GetCandidate(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")
	if !ValidKind(kind) {
		httpx.NotFound(w)
		return
	}
	id, ok := httpx.Int64Param(w, r, "id")
	if !ok {
		return
	}
	if !h.stagingReady(r.Context()) {
		httpx.NotFound(w)
		return
	}
	sql := questionDetailSQL
	if kind == KindMaterial {
		sql = materialDetailSQL
	}
	rows, err := h.DB.Query(r.Context(), sql, id)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	row, err := pgx.CollectOneRow(rows, pgx.RowToStructByPos[detailRow])
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"candidate": row.toDetail(kind)})
}

func (h Handler) DecideCandidate(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")
	if !ValidKind(kind) {
		httpx.NotFound(w)
		return
	}
	id, ok := httpx.Int64Param(w, r, "id")
	if !ok {
		return
	}
	var in struct {
		Decision  string          `json:"decision"`
		Checklist map[string]bool `json:"checklist"`
		Notes     string          `json:"notes"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if err := ValidateDecision(kind, in.Decision, in.Checklist); err != nil {
		httpx.Invalid(w, map[string]string{"decision": err.Error()})
		return
	}
	if !h.stagingReady(r.Context()) {
		httpx.Error(w, http.StatusConflict, "pipeline_data_unavailable", "Data pipeline belum tersedia.")
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

	table := "question_candidates"
	if kind == KindMaterial {
		table = "material_candidates"
	}
	var sourceID int
	err = tx.QueryRow(ctx, `SELECT source_id FROM content_staging.`+table+` WHERE id = $1`, id).Scan(&sourceID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}

	var current *string
	if err := tx.QueryRow(ctx, `SELECT status FROM candidate_publications WHERE kind = $1 AND candidate_id = $2`, kind, id).Scan(&current); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		httpx.Internal(w, r, err)
		return
	}
	if current != nil && (*current == StatusPublished || *current == StatusPublishing) {
		httpx.Error(w, http.StatusConflict, "already_published", "Kandidat sudah dipublikasikan.")
		return
	}

	status, _ := DecisionStatus(kind, in.Decision)
	checklist := in.Checklist
	if checklist == nil {
		checklist = map[string]bool{}
	}
	checklistJSON, err := json.Marshal(checklist)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO candidate_publications
			(kind, candidate_id, source_id, decision, decided_by, decided_at, notes, checklist, status, last_error, updated_at)
		VALUES ($1, $2, $3, $4, $5, now(), $6, $7, $8, '', now())
		ON CONFLICT (kind, candidate_id) DO UPDATE SET
			decision = EXCLUDED.decision, decided_by = EXCLUDED.decided_by, decided_at = now(),
			notes = EXCLUDED.notes, checklist = EXCLUDED.checklist, status = EXCLUDED.status,
			canonical_question_id = NULL, attempts = 0, last_error = '', published_at = NULL,
			updated_at = now()`,
		kind, id, sourceID, in.Decision, u.ID, in.Notes, checklistJSON, status); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	audit.Log(ctx, tx, u.ID, "candidate.decision", "candidate", fmt.Sprintf("%s:%d", kind, id),
		map[string]any{"decision": in.Decision, "status": status})
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	pub, err := h.getPublication(ctx, kind, id)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"publication": pub})
}

func (h Handler) getPublication(ctx context.Context, kind string, id int64) (*Publication, error) {
	var p Publication
	var decidedAt *time.Time
	err := h.DB.QueryRow(ctx, `
		SELECT p.id, p.kind, p.candidate_id, p.decision, p.status, p.notes, p.checklist,
		       u.display_name, p.decided_at, p.canonical_question_id, p.attempts, p.last_error, p.published_at
		FROM candidate_publications p LEFT JOIN users u ON u.id = p.decided_by
		WHERE p.kind = $1 AND p.candidate_id = $2`, kind, id).
		Scan(&p.ID, &p.Kind, &p.CandidateID, &p.Decision, &p.Status, &p.Notes, &p.Checklist,
			&p.Reviewer, &decidedAt, &p.CanonicalQuestionID, &p.Attempts, &p.LastError, &p.PublishedAt)
	if err != nil {
		return nil, err
	}
	if decidedAt != nil {
		p.DecidedAt = *decidedAt
	}
	return &p, nil
}
