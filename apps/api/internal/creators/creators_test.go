package creators

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/attribution"
	"github.com/nocify/eduplay/api/internal/db"
	"github.com/nocify/eduplay/api/internal/seed"
)

// Set TEST_DATABASE_URL to a throwaway database; these tests reset its schema.

type fixtures struct {
	pool     *pgxpool.Pool
	question uuid.UUID
	skillID  int
	actor    uuid.UUID
}

func setupEnv(t *testing.T) fixtures {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	for _, stmt := range []string{
		`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`,
		`DROP SCHEMA IF EXISTS content_staging CASCADE;`,
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := seed.Run(ctx, pool, seed.Options{ContentDir: "../../../../content", AssetsDir: "../../../../assets", Publish: true}); err != nil {
		t.Fatal(err)
	}
	var f fixtures
	f.pool = pool
	if err := pool.QueryRow(ctx, `SELECT id, skill_id FROM questions WHERE status = 'published' ORDER BY code LIMIT 1`).
		Scan(&f.question, &f.skillID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, password_hash, display_name, role)
		VALUES ('editor@test', 'x', 'Editor', 'content_editor') RETURNING id`).Scan(&f.actor); err != nil {
		t.Fatal(err)
	}
	return f
}

type item struct {
	finalized bool
	correct   bool
	firstTry  bool
	hint      bool
}

// makeEvidence creates a child, an assessment and the given items for one question.
func makeEvidence(t *testing.T, pool *pgxpool.Pool, questionID uuid.UUID, skillID int, items []item) {
	t.Helper()
	ctx := context.Background()
	var parent, child, versionID, assessmentID uuid.UUID
	var gradeID, subjectID int
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, password_hash, display_name, role)
		VALUES ($1, 'x', 'Parent', 'parent') RETURNING id`, fmt.Sprintf("p-%s@test", uuid.NewString()[:8])).Scan(&parent); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM grades ORDER BY id LIMIT 1`).Scan(&gradeID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM subjects ORDER BY id LIMIT 1`).Scan(&subjectID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO children (parent_id, nickname, grade_id) VALUES ($1, 'Anak', $2) RETURNING id`,
		parent, gradeID).Scan(&child); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT COALESCE(published_version_id, current_version_id) FROM questions WHERE id = $1`,
		questionID).Scan(&versionID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO assessments (child_id, subject_id, grade_id, status, planned_count)
		VALUES ($1, $2, $3, 'in_progress', $4) RETURNING id`, child, subjectID, gradeID, len(items)).Scan(&assessmentID); err != nil {
		t.Fatal(err)
	}
	for i, it := range items {
		var finalizedAt, finalCorrect, firstTry any
		if it.finalized {
			finalizedAt, finalCorrect, firstTry = time.Now(), it.correct, it.firstTry
		}
		if _, err := pool.Exec(ctx, `INSERT INTO assessment_items
			(assessment_id, position, question_id, question_version_id, skill_id, difficulty, hint_shown, attempts,
			 final_correct, first_try_correct, finalized_at)
			VALUES ($1, $2, $3, $4, $5, 1, $6, 1, $7, $8, $9)`,
			assessmentID, i+1, questionID, versionID, skillID, it.hint, finalCorrect, firstTry, finalizedAt); err != nil {
			t.Fatal(err)
		}
	}
}

func attributionOf(t *testing.T, pool *pgxpool.Pool, questionID uuid.UUID) (usage, completion, correct, firstTry, hint int, quality float64, creatorID *uuid.UUID) {
	t.Helper()
	err := pool.QueryRow(context.Background(), `SELECT usage_count, completion_count, correct_count,
		first_try_count, hint_count, quality_score::float8, creator_id FROM question_attribution WHERE question_id = $1`,
		questionID).Scan(&usage, &completion, &correct, &firstTry, &hint, &quality, &creatorID)
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestSystemCreatorSeeded(t *testing.T) {
	f := setupEnv(t)
	var code, kind, status string
	var share float64
	if err := f.pool.QueryRow(context.Background(), `SELECT code, kind, status, share_pct::float8 FROM creators WHERE id = $1`,
		SystemCreatorID).Scan(&code, &kind, &status, &share); err != nil {
		t.Fatalf("system creator not seeded: %v", err)
	}
	if code != "system" || kind != KindSystem || status != StatusActive || share != 0 {
		t.Fatalf("system creator = %s/%s/%s/%v", code, kind, status, share)
	}
}

func TestRecomputeRollupIsIdempotent(t *testing.T) {
	f := setupEnv(t)
	ctx := context.Background()
	makeEvidence(t, f.pool, f.question, f.skillID, []item{
		{finalized: true, correct: true, firstTry: true},
		{finalized: true, correct: false, firstTry: false, hint: true},
		{finalized: false},
	})
	for run := 0; run < 2; run++ {
		if err := attribution.RecomputeQuestions(ctx, f.pool, []uuid.UUID{f.question}); err != nil {
			t.Fatal(err)
		}
		usage, completion, correct, firstTry, hint, quality, _ := attributionOf(t, f.pool, f.question)
		if usage != 3 || completion != 2 || correct != 1 || firstTry != 1 || hint != 1 {
			t.Fatalf("run %d stats = %d/%d/%d/%d/%d", run, usage, completion, correct, firstTry, hint)
		}
		if quality != 53.33 {
			t.Fatalf("run %d quality = %v, want 53.33", run, quality)
		}
	}
	// A full rebuild agrees with the incremental one.
	if err := attribution.RecomputeAll(ctx, f.pool); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, quality, _ := attributionOf(t, f.pool, f.question); quality != 53.33 {
		t.Fatalf("full rebuild quality = %v", quality)
	}
}

// driftCounts reports rollup-vs-evidence mismatches: questions with evidence but
// no rollup row, orphan rollups, and rows whose counts disagree with the raw
// assessment_items. Zero means the rollup is consistent.
func driftCounts(t *testing.T, pool *pgxpool.Pool) (missing, orphans, values int) {
	t.Helper()
	if err := pool.QueryRow(context.Background(), `
		SELECT
		 (SELECT count(*)::int FROM (SELECT DISTINCT question_id FROM assessment_items) e
		    LEFT JOIN question_attribution a ON a.question_id = e.question_id
		    WHERE a.question_id IS NULL),
		 (SELECT count(*)::int FROM question_attribution a
		    WHERE NOT EXISTS (SELECT 1 FROM assessment_items i WHERE i.question_id = a.question_id)),
		 (SELECT count(*)::int FROM (
		    SELECT i.question_id,
		           count(*)::int AS usage,
		           count(*) FILTER (WHERE i.finalized_at IS NOT NULL)::int AS completion,
		           count(*) FILTER (WHERE i.final_correct)::int AS correct,
		           count(*) FILTER (WHERE i.first_try_correct)::int AS first_try,
		           count(*) FILTER (WHERE i.hint_shown)::int AS hint
		    FROM assessment_items i GROUP BY i.question_id) e
		    JOIN question_attribution a ON a.question_id = e.question_id
		    WHERE a.usage_count <> e.usage OR a.completion_count <> e.completion
		       OR a.correct_count <> e.correct OR a.first_try_count <> e.first_try
		       OR a.hint_count <> e.hint)`).
		Scan(&missing, &orphans, &values); err != nil {
		t.Fatal(err)
	}
	return missing, orphans, values
}

// TestRollupMatchesEvidence guards the rollup contract: after a recompute, every
// question with evidence has a row whose counts match the raw items — including
// a served-but-unanswered item, which still counts as usage. The session-complete
// hook must therefore recompute every SERVED question, not only the answered ones.
func TestRollupMatchesEvidence(t *testing.T) {
	f := setupEnv(t)
	ctx := context.Background()
	var q2 uuid.UUID
	var skill2 int
	if err := f.pool.QueryRow(ctx,
		`SELECT id, skill_id FROM questions WHERE status = 'published' AND id <> $1 ORDER BY code LIMIT 1`,
		f.question).Scan(&q2, &skill2); err != nil {
		t.Fatal(err)
	}
	makeEvidence(t, f.pool, f.question, f.skillID, []item{
		{finalized: true, correct: true, firstTry: true},
		{finalized: false},
	})
	makeEvidence(t, f.pool, q2, skill2, []item{{finalized: true, correct: false, firstTry: false}})

	if err := attribution.RecomputeQuestions(ctx, f.pool, []uuid.UUID{f.question, q2}); err != nil {
		t.Fatal(err)
	}
	if missing, orphans, values := driftCounts(t, f.pool); missing != 0 || orphans != 0 || values != 0 {
		t.Fatalf("rollup drift after recompute: missing=%d orphans=%d values=%d", missing, orphans, values)
	}
	usage, completion, _, _, _, _, _ := attributionOf(t, f.pool, f.question)
	if usage != 2 || completion != 1 {
		t.Fatalf("usage/completion = %d/%d, want 2/1 (served item counts as usage)", usage, completion)
	}
}

func TestAttributionCascadesWithQuestion(t *testing.T) {
	f := setupEnv(t)
	ctx := context.Background()
	var qid uuid.UUID
	if err := f.pool.QueryRow(ctx, `INSERT INTO questions (code, skill_id, type, difficulty, learning_objective)
		VALUES ('Q-CASCADE', $1, 'short_answer', 1, 'uji') RETURNING id`, f.skillID).Scan(&qid); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO question_attribution (question_id, usage_count) VALUES ($1, 1)`, qid); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `DELETE FROM questions WHERE id = $1`, qid); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*)::int FROM question_attribution WHERE question_id = $1`, qid).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("attribution row survived question delete: %d", count)
	}
}

func TestAssignAndUnassignFollowsRollup(t *testing.T) {
	f := setupEnv(t)
	ctx := context.Background()
	makeEvidence(t, f.pool, f.question, f.skillID, []item{{finalized: true, correct: true, firstTry: true}})
	if err := attribution.RecomputeQuestions(ctx, f.pool, []uuid.UUID{f.question}); err != nil {
		t.Fatal(err)
	}
	var creatorID uuid.UUID
	if err := f.pool.QueryRow(ctx, `INSERT INTO creators (code, display_name, kind) VALUES ('C-TEST', 'Penulis Uji', 'external') RETURNING id`).
		Scan(&creatorID); err != nil {
		t.Fatal(err)
	}
	h := Handler{DB: f.pool}
	if err := h.setCreator(ctx, f.question, &creatorID, f.actor); err != nil {
		t.Fatal(err)
	}
	var gotQuestion *uuid.UUID
	if err := f.pool.QueryRow(ctx, `SELECT creator_id FROM questions WHERE id = $1`, f.question).Scan(&gotQuestion); err != nil {
		t.Fatal(err)
	}
	_, _, _, _, _, _, gotAttribution := attributionOf(t, f.pool, f.question)
	if gotQuestion == nil || *gotQuestion != creatorID {
		t.Fatalf("question creator = %v, want %v", gotQuestion, creatorID)
	}
	if gotAttribution == nil || *gotAttribution != creatorID {
		t.Fatalf("attribution creator = %v, want %v", gotAttribution, creatorID)
	}

	if err := h.setCreator(ctx, f.question, nil, f.actor); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, _, after := attributionOf(t, f.pool, f.question); after != nil {
		t.Fatalf("attribution creator after unassign = %v", after)
	}
}

func TestListAndGetCreator(t *testing.T) {
	f := setupEnv(t)
	ctx := context.Background()
	makeEvidence(t, f.pool, f.question, f.skillID, []item{
		{finalized: true, correct: true, firstTry: true},
		{finalized: true, correct: true, firstTry: false, hint: true},
	})
	var creatorID uuid.UUID
	if err := f.pool.QueryRow(ctx, `INSERT INTO creators (code, display_name, kind, share_pct)
		VALUES ('C-LIST', 'Bank Soal Uji', 'external', 12.5) RETURNING id`).Scan(&creatorID); err != nil {
		t.Fatal(err)
	}
	if err := (Handler{DB: f.pool}).setCreator(ctx, f.question, &creatorID, f.actor); err != nil {
		t.Fatal(err)
	}

	h := Handler{DB: f.pool}
	r := chi.NewRouter()
	r.Get("/creators", h.ListCreators)
	r.Get("/creators/{creatorID}", h.GetCreator)
	srv := httptest.NewServer(r)
	defer srv.Close()

	var list struct {
		Creators []CreatorSummary `json:"creators"`
		Counts   map[string]int   `json:"counts"`
	}
	getJSON(t, srv.URL+"/creators", &list)
	if list.Counts[StatusActive] != 2 { // the system creator + C-LIST
		t.Fatalf("counts = %+v", list.Counts)
	}
	var found *CreatorSummary
	for i := range list.Creators {
		if list.Creators[i].ID == creatorID {
			found = &list.Creators[i]
		}
	}
	if found == nil {
		t.Fatalf("creator missing from list: %+v", list.Creators)
	}
	if found.Questions != 1 || found.UsageCount != 2 || found.CompletionCount != 2 || found.SharePct != 12.5 {
		t.Fatalf("summary = %+v", *found)
	}
	if found.AvgQuality != 75 { // 100*(0.5*1 + 0.3*0.5 + 0.2*0.5)
		t.Fatalf("avg quality = %v, want 75", found.AvgQuality)
	}

	var detail struct {
		Creator   CreatorSummary   `json:"creator"`
		Questions []AttributionRow `json:"questions"`
	}
	getJSON(t, fmt.Sprintf("%s/creators/%s", srv.URL, creatorID), &detail)
	if detail.Creator.ID != creatorID || len(detail.Questions) != 1 {
		t.Fatalf("detail = %+v", detail)
	}
	if detail.Questions[0].QuestionID != f.question || detail.Questions[0].QualityScore != 75 {
		t.Fatalf("question row = %+v", detail.Questions[0])
	}
}

func getJSON(t *testing.T, url string, v any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatal(err)
	}
}
