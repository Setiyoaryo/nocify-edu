package candidates

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Content Studio tests. DB-gated via setupPublishEnv (TEST_DATABASE_URL).

const figuresPayload = `{"code":"Q-FIG","skill":"x","type":"image_choice","difficulty":2,
	"learning_objective":"Anak dapat membandingkan.","content":{"prompt":"Mana lebih banyak?",
	"audio_text":"Mana lebih banyak?","options":[
	{"id":"a","label":"3","figures":[{"icon":"orange","count":3}]},
	{"id":"b","label":"5","figures":[{"icon":"orange","count":5}]}]},
	"answer":{"option_id":"b"},"explanation":"5 lebih banyak."}`

func studioRouter(h Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/content-studio/tree", h.ContentStudioTree)
	r.Get("/content-studio/lessons/{skillID}", h.ContentStudioLesson)
	return r
}

func getJSON(t *testing.T, h http.Handler, path string, out any) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if out != nil && rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("decode %s: %v (body=%s)", path, err, rec.Body.String())
		}
	}
	return rec.Code
}

func newChunk(t *testing.T, pool *pgxpool.Pool, f stagingFixtures, hash string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO content_staging.document_chunks
		(document_id, heading, sequence, text, hash)
		VALUES ($1, 'Bab', (SELECT COALESCE(max(sequence), -1) + 1 FROM content_staging.document_chunks WHERE document_id = $1),
		        'isi chunk', $2) RETURNING id`, f.documentID, hash).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func insertMaterial(t *testing.T, pool *pgxpool.Pool, f stagingFixtures, skillID int, chunkID int64, title string, visual bool) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO content_staging.material_candidates
		(document_id, chunk_id, source_id, skill_id, phase_code, grade_code, subject_code,
		 title, summary, learning_objective, difficulty_hint, language, visual_required, audio_required,
		 content_hash, confidence, qa_issues, status)
		VALUES ($1,$2,$3,$4,'','','',$5,'ringkasan','Tujuan',2,'id',$6,false,$7,0.8,'[]'::jsonb,'needs_review')
		RETURNING id`, f.documentID, chunkID, f.sourceID, skillID, title, visual, title+"-hash").Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func insertQuestion(t *testing.T, pool *pgxpool.Pool, f stagingFixtures, skillID int, chunkID int64, template, code, payload string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(), `INSERT INTO content_staging.question_candidates
		(document_id, chunk_id, source_id, skill_id, phase_code, grade_code, subject_code,
		 template, type, difficulty, learning_objective, prompt, payload, content_hash, confidence, qa_issues, status)
		VALUES ($1,$2,$3,$4,'','','',$5,'short_answer',3,'Tujuan','Lengkapi: _____',$6::jsonb,$7,0.9,'[]'::jsonb,'needs_review')
		RETURNING id`, f.documentID, chunkID, f.sourceID, skillID, template, payload, code+"-hash").Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// skillWithPublished returns a seeded skill that has at least one published
// canonical question, so lesson/degraded tests always have canonical content.
func skillWithPublished(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var id int
	if err := pool.QueryRow(context.Background(),
		`SELECT skill_id FROM questions WHERE status='published' GROUP BY skill_id ORDER BY count(*) DESC, skill_id LIMIT 1`).
		Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func publishedCount(t *testing.T, pool *pgxpool.Pool, skillID int) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*)::int FROM questions WHERE skill_id=$1 AND status='published'`, skillID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func findSkill(grades []StudioGrade, id int) (StudioSkill, bool) {
	for _, g := range grades {
		for _, s := range g.Subjects {
			for _, c := range s.Chapters {
				for _, k := range c.Skills {
					if k.ID == id {
						return k, true
					}
				}
			}
		}
	}
	return StudioSkill{}, false
}

func TestContentStudioTree(t *testing.T) {
	pool, f := setupPublishEnv(t)
	skillID := skillWithPublished(t, pool)

	materialID := insertMaterial(t, pool, f, skillID, newChunk(t, pool, f, "tree-mat"), "Materi A", true)
	insertQuestion(t, pool, f, skillID, newChunk(t, pool, f, "tree-q1"), "cloze", "Q-A1", payloadJSON(f.skillCode, "Q-A1"))
	insertQuestion(t, pool, f, skillID, newChunk(t, pool, f, "tree-q2"), "cloze", "Q-A2", payloadJSON(f.skillCode, "Q-A2"))
	// One unmapped candidate (skill_id NULL) must land in the unmapped bucket.
	unmappedID := insertQuestion(t, pool, f, 0, newChunk(t, pool, f, "tree-unmapped"), "cloze", "Q-U", payloadJSON(f.skillCode, "Q-U"))
	if _, err := pool.Exec(context.Background(), `UPDATE content_staging.question_candidates SET skill_id = NULL WHERE id = $1`, unmappedID); err != nil {
		t.Fatal(err)
	}
	// Decide the material: needs_review should then exclude it.
	if _, err := pool.Exec(context.Background(), `INSERT INTO candidate_publications
		(kind, candidate_id, source_id, decision, status) VALUES ('material', $1, $2, 'approved', 'not_applicable')`,
		materialID, f.sourceID); err != nil {
		t.Fatal(err)
	}

	var tree struct {
		Available bool           `json:"available"`
		Unmapped  StudioUnmapped `json:"unmapped"`
		Grades    []StudioGrade  `json:"grades"`
	}
	if code := getJSON(t, studioRouter(Handler{DB: pool}), "/content-studio/tree", &tree); code != http.StatusOK {
		t.Fatalf("tree status = %d", code)
	}
	if !tree.Available {
		t.Fatal("tree should be available")
	}
	k, ok := findSkill(tree.Grades, skillID)
	if !ok {
		t.Fatalf("skill %d missing from tree", skillID)
	}
	if k.Materials != 1 || k.Questions != 2 {
		t.Fatalf("counts = materials:%d questions:%d, want 1/2", k.Materials, k.Questions)
	}
	if k.NeedsReview != 2 {
		t.Fatalf("needs_review = %d, want 2 (material is decided)", k.NeedsReview)
	}
	if want := publishedCount(t, pool, skillID); k.Published != want {
		t.Fatalf("published = %d, want %d", k.Published, want)
	}
	if tree.Unmapped.Questions != 1 {
		t.Fatalf("unmapped questions = %d, want 1", tree.Unmapped.Questions)
	}
}

func TestContentStudioLesson(t *testing.T) {
	pool, f := setupPublishEnv(t)
	skillID := skillWithPublished(t, pool)

	chunk := newChunk(t, pool, f, "lesson-mat")
	insertMaterial(t, pool, f, skillID, chunk, "Materi B", true)
	insertQuestion(t, pool, f, skillID, chunk, "cloze", "Q-B1", payloadJSON(f.skillCode, "Q-B1"))
	insertQuestion(t, pool, f, skillID, chunk, "scenario", "Q-B2", figuresPayload)
	insertQuestion(t, pool, f, skillID, newChunk(t, pool, f, "lesson-orphan"), "cloze", "Q-ORPH", payloadJSON(f.skillCode, "Q-ORPH"))

	var lesson struct {
		Available       bool                  `json:"available"`
		Skill           StudioLesson          `json:"skill"`
		Materials       []StudioMaterialGroup `json:"materials"`
		OrphanQuestions []CandidateDetail     `json:"orphan_questions"`
		Published       []CanonicalQuestion   `json:"published"`
	}
	if code := getJSON(t, studioRouter(Handler{DB: pool}), "/content-studio/lessons/"+strconv.Itoa(skillID), &lesson); code != http.StatusOK {
		t.Fatalf("lesson status = %d", code)
	}
	if lesson.Skill.ID != skillID {
		t.Fatalf("skill id = %d, want %d", lesson.Skill.ID, skillID)
	}
	if len(lesson.Materials) != 1 {
		t.Fatalf("materials = %d, want 1", len(lesson.Materials))
	}
	if got := len(lesson.Materials[0].Questions); got != 2 {
		t.Fatalf("derived questions = %d, want 2", got)
	}
	if len(lesson.OrphanQuestions) != 1 || lesson.OrphanQuestions[0].Prompt == "" {
		t.Fatalf("orphan_questions = %d, want 1", len(lesson.OrphanQuestions))
	}
	if got, want := len(lesson.Published), publishedCount(t, pool, skillID); got != want {
		t.Fatalf("published = %d, want %d", got, want)
	}
	// figures payload → has_visual; material flagged visual_required → has_visual.
	var sawVisual bool
	for _, q := range lesson.Materials[0].Questions {
		if q.HasVisual {
			sawVisual = true
		}
	}
	if !sawVisual {
		t.Fatal("expected at least one derived question with has_visual")
	}
	if !lesson.Materials[0].Material.HasVisual {
		t.Fatal("material visual_required should set has_visual")
	}
}

func TestContentStudioLessonNotFound(t *testing.T) {
	pool, _ := setupPublishEnv(t)
	if code := getJSON(t, studioRouter(Handler{DB: pool}), "/content-studio/lessons/99999999", nil); code != http.StatusNotFound {
		t.Fatalf("unknown skill status = %d, want 404", code)
	}
}

func TestContentStudioDegraded(t *testing.T) {
	pool, _ := setupPublishEnv(t)
	skillID := skillWithPublished(t, pool)
	if _, err := pool.Exec(context.Background(), `DROP SCHEMA content_staging CASCADE`); err != nil {
		t.Fatal(err)
	}

	var tree struct {
		Available bool          `json:"available"`
		Grades    []StudioGrade `json:"grades"`
	}
	if code := getJSON(t, studioRouter(Handler{DB: pool}), "/content-studio/tree", &tree); code != http.StatusOK {
		t.Fatalf("degraded tree status = %d", code)
	}
	if tree.Available || len(tree.Grades) != 0 {
		t.Fatalf("degraded tree should be available:false with no grades")
	}

	var lesson struct {
		Available bool                `json:"available"`
		Published []CanonicalQuestion `json:"published"`
	}
	if code := getJSON(t, studioRouter(Handler{DB: pool}), "/content-studio/lessons/"+strconv.Itoa(skillID), &lesson); code != http.StatusOK {
		t.Fatalf("degraded lesson status = %d", code)
	}
	if lesson.Available {
		t.Fatal("degraded lesson should be available:false")
	}
	if len(lesson.Published) == 0 {
		t.Fatal("canonical published questions must still render when staging is absent")
	}
}

func TestPayloadHasVisual(t *testing.T) {
	cases := []struct {
		name  string
		raw   string
		vis   bool
		audio bool
	}{
		{"figures", `{"content":{"figures":[{"icon":"orange","count":3}]}}`, true, false},
		{"option figures", `{"content":{"options":[{"figures":[{"icon":"orange","count":2}]}]}}`, true, false},
		{"option image", `{"content":{"options":[{"image":"/api/v1/media/x"}]}}`, true, false},
		{"image", `{"content":{"image":"/api/v1/media/x"}}`, true, false},
		{"audio only", `{"content":{"audio_text":"Baca"}}`, false, true},
		{"text only", `{"content":{"prompt":"x"}}`, false, false},
		{"empty", ``, false, false},
		{"malformed", `{`, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := payloadHasVisual([]byte(tc.raw)); got != tc.vis {
				t.Errorf("payloadHasVisual = %v, want %v", got, tc.vis)
			}
			if got := payloadHasAudio([]byte(tc.raw)); got != tc.audio {
				t.Errorf("payloadHasAudio = %v, want %v", got, tc.audio)
			}
		})
	}
}
