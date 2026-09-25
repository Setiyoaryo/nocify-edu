package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nocify/eduplay/api/internal/audit"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/httpx"
)

// Curriculum returns the whole hierarchy as flat lists; the CMS builds the tree.
func (h Handler) Curriculum(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	out := map[string]any{}
	queries := map[string]string{
		"versions": `SELECT json_agg(t ORDER BY t.id) FROM (SELECT id, code, name, is_active FROM curriculum_versions) t`,
		"levels":   `SELECT json_agg(t ORDER BY t.sort) FROM (SELECT id, code, name, sort FROM education_levels) t`,
		"grades":   `SELECT json_agg(t ORDER BY t.sort) FROM (SELECT id, education_level_id, code, name, sort, session_size, ux_band FROM grades) t`,
		"subjects": `SELECT json_agg(t ORDER BY t.sort) FROM (SELECT id, code, name, motif, sort FROM subjects) t`,
		"chapters": `SELECT json_agg(t ORDER BY t.sort) FROM (SELECT id, curriculum_version_id, grade_id, subject_id, code, name, sort FROM chapters) t`,
		"skills": `SELECT json_agg(t ORDER BY t.sort) FROM (SELECT k.id, k.chapter_id, k.code, k.name, k.description, k.sort,
			(SELECT count(*) FROM questions q WHERE q.skill_id = k.id) AS questions,
			(SELECT count(*) FROM questions q WHERE q.skill_id = k.id AND q.published_version_id IS NOT NULL) AS live FROM skills k) t`,
	}
	for key, sql := range queries {
		var raw json.RawMessage
		if err := h.DB.QueryRow(ctx, sql).Scan(&raw); err != nil {
			httpx.Internal(w, r, err)
			return
		}
		if raw == nil {
			raw = json.RawMessage("[]")
		}
		out[key] = raw
	}
	httpx.JSON(w, http.StatusOK, out)
}

type curriculumInput struct {
	Code        *string `json:"code"`
	Name        *string `json:"name"`
	Motif       *string `json:"motif"`
	Description *string `json:"description"`
	Sort        *int    `json:"sort"`
	GradeID     *int    `json:"grade_id"`
	SubjectID   *int    `json:"subject_id"`
	ChapterID   *int    `json:"chapter_id"`
	SessionSize *int    `json:"session_size"`
	UXBand      *string `json:"ux_band"`
}

func trimmed(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	return &s
}

func (h Handler) curriculumWrite(w http.ResponseWriter, r *http.Request, entity, action string, exec func(in curriculumInput) (pgx.Row, error)) {
	var in curriculumInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.Code, in.Name = trimmed(in.Code), trimmed(in.Name)
	if (in.Code != nil && *in.Code == "") || (in.Name != nil && *in.Name == "") {
		httpx.Invalid(w, map[string]string{"name": "Kode dan nama tidak boleh kosong."})
		return
	}
	row, err := exec(in)
	if err != nil {
		httpx.Invalid(w, map[string]string{"input": err.Error()})
		return
	}
	var id int
	if err := row.Scan(&id); err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			httpx.NotFound(w)
		case errors.As(err, &pgErr) && pgErr.Code == "23505":
			httpx.Invalid(w, map[string]string{"code": "Kode sudah dipakai."})
		case errors.As(err, &pgErr) && (pgErr.Code == "23502" || pgErr.Code == "23503" || pgErr.Code == "23514"):
			httpx.Invalid(w, map[string]string{"input": "Data wajib belum lengkap atau referensi tidak valid."})
		default:
			httpx.Internal(w, r, err)
		}
		return
	}
	audit.Log(r.Context(), h.DB, auth.MustUser(r.Context()).ID, entity+"."+action, entity, fmt.Sprint(id), nil)
	httpx.JSON(w, http.StatusOK, map[string]int{"id": id})
}

func (h Handler) CreateSubject(w http.ResponseWriter, r *http.Request) {
	h.curriculumWrite(w, r, "subject", "create", func(in curriculumInput) (pgx.Row, error) {
		return h.DB.QueryRow(r.Context(), `INSERT INTO subjects (code, name, motif, sort) VALUES ($1, $2, COALESCE($3, ''), COALESCE($4, 0)) RETURNING id`,
			in.Code, in.Name, in.Motif, in.Sort), nil
	})
}

func (h Handler) UpdateSubject(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.IntParam(w, r, "id")
	if !ok {
		return
	}
	h.curriculumWrite(w, r, "subject", "update", func(in curriculumInput) (pgx.Row, error) {
		return h.DB.QueryRow(r.Context(), `UPDATE subjects SET name = COALESCE($2, name), motif = COALESCE($3, motif), sort = COALESCE($4, sort)
			WHERE id = $1 RETURNING id`, id, in.Name, in.Motif, in.Sort), nil
	})
}

func (h Handler) UpdateGrade(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.IntParam(w, r, "id")
	if !ok {
		return
	}
	h.curriculumWrite(w, r, "grade", "update", func(in curriculumInput) (pgx.Row, error) {
		return h.DB.QueryRow(r.Context(), `UPDATE grades SET name = COALESCE($2, name), session_size = COALESCE($3, session_size), ux_band = COALESCE($4, ux_band)
			WHERE id = $1 RETURNING id`, id, in.Name, in.SessionSize, in.UXBand), nil
	})
}

func (h Handler) CreateChapter(w http.ResponseWriter, r *http.Request) {
	h.curriculumWrite(w, r, "chapter", "create", func(in curriculumInput) (pgx.Row, error) {
		return h.DB.QueryRow(r.Context(), `INSERT INTO chapters (curriculum_version_id, grade_id, subject_id, code, name, sort)
			VALUES ((SELECT id FROM curriculum_versions WHERE is_active LIMIT 1), $1, $2, $3, $4, COALESCE($5, 0)) RETURNING id`,
			in.GradeID, in.SubjectID, in.Code, in.Name, in.Sort), nil
	})
}

func (h Handler) UpdateChapter(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.IntParam(w, r, "id")
	if !ok {
		return
	}
	h.curriculumWrite(w, r, "chapter", "update", func(in curriculumInput) (pgx.Row, error) {
		return h.DB.QueryRow(r.Context(), `UPDATE chapters SET name = COALESCE($2, name), sort = COALESCE($3, sort) WHERE id = $1 RETURNING id`,
			id, in.Name, in.Sort), nil
	})
}

func (h Handler) CreateSkill(w http.ResponseWriter, r *http.Request) {
	h.curriculumWrite(w, r, "skill", "create", func(in curriculumInput) (pgx.Row, error) {
		return h.DB.QueryRow(r.Context(), `INSERT INTO skills (chapter_id, code, name, description, sort)
			VALUES ($1, $2, $3, COALESCE($4, ''), COALESCE($5, 0)) RETURNING id`, in.ChapterID, in.Code, in.Name, in.Description, in.Sort), nil
	})
}

func (h Handler) UpdateSkill(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.IntParam(w, r, "id")
	if !ok {
		return
	}
	h.curriculumWrite(w, r, "skill", "update", func(in curriculumInput) (pgx.Row, error) {
		return h.DB.QueryRow(r.Context(), `UPDATE skills SET name = COALESCE($2, name), description = COALESCE($3, description), sort = COALESCE($4, sort)
			WHERE id = $1 RETURNING id`, id, in.Name, in.Description, in.Sort), nil
	})
}

func (h Handler) AuditLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, err := h.DB.Query(r.Context(), `SELECT l.id, COALESCE(u.display_name, 'system'), l.action, l.entity_type, l.entity_id, l.data, l.created_at
		FROM audit_logs l LEFT JOIN users u ON u.id = l.actor_id
		WHERE ($1 = '' OR l.entity_type = $1) AND ($2 = '' OR l.entity_id = $2)
		ORDER BY l.id DESC LIMIT 200`, q.Get("entity_type"), q.Get("entity_id"))
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	type entry struct {
		ID         int64           `json:"id"`
		Actor      string          `json:"actor"`
		Action     string          `json:"action"`
		EntityType string          `json:"entity_type"`
		EntityID   string          `json:"entity_id"`
		Data       json.RawMessage `json:"data"`
		CreatedAt  time.Time       `json:"created_at"`
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[entry])
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"logs": out})
}

// Stats is the admin overview, including the core business KPIs of §51.
func (h Handler) Stats(w http.ResponseWriter, r *http.Request) {
	var raw json.RawMessage
	err := h.DB.QueryRow(r.Context(), `SELECT json_build_object(
		'questions', (SELECT COALESCE(json_object_agg(status, n), '{}') FROM (SELECT status, count(*) n FROM questions GROUP BY status) t),
		'assets', (SELECT COALESCE(json_object_agg(status, n), '{}') FROM (SELECT status, count(*) n FROM assets GROUP BY status) t),
		'parents', (SELECT count(*) FROM users WHERE role = 'parent'),
		'children', (SELECT count(*) FROM children),
		'active_subscribers', (SELECT count(*) FROM subscriptions WHERE status IN ('active','trial','cancelled','past_due') AND current_period_end > now()),
		'activated_parents', (SELECT count(DISTINCT c.parent_id) FROM children c JOIN assessments a ON a.child_id = c.id AND a.status = 'completed'),
		'challenges_started_7d', (SELECT count(*) FROM assessments WHERE started_at > now() - interval '7 days'),
		'challenges_completed_7d', (SELECT count(*) FROM assessments WHERE status = 'completed' AND started_at > now() - interval '7 days'),
		'payment_failures_7d', (SELECT count(*) FROM payment_checkouts WHERE status = 'failed' AND created_at > now() - interval '7 days'),
		'webhook_errors', (SELECT count(*) FROM payment_events WHERE error IS NOT NULL AND processed_at IS NULL)
	)`).Scan(&raw)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, raw)
}
