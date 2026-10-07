package server_test

// Critical E2E flow of §48 against a real PostgreSQL:
//
//	Register → Create Child → Start Challenge → Answer → Complete → Result →
//	Parent Progress → Subscribe → Payment Webhook → Subscription Active
//
// Set TEST_DATABASE_URL to a throwaway database; the test resets its schema.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/attribution"
	"github.com/nocify/eduplay/api/internal/config"
	"github.com/nocify/eduplay/api/internal/db"
	"github.com/nocify/eduplay/api/internal/media"
	"github.com/nocify/eduplay/api/internal/rewards"
	"github.com/nocify/eduplay/api/internal/seed"
	"github.com/nocify/eduplay/api/internal/server"
	"github.com/nocify/eduplay/api/internal/subscriptions"
)

const (
	origin = "http://app.test"
	// testPassword is the fixture password for registered test accounts. The
	// literal trips GitGuardian's generic-password detector; it is not a secret.
	testPassword = "rahasia123" // ggignore
)

type env struct {
	t    *testing.T
	srv  *httptest.Server
	pool *pgxpool.Pool
	cfg  config.Config
}

func setup(t *testing.T) *env {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	var currentDB string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&currentDB); err != nil {
		t.Fatalf("failed to query current_database: %v", err)
	}
	if !strings.Contains(strings.ToLower(currentDB), "test") {
		t.Fatalf("safety guard: refusing to DROP SCHEMA on non-test database %q (database name must contain 'test')", currentDB)
	}
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := seed.Run(ctx, pool, seed.Options{ContentDir: "../../../../content", AssetsDir: "../../../../assets", Publish: true}); err != nil {
		t.Fatal(err)
	}
	if err := rewards.LoadFreeAvatars(ctx, pool); err != nil {
		t.Fatal(err)
	}
	tz, _ := time.LoadLocation("Asia/Jakarta")
	cfg := config.Config{Env: "test", SessionSecret: "s", SessionTTL: time.Hour, PublicURL: origin,
		PaymentProvider: "mock", PaymentSecret: "whsec", MonthlyPriceIDR: 49000,
		FreeDailyChallenges: 2, FreeMaxChildren: 1, PaidMaxChildren: 5, StorageDir: t.TempDir(), Timezone: tz}
	srv := httptest.NewServer(server.New(cfg, pool, media.LocalStorage{Dir: t.TempDir()}))
	t.Cleanup(func() { srv.Close(); pool.Close() })
	return &env{t: t, srv: srv, pool: pool, cfg: cfg}
}

type client struct {
	e *env
	c *http.Client
}

func (e *env) client() *client {
	jar, _ := cookiejar.New(nil)
	return &client{e: e, c: &http.Client{Jar: jar}}
}

func (c *client) do(method, path string, body any, headers ...string) (int, map[string]any) {
	c.e.t.Helper()
	var rd io.Reader
	if body != nil {
		if b, ok := body.([]byte); ok {
			rd = bytes.NewReader(b)
		} else {
			b, _ := json.Marshal(body)
			rd = bytes.NewReader(b)
		}
	}
	req, _ := http.NewRequest(method, c.e.srv.URL+"/api/v1"+path, rd)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", origin)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := c.c.Do(req)
	if err != nil {
		c.e.t.Fatal(err)
	}
	defer res.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func (c *client) must(want int, method, path string, body any) map[string]any {
	c.e.t.Helper()
	code, out := c.do(method, path, body)
	if code != want {
		c.e.t.Fatalf("%s %s: status %d, want %d: %v", method, path, code, want, out)
	}
	return out
}

func (c *client) register(email string) map[string]any {
	return c.must(201, "POST", "/auth/register", map[string]any{"email": email, "password": testPassword, "display_name": "Ibu Test", "consent": true})
}

func gradeID(e *env, code string) int {
	var id int
	if err := e.pool.QueryRow(context.Background(), `SELECT id FROM grades WHERE code = $1`, code).Scan(&id); err != nil {
		e.t.Fatal(err)
	}
	return id
}

func subjectID(e *env, code string) int {
	var id int
	if err := e.pool.QueryRow(context.Background(), `SELECT id FROM subjects WHERE code = $1`, code).Scan(&id); err != nil {
		e.t.Fatal(err)
	}
	return id
}

// correctResponse reads the stored answer for an item, standing in for a child who knows it.
func correctResponse(e *env, itemID string) json.RawMessage {
	var answer json.RawMessage
	if err := e.pool.QueryRow(context.Background(), `SELECT qv.answer FROM assessment_items i JOIN question_versions qv ON qv.id = i.question_version_id WHERE i.id = $1`, itemID).Scan(&answer); err != nil {
		e.t.Fatal(err)
	}
	var a map[string]any
	_ = json.Unmarshal(answer, &a)
	if acc, ok := a["accepted"].([]any); ok {
		b, _ := json.Marshal(map[string]any{"text": acc[0]})
		return b
	}
	return answer
}

func TestCriticalFlow(t *testing.T) {
	e := setup(t)
	parent := e.client()

	// Register + unauthenticated guard
	if code, _ := e.client().do("GET", "/me", nil); code != 401 {
		t.Fatalf("anonymous /me should be 401, got %d", code)
	}
	parent.register("parent@example.test")
	me := parent.must(200, "GET", "/me", nil)
	if me["user"].(map[string]any)["role"] != "parent" {
		t.Fatal("new accounts must be parents")
	}

	// Create child (free: max 1)
	child := parent.must(201, "POST", "/children", map[string]any{"nickname": "Adi", "grade_id": gradeID(e, "sd-1"), "avatar": "dog"})["child"].(map[string]any)
	childID := child["id"].(string)
	if code, out := parent.do("POST", "/children", map[string]any{"nickname": "Bima", "grade_id": gradeID(e, "sd-1")}); code != 402 || out["error"] != "upgrade_required" {
		t.Fatalf("second child on free plan: %d %v", code, out)
	}

	// Another parent cannot see or use this child
	stranger := e.client()
	stranger.register("stranger@example.test")
	if code, _ := stranger.do("GET", "/children/"+childID+"/progress", nil); code != 404 {
		t.Fatalf("cross-parent access must be 404, got %d", code)
	}
	if code, _ := stranger.do("POST", "/assessments", map[string]any{"child_id": childID, "subject_id": subjectID(e, "matematika")}); code != 404 {
		t.Fatalf("cross-parent assessment must be 404, got %d", code)
	}

	// Home lists subjects for the child's grade
	home := parent.must(200, "GET", "/children/"+childID+"/home", nil)
	if n := len(home["subjects"].([]any)); n != 2 {
		t.Fatalf("SD 1 should have 2 subjects with content, got %d", n)
	}

	// Start a challenge
	asm := parent.must(201, "POST", "/assessments", map[string]any{"child_id": childID, "subject_id": subjectID(e, "matematika")})["assessment"].(map[string]any)
	asmID := asm["id"].(string)
	planned := int(asm["planned_count"].(float64))
	if planned != 8 {
		t.Fatalf("SD 1 session size should be 8, got %d", planned)
	}
	cur := asm["current"].(map[string]any)
	if _, leaked := cur["content"].(map[string]any)["answer"]; leaked {
		t.Fatal("answer must never be sent to the client")
	}

	// Resuming returns the same session instead of spending quota
	again := parent.must(200, "POST", "/assessments", map[string]any{"child_id": childID, "subject_id": subjectID(e, "matematika")})["assessment"].(map[string]any)
	if again["id"] != asmID {
		t.Fatal("in-progress session should be resumed")
	}

	// Item 1: wrong → hint + retry → correct (retry)
	res := parent.must(200, "POST", "/assessments/"+asmID+"/answers", map[string]any{"item_id": cur["id"], "response": map[string]any{"value": -999, "option_id": "zzz"}, "response_time_ms": 3000})
	if res["correct"] != false || res["final"] != false || res["hint"] == "" || !strings.HasPrefix(res["feedback"].(string), "Hampir benar") {
		t.Fatalf("wrong first attempt should give hint + retry: %v", res)
	}
	res = parent.must(200, "POST", "/assessments/"+asmID+"/answers", map[string]any{"item_id": cur["id"], "response": correctResponse(e, cur["id"].(string)), "response_time_ms": 2000})
	if res["correct"] != true || res["final"] != true || res["explanation"] == "" {
		t.Fatalf("correct retry should finalize: %v", res)
	}
	// A finalized item cannot be answered again (attempts are immutable evidence)
	if code, _ := parent.do("POST", "/assessments/"+asmID+"/answers", map[string]any{"item_id": cur["id"], "response": correctResponse(e, cur["id"].(string))}); code != 409 {
		t.Fatalf("re-answering a finalized item must be 409, got %d", code)
	}

	// Remaining items: answer correctly
	for i := 2; i <= planned; i++ {
		view := res["assessment"].(map[string]any)
		cur = view["current"].(map[string]any)
		if int(cur["position"].(float64)) != i {
			t.Fatalf("expected position %d, got %v", i, cur["position"])
		}
		res = parent.must(200, "POST", "/assessments/"+asmID+"/answers", map[string]any{"item_id": cur["id"], "response": correctResponse(e, cur["id"].(string)), "response_time_ms": 1500})
		if res["correct"] != true {
			t.Fatalf("item %d graded wrong: %v", i, res)
		}
		if i == 3 && res["mini_reward"] != true {
			t.Fatal("mini reward expected after challenge 3")
		}
	}
	if res["done"] != true {
		t.Fatalf("session should be done after %d items", planned)
	}

	// Complete: score, stars, XP, badges, skill mastery
	done := parent.must(200, "POST", "/assessments/"+asmID+"/complete", nil)
	result := done["result"].(map[string]any)
	// 7 first-try correct + 1 correct on retry = 7.5/8 = 94
	if result["score"].(float64) != 94 || result["stars"].(float64) != 3 || result["accuracy"].(float64) != 87.5 {
		t.Fatalf("unexpected result: %v", result)
	}
	unlocked := map[string]bool{}
	for _, u := range done["unlocks"].([]any) {
		unlocked[u.(map[string]any)["code"].(string)] = true
	}
	if !unlocked["first_challenge"] {
		t.Fatalf("first_challenge badge expected, got %v", done["unlocks"])
	}
	if len(done["skills"].([]any)) == 0 {
		t.Fatal("completion should report updated skills")
	}
	// Idempotent completion
	again2 := parent.must(200, "POST", "/assessments/"+asmID+"/complete", nil)
	if again2["already_completed"] != true || len(again2["unlocks"].([]any)) != 0 {
		t.Fatal("second complete must not award again")
	}

	// Parent progress (free tier: teaser only, skills locked)
	prog := parent.must(200, "GET", "/children/"+childID+"/progress", nil)
	if prog["overall"] == nil || prog["premium"] != false || prog["locked"].(map[string]any)["skills"] != true {
		t.Fatalf("free progress view unexpected: %v", prog)
	}
	hist := parent.must(200, "GET", "/children/"+childID+"/history", nil)
	if len(hist["history"].([]any)) != 1 {
		t.Fatal("history should contain the completed session")
	}
	parent.must(200, "GET", "/children/"+childID+"/recommendations", nil)
	recap := parent.must(200, "GET", "/children/"+childID+"/recap", nil)
	if recap["archetype"] == nil || recap["stats"] == nil {
		t.Fatal("recap must include archetype and stats")
	}

	// Free daily limit (2): one used; one more allowed, third blocked
	parent.must(201, "POST", "/assessments", map[string]any{"child_id": childID, "subject_id": subjectID(e, "bahasa-indonesia")})
	if code, out := parent.do("POST", "/assessments", map[string]any{"child_id": childID, "subject_id": subjectID(e, "matematika")}); code != 402 || out["error"] != "daily_limit" {
		t.Fatalf("free daily limit should block: %d %v", code, out)
	}

	// Subscribe: checkout never activates by itself
	chk := parent.must(201, "POST", "/subscription/checkout", map[string]any{"plan": "monthly"})
	ref := chk["checkout_ref"].(string)
	if sub := parent.must(200, "GET", "/subscription", nil)["subscription"].(map[string]any); sub["premium"] != false {
		t.Fatal("checkout alone must not activate")
	}

	// Webhook with a bad signature is rejected
	body, _ := json.Marshal(subscriptions.Event{ID: "evt_1", Type: "payment.succeeded", CheckoutRef: ref, Amount: 49000, Currency: "IDR"})
	anon := e.client()
	if code, _ := anon.do("POST", "/webhooks/payment", body, subscriptions.MockSignatureHeader, "deadbeef"); code != 401 {
		t.Fatalf("bad signature must be 401, got %d", code)
	}
	// Tampered amount is refused even with a valid signature
	bad, _ := json.Marshal(subscriptions.Event{ID: "evt_0", Type: "payment.succeeded", CheckoutRef: ref, Amount: 1, Currency: "IDR"})
	mock := subscriptions.MockProvider{Secret: []byte(e.cfg.PaymentSecret)}
	if code, _ := anon.do("POST", "/webhooks/payment", bad, subscriptions.MockSignatureHeader, mock.Sign(bad)); code != 500 {
		t.Fatalf("amount mismatch must fail (so provider retries/alerts), got %d", code)
	}
	// Valid, signed webhook activates; replay is a no-op
	code, out := anon.do("POST", "/webhooks/payment", body, subscriptions.MockSignatureHeader, mock.Sign(body))
	if code != 200 || out["result"] != "processed" {
		t.Fatalf("webhook: %d %v", code, out)
	}
	code, out = anon.do("POST", "/webhooks/payment", body, subscriptions.MockSignatureHeader, mock.Sign(body))
	if code != 200 || out["result"] != "duplicate" {
		t.Fatalf("replayed webhook must be idempotent: %d %v", code, out)
	}
	sub := parent.must(200, "GET", "/subscription", nil)["subscription"].(map[string]any)
	if sub["premium"] != true || sub["status"] != "active" {
		t.Fatalf("subscription should be active: %v", sub)
	}
	var periods int
	_ = e.pool.QueryRow(context.Background(), `SELECT count(*) FROM subscriptions WHERE current_period_end BETWEEN now() + interval '27 days' AND now() + interval '32 days'`).Scan(&periods)
	if periods != 1 {
		t.Fatal("duplicate webhook must not extend the period twice")
	}

	// Premium unlocks: full skills, more children, unlimited challenges
	prog = parent.must(200, "GET", "/children/"+childID+"/progress", nil)
	if prog["premium"] != true || prog["trend"] == nil {
		t.Fatalf("premium progress should include trend: %v", prog)
	}
	parent.must(201, "POST", "/children", map[string]any{"nickname": "Bima", "grade_id": gradeID(e, "tk-b")})
	parent.must(201, "POST", "/assessments", map[string]any{"child_id": childID, "subject_id": subjectID(e, "matematika")})

	// Missions: parent assigns a skill; the child plays a focused session; it counts.
	var skillID int
	_ = e.pool.QueryRow(context.Background(), `SELECT id FROM skills WHERE code = 'sd1-pengurangan-1-20'`).Scan(&skillID)
	ms := parent.must(200, "POST", "/children/"+childID+"/missions", map[string]any{"skill_id": skillID, "target": 1})["missions"].([]any)
	missionID := ms[0].(map[string]any)["id"].(string)
	if code, _ := parent.do("POST", "/children/"+childID+"/missions", map[string]any{"skill_id": skillID}); code != 409 {
		t.Fatal("duplicate active mission for the same skill must be refused")
	}
	home = parent.must(200, "GET", "/children/"+childID+"/home", nil)
	if len(home["missions"].([]any)) != 1 {
		t.Fatal("child home should list the active mission")
	}
	focused := parent.must(201, "POST", "/assessments", map[string]any{"child_id": childID, "mission_id": missionID})["assessment"].(map[string]any)
	if focused["focus_skill"] != "Pengurangan 1–20" {
		t.Fatalf("mission session must be skill-focused: %v", focused["focus_skill"])
	}
	fid := focused["id"].(string)
	res = map[string]any{"assessment": focused}
	for {
		cur := res["assessment"].(map[string]any)["current"]
		if cur == nil {
			break
		}
		item := cur.(map[string]any)
		var sk int
		_ = e.pool.QueryRow(context.Background(), `SELECT skill_id FROM assessment_items WHERE id = $1`, item["id"]).Scan(&sk)
		if sk != skillID {
			t.Fatal("focused session served a question from another skill")
		}
		res = parent.must(200, "POST", "/assessments/"+fid+"/answers", map[string]any{"item_id": item["id"], "response": correctResponse(e, item["id"].(string))})
		if res["done"] == true {
			break
		}
	}
	done = parent.must(200, "POST", "/assessments/"+fid+"/complete", nil)
	if m := done["mission"].(map[string]any); m["status"] != "completed" {
		t.Fatalf("mission should be completed: %v", m)
	}
	gotBadge := false
	for _, u := range done["unlocks"].([]any) {
		gotBadge = gotBadge || u.(map[string]any)["code"] == "mission_done"
	}
	if !gotBadge {
		t.Fatal("mission_done badge expected")
	}
	prog = parent.must(200, "GET", "/children/"+childID+"/progress", nil)
	if prog["streak"].(float64) < 1 || len(prog["chapters"].([]any)) == 0 {
		t.Fatalf("progress should include streak and chapters: %v %v", prog["streak"], prog["chapters"])
	}

	// Cancel keeps access until period end
	sub = parent.must(200, "POST", "/subscription/cancel", nil)["subscription"].(map[string]any)
	if sub["status"] != "cancelled" || sub["premium"] != true {
		t.Fatalf("cancelled sub keeps access until period end: %v", sub)
	}

	// Raw attempts are immutable in the database itself
	if _, err := e.pool.Exec(context.Background(), `UPDATE question_attempts SET is_correct = true`); err == nil {
		t.Fatal("question_attempts must reject updates")
	}
}

func TestCSRFAndAdminRBAC(t *testing.T) {
	e := setup(t)
	parent := e.client()
	parent.register("p@example.test")

	// Cross-site POST with the victim's cookie is refused
	code, _ := parent.do("POST", "/children", map[string]any{"nickname": "X", "grade_id": gradeID(e, "sd-1")}, "Origin", "https://evil.example")
	if code != 403 {
		t.Fatalf("foreign Origin must be 403, got %d", code)
	}
	// Parents cannot reach the CMS
	if code, _ := parent.do("GET", "/admin/questions", nil); code != 403 {
		t.Fatalf("parent on admin must be 403, got %d", code)
	}

	// Editor authors a question; reviewer approves and publishes; editor cannot publish.
	ctx := context.Background()
	mkStaff := func(email, role string) *client {
		c := e.client()
		c.register(email)
		if _, err := e.pool.Exec(ctx, `UPDATE users SET role = $2 WHERE email = $1`, email, role); err != nil {
			t.Fatal(err)
		}
		return c
	}
	editor := mkStaff("editor@example.test", "content_editor")
	reviewer := mkStaff("reviewer@example.test", "reviewer")
	var skillID int
	_ = e.pool.QueryRow(ctx, `SELECT id FROM skills WHERE code = 'sd1-penjumlahan-1-20'`).Scan(&skillID)

	q := editor.must(201, "POST", "/admin/questions", map[string]any{
		"skill_id": skillID, "type": "number_input", "difficulty": 1, "learning_objective": "Penjumlahan",
		"source": "Tim konten, rev 1", "content": map[string]any{"prompt": "2 + 2 = ?"}, "answer": map[string]any{"value": 4},
		"explanation": "2 + 2 = 4", "hint": "Hitung jari."})["question"].(map[string]any)
	qid := q["id"].(string)
	editor.must(200, "POST", "/admin/questions/"+qid+"/transition", map[string]any{"to": "review"})
	if code, _ := editor.do("POST", "/admin/questions/"+qid+"/transition", map[string]any{"to": "approved"}); code != 409 {
		t.Fatal("editor must not approve")
	}
	if code, _ := reviewer.do("POST", "/admin/questions/"+qid+"/transition", map[string]any{"to": "approved", "checklist": map[string]bool{"academic_answer": true}}); code != 409 {
		t.Fatal("approval without full checklist must fail")
	}
	checklist := map[string]bool{}
	for _, k := range []string{"academic_answer", "academic_explanation", "academic_curriculum", "academic_difficulty", "ux_instruction", "ux_readable", "ux_touch", "ux_visual", "a11y_color", "a11y_alternatives", "a11y_contrast"} {
		checklist[k] = true
	}
	reviewer.must(200, "POST", "/admin/questions/"+qid+"/transition", map[string]any{"to": "approved", "checklist": checklist})
	if code, _ := editor.do("POST", "/admin/questions/"+qid+"/transition", map[string]any{"to": "published"}); code != 403 && code != 409 {
		t.Fatalf("editor must not publish, got %d", code)
	}
	pub := reviewer.must(200, "POST", "/admin/questions/"+qid+"/transition", map[string]any{"to": "published"})["question"].(map[string]any)
	if pub["published_version_id"] == nil {
		t.Fatal("published question must have a live version")
	}

	// Editing a live question keeps serving v1 until v2 is published
	editor.must(200, "PUT", "/admin/questions/"+qid, map[string]any{
		"skill_id": skillID, "type": "number_input", "difficulty": 1, "learning_objective": "Penjumlahan",
		"source": "Tim konten, rev 2", "content": map[string]any{"prompt": "3 + 3 = ?"}, "answer": map[string]any{"value": 6},
		"explanation": "3 + 3 = 6", "hint": ""})
	got := reviewer.must(200, "GET", "/admin/questions/"+qid, nil)["question"].(map[string]any)
	if got["status"] != "draft" || got["published_version_id"] != pub["published_version_id"] {
		t.Fatalf("edit should create a draft v2 while v1 stays live: %v", got)
	}

	// Preview grades like the child app
	pv := editor.must(200, "POST", "/admin/questions/"+qid+"/preview", map[string]any{"response": map[string]any{"value": 6}})
	if pv["correct"] != true {
		t.Fatalf("preview grading: %v", pv)
	}

	// Asset governance: AI-generated imagery cannot be approved
	var licID int
	_ = e.pool.QueryRow(ctx, `SELECT id FROM asset_licenses WHERE name = 'DrawKit License'`).Scan(&licID)
	var assetID string
	_ = e.pool.QueryRow(ctx, `INSERT INTO assets (code, name, kind, source, creator, license_id, license_url, ai_generated)
		VALUES ('ILL-9001','x.svg','illustration','Generator','AI',$1,'https://www.drawkit.com/license',true) RETURNING id`, licID).Scan(&assetID)
	reviewer.must(200, "POST", "/admin/assets/"+assetID+"/transition", map[string]any{"to": "license_verified"})
	if code, _ := reviewer.do("POST", "/admin/assets/"+assetID+"/transition", map[string]any{"to": "design_approved"}); code != 409 {
		t.Fatal("AI-generated asset must not be design-approved")
	}

	// Tags are editable and filterable; a tag-only edit must not reset the status.
	tagged := reviewer.must(200, "PATCH", "/admin/assets/"+assetID, map[string]any{"tags": []string{"buah", "hitung"}})["asset"].(map[string]any)
	if tagged["status"] != "license_verified" {
		t.Fatalf("tag-only edit must not reset status, got %v", tagged["status"])
	}
	if got := tagged["tags"].([]any); len(got) != 2 || got[0] != "buah" {
		t.Fatalf("tags not stored: %v", tagged["tags"])
	}
	hit := reviewer.must(200, "GET", "/admin/assets?tag=buah", nil)["assets"].([]any)
	if len(hit) != 1 || hit[0].(map[string]any)["id"] != assetID {
		t.Fatalf("tag filter should return the tagged asset, got %v", hit)
	}
	miss := reviewer.must(200, "GET", "/admin/assets?tag=tidak-ada", nil)["assets"].([]any)
	if len(miss) != 0 {
		t.Fatalf("unknown tag must match nothing, got %v", miss)
	}
}

// applyStagingDDL creates the Python-owned staging schema by replaying its SQL
// migrations (the pipeline is a separate service and never runs inside these
// tests). Same relative path as the candidates package tests.
func applyStagingDDL(t *testing.T, e *env) {
	t.Helper()
	files, err := filepath.Glob("../../../../services/content-pipeline/sql/0*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("staging migrations not found: %v", err)
	}
	sort.Strings(files)
	for _, path := range files {
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.pool.Exec(context.Background(), string(sqlBytes)); err != nil {
			t.Fatalf("apply %s: %v", filepath.Base(path), err)
		}
	}
}

// seedCandidate inserts one source document/chunk and a question candidate,
// returning the candidate id. Mirrors what the Python pipeline would produce.
func seedCandidate(t *testing.T, e *env) int64 {
	t.Helper()
	ctx := context.Background()
	var sourceID int
	if err := e.pool.QueryRow(ctx, `INSERT INTO content_staging.source_registry
		(code, name, base_url, source_type, tier, enabled, status)
		VALUES ('test-src', 'Test Source', 'https://x.test', 'official_curriculum', 1, true, 'active')
		RETURNING id`).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	var runID int64
	if err := e.pool.QueryRow(ctx, `INSERT INTO content_staging.crawl_runs (source_id) VALUES ($1) RETURNING id`, sourceID).Scan(&runID); err != nil {
		t.Fatal(err)
	}
	var itemID int64
	if err := e.pool.QueryRow(ctx, `INSERT INTO content_staging.crawl_items
		(crawl_run_id, source_id, url, canonical_url, status) VALUES ($1, $2, 'https://x.test/a', 'https://x.test/a', 'fetched')
		RETURNING id`, runID, sourceID).Scan(&itemID); err != nil {
		t.Fatal(err)
	}
	var documentID int64
	if err := e.pool.QueryRow(ctx, `INSERT INTO content_staging.source_documents
		(source_id, crawl_item_id, title, canonical_url, document_type, content_hash, retrieved_at, license)
		VALUES ($1, $2, 'Dokumen Uji', 'https://x.test/a', 'html', 'doc-hash', '2026-09-30T00:00:00Z', 'public')
		RETURNING id`, sourceID, itemID).Scan(&documentID); err != nil {
		t.Fatal(err)
	}
	var chunkID int64
	if err := e.pool.QueryRow(ctx, `INSERT INTO content_staging.document_chunks
		(document_id, heading, sequence, text, hash) VALUES ($1, 'Bab', 0, 'isi chunk', 'chunk-hash')
		RETURNING id`, documentID).Scan(&chunkID); err != nil {
		t.Fatal(err)
	}
	var skillCode string
	if err := e.pool.QueryRow(ctx, `SELECT code FROM skills ORDER BY id LIMIT 1`).Scan(&skillCode); err != nil {
		t.Fatal(err)
	}
	payload := fmt.Sprintf(`{"code":"Q-E2E-1","skill":%q,"type":"short_answer","difficulty":3,`+
		`"learning_objective":"Peserta didik dapat X.","content":{"prompt":"Lengkapi: _____"},`+
		`"answer":{"accepted":["x"]},"explanation":"Jawaban: x."}`, skillCode)
	var candidateID int64
	if err := e.pool.QueryRow(ctx, `INSERT INTO content_staging.question_candidates
		(document_id, chunk_id, source_id, skill_id, phase_code, grade_code, subject_code,
		 template, type, difficulty, learning_objective, prompt, payload, content_hash, confidence, qa_issues, status)
		VALUES ($1, $2, $3, NULL, '', '', '', 'cloze', 'short_answer', 3, 'Peserta didik dapat X.',
		 'Lengkapi: _____', $4::jsonb, 'hash-e2e', 0.9, '[]'::jsonb, 'draft') RETURNING id`,
		documentID, chunkID, sourceID, payload).Scan(&candidateID); err != nil {
		t.Fatal(err)
	}
	return candidateID
}

func fullChecklist() map[string]bool {
	c := map[string]bool{}
	for _, k := range []string{"academic_answer", "academic_explanation", "academic_curriculum", "academic_difficulty", "ux_instruction", "ux_readable", "ux_touch", "ux_visual", "a11y_color", "a11y_alternatives", "a11y_contrast"} {
		c[k] = true
	}
	return c
}

// TestCandidateReviewRBAC exercises the M5 reviewer CMS: the queue degrades
// cleanly when the pipeline has not run, parents/editors cannot decide, and an
// approval with the full checklist queues a question for publish.
func TestCandidateReviewRBAC(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	mkStaff := func(email, role string) *client {
		c := e.client()
		c.register(email)
		if _, err := e.pool.Exec(ctx, `UPDATE users SET role = $2 WHERE email = $1`, email, role); err != nil {
			t.Fatal(err)
		}
		return c
	}
	reviewer := mkStaff("reviewer@example.test", "reviewer")
	editor := mkStaff("editor@example.test", "content_editor")

	// Without the pipeline schema the queue is empty, not a 500.
	if _, err := e.pool.Exec(ctx, `DROP SCHEMA IF EXISTS content_staging CASCADE`); err != nil {
		t.Fatal(err)
	}
	empty := reviewer.must(200, "GET", "/admin/candidates", nil)
	if empty["available"] != false {
		t.Fatalf("queue without pipeline should report available:false, got %v", empty)
	}

	applyStagingDDL(t, e)
	candidateID := seedCandidate(t, e)
	path := fmt.Sprintf("/admin/candidates/question/%d", candidateID)
	decision := path + "/decision"

	// Parents never reach the CMS.
	parent := e.client()
	parent.register("p@example.test")
	if code, _ := parent.do("GET", "/admin/candidates", nil); code != 403 {
		t.Fatalf("parent on review queue must be 403, got %d", code)
	}

	// Editors can read the queue but not decide.
	list := editor.must(200, "GET", "/admin/candidates", nil)
	if list["available"] != true || len(list["candidates"].([]any)) != 1 {
		t.Fatalf("queue unexpected: %v", list)
	}
	if code, _ := editor.do("POST", decision, map[string]any{"decision": "approved", "checklist": fullChecklist()}); code != 403 {
		t.Fatalf("editor must not decide, got %d", code)
	}

	// The reviewer sees source lineage + mapping in one screen.
	detail := reviewer.must(200, "GET", path, nil)["candidate"].(map[string]any)
	if detail["kind"] != "question" || detail["source"].(map[string]any)["code"] != "test-src" {
		t.Fatalf("detail missing source: %v", detail)
	}
	if detail["lineage"].(map[string]any)["document_title"] != "Dokumen Uji" {
		t.Fatalf("detail missing document: %v", detail["lineage"])
	}

	// Approving without the full checklist is refused.
	if code, _ := reviewer.do("POST", decision, map[string]any{"decision": "approved", "checklist": map[string]bool{"academic_answer": true}}); code != 422 {
		t.Fatalf("incomplete checklist must be 422, got %d", code)
	}
	// With the full checklist the question is queued for publish.
	pub := reviewer.must(200, "POST", decision, map[string]any{"decision": "approved", "checklist": fullChecklist(), "notes": "ok"})["publication"].(map[string]any)
	if pub["status"] != "queued" || pub["decision"] != "approved" {
		t.Fatalf("publication unexpected: %v", pub)
	}
	row := reviewer.must(200, "GET", "/admin/candidates", nil)["candidates"].([]any)[0].(map[string]any)
	if row["publication_status"] != "queued" {
		t.Fatalf("queue should reflect the decision: %v", row)
	}
	// Once published, the decision is frozen.
	if _, err := e.pool.Exec(ctx, `UPDATE candidate_publications SET status = 'published' WHERE kind = 'question' AND candidate_id = $1`, candidateID); err != nil {
		t.Fatal(err)
	}
	if code, _ := reviewer.do("POST", decision, map[string]any{"decision": "rejected"}); code != 409 {
		t.Fatalf("deciding after publish must be 409, got %d", code)
	}
}

// seedEvidence inserts one finalized, correct assessment item for childID on a
// published question, rebuilds that question's attribution rollup, and returns
// the question id.
func seedEvidence(t *testing.T, e *env, childID string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	child, err := uuid.Parse(childID)
	if err != nil {
		t.Fatal(err)
	}
	var qid, versionID uuid.UUID
	var skillID, subjectID, childGrade int
	if err := e.pool.QueryRow(ctx, `SELECT id, skill_id FROM questions WHERE status = 'published' ORDER BY code LIMIT 1`).
		Scan(&qid, &skillID); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(ctx, `SELECT id FROM subjects ORDER BY id LIMIT 1`).Scan(&subjectID); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(ctx, `SELECT COALESCE(published_version_id, current_version_id) FROM questions WHERE id = $1`, qid).
		Scan(&versionID); err != nil {
		t.Fatal(err)
	}
	if err := e.pool.QueryRow(ctx, `SELECT grade_id FROM children WHERE id = $1`, child).Scan(&childGrade); err != nil {
		t.Fatal(err)
	}
	var assessmentID uuid.UUID
	if err := e.pool.QueryRow(ctx, `INSERT INTO assessments (child_id, subject_id, grade_id, status, planned_count)
		VALUES ($1, $2, $3, 'in_progress', 1) RETURNING id`, child, subjectID, childGrade).Scan(&assessmentID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO assessment_items
		(assessment_id, position, question_id, question_version_id, skill_id, difficulty, attempts,
		 final_correct, first_try_correct, finalized_at)
		VALUES ($1, 1, $2, $3, $4, 1, 1, true, true, now())`, assessmentID, qid, versionID, skillID); err != nil {
		t.Fatal(err)
	}
	if err := attribution.RecomputeQuestions(ctx, e.pool, []uuid.UUID{qid}); err != nil {
		t.Fatal(err)
	}
	return qid
}

func rollupRows(t *testing.T, e *env, qid uuid.UUID) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM question_attribution WHERE question_id = $1`, qid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestErasureRefreshesAttribution: erasing a child, or the whole account,
// cascades assessment evidence away. The derived attribution rollup must not
// keep the vanished evidence (§39).
func TestErasureRefreshesAttribution(t *testing.T) {
	e := setup(t)
	parent := e.client()
	parent.register("erase@test")

	// Child erasure.
	childID := parent.must(201, "POST", "/children", map[string]any{"nickname": "Hapus", "grade_id": gradeID(e, "sd-1")})["child"].(map[string]any)["id"].(string)
	qid := seedEvidence(t, e, childID)
	if n := rollupRows(t, e, qid); n != 1 {
		t.Fatalf("pre-erasure rollup rows = %d, want 1", n)
	}
	parent.must(204, "DELETE", "/children/"+childID, nil)
	if n := rollupRows(t, e, qid); n != 0 {
		t.Fatalf("child erasure left %d stale attribution rows", n)
	}

	// Whole-account erasure.
	child2 := parent.must(201, "POST", "/children", map[string]any{"nickname": "Hapus2", "grade_id": gradeID(e, "sd-1")})["child"].(map[string]any)["id"].(string)
	qid2 := seedEvidence(t, e, child2)
	if n := rollupRows(t, e, qid2); n != 1 {
		t.Fatalf("pre-erasure rollup rows = %d, want 1", n)
	}
	parent.must(204, "DELETE", "/me", map[string]any{"password": testPassword})
	if n := rollupRows(t, e, qid2); n != 0 {
		t.Fatalf("account erasure left %d stale attribution rows", n)
	}
}

// TestErasureSurvivesAttributionFailure: erasure must honor the data-subject
// right (UU PDP) even when the derived rollup cannot be refreshed. The recompute
// runs in a savepoint, so a rollup failure rolls back only the rollup — the
// child/account delete still commits. We force the failure by dropping the
// rollup table after seeding evidence.
func TestErasureSurvivesAttributionFailure(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	// Child-erasure path.
	parentA := e.client()
	parentA.register("erasefail-a@test")
	childA := parentA.must(201, "POST", "/children", map[string]any{"nickname": "Hapus1", "grade_id": gradeID(e, "sd-1")})["child"].(map[string]any)["id"].(string)
	seedEvidence(t, e, childA)

	// Account-erasure path (separate parent: the free plan allows one child each).
	parentB := e.client()
	parentB.register("erasefail-b@test")
	childB := parentB.must(201, "POST", "/children", map[string]any{"nickname": "Hapus2", "grade_id": gradeID(e, "sd-1")})["child"].(map[string]any)["id"].(string)
	seedEvidence(t, e, childB)

	// Make every rollup recompute fail inside the savepoint.
	if _, err := e.pool.Exec(ctx, `DROP TABLE question_attribution`); err != nil {
		t.Fatal(err)
	}

	parentA.must(204, "DELETE", "/children/"+childA, nil)
	var n int
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM children WHERE id = $1`, childA).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("child not erased after rollup failure: %d rows", n)
	}

	parentB.must(204, "DELETE", "/me", map[string]any{"password": testPassword})
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE email = $1`, "erasefail-b@test").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("account not erased after rollup failure: %d rows", n)
	}
}

// TestCompletionSurvivesAttributionFailure: the attribution rollup is
// best-effort. Its recompute runs in a savepoint, so a DB error must roll back
// only the rollup — not abort the session-complete transaction. We force the
// failure by dropping the rollup table mid-session and assert the child still
// completes (and daily progress is committed).
func TestCompletionSurvivesAttributionFailure(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	parent := e.client()
	parent.register("savepoint@test")
	childID := parent.must(201, "POST", "/children", map[string]any{"nickname": "Adi", "grade_id": gradeID(e, "sd-1")})["child"].(map[string]any)["id"].(string)

	asm := parent.must(201, "POST", "/assessments", map[string]any{"child_id": childID, "subject_id": subjectID(e, "matematika")})["assessment"].(map[string]any)
	asmID := asm["id"].(string)
	cur := asm["current"].(map[string]any)
	parent.must(200, "POST", "/assessments/"+asmID+"/answers",
		map[string]any{"item_id": cur["id"], "response": correctResponse(e, cur["id"].(string)), "response_time_ms": 1000})

	// Make the rollup recompute fail inside the savepoint.
	if _, err := e.pool.Exec(ctx, `DROP TABLE question_attribution`); err != nil {
		t.Fatal(err)
	}

	if code, out := parent.do("POST", "/assessments/"+asmID+"/complete", nil); code != 200 {
		t.Fatalf("session completion must survive a rollup failure, got %d: %v", code, out)
	}
	var n int
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM daily_progress WHERE child_id = $1`, childID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("daily progress not committed after savepoint rollback: %d rows", n)
	}
}

// TestContentStudioRBAC covers the M9 read endpoints: anonymous is rejected,
// parents never reach the CMS, editors can browse, the view degrades cleanly
// when the pipeline has not run, and an unknown skill is a clean 404.
func TestContentStudioRBAC(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	mkStaff := func(email, role string) *client {
		c := e.client()
		c.register(email)
		if _, err := e.pool.Exec(ctx, `UPDATE users SET role = $2 WHERE email = $1`, email, role); err != nil {
			t.Fatal(err)
		}
		return c
	}
	editor := mkStaff("studio-editor@example.test", "content_editor")

	var skillID int
	if err := e.pool.QueryRow(ctx, `SELECT id FROM skills ORDER BY id LIMIT 1`).Scan(&skillID); err != nil {
		t.Fatal(err)
	}
	lessonPath := fmt.Sprintf("/admin/content-studio/lessons/%d", skillID)

	if code, _ := e.client().do("GET", "/admin/content-studio/tree", nil); code != 401 {
		t.Fatalf("anonymous tree must be 401, got %d", code)
	}

	parent := e.client()
	parent.register("studio-parent@example.test")
	if code, _ := parent.do("GET", "/admin/content-studio/tree", nil); code != 403 {
		t.Fatalf("parent tree must be 403, got %d", code)
	}
	if code, _ := parent.do("GET", lessonPath, nil); code != 403 {
		t.Fatalf("parent lesson must be 403, got %d", code)
	}

	// Without the pipeline schema both views degrade to available:false, not 500.
	if _, err := e.pool.Exec(ctx, `DROP SCHEMA IF EXISTS content_staging CASCADE`); err != nil {
		t.Fatal(err)
	}
	if tree := editor.must(200, "GET", "/admin/content-studio/tree", nil); tree["available"] != false {
		t.Fatalf("tree without pipeline should be available:false, got %v", tree["available"])
	}
	lesson := editor.must(200, "GET", lessonPath, nil)
	if lesson["available"] != false {
		t.Fatalf("lesson without pipeline should be available:false, got %v", lesson["available"])
	}
	if _, ok := lesson["published"].([]any); !ok {
		t.Fatalf("lesson must still return a published array, got %v", lesson["published"])
	}

	if code, _ := editor.do("GET", "/admin/content-studio/lessons/999999", nil); code != 404 {
		t.Fatalf("unknown skill must be 404, got %d", code)
	}
}

// ensurePipelineStaging creates the Python-owned staging schema the Go control
// API reads, reusing the pipeline's own migrations so the DDL cannot drift.
func ensurePipelineStaging(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	for _, name := range []string{"0001_content_staging.sql", "0008_pipeline_jobs.sql", "0009_worker_heartbeats.sql"} {
		ddl, err := os.ReadFile("../../../../services/content-pipeline/sql/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(ddl)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

// TestPipelineControl covers the M11 UI control surface: RBAC, graceful
// degradation without the pipeline schema, and the enqueue/list/cancel/retry
// round-trip against a real queue table.
func TestPipelineControl(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `DROP SCHEMA IF EXISTS content_staging CASCADE`)
	})

	mkStaff := func(email, role string) *client {
		c := e.client()
		c.register(email)
		if _, err := e.pool.Exec(ctx, `UPDATE users SET role = $2 WHERE email = $1`, email, role); err != nil {
			t.Fatal(err)
		}
		return c
	}
	editor := mkStaff("pipe-editor@example.test", "content_editor")
	reviewer := mkStaff("pipe-reviewer@example.test", "reviewer")

	// RBAC: anonymous and parents are rejected; publish is reviewer-only.
	if code, _ := e.client().do("GET", "/admin/pipeline/sources", nil); code != 401 {
		t.Fatalf("anonymous sources must be 401, got %d", code)
	}
	parent := e.client()
	parent.register("pipe-parent@example.test")
	if code, _ := parent.do("GET", "/admin/pipeline/sources", nil); code != 403 {
		t.Fatalf("parent sources must be 403, got %d", code)
	}
	if code, _ := parent.do("POST", "/admin/pipeline/jobs", map[string]any{"kind": "run_pipeline"}); code != 403 {
		t.Fatalf("parent enqueue must be 403, got %d", code)
	}
	if code, _ := editor.do("POST", "/admin/pipeline/publish", map[string]any{"all": true}); code != 403 {
		t.Fatalf("editor publish must be 403, got %d", code)
	}

	// Degrade: without the queue table the GETs are available:false and enqueue 409.
	if _, err := e.pool.Exec(ctx, `DROP SCHEMA IF EXISTS content_staging CASCADE`); err != nil {
		t.Fatal(err)
	}
	if out := editor.must(200, "GET", "/admin/pipeline/sources", nil); out["available"] != false {
		t.Fatalf("sources without pipeline should be available:false, got %v", out["available"])
	}
	if out := editor.must(200, "GET", "/admin/pipeline/jobs", nil); out["available"] != false {
		t.Fatalf("jobs without pipeline should be available:false, got %v", out["available"])
	}
	if out := editor.must(200, "GET", "/admin/pipeline/worker-status", nil); out["available"] != false {
		t.Fatalf("worker-status without pipeline should be available:false, got %v", out["available"])
	}
	if code, _ := editor.do("POST", "/admin/pipeline/jobs", map[string]any{"kind": "run_pipeline"}); code != 409 {
		t.Fatalf("enqueue without pipeline must be 409, got %d", code)
	}

	// Happy path.
	ensurePipelineStaging(t, e.pool)
	if out := editor.must(200, "GET", "/admin/pipeline/sources", nil); out["available"] != true {
		t.Fatalf("sources should be available:true, got %v", out["available"])
	}
	if out := editor.must(200, "GET", "/admin/pipeline/worker-status", nil); out["available"] != true {
		t.Fatalf("worker-status should be available:true, got %v", out["available"])
	}

	if _, err := e.pool.Exec(ctx, `INSERT INTO content_staging.worker_heartbeats
		(worker_id, hostname, pid, status, last_seen_at)
		VALUES ('test-w-1', 'host-1', 1234, 'idle', now())`); err != nil {
		t.Fatal(err)
	}
	if ws := editor.must(200, "GET", "/admin/pipeline/worker-status", nil); ws["available"] != true {
		t.Fatalf("worker-status with heartbeat should be available:true, got %v", ws)
	} else if w, ok := ws["worker"].(map[string]any); !ok || w["status"] != "idle" {
		t.Fatalf("expected worker status idle, got %v", ws["worker"])
	}

	created := editor.must(201, "POST", "/admin/pipeline/jobs", map[string]any{
		"kind": "run_pipeline", "params": map[string]any{"sources": []string{"kemendikdasmen-kurikulum"}},
	})
	job, _ := created["job"].(map[string]any)
	jobID, _ := job["id"].(float64)
	if job["status"] != "queued" {
		t.Fatalf("new job should be queued, got %v", job["status"])
	}
	list := editor.must(200, "GET", "/admin/pipeline/jobs", nil)
	if jobs, _ := list["jobs"].([]any); len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	editor.must(200, "GET", fmt.Sprintf("/admin/pipeline/jobs/%d", int64(jobID)), nil)

	if code, _ := editor.do("POST", "/admin/pipeline/jobs", map[string]any{"kind": "bogus"}); code != 422 {
		t.Fatalf("invalid kind must be 422, got %d", code)
	}

	cancelled := editor.must(200, "POST", fmt.Sprintf("/admin/pipeline/jobs/%d/cancel", int64(jobID)), nil)
	if cj, _ := cancelled["job"].(map[string]any); cj["status"] != "cancelled" {
		t.Fatalf("job should be cancelled, got %v", cj["status"])
	}
	retried := editor.must(200, "POST", fmt.Sprintf("/admin/pipeline/jobs/%d/retry", int64(jobID)), nil)
	if rj, _ := retried["job"].(map[string]any); rj["status"] != "queued" {
		t.Fatalf("job should be queued after retry, got %v", rj["status"])
	}

	// Source governance change is expressed as a job, so Go never writes source_registry.
	if _, err := e.pool.Exec(ctx, `INSERT INTO content_staging.source_registry
		(code, name, base_url, source_type, tier) VALUES ('test-src','Test','https://x.test','official_curriculum',1)`); err != nil {
		t.Fatal(err)
	}
	src := editor.must(202, "POST", "/admin/pipeline/sources/test-src/status",
		map[string]any{"enabled": true, "crawl_frequency": "daily"})
	if sj, _ := src["job"].(map[string]any); sj["kind"] != "set_source_status" {
		t.Fatalf("expected a set_source_status job, got %v", sj["kind"])
	}
	if code, _ := editor.do("POST", "/admin/pipeline/sources/test-src/status", map[string]any{"crawl_frequency": "hourly"}); code != 422 {
		t.Fatalf("invalid crawl_frequency must be 422, got %d", code)
	}

	// Publish: reviewer-only; an empty queue yields a zeroed report.
	reviewer.must(200, "POST", "/admin/pipeline/publish", map[string]any{"all": true})
}
