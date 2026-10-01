package changes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/db"
	"github.com/nocify/eduplay/api/internal/seed"
)

// Set TEST_DATABASE_URL to a throwaway database; these tests reset its schema.

type fixtures struct {
	pool       *pgxpool.Pool
	reviewer   uuid.UUID
	documentID uuid.UUID
	eventID    int64
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
	applyStagingDDL(t, pool)

	var f fixtures
	f.pool = pool
	if err := pool.QueryRow(ctx, `INSERT INTO users (email, password_hash, display_name, role)
		VALUES ('reviewer@test', 'x', 'Rev', 'reviewer') RETURNING id`).Scan(&f.reviewer); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO curriculum_source_documents (code, title, source_url, document_hash)
		VALUES ('test-doc', 'Dokumen Uji', 'https://x.test/kurikulum', 'baseline-hash') RETURNING id`).Scan(&f.documentID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO content_staging.curriculum_change_events
		(source_document_id, source_document_code, curriculum_version_id, url,
		 baseline_hash, observed_hash, detection, document_type, cp_added, cp_changed, cp_removed, status)
		VALUES ($1, 'test-doc', NULL, 'https://x.test/kurikulum',
		 'baseline-hash', 'observed-hash', 'content_hash', 'text', 1, 1, 0, 'analyzed')
		RETURNING id`, f.documentID).Scan(&f.eventID); err != nil {
		t.Fatal(err)
	}
	insertImpact(t, pool, f.eventID, "CP-A-MAT-BARU", "added", "Ditambahkan.", 0)
	insertImpact(t, pool, f.eventID, "CP-A-MAT-BILANGAN", "changed", "Diubah.", 2)
	return f
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

func insertImpact(t *testing.T, pool *pgxpool.Pool, eventID int64, code, changeType, statement string, questions int) {
	t.Helper()
	ctx := context.Background()
	var impactID int64
	if err := pool.QueryRow(ctx, `INSERT INTO content_staging.curriculum_change_impacts
		(change_event_id, cp_id, cp_code, change_type, old_statement, new_statement, statement_hash,
		 affected_skills, affected_questions)
		VALUES ($1, NULL, $2, $3, 'lama', $4, 'hash', 1, $5) RETURNING id`,
		eventID, code, changeType, statement, questions).Scan(&impactID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < questions; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO content_staging.curriculum_change_questions
			(impact_id, question_id, question_code, skill_id, match, status)
			VALUES ($1, $2, $3, 1, 'skill', 'published')`,
			impactID, uuid.New(), fmt.Sprintf("Q-%s-%d", code, i)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRecordDecisionUpsertsAndIsIdempotent(t *testing.T) {
	f := setupEnv(t)
	ctx := context.Background()

	first, err := RecordDecision(ctx, f.pool, DecisionInput{
		EventID: f.eventID, Status: StatusAcknowledged, Notes: "akan diperbarui", UserID: f.reviewer,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != StatusAcknowledged || first.Reviewer == nil || *first.Reviewer != "Rev" {
		t.Fatalf("review = %+v", first)
	}

	second, err := RecordDecision(ctx, f.pool, DecisionInput{
		EventID: f.eventID, Status: StatusDismissed, Notes: "ternyata tidak relevan", UserID: f.reviewer,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("decision forked the review row: %d != %d", second.ID, first.ID)
	}

	var count int
	var status, notes string
	if err := f.pool.QueryRow(ctx, `SELECT count(*)::int, max(status), max(notes)
		FROM curriculum_change_reviews WHERE change_event_id = $1`, f.eventID).Scan(&count, &status, &notes); err != nil {
		t.Fatal(err)
	}
	if count != 1 || status != StatusDismissed || notes != "ternyata tidak relevan" {
		t.Fatalf("ledger row = count:%d status:%s notes:%s", count, status, notes)
	}
	// The snapshot is copied by value from staging (D4).
	var observed string
	var affected int
	if err := f.pool.QueryRow(ctx, `SELECT observed_hash, affected_questions
		FROM curriculum_change_reviews WHERE change_event_id = $1`, f.eventID).Scan(&observed, &affected); err != nil {
		t.Fatal(err)
	}
	if observed != "observed-hash" || affected != 2 {
		t.Fatalf("snapshot = %s / %d", observed, affected)
	}
}

func TestRecordDecisionUnknownEvent(t *testing.T) {
	f := setupEnv(t)
	_, err := RecordDecision(context.Background(), f.pool, DecisionInput{
		EventID: 999999, Status: StatusAcknowledged, UserID: f.reviewer,
	})
	if err != ErrChangeNotFound {
		t.Fatalf("err = %v, want ErrChangeNotFound", err)
	}
}

func newRouter(h Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/changes", h.ListChanges)
	r.Get("/changes/{id}", h.GetChange)
	return r
}

func TestListAndGetChange(t *testing.T) {
	f := setupEnv(t)
	h := Handler{DB: f.pool}
	srv := httptest.NewServer(newRouter(h))
	defer srv.Close()

	var list struct {
		Changes   []ChangeSummary `json:"changes"`
		Counts    map[string]int  `json:"counts"`
		Available bool            `json:"available"`
	}
	getJSON(t, srv.URL+"/changes", &list)
	if !list.Available || len(list.Changes) != 1 {
		t.Fatalf("list = %+v", list)
	}
	if list.Changes[0].ReviewStatus != StatusPending || list.Changes[0].CPAdded != 1 || list.Changes[0].AffectedQuestions != 2 {
		t.Fatalf("summary = %+v", list.Changes[0])
	}
	if list.Counts[StatusPending] != 1 {
		t.Fatalf("counts = %+v", list.Counts)
	}

	var detail struct {
		Change ChangeDetail `json:"change"`
	}
	getJSON(t, fmt.Sprintf("%s/changes/%d", srv.URL, f.eventID), &detail)
	if len(detail.Change.Impacts) != 2 {
		t.Fatalf("impacts = %+v", detail.Change.Impacts)
	}
	var totalQuestions int
	for _, imp := range detail.Change.Impacts {
		totalQuestions += len(imp.Questions)
	}
	if totalQuestions != 2 {
		t.Fatalf("question snapshot = %d", totalQuestions)
	}
	if detail.Change.Review != nil {
		t.Fatalf("undecided change should have no review: %+v", detail.Change.Review)
	}

	if _, err := RecordDecision(context.Background(), f.pool, DecisionInput{
		EventID: f.eventID, Status: StatusDeferred, Notes: "nanti", UserID: f.reviewer,
	}); err != nil {
		t.Fatal(err)
	}
	var after struct {
		Change ChangeDetail `json:"change"`
	}
	getJSON(t, fmt.Sprintf("%s/changes/%d", srv.URL, f.eventID), &after)
	if after.Change.Review == nil || after.Change.ReviewStatus != StatusDeferred {
		t.Fatalf("review after decision = %+v", after.Change)
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
