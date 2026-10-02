package candidates

// Content Studio: browse the content pipeline per lesson (grade → subject →
// chapter → skill). A lesson is a skill — the smallest curriculum unit — and
// shows its material candidates, the question candidates derived from each
// material (lineage = same chunk), orphan questions, and the canonical
// questions already published for that skill. Read-only; decisions reuse
// DecideCandidate.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nocify/eduplay/api/internal/httpx"
)

// --- Browse tree -----------------------------------------------------------

type StudioSkill struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Materials   int    `json:"materials"`
	Questions   int    `json:"questions"`
	NeedsReview int    `json:"needs_review"`
	Published   int    `json:"published"`
}

type StudioChapter struct {
	ID     int           `json:"id"`
	Code   string        `json:"code"`
	Name   string        `json:"name"`
	Sort   int           `json:"sort"`
	Skills []StudioSkill `json:"skills"`
}

type StudioSubject struct {
	ID       int             `json:"id"`
	Code     string          `json:"code"`
	Name     string          `json:"name"`
	Sort     int             `json:"sort"`
	Chapters []StudioChapter `json:"chapters"`
}

type StudioGrade struct {
	ID       int             `json:"id"`
	Code     string          `json:"code"`
	Name     string          `json:"name"`
	Sort     int             `json:"sort"`
	Subjects []StudioSubject `json:"subjects"`
}

type StudioUnmapped struct {
	Materials int `json:"materials"`
	Questions int `json:"questions"`
}

// treeSkillRow is scanned positionally; its field order must match
// treeSkillSQL's SELECT exactly.
type treeSkillRow struct {
	GradeID     int
	GradeCode   string
	GradeName   string
	GradeSort   int
	SubjectID   int
	SubjectCode string
	SubjectName string
	SubjectSort int
	ChapterID   int
	ChapterCode string
	ChapterName string
	ChapterSort int
	SkillID     int
	SkillCode   string
	SkillName   string
	SkillSort   int
	Materials   int
	Questions   int
	NeedsReview int
	Published   int
}

// treeSkillSQL returns one row per skill with per-skill pipeline counts.
// needs_review counts candidates with no decision row yet (matching the review
// queue); published counts canonical questions already live for the skill.
const treeSkillSQL = `
	WITH cand AS (
	    SELECT m.skill_id, 'material' AS kind, p.status AS pub_status
	    FROM content_staging.material_candidates m
	    LEFT JOIN candidate_publications p ON p.kind = 'material' AND p.candidate_id = m.id
	    UNION ALL
	    SELECT q.skill_id, 'question', p.status
	    FROM content_staging.question_candidates q
	    LEFT JOIN candidate_publications p ON p.kind = 'question' AND p.candidate_id = q.id
	),
	agg AS (
	    SELECT skill_id,
	           count(*) FILTER (WHERE kind = 'material')::int AS materials,
	           count(*) FILTER (WHERE kind = 'question')::int AS questions,
	           count(*) FILTER (WHERE pub_status IS NULL)::int AS needs_review
	    FROM cand WHERE skill_id IS NOT NULL GROUP BY skill_id
	)
	SELECT g.id, g.code, g.name, g.sort,
	       sub.id, sub.code, sub.name, sub.sort,
	       ch.id, ch.code, ch.name, ch.sort,
	       k.id, k.code, k.name, k.sort,
	       COALESCE(a.materials, 0), COALESCE(a.questions, 0), COALESCE(a.needs_review, 0),
	       (SELECT count(*)::int FROM questions qq WHERE qq.skill_id = k.id AND qq.status = 'published')
	FROM skills k
	JOIN chapters ch ON ch.id = k.chapter_id
	JOIN grades g ON g.id = ch.grade_id
	JOIN subjects sub ON sub.id = ch.subject_id
	LEFT JOIN agg a ON a.skill_id = k.id
	ORDER BY g.sort, g.id, sub.sort, sub.id, ch.sort, ch.id, k.sort, k.id`

func (h Handler) ContentStudioTree(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !h.stagingReady(ctx) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"available": false, "unmapped": StudioUnmapped{}, "grades": []StudioGrade{},
		})
		return
	}
	rows, err := h.DB.Query(ctx, treeSkillSQL)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	flat, err := pgx.CollectRows(rows, pgx.RowToStructByPos[treeSkillRow])
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var unmapped StudioUnmapped
	if err := h.DB.QueryRow(ctx, `
		SELECT (SELECT count(*)::int FROM content_staging.material_candidates WHERE skill_id IS NULL),
		       (SELECT count(*)::int FROM content_staging.question_candidates WHERE skill_id IS NULL)`).
		Scan(&unmapped.Materials, &unmapped.Questions); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"available": true, "unmapped": unmapped, "grades": nestTree(flat),
	})
}

// nestTree turns the ordered flat rows into grade → subject → chapter → skill.
func nestTree(rows []treeSkillRow) []StudioGrade {
	grades := []StudioGrade{}
	var g *StudioGrade
	var s *StudioSubject
	var c *StudioChapter
	for _, row := range rows {
		if g == nil || g.ID != row.GradeID {
			grades = append(grades, StudioGrade{
				ID: row.GradeID, Code: row.GradeCode, Name: row.GradeName, Sort: row.GradeSort,
				Subjects: []StudioSubject{},
			})
			g = &grades[len(grades)-1]
			s, c = nil, nil
		}
		if s == nil || s.ID != row.SubjectID {
			g.Subjects = append(g.Subjects, StudioSubject{
				ID: row.SubjectID, Code: row.SubjectCode, Name: row.SubjectName, Sort: row.SubjectSort,
				Chapters: []StudioChapter{},
			})
			s = &g.Subjects[len(g.Subjects)-1]
			c = nil
		}
		if c == nil || c.ID != row.ChapterID {
			s.Chapters = append(s.Chapters, StudioChapter{
				ID: row.ChapterID, Code: row.ChapterCode, Name: row.ChapterName, Sort: row.ChapterSort,
				Skills: []StudioSkill{},
			})
			c = &s.Chapters[len(s.Chapters)-1]
		}
		c.Skills = append(c.Skills, StudioSkill{
			ID: row.SkillID, Code: row.SkillCode, Name: row.SkillName,
			Materials: row.Materials, Questions: row.Questions,
			NeedsReview: row.NeedsReview, Published: row.Published,
		})
	}
	return grades
}

// --- Lesson detail ---------------------------------------------------------

type StudioLesson struct {
	ID          int    `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ChapterID   int    `json:"chapter_id"`
	ChapterName string `json:"chapter_name"`
	GradeID     int    `json:"grade_id"`
	GradeCode   string `json:"grade_code"`
	GradeName   string `json:"grade_name"`
	SubjectID   int    `json:"subject_id"`
	SubjectCode string `json:"subject_code"`
	SubjectName string `json:"subject_name"`
}

type StudioMaterialGroup struct {
	Material  CandidateDetail   `json:"material"`
	Questions []CandidateDetail `json:"questions"`
}

type CanonicalQuestion struct {
	ID          uuid.UUID       `json:"id"`
	Code        string          `json:"code"`
	Type        string          `json:"type"`
	Difficulty  *int            `json:"difficulty"`
	Content     json.RawMessage `json:"content"`
	Answer      json.RawMessage `json:"answer"`
	Explanation string          `json:"explanation"`
	Hint        string          `json:"hint"`
	PublishedAt *time.Time      `json:"published_at"`
	Reviewer    string          `json:"reviewer"`
	HasVisual   bool            `json:"has_visual"`
	HasAudio    bool            `json:"has_audio"`
}

// canonicalRow is scanned positionally into CanonicalQuestion.
type canonicalRow struct {
	ID          uuid.UUID
	Code        string
	Type        string
	Difficulty  *int
	Content     []byte
	Answer      []byte
	Explanation string
	Hint        string
	PublishedAt *time.Time
	Reviewer    string
}

const questionDetailBySkillSQL = questionDetailSelect + `
	WHERE q.skill_id = $1 ORDER BY q.chunk_id, q.id LIMIT $2`

const materialDetailBySkillSQL = materialDetailSelect + `
	WHERE m.skill_id = $1 ORDER BY m.chunk_id, m.id LIMIT $2`

const publishedBySkillSQL = `
	SELECT q.id, q.code, q.type, q.difficulty, v.content, v.answer, v.explanation, v.hint,
	       q.published_at, COALESCE(u.display_name, '')
	FROM questions q
	JOIN question_versions v ON v.id = q.published_version_id
	LEFT JOIN users u ON u.id = q.reviewer_id
	WHERE q.skill_id = $1 AND q.status = 'published'
	ORDER BY q.code LIMIT $2`

func (h Handler) ContentStudioLesson(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	skillID, ok := httpx.IntParam(w, r, "skillID")
	if !ok {
		return
	}
	limit := httpx.QueryInt(r, "limit", 200)
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	var skill StudioLesson
	err := h.DB.QueryRow(ctx, `
		SELECT k.id, k.code, k.name, k.description,
		       ch.id, ch.name, g.id, g.code, g.name, sub.id, sub.code, sub.name
		FROM skills k
		JOIN chapters ch ON ch.id = k.chapter_id
		JOIN grades g ON g.id = ch.grade_id
		JOIN subjects sub ON sub.id = ch.subject_id
		WHERE k.id = $1`, skillID).
		Scan(&skill.ID, &skill.Code, &skill.Name, &skill.Description,
			&skill.ChapterID, &skill.ChapterName,
			&skill.GradeID, &skill.GradeCode, &skill.GradeName,
			&skill.SubjectID, &skill.SubjectCode, &skill.SubjectName)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}

	// Canonical published questions live in `public`, so they render even when
	// the pipeline has not been run in this environment.
	published, err := h.publishedForSkill(ctx, skillID, limit)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}

	if !h.stagingReady(ctx) {
		httpx.JSON(w, http.StatusOK, map[string]any{
			"available": false, "skill": skill,
			"materials": []StudioMaterialGroup{}, "orphan_questions": []CandidateDetail{},
			"published": published,
		})
		return
	}

	materials, err := h.candidatesBySkill(ctx, KindMaterial, materialDetailBySkillSQL, skillID, limit)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	questions, err := h.candidatesBySkill(ctx, KindQuestion, questionDetailBySkillSQL, skillID, limit)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}

	groups, orphans := groupByChunk(materials, questions)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"available": true, "skill": skill,
		"materials": groups, "orphan_questions": orphans, "published": published,
	})
}

func (h Handler) candidatesBySkill(ctx context.Context, kind, sql string, skillID, limit int) ([]CandidateDetail, error) {
	rows, err := h.DB.Query(ctx, sql, skillID, limit)
	if err != nil {
		return nil, err
	}
	flat, err := pgx.CollectRows(rows, pgx.RowToStructByPos[detailRow])
	if err != nil {
		return nil, err
	}
	out := make([]CandidateDetail, 0, len(flat))
	for _, row := range flat {
		out = append(out, row.toDetail(kind))
	}
	return out, nil
}

func (h Handler) publishedForSkill(ctx context.Context, skillID, limit int) ([]CanonicalQuestion, error) {
	rows, err := h.DB.Query(ctx, publishedBySkillSQL, skillID, limit)
	if err != nil {
		return nil, err
	}
	flat, err := pgx.CollectRows(rows, pgx.RowToStructByPos[canonicalRow])
	if err != nil {
		return nil, err
	}
	out := make([]CanonicalQuestion, 0, len(flat))
	for _, row := range flat {
		q := CanonicalQuestion{
			ID: row.ID, Code: row.Code, Type: row.Type, Difficulty: row.Difficulty,
			Explanation: row.Explanation, Hint: row.Hint, PublishedAt: row.PublishedAt,
			Reviewer:  row.Reviewer,
			HasVisual: payloadHasVisual(row.Content), HasAudio: payloadHasAudio(row.Content),
		}
		if len(row.Content) > 0 {
			q.Content = json.RawMessage(row.Content)
		}
		if len(row.Answer) > 0 {
			q.Answer = json.RawMessage(row.Answer)
		}
		out = append(out, q)
	}
	return out, nil
}

// groupByChunk nests each question under the material that shares its chunk
// (the pipeline generates both from the same chunk). Questions whose chunk has
// no material are returned as orphans.
func groupByChunk(materials, questions []CandidateDetail) ([]StudioMaterialGroup, []CandidateDetail) {
	byChunk := make(map[int64][]CandidateDetail, len(materials))
	for _, q := range questions {
		byChunk[q.Lineage.ChunkID] = append(byChunk[q.Lineage.ChunkID], q)
	}
	groups := make([]StudioMaterialGroup, 0, len(materials))
	used := make(map[int64]bool, len(materials))
	for _, m := range materials {
		chunk := m.Lineage.ChunkID
		qs := byChunk[chunk]
		if qs == nil {
			qs = []CandidateDetail{}
		}
		used[chunk] = true
		groups = append(groups, StudioMaterialGroup{Material: m, Questions: qs})
	}
	orphans := []CandidateDetail{}
	for _, q := range questions {
		if !used[q.Lineage.ChunkID] {
			orphans = append(orphans, q)
		}
	}
	return groups, orphans
}
