package candidates

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/creators"
	"github.com/nocify/eduplay/api/internal/db"
	"github.com/nocify/eduplay/api/internal/seed"
)

// Set TEST_DATABASE_URL to a throwaway database; these tests reset its schema.

type stagingFixtures struct {
	sourceID   int
	documentID int64
	chunkID    int64
	skillCode  string
}

func setupPublishEnv(t *testing.T) (*pgxpool.Pool, stagingFixtures) {
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
	applyStagingDDL(t, pool)

	var f stagingFixtures
	if err := pool.QueryRow(ctx, `SELECT code FROM skills ORDER BY id LIMIT 1`).Scan(&f.skillCode); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO content_staging.source_registry
		(code, name, base_url, source_type, tier, enabled, status)
		VALUES ('test-src', 'Test Source', 'https://x.test', 'official_curriculum', 1, true, 'active')
		RETURNING id`).Scan(&f.sourceID); err != nil {
		t.Fatal(err)
	}
	var runID int64
	if err := pool.QueryRow(ctx, `INSERT INTO content_staging.crawl_runs (source_id) VALUES ($1) RETURNING id`, f.sourceID).Scan(&runID); err != nil {
		t.Fatal(err)
	}
	var itemID int64
	if err := pool.QueryRow(ctx, `INSERT INTO content_staging.crawl_items
		(crawl_run_id, source_id, url, canonical_url, status) VALUES ($1, $2, 'https://x.test/a', 'https://x.test/a', 'fetched')
		RETURNING id`, runID, f.sourceID).Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO content_staging.source_documents
		(source_id, crawl_item_id, title, canonical_url, document_type, content_hash, retrieved_at, license)
		VALUES ($1, $2, 'Dokumen Uji', 'https://x.test/a', 'html', 'doc-hash', '2026-09-30T00:00:00Z', 'public')
		RETURNING id`, f.sourceID, itemID).Scan(&f.documentID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO content_staging.document_chunks
		(document_id, heading, sequence, text, hash) VALUES ($1, 'Bab', 0, 'isi chunk', 'chunk-hash')
		RETURNING id`, f.documentID).Scan(&f.chunkID); err != nil {
		t.Fatal(err)
	}
	return pool, f
}

func applyStagingDDL(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	files, err := filepath.Glob("../../../../services/content-pipeline/sql/0*.sql")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatal("no staging migrations found")
	}
	for _, path := range files {
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), string(sqlBytes)); err != nil {
			t.Fatalf("apply %s: %v", filepath.Base(path), err)
		}
	}
}

func insertReviewer(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := pool.QueryRow(context.Background(), `INSERT INTO users (email, password_hash, display_name, role)
		VALUES ('reviewer@test', 'x', 'Rev', 'reviewer') RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func payloadJSON(skillCode, code string) string {
	return fmt.Sprintf(`{"code":%q,"skill":%q,"type":"short_answer","difficulty":3,`+
		`"learning_objective":"Peserta didik dapat X.","content":{"prompt":"Lengkapi: _____"},`+
		`"answer":{"accepted":["x"]},"explanation":"Jawaban: x."}`, code, skillCode)
}

func insertCandidate(t *testing.T, pool *pgxpool.Pool, f stagingFixtures, code, contentHash string) int64 {
	t.Helper()
	ctx := context.Background()
	var chunkID int64
	if err := pool.QueryRow(ctx, `INSERT INTO content_staging.document_chunks
		(document_id, heading, sequence, text, hash)
		VALUES ($1, 'Bab', (SELECT COALESCE(max(sequence), -1) + 1 FROM content_staging.document_chunks WHERE document_id = $1),
		        'isi chunk', $2) RETURNING id`, f.documentID, code+"-chunk").Scan(&chunkID); err != nil {
		t.Fatal(err)
	}
	var id int64
	if err := pool.QueryRow(ctx, `INSERT INTO content_staging.question_candidates
		(document_id, chunk_id, source_id, skill_id, phase_code, grade_code, subject_code,
		 template, type, difficulty, learning_objective, prompt, payload, content_hash, confidence, qa_issues, status)
		VALUES ($1, $2, $3, NULL, '', '', '', 'cloze', 'short_answer', 3, 'Peserta didik dapat X.',
		 'Lengkapi: _____', $4::jsonb, $5, 0.9, '[]'::jsonb, 'draft') RETURNING id`,
		f.documentID, chunkID, f.sourceID, payloadJSON(f.skillCode, code), contentHash).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func queuePublication(t *testing.T, pool *pgxpool.Pool, f stagingFixtures, candidateID int64, decidedBy *uuid.UUID) {
	t.Helper()
	checklist := `{"academic_answer":true}`
	if _, err := pool.Exec(context.Background(), `INSERT INTO candidate_publications
		(kind, candidate_id, source_id, decision, decided_by, notes, checklist, status)
		VALUES ('question', $1, $2, 'approved', $3, '', $4::jsonb, 'queued')`,
		candidateID, f.sourceID, decidedBy, checklist); err != nil {
		t.Fatal(err)
	}
}

func TestPublishHappyPathAndIdempotent(t *testing.T) {
	pool, f := setupPublishEnv(t)
	ctx := context.Background()
	reviewer := insertReviewer(t, pool)
	cid := insertCandidate(t, pool, f, "Q-TEST-HAPPY", "hash-happy")
	queuePublication(t, pool, f, cid, &reviewer)

	dry, err := Publish(ctx, pool, PublishOptions{DryRun: true, All: true})
	if err != nil {
		t.Fatal(err)
	}
	if dry.Published != 1 || dry.Remaining != 1 {
		t.Fatalf("dry run report = %+v", dry)
	}
	var count int
	_ = pool.QueryRow(ctx, `SELECT count(*)::int FROM questions WHERE code = 'Q-TEST-HAPPY'`).Scan(&count)
	if count != 0 {
		t.Fatal("dry run wrote a canonical question")
	}

	rep, err := Publish(ctx, pool, PublishOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Published != 1 || rep.Failed != 0 {
		t.Fatalf("publish report = %+v", rep)
	}

	var status string
	var publishedVersion *uuid.UUID
	var reviewerID *uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT status, published_version_id, reviewer_id FROM questions WHERE code = 'Q-TEST-HAPPY'`).
		Scan(&status, &publishedVersion, &reviewerID); err != nil {
		t.Fatal(err)
	}
	if status != "published" || publishedVersion == nil || reviewerID == nil || *reviewerID != reviewer {
		t.Fatalf("question not published correctly: status=%s version=%v reviewer=%v", status, publishedVersion, reviewerID)
	}

	var versions int
	_ = pool.QueryRow(ctx, `SELECT count(*)::int FROM question_versions v JOIN questions q ON q.id = v.question_id WHERE q.code = 'Q-TEST-HAPPY'`).Scan(&versions)
	if versions != 1 {
		t.Fatalf("versions = %d, want 1", versions)
	}
	var provHash, provSource string
	var provSkill *int
	if err := pool.QueryRow(ctx, `SELECT p.content_hash, p.source_code, p.skill_id FROM question_provenance p
		JOIN questions q ON q.id = p.question_id WHERE q.code = 'Q-TEST-HAPPY'`).Scan(&provHash, &provSource, &provSkill); err != nil {
		t.Fatal(err)
	}
	if provHash != "hash-happy" || provSource != "test-src" || provSkill == nil {
		t.Fatalf("provenance wrong: hash=%s source=%s skill=%v", provHash, provSource, provSkill)
	}
	// M8: pipeline content is attributed to the system creator, and the
	// adaptation lineage records the reviewing human (§39).
	var creatorID, lineageReviewer *uuid.UUID
	var adaptation string
	if err := pool.QueryRow(ctx, `SELECT q.creator_id, p.adaptation_type, p.reviewed_by
		FROM questions q JOIN question_provenance p ON p.question_id = q.id
		WHERE q.code = 'Q-TEST-HAPPY'`).Scan(&creatorID, &adaptation, &lineageReviewer); err != nil {
		t.Fatal(err)
	}
	if creatorID == nil || *creatorID != creators.SystemCreatorID {
		t.Fatalf("pipeline question not attributed to the system creator: %v", creatorID)
	}
	if adaptation != "original" || lineageReviewer == nil || *lineageReviewer != reviewer {
		t.Fatalf("adaptation lineage wrong: type=%s reviewer=%v", adaptation, lineageReviewer)
	}
	var toStatus string
	var checklist []byte
	if err := pool.QueryRow(ctx, `SELECT to_status, checklist FROM question_reviews r
		JOIN questions q ON q.id = r.question_id WHERE q.code = 'Q-TEST-HAPPY'`).Scan(&toStatus, &checklist); err != nil {
		t.Fatal(err)
	}
	if toStatus != "published" || !strings.Contains(string(checklist), "academic_answer") {
		t.Fatalf("review row wrong: %s %s", toStatus, checklist)
	}
	var pubStatus string
	var canonical *uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT status, canonical_question_id FROM candidate_publications WHERE candidate_id = $1`, cid).
		Scan(&pubStatus, &canonical); err != nil {
		t.Fatal(err)
	}
	if pubStatus != "published" || canonical == nil {
		t.Fatalf("publication not marked published: %s %v", pubStatus, canonical)
	}

	again, err := Publish(ctx, pool, PublishOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if again.Claimed != 0 || again.Published != 0 {
		t.Fatalf("second run should be a no-op: %+v", again)
	}
}

func TestPublishLinksExistingQuestion(t *testing.T) {
	pool, f := setupPublishEnv(t)
	ctx := context.Background()
	reviewer := insertReviewer(t, pool)

	first := insertCandidate(t, pool, f, "Q-TEST-LINK", "hash-link")
	queuePublication(t, pool, f, first, &reviewer)
	if _, err := Publish(ctx, pool, PublishOptions{All: true}); err != nil {
		t.Fatal(err)
	}

	second := insertCandidate(t, pool, f, "Q-TEST-LINK", "hash-link")
	queuePublication(t, pool, f, second, &reviewer)
	rep, err := Publish(ctx, pool, PublishOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Linked != 1 || rep.Published != 0 {
		t.Fatalf("expected link, got %+v", rep)
	}
	var questions, versions int
	_ = pool.QueryRow(ctx, `SELECT count(*)::int FROM questions WHERE code = 'Q-TEST-LINK'`).Scan(&questions)
	_ = pool.QueryRow(ctx, `SELECT count(*)::int FROM question_versions v JOIN questions q ON q.id = v.question_id WHERE q.code = 'Q-TEST-LINK'`).Scan(&versions)
	if questions != 1 || versions != 1 {
		t.Fatalf("link created duplicates: questions=%d versions=%d", questions, versions)
	}
}

func TestPublishFailsOnCodeCollision(t *testing.T) {
	pool, f := setupPublishEnv(t)
	ctx := context.Background()
	reviewer := insertReviewer(t, pool)

	first := insertCandidate(t, pool, f, "Q-TEST-COLLIDE", "hash-a")
	queuePublication(t, pool, f, first, &reviewer)
	if _, err := Publish(ctx, pool, PublishOptions{All: true}); err != nil {
		t.Fatal(err)
	}
	second := insertCandidate(t, pool, f, "Q-TEST-COLLIDE", "hash-b")
	queuePublication(t, pool, f, second, &reviewer)
	rep, err := Publish(ctx, pool, PublishOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Failed != 1 {
		t.Fatalf("expected failure, got %+v", rep)
	}
	var lastError string
	_ = pool.QueryRow(ctx, `SELECT last_error FROM candidate_publications WHERE candidate_id = $1`, second).Scan(&lastError)
	if !strings.Contains(lastError, "collision") {
		t.Fatalf("last_error = %q", lastError)
	}
}

func TestPublishFailsOnMissingSkill(t *testing.T) {
	pool, f := setupPublishEnv(t)
	ctx := context.Background()
	reviewer := insertReviewer(t, pool)
	var cid int64
	if err := pool.QueryRow(ctx, `INSERT INTO content_staging.question_candidates
		(document_id, chunk_id, source_id, phase_code, grade_code, subject_code, template, type,
		 difficulty, learning_objective, prompt, payload, content_hash, confidence, qa_issues, status)
		VALUES ($1, $2, $3, '', '', '', 'cloze', 'short_answer', 3, 'o', 'p', $4::jsonb, 'hash-miss', 0.9, '[]'::jsonb, 'draft')
		RETURNING id`, f.documentID, f.chunkID, f.sourceID, payloadJSON("NOPE-999", "Q-TEST-MISS")).Scan(&cid); err != nil {
		t.Fatal(err)
	}
	queuePublication(t, pool, f, cid, &reviewer)
	rep, err := Publish(ctx, pool, PublishOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Failed != 1 {
		t.Fatalf("expected failure, got %+v", rep)
	}
	var lastError string
	_ = pool.QueryRow(ctx, `SELECT last_error FROM candidate_publications WHERE candidate_id = $1`, cid).Scan(&lastError)
	if !strings.Contains(lastError, "not found") {
		t.Fatalf("last_error = %q", lastError)
	}
}

func TestPublishFailsWithoutReviewer(t *testing.T) {
	pool, f := setupPublishEnv(t)
	ctx := context.Background()
	cid := insertCandidate(t, pool, f, "Q-TEST-NOREV", "hash-norev")
	queuePublication(t, pool, f, cid, nil)
	rep, err := Publish(ctx, pool, PublishOptions{All: true})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Failed != 1 {
		t.Fatalf("expected failure, got %+v", rep)
	}
	var lastError string
	_ = pool.QueryRow(ctx, `SELECT last_error FROM candidate_publications WHERE candidate_id = $1`, cid).Scan(&lastError)
	if !strings.Contains(lastError, "no reviewer") {
		t.Fatalf("last_error = %q", lastError)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*)::int FROM questions WHERE code = 'Q-TEST-NOREV'`).Scan(&n)
	if n != 0 {
		t.Fatal("question created despite missing reviewer")
	}
}

func TestPublishRetryFailed(t *testing.T) {
	pool, f := setupPublishEnv(t)
	ctx := context.Background()
	reviewer := insertReviewer(t, pool)
	cid := insertCandidate(t, pool, f, "Q-TEST-RETRY", "hash-retry")
	queuePublication(t, pool, f, cid, nil)
	if _, err := Publish(ctx, pool, PublishOptions{All: true}); err != nil {
		t.Fatal(err)
	}
	// The reviewer is now set; retrying should publish.
	if _, err := pool.Exec(ctx, `UPDATE candidate_publications SET decided_by = $2 WHERE candidate_id = $1`, cid, reviewer); err != nil {
		t.Fatal(err)
	}
	rep, err := Publish(ctx, pool, PublishOptions{All: true, RetryFailed: true})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Published != 1 {
		t.Fatalf("retry should publish, got %+v", rep)
	}
}

func TestCandidatePayloadRoundTrip(t *testing.T) {
	var p CandidatePayload
	if err := json.Unmarshal([]byte(payloadJSON("SK-1", "Q-1")), &p); err != nil {
		t.Fatal(err)
	}
	if p.Code != "Q-1" || p.Skill != "SK-1" || p.Type != "short_answer" || p.Difficulty != 3 {
		t.Fatalf("payload = %+v", p)
	}
}
