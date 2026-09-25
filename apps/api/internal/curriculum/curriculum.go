// Package curriculum serves the Curriculum → Level → Grade → Subject → Chapter → Skill
// hierarchy (§14). Nothing about grades or subjects is hardcoded in Go.
package curriculum

import (
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/httpx"
)

type Grade struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Level       string `json:"level"`
	LevelName   string `json:"level_name"`
	SessionSize int    `json:"session_size"`
	UXBand      string `json:"ux_band"`
}

type Subject struct {
	ID        int    `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Motif     string `json:"motif"`
	Questions int    `json:"questions,omitempty"`
}

type Skill struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ChapterID   int    `json:"chapter_id"`
	ChapterName string `json:"chapter_name"`
	SubjectID   int    `json:"subject_id"`
}

type Handler struct{ DB *pgxpool.Pool }

func (h Handler) Grades(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT g.id, g.code, g.name, l.code, l.name, g.session_size, g.ux_band
		FROM grades g JOIN education_levels l ON l.id = g.education_level_id
		ORDER BY l.sort, g.sort`)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Grade, error) {
		var g Grade
		err := row.Scan(&g.ID, &g.Code, &g.Name, &g.Level, &g.LevelName, &g.SessionSize, &g.UXBand)
		return g, err
	})
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"grades": out})
}

// Subjects lists subjects; with ?grade_id it lists only subjects that have
// published questions for that grade in the active curriculum.
func (h Handler) Subjects(w http.ResponseWriter, r *http.Request) {
	gradeID := httpx.QueryInt(r, "grade_id", 0)
	var rows pgx.Rows
	var err error
	if gradeID == 0 {
		rows, err = h.DB.Query(r.Context(), `SELECT id, code, name, motif, 0 FROM subjects ORDER BY sort, name`)
	} else {
		rows, err = h.DB.Query(r.Context(), SubjectsForGradeSQL, gradeID)
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	out, err := CollectSubjects(rows)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"subjects": out})
}

const SubjectsForGradeSQL = `
	SELECT s.id, s.code, s.name, s.motif, count(q.id)::int
	FROM subjects s
	JOIN chapters c ON c.subject_id = s.id AND c.grade_id = $1
	JOIN curriculum_versions cv ON cv.id = c.curriculum_version_id AND cv.is_active
	JOIN skills k ON k.chapter_id = c.id
	JOIN questions q ON q.skill_id = k.id AND q.published_version_id IS NOT NULL
	GROUP BY s.id ORDER BY s.sort, s.name`

func CollectSubjects(rows pgx.Rows) ([]Subject, error) {
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Subject, error) {
		var s Subject
		err := row.Scan(&s.ID, &s.Code, &s.Name, &s.Motif, &s.Questions)
		return s, err
	})
}

func (h Handler) Skills(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT k.id, k.code, k.name, k.description, c.id, c.name, c.subject_id
		FROM skills k JOIN chapters c ON c.id = k.chapter_id
		JOIN curriculum_versions cv ON cv.id = c.curriculum_version_id AND cv.is_active
		WHERE ($1 = 0 OR c.grade_id = $1) AND ($2 = 0 OR c.subject_id = $2)
		ORDER BY c.sort, k.sort`, httpx.QueryInt(r, "grade_id", 0), httpx.QueryInt(r, "subject_id", 0))
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Skill, error) {
		var k Skill
		err := row.Scan(&k.ID, &k.Code, &k.Name, &k.Description, &k.ChapterID, &k.ChapterName, &k.SubjectID)
		return k, err
	})
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"skills": out})
}
