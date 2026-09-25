package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/audit"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/db"
	"github.com/nocify/eduplay/api/internal/httpx"
	"github.com/nocify/eduplay/api/internal/questions"
)

type Handler struct{ DB *pgxpool.Pool }

func (h Handler) QuestionTypes(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"types": questions.Types(), "checklist": QualityChecklist})
}

type QuestionRow struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Type        string    `json:"type"`
	Difficulty  int       `json:"difficulty"`
	Status      string    `json:"status"`
	Live        bool      `json:"live"`    // a version is being served to children
	Pending     bool      `json:"pending"` // current version differs from the live one
	Prompt      string    `json:"prompt"`
	SkillID     int       `json:"skill_id"`
	SkillName   string    `json:"skill_name"`
	ChapterName string    `json:"chapter_name"`
	SubjectName string    `json:"subject_name"`
	GradeName   string    `json:"grade_name"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (h Handler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, err := h.DB.Query(r.Context(), `
		SELECT q.id, q.code, q.type, q.difficulty, q.status, q.published_version_id IS NOT NULL,
		       q.published_version_id IS DISTINCT FROM q.current_version_id AND q.published_version_id IS NOT NULL,
		       COALESCE(qv.content->>'prompt', ''), k.id, k.name, c.name, s.name, g.name, q.updated_at
		FROM questions q
		JOIN question_versions qv ON qv.id = q.current_version_id
		JOIN skills k ON k.id = q.skill_id
		JOIN chapters c ON c.id = k.chapter_id
		JOIN subjects s ON s.id = c.subject_id
		JOIN grades g ON g.id = c.grade_id
		WHERE ($1 = '' OR q.status = $1) AND ($2 = 0 OR k.id = $2) AND ($3 = 0 OR s.id = $3) AND ($4 = 0 OR g.id = $4)
		  AND ($5 = '' OR q.code ILIKE '%' || $5 || '%' OR qv.content->>'prompt' ILIKE '%' || $5 || '%')
		ORDER BY q.updated_at DESC LIMIT 300`,
		q.Get("status"), httpx.QueryInt(r, "skill_id", 0), httpx.QueryInt(r, "subject_id", 0), httpx.QueryInt(r, "grade_id", 0), q.Get("q"))
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[QuestionRow])
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var counts = map[string]int{}
	crows, err := h.DB.Query(r.Context(), `SELECT status, count(*)::int FROM questions GROUP BY status`)
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
	httpx.JSON(w, http.StatusOK, map[string]any{"questions": out, "counts": counts})
}

type QuestionInput struct {
	Code              string          `json:"code"`
	SkillID           int             `json:"skill_id"`
	Type              string          `json:"type"`
	Difficulty        int             `json:"difficulty"`
	LearningObjective string          `json:"learning_objective"`
	Source            string          `json:"source"`
	Content           json.RawMessage `json:"content"`
	Answer            json.RawMessage `json:"answer"`
	Explanation       string          `json:"explanation"`
	Hint              string          `json:"hint"`
}

func (in *QuestionInput) Validate(ctx context.Context, q db.DBTX) map[string]string {
	f := map[string]string{}
	in.LearningObjective = strings.TrimSpace(in.LearningObjective)
	in.Explanation = strings.TrimSpace(in.Explanation)
	in.Source = strings.TrimSpace(in.Source)
	if in.LearningObjective == "" {
		f["learning_objective"] = "Tujuan pembelajaran wajib."
	}
	if in.Explanation == "" {
		f["explanation"] = "Penjelasan wajib."
	}
	if in.Source == "" {
		f["source"] = "Sumber/revisi wajib dicatat."
	}
	if in.Difficulty < 1 || in.Difficulty > 5 {
		f["difficulty"] = "Kesulitan 1–5."
	}
	var ok bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM skills WHERE id = $1)`, in.SkillID).Scan(&ok); err != nil || !ok {
		f["skill_id"] = "Skill tidak valid."
	}
	if err := questions.Validate(in.Type, in.Content, in.Answer); err != nil {
		f["content"] = err.Error()
	}
	return f
}

var mediaRef = regexp.MustCompile(`/api/v1/media/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)

// linkMedia records which assets a version uses, derived from URLs in content,
// so asset usage and publish gates cannot drift from the actual content.
func linkMedia(ctx context.Context, tx pgx.Tx, versionID uuid.UUID, content json.RawMessage) error {
	for _, m := range mediaRef.FindAllStringSubmatch(string(content), -1) {
		tag, err := tx.Exec(ctx, `INSERT INTO question_media (question_version_id, asset_id)
			SELECT $1, id FROM assets WHERE id = $2 ON CONFLICT DO NOTHING`, versionID, m[1])
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			var exists bool
			_ = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM question_media WHERE question_version_id = $1 AND asset_id = $2)`, versionID, m[1]).Scan(&exists)
			if !exists {
				return fmt.Errorf("unknown asset %s", m[1])
			}
		}
	}
	return nil
}

// CreateQuestion is exported for the seeder, which runs the same validation.
func CreateQuestion(ctx context.Context, tx pgx.Tx, in QuestionInput, author uuid.UUID) (uuid.UUID, error) {
	if in.Code == "" {
		in.Code = "Q-" + strings.ToUpper(uuid.NewString()[:8])
	}
	var authorArg any
	if author != uuid.Nil {
		authorArg = author
	}
	var qid, vid uuid.UUID
	if err := tx.QueryRow(ctx, `INSERT INTO questions (code, skill_id, type, difficulty, learning_objective, source, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, in.Code, in.SkillID, in.Type, in.Difficulty, in.LearningObjective, in.Source, authorArg).Scan(&qid); err != nil {
		return qid, err
	}
	if err := tx.QueryRow(ctx, `INSERT INTO question_versions (question_id, version, content, answer, explanation, hint, created_by)
		VALUES ($1, 1, $2, $3, $4, $5, $6) RETURNING id`, qid, in.Content, in.Answer, in.Explanation, in.Hint, authorArg).Scan(&vid); err != nil {
		return qid, err
	}
	if _, err := tx.Exec(ctx, `UPDATE questions SET current_version_id = $2 WHERE id = $1`, qid, vid); err != nil {
		return qid, err
	}
	return qid, linkMedia(ctx, tx, vid, in.Content)
}

func (h Handler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	var in QuestionInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	if f := in.Validate(r.Context(), h.DB); len(f) > 0 {
		httpx.Invalid(w, f)
		return
	}
	u := auth.MustUser(r.Context())
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())
	id, err := CreateQuestion(r.Context(), tx, in, u.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpx.Invalid(w, map[string]string{"code": "Kode soal sudah dipakai."})
			return
		}
		httpx.Invalid(w, map[string]string{"content": err.Error()})
		return
	}
	audit.Log(r.Context(), tx, u.ID, "question.create", "question", id.String(), nil)
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	h.getQuestion(w, r, id, http.StatusCreated)
}

type Version struct {
	ID          uuid.UUID       `json:"id"`
	Version     int             `json:"version"`
	Content     json.RawMessage `json:"content"`
	Answer      json.RawMessage `json:"answer"`
	Explanation string          `json:"explanation"`
	Hint        string          `json:"hint"`
	CreatedBy   *string         `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
}

func (h Handler) getQuestion(w http.ResponseWriter, r *http.Request, id uuid.UUID, status int) {
	ctx := r.Context()
	var q struct {
		ID                 uuid.UUID  `json:"id"`
		Code               string     `json:"code"`
		SkillID            int        `json:"skill_id"`
		Type               string     `json:"type"`
		Difficulty         int        `json:"difficulty"`
		Status             string     `json:"status"`
		LearningObjective  string     `json:"learning_objective"`
		Source             string     `json:"source"`
		CurrentVersionID   *uuid.UUID `json:"current_version_id"`
		PublishedVersionID *uuid.UUID `json:"published_version_id"`
		Reviewer           *string    `json:"reviewer"`
		PublishedAt        *time.Time `json:"published_at"`
		UpdatedAt          time.Time  `json:"updated_at"`
	}
	err := h.DB.QueryRow(ctx, `SELECT q.id, q.code, q.skill_id, q.type, q.difficulty, q.status, q.learning_objective, q.source,
		q.current_version_id, q.published_version_id, u.display_name, q.published_at, q.updated_at
		FROM questions q LEFT JOIN users u ON u.id = q.reviewer_id WHERE q.id = $1`, id).
		Scan(&q.ID, &q.Code, &q.SkillID, &q.Type, &q.Difficulty, &q.Status, &q.LearningObjective, &q.Source,
			&q.CurrentVersionID, &q.PublishedVersionID, &q.Reviewer, &q.PublishedAt, &q.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	rows, err := h.DB.Query(ctx, `SELECT v.id, v.version, v.content, v.answer, v.explanation, v.hint, u.display_name, v.created_at
		FROM question_versions v LEFT JOIN users u ON u.id = v.created_by WHERE v.question_id = $1 ORDER BY v.version DESC`, id)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	versions, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Version])
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	type review struct {
		Reviewer  string          `json:"reviewer"`
		From      string          `json:"from"`
		To        string          `json:"to"`
		Version   int             `json:"version"`
		Checklist json.RawMessage `json:"checklist"`
		Notes     string          `json:"notes"`
		CreatedAt time.Time       `json:"created_at"`
	}
	rrows, err := h.DB.Query(ctx, `SELECT u.display_name, r.from_status, r.to_status, v.version, r.checklist, r.notes, r.created_at
		FROM question_reviews r JOIN users u ON u.id = r.reviewer_id JOIN question_versions v ON v.id = r.question_version_id
		WHERE r.question_id = $1 ORDER BY r.created_at DESC`, id)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	reviews, err := pgx.CollectRows(rrows, pgx.RowToStructByPos[review])
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, status, map[string]any{"question": q, "versions": versions, "reviews": reviews})
}

func (h Handler) GetQuestion(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "questionID")
	if !ok {
		return
	}
	h.getQuestion(w, r, id, http.StatusOK)
}

// UpdateQuestion appends a new version. A version already served to children
// keeps being served until the new one is reviewed and published.
func (h Handler) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "questionID")
	if !ok {
		return
	}
	var in QuestionInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	if f := in.Validate(r.Context(), h.DB); len(f) > 0 {
		httpx.Invalid(w, f)
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
	var status string
	var next int
	err = tx.QueryRow(ctx, `SELECT q.status, (SELECT max(version) + 1 FROM question_versions WHERE question_id = q.id)
		FROM questions q WHERE q.id = $1 FOR UPDATE`, id).Scan(&status, &next)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if status == "retired" {
		httpx.Error(w, http.StatusConflict, "retired", "Aktifkan kembali soal (ke Draft) sebelum mengedit.")
		return
	}
	var vid uuid.UUID
	if err := tx.QueryRow(ctx, `INSERT INTO question_versions (question_id, version, content, answer, explanation, hint, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, id, next, in.Content, in.Answer, in.Explanation, in.Hint, u.ID).Scan(&vid); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if err := linkMedia(ctx, tx, vid, in.Content); err != nil {
		httpx.Invalid(w, map[string]string{"content": err.Error()})
		return
	}
	// Any content change restarts the workflow for the new version.
	if _, err := tx.Exec(ctx, `UPDATE questions SET current_version_id = $2, skill_id = $3, type = $4, difficulty = $5,
		learning_objective = $6, source = $7, status = 'draft', updated_at = now() WHERE id = $1`,
		id, vid, in.SkillID, in.Type, in.Difficulty, in.LearningObjective, in.Source); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	audit.Log(ctx, tx, u.ID, "question.update", "question", id.String(), map[string]any{"version": next})
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	h.getQuestion(w, r, id, http.StatusOK)
}

func (h Handler) TransitionQuestion(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "questionID")
	if !ok {
		return
	}
	var in struct {
		To        string          `json:"to"`
		Checklist map[string]bool `json:"checklist"`
		Notes     string          `json:"notes"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	ctx := r.Context()
	u := auth.MustUser(ctx)
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(ctx)
	var status, typ string
	var vid uuid.UUID
	var author *uuid.UUID
	var content, answer json.RawMessage
	err = tx.QueryRow(ctx, `SELECT q.status, q.type, q.current_version_id, v.created_by, v.content, v.answer
		FROM questions q JOIN question_versions v ON v.id = q.current_version_id WHERE q.id = $1 FOR UPDATE OF q`, id).
		Scan(&status, &typ, &vid, &author, &content, &answer)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var mediaPending int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM question_media m JOIN assets a ON a.id = m.asset_id
		WHERE m.question_version_id = $1 AND a.status <> 'published'`, vid).Scan(&mediaPending); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	tc := TransitionCtx{From: status, To: in.To, Actor: u, Checklist: in.Checklist, MediaReady: mediaPending == 0}
	if author != nil {
		tc.AuthorOfHead = author.String()
	}
	if err := CheckQuestionTransition(tc); err != nil {
		httpx.Error(w, http.StatusConflict, "gate_failed", err.Error())
		return
	}
	if in.To == "review" || in.To == "approved" || in.To == "published" {
		if err := questions.Validate(typ, content, answer); err != nil {
			httpx.Error(w, http.StatusConflict, "invalid_content", err.Error())
			return
		}
	}
	switch in.To {
	case "published":
		_, err = tx.Exec(ctx, `UPDATE questions SET status = 'published', published_version_id = current_version_id,
			published_at = now(), updated_at = now() WHERE id = $1`, id)
	case "retired":
		_, err = tx.Exec(ctx, `UPDATE questions SET status = 'retired', published_version_id = NULL, updated_at = now() WHERE id = $1`, id)
	case "approved":
		_, err = tx.Exec(ctx, `UPDATE questions SET status = 'approved', reviewer_id = $2, updated_at = now() WHERE id = $1`, id, u.ID)
	default:
		_, err = tx.Exec(ctx, `UPDATE questions SET status = $2, updated_at = now() WHERE id = $1`, id, in.To)
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	checklist := in.Checklist
	if checklist == nil {
		checklist = map[string]bool{}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO question_reviews (question_id, question_version_id, reviewer_id, from_status, to_status, checklist, notes)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, id, vid, u.ID, status, in.To, checklist, in.Notes); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	audit.Log(ctx, tx, u.ID, "question.transition", "question", id.String(), map[string]any{"from": status, "to": in.To})
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	h.getQuestion(w, r, id, http.StatusOK)
}

// Preview renders the child-facing content of a version and optionally grades a
// test response, so editors can try a question exactly as a child would.
func (h Handler) Preview(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "questionID")
	if !ok {
		return
	}
	var in struct {
		VersionID *uuid.UUID      `json:"version_id"`
		Response  json.RawMessage `json:"response"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	var typ, explanation, hint string
	var content, answer json.RawMessage
	err := h.DB.QueryRow(r.Context(), `SELECT q.type, v.content, v.answer, v.explanation, v.hint FROM questions q
		JOIN question_versions v ON v.question_id = q.id AND v.id = COALESCE($2, q.current_version_id) WHERE q.id = $1`, id, in.VersionID).
		Scan(&typ, &content, &answer, &explanation, &hint)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	pub, err := questions.PublicContent(typ, content, "preview")
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	out := map[string]any{"type": typ, "content": pub, "hint": hint, "explanation": explanation}
	if len(in.Response) > 0 {
		correct, _ := questions.Grade(typ, content, answer, in.Response)
		out["correct"] = correct
	}
	httpx.JSON(w, http.StatusOK, out)
}
