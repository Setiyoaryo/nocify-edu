package assessment

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/nocify/eduplay/api/internal/analytics"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/children"
	"github.com/nocify/eduplay/api/internal/creators"
	"github.com/nocify/eduplay/api/internal/db"
	"github.com/nocify/eduplay/api/internal/httpx"
	"github.com/nocify/eduplay/api/internal/progress"
	"github.com/nocify/eduplay/api/internal/questions"
	"github.com/nocify/eduplay/api/internal/rewards"
	"github.com/nocify/eduplay/api/internal/subscriptions"
)

var (
	sessionsStarted   = promauto.NewCounter(prometheus.CounterOpts{Name: "eduplay_assessments_started_total", Help: "Challenge sessions started."})
	sessionsCompleted = promauto.NewCounter(prometheus.CounterOpts{Name: "eduplay_assessments_completed_total", Help: "Challenge sessions completed."})
	questionsAnswered = promauto.NewCounterVec(prometheus.CounterOpts{Name: "eduplay_question_attempts_total", Help: "Question attempts by correctness."}, []string{"correct"})
)

// MaxAttempts per item: wrong → hint → one retry → explanation (§10).
const MaxAttempts = 2

type Handler struct {
	DB     *pgxpool.Pool
	Policy subscriptions.Policy
	TZ     *time.Location
}

// ---------------------------------------------------------------- views

type ItemView struct {
	ID        uuid.UUID       `json:"id"`
	Position  int             `json:"position"`
	Type      string          `json:"type"`
	Content   json.RawMessage `json:"content"`
	Attempts  int             `json:"attempts"`
	HintShown bool            `json:"hint_shown"`
	Hint      string          `json:"hint,omitempty"`
	SkillName string          `json:"skill_name"`
}

type View struct {
	ID           uuid.UUID  `json:"id"`
	ChildID      uuid.UUID  `json:"child_id"`
	SubjectID    int        `json:"subject_id"`
	SubjectName  string     `json:"subject_name"`
	UXBand       string     `json:"ux_band"`
	FocusSkill   *string    `json:"focus_skill"` // set for skill-focused practice
	Status       string     `json:"status"`
	PlannedCount int        `json:"planned_count"`
	Answered     int        `json:"answered"`
	Results      []bool     `json:"results"` // final correctness per finalized position
	Current      *ItemView  `json:"current"`
	Result       *Score     `json:"result,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// loadOwned fetches an assessment the signed-in parent may act on.
func (h Handler) loadOwned(ctx context.Context, q db.DBTX, parentID, id uuid.UUID, lock bool) (View, error) {
	var v View
	sql := `SELECT a.id, a.child_id, a.subject_id, s.name, g.ux_band, a.status, a.planned_count, a.started_at, a.completed_at,
	               a.score, a.accuracy::float8, a.stars, a.xp_earned, fk.name
		FROM assessments a
		LEFT JOIN skills fk ON fk.id = a.skill_id
		JOIN children c ON c.id = a.child_id
		JOIN subjects s ON s.id = a.subject_id
		JOIN grades g ON g.id = a.grade_id
		WHERE a.id = $1 AND c.parent_id = $2`
	if lock {
		sql += ` FOR UPDATE OF a`
	}
	var score, stars, xp *int
	var acc *float64
	err := q.QueryRow(ctx, sql, id, parentID).Scan(&v.ID, &v.ChildID, &v.SubjectID, &v.SubjectName, &v.UXBand, &v.Status,
		&v.PlannedCount, &v.StartedAt, &v.CompletedAt, &score, &acc, &stars, &xp, &v.FocusSkill)
	if err != nil {
		return v, err
	}
	if score != nil {
		v.Result = &Score{Score: *score, Accuracy: *acc, Stars: *stars, XP: *xp}
	}
	return v, nil
}

func (h Handler) fillItems(ctx context.Context, q db.DBTX, v *View) error {
	rows, err := q.Query(ctx, `
		SELECT i.id, i.position, qq.type, qv.content, i.attempts, i.hint_shown, qv.hint, k.name, i.final_correct
		FROM assessment_items i
		JOIN questions qq ON qq.id = i.question_id
		JOIN question_versions qv ON qv.id = i.question_version_id
		JOIN skills k ON k.id = i.skill_id
		WHERE i.assessment_id = $1 ORDER BY i.position`, v.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	v.Results = []bool{}
	for rows.Next() {
		var it ItemView
		var hint string
		var final *bool
		if err := rows.Scan(&it.ID, &it.Position, &it.Type, &it.Content, &it.Attempts, &it.HintShown, &hint, &it.SkillName, &final); err != nil {
			return err
		}
		if final != nil {
			v.Results = append(v.Results, *final)
			continue
		}
		if it.HintShown {
			it.Hint = hintOrDefault(hint)
		}
		it.Content, err = questions.PublicContent(it.Type, it.Content, it.ID.String())
		if err != nil {
			return err
		}
		v.Current = &it
	}
	v.Answered = len(v.Results)
	return rows.Err()
}

func hintOrDefault(h string) string {
	if h == "" {
		return "Baca soalnya pelan-pelan, lalu coba lagi."
	}
	return h
}

// ---------------------------------------------------------------- selection

type poolRow struct {
	Candidate
	VersionID uuid.UUID
}

// scope is what a session draws from: a whole subject, or one skill for
// focused practice (missions, "latih skill ini").
type scope struct {
	GradeID, SubjectID, SkillID int // SkillID 0 = whole subject
}

func (h Handler) loadPool(ctx context.Context, q db.DBTX, childID uuid.UUID, sc scope) ([]poolRow, []SkillState, error) {
	rows, err := q.Query(ctx, `
		SELECT q.id, q.skill_id, q.difficulty, q.published_version_id,
		       q.id IN (SELECT question_id FROM question_attempts WHERE child_id = $1 ORDER BY created_at DESC LIMIT 60)
		FROM questions q
		JOIN skills k ON k.id = q.skill_id
		JOIN chapters c ON c.id = k.chapter_id
		JOIN curriculum_versions cv ON cv.id = c.curriculum_version_id AND cv.is_active
		WHERE q.published_version_id IS NOT NULL AND c.grade_id = $2 AND c.subject_id = $3 AND ($4 = 0 OR q.skill_id = $4)`,
		childID, sc.GradeID, sc.SubjectID, sc.SkillID)
	if err != nil {
		return nil, nil, err
	}
	var pool []poolRow
	skillSet := map[int]bool{}
	for rows.Next() {
		var p poolRow
		var qid uuid.UUID
		if err := rows.Scan(&qid, &p.SkillID, &p.Difficulty, &p.VersionID, &p.RecentSeen); err != nil {
			rows.Close()
			return nil, nil, err
		}
		p.QuestionID = qid.String()
		pool = append(pool, p)
		skillSet[p.SkillID] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	ids := make([]int, 0, len(skillSet))
	for id := range skillSet {
		ids = append(ids, id)
	}
	mastery := map[int]float64{}
	srows, err := q.Query(ctx, `SELECT skill_id, mastery::float8 FROM skill_progress WHERE child_id = $1 AND skill_id = ANY($2)`, childID, ids)
	if err != nil {
		return nil, nil, err
	}
	for srows.Next() {
		var id int
		var m float64
		if err := srows.Scan(&id, &m); err != nil {
			srows.Close()
			return nil, nil, err
		}
		mastery[id] = m
	}
	srows.Close()
	states := make([]SkillState, 0, len(ids))
	for _, id := range ids {
		m, ok := mastery[id]
		states = append(states, SkillState{SkillID: id, Mastery: m, HasEvidence: ok})
	}
	return pool, states, srows.Err()
}

func baseDifficulty(s SkillState) int { return progress.TargetDifficulty(s.Mastery, s.HasEvidence) }

// addNextItem picks and inserts the item for the next position; false when the pool is exhausted.
func (h Handler) addNextItem(ctx context.Context, tx pgx.Tx, assessmentID, childID uuid.UUID, sc scope) (bool, error) {
	pool, states, err := h.loadPool(ctx, tx, childID, sc)
	if err != nil {
		return false, err
	}
	rows, err := tx.Query(ctx, `SELECT question_id, skill_id, difficulty, COALESCE(first_try_correct, false)
		FROM assessment_items WHERE assessment_id = $1 ORDER BY position`, assessmentID)
	if err != nil {
		return false, err
	}
	used := map[string]bool{}
	var history []Outcome
	for rows.Next() {
		var qid uuid.UUID
		var o Outcome
		if err := rows.Scan(&qid, &o.SkillID, &o.Difficulty, &o.FirstTryCorrect); err != nil {
			rows.Close()
			return false, err
		}
		used[qid.String()] = true
		history = append(history, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, err
	}

	skillID, diff := NextTarget(len(history), states, history, baseDifficulty)
	cands := make([]Candidate, len(pool))
	for i, p := range pool {
		cands[i] = p.Candidate
	}
	pick := Pick(cands, used, skillID, diff, rand.New(rand.NewSource(time.Now().UnixNano())))
	if pick == "" {
		return false, nil
	}
	for _, p := range pool {
		if p.QuestionID == pick {
			_, err := tx.Exec(ctx, `INSERT INTO assessment_items (assessment_id, position, question_id, question_version_id, skill_id, difficulty)
				VALUES ($1, $2, $3, $4, $5, $6)`, assessmentID, len(history)+1, pick, p.VersionID, p.SkillID, p.Difficulty)
			return err == nil, err
		}
	}
	return false, nil
}

// ---------------------------------------------------------------- handlers

// MinFocusedPool is the fewest live questions a skill needs before it can be
// practised on its own; below that the session falls back to the whole subject.
const MinFocusedPool = 3

func (h Handler) Start(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ChildID   uuid.UUID  `json:"child_id"`
		SubjectID int        `json:"subject_id"`
		SkillID   int        `json:"skill_id"`
		MissionID *uuid.UUID `json:"mission_id"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	ctx := r.Context()
	u := auth.MustUser(ctx)
	child, err := children.LoadOwned(ctx, h.DB, u.ID, in.ChildID)
	if errors.Is(err, children.ErrNotFound) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}

	// A mission fixes the skill. Missions belong to the child, so ownership is already proven.
	if in.MissionID != nil {
		err := h.DB.QueryRow(ctx, `SELECT skill_id FROM child_missions WHERE id = $1 AND child_id = $2 AND status = 'active'`,
			*in.MissionID, child.ID).Scan(&in.SkillID)
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(w, http.StatusConflict, "mission_closed", "Misi ini sudah selesai.")
			return
		}
		if err != nil {
			httpx.Internal(w, r, err)
			return
		}
	}
	sc := scope{GradeID: child.GradeID, SubjectID: in.SubjectID, SkillID: in.SkillID}
	if in.SkillID != 0 {
		// Resolve the subject from the skill and make sure it belongs to the child's grade.
		err := h.DB.QueryRow(ctx, `SELECT c.subject_id FROM skills k JOIN chapters c ON c.id = k.chapter_id
			WHERE k.id = $1 AND c.grade_id = $2`, in.SkillID, child.GradeID).Scan(&sc.SubjectID)
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Invalid(w, map[string]string{"skill_id": "Skill tidak tersedia untuk kelas ini."})
			return
		}
		if err != nil {
			httpx.Internal(w, r, err)
			return
		}
	}

	// Resume an unfinished session of the same scope instead of burning quota.
	var existing uuid.UUID
	err = h.DB.QueryRow(ctx, `SELECT id FROM assessments WHERE child_id = $1 AND subject_id = $2 AND COALESCE(skill_id, 0) = $3
		AND status = 'in_progress' AND started_at > now() - interval '3 hours' ORDER BY started_at DESC LIMIT 1`,
		child.ID, sc.SubjectID, sc.SkillID).Scan(&existing)
	if err == nil {
		h.respond(w, r, u.ID, existing, http.StatusOK)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		httpx.Internal(w, r, err)
		return
	}

	ent, err := h.Policy.Get(ctx, h.DB, u.ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if ent.Limits.DailyChallenges > 0 {
		var today int
		if err := h.DB.QueryRow(ctx, `SELECT count(*) FROM assessments a JOIN children c ON c.id = a.child_id
			WHERE c.parent_id = $1 AND a.started_at >= $2`, u.ID, subscriptions.StartOfDay(h.TZ)).Scan(&today); err != nil {
			httpx.Internal(w, r, err)
			return
		}
		if today >= ent.Limits.DailyChallenges {
			httpx.Error(w, http.StatusPaymentRequired, "daily_limit", "Challenge gratis hari ini sudah habis. Sampai jumpa besok!")
			return
		}
	}

	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(ctx)

	var sessionSize int
	if err := tx.QueryRow(ctx, `SELECT session_size FROM grades WHERE id = $1`, child.GradeID).Scan(&sessionSize); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	countPool := func(sc scope) (int, error) {
		var n int
		err := tx.QueryRow(ctx, `SELECT count(*) FROM questions q JOIN skills k ON k.id = q.skill_id JOIN chapters c ON c.id = k.chapter_id
			JOIN curriculum_versions cv ON cv.id = c.curriculum_version_id AND cv.is_active
			WHERE q.published_version_id IS NOT NULL AND c.grade_id = $1 AND c.subject_id = $2 AND ($3 = 0 OR q.skill_id = $3)`,
			sc.GradeID, sc.SubjectID, sc.SkillID).Scan(&n)
		return n, err
	}
	poolSize, err := countPool(sc)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if sc.SkillID != 0 && poolSize < MinFocusedPool && in.MissionID == nil {
		sc.SkillID = 0
		if poolSize, err = countPool(sc); err != nil {
			httpx.Internal(w, r, err)
			return
		}
	}
	if poolSize == 0 {
		httpx.Error(w, http.StatusConflict, "no_content", "Belum ada challenge untuk pelajaran ini.")
		return
	}
	planned := min(sessionSize, poolSize)

	if _, err := tx.Exec(ctx, `UPDATE assessments SET status = 'abandoned' WHERE child_id = $1 AND status = 'in_progress'`, child.ID); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var skillArg any
	if sc.SkillID != 0 {
		skillArg = sc.SkillID
	}
	var id uuid.UUID
	if err := tx.QueryRow(ctx, `INSERT INTO assessments (child_id, subject_id, grade_id, planned_count, skill_id, mission_id)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		child.ID, sc.SubjectID, child.GradeID, planned, skillArg, in.MissionID).Scan(&id); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if _, err := h.addNextItem(ctx, tx, id, child.ID, sc); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	sessionsStarted.Inc()
	analytics.Track(ctx, h.DB, "challenge_started", u.ID, child.ID, map[string]any{"subject_id": sc.SubjectID, "skill_id": sc.SkillID})
	h.respond(w, r, u.ID, id, http.StatusCreated)
}

func (h Handler) respond(w http.ResponseWriter, r *http.Request, parentID, id uuid.UUID, status int) {
	v, err := h.loadOwned(r.Context(), h.DB, parentID, id, false)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if err := h.fillItems(r.Context(), h.DB, &v); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, status, map[string]any{"assessment": v})
}

func (h Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "assessmentID")
	if !ok {
		return
	}
	h.respond(w, r, auth.MustUser(r.Context()).ID, id, http.StatusOK)
}

func (h Handler) Hint(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "assessmentID")
	if !ok {
		return
	}
	itemID, ok := httpx.UUIDParam(w, r, "itemID")
	if !ok {
		return
	}
	var hint string
	err := h.DB.QueryRow(r.Context(), `
		UPDATE assessment_items i SET hint_shown = true
		FROM assessments a, children c, question_versions qv
		WHERE i.id = $1 AND i.assessment_id = $2 AND a.id = i.assessment_id AND c.id = a.child_id AND c.parent_id = $3
		  AND a.status = 'in_progress' AND i.finalized_at IS NULL AND qv.id = i.question_version_id
		RETURNING qv.hint`, itemID, id, auth.MustUser(r.Context()).ID).Scan(&hint)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"hint": hintOrDefault(hint)})
}

var praise = []string{"Hebat! Jawabanmu benar.", "Keren! Kamu berhasil.", "Tepat sekali! Terus semangat."}

type AnswerResult struct {
	Correct       bool            `json:"correct"`
	Final         bool            `json:"final"` // item finished (correct, or out of attempts)
	Feedback      string          `json:"feedback"`
	Hint          string          `json:"hint,omitempty"`
	Explanation   string          `json:"explanation,omitempty"`
	CorrectAnswer json.RawMessage `json:"correct_answer,omitempty"`
	Done          bool            `json:"done"`        // no more items: call complete
	MiniReward    bool            `json:"mini_reward"` // celebrate mid-session (§8.2)
	Assessment    *View           `json:"assessment,omitempty"`
}

func (h Handler) Answer(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "assessmentID")
	if !ok {
		return
	}
	var in struct {
		ItemID         uuid.UUID       `json:"item_id"`
		Response       json.RawMessage `json:"response"`
		ResponseTimeMS int             `json:"response_time_ms"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if len(in.Response) == 0 || len(in.Response) > 4096 {
		httpx.Invalid(w, map[string]string{"response": "Jawaban wajib diisi."})
		return
	}
	in.ResponseTimeMS = max(0, min(in.ResponseTimeMS, 10*60*1000))

	ctx := r.Context()
	u := auth.MustUser(ctx)
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(ctx)

	v, err := h.loadOwned(ctx, tx, u.ID, id, true)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if v.Status != "in_progress" {
		httpx.Error(w, http.StatusConflict, "not_in_progress", "Challenge ini sudah selesai.")
		return
	}

	var (
		position, attempts, difficulty, skillID int
		hintShown                               bool
		qid, qvid                               uuid.UUID
		typ, explanation, hint                  string
		content, answer                         json.RawMessage
		gradeID, scopeSkill                     int
	)
	err = tx.QueryRow(ctx, `
		SELECT i.position, i.attempts, i.difficulty, i.skill_id, i.hint_shown, i.question_id, i.question_version_id,
		       q.type, qv.explanation, qv.hint, qv.content, qv.answer, a.grade_id, COALESCE(a.skill_id, 0)
		FROM assessment_items i
		JOIN questions q ON q.id = i.question_id
		JOIN question_versions qv ON qv.id = i.question_version_id
		JOIN assessments a ON a.id = i.assessment_id
		WHERE i.id = $1 AND i.assessment_id = $2 AND i.finalized_at IS NULL
		FOR UPDATE OF i`, in.ItemID, id).
		Scan(&position, &attempts, &difficulty, &skillID, &hintShown, &qid, &qvid, &typ, &explanation, &hint, &content, &answer, &gradeID, &scopeSkill)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusConflict, "item_closed", "Soal ini sudah dijawab.")
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}

	correct, err := questions.Grade(typ, content, answer, in.Response)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	attempts++
	if _, err := tx.Exec(ctx, `INSERT INTO question_attempts
		(assessment_item_id, child_id, question_id, question_version_id, skill_id, difficulty, attempt_no, response, is_correct, response_time_ms, hint_used)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		in.ItemID, v.ChildID, qid, qvid, skillID, difficulty, attempts, in.Response, correct, in.ResponseTimeMS, hintShown); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	questionsAnswered.WithLabelValues(map[bool]string{true: "true", false: "false"}[correct]).Inc()

	res := AnswerResult{Correct: correct}
	final := correct || attempts >= MaxAttempts
	if !final {
		// Wrong → hint → retry. The hint is revealed now and counts as used on the retry.
		res.Feedback = "Hampir benar. Coba lihat lagi petunjuknya."
		res.Hint = hintOrDefault(hint)
		if _, err := tx.Exec(ctx, `UPDATE assessment_items SET attempts = $2, hint_shown = true, first_try_correct = false WHERE id = $1`, in.ItemID, attempts); err != nil {
			httpx.Internal(w, r, err)
			return
		}
	} else {
		res.Final = true
		res.Explanation = explanation
		switch {
		case correct && attempts == 1 && !hintShown:
			res.Feedback = praise[position%len(praise)]
		case correct:
			res.Feedback = "Bagus! Kamu berhasil setelah mencoba lagi."
		default:
			res.Feedback = "Tidak apa-apa, ini jawaban yang benar. Yuk lihat penjelasannya."
			res.CorrectAnswer = answer
		}
		if _, err := tx.Exec(ctx, `UPDATE assessment_items SET attempts = $2, final_correct = $3,
			first_try_correct = COALESCE(first_try_correct, $3), finalized_at = now() WHERE id = $1`, in.ItemID, attempts, correct); err != nil {
			httpx.Internal(w, r, err)
			return
		}
		res.MiniReward = position == 3 && v.PlannedCount > 4
		if position >= v.PlannedCount {
			res.Done = true
		} else {
			added, err := h.addNextItem(ctx, tx, id, v.ChildID, scope{GradeID: gradeID, SubjectID: v.SubjectID, SkillID: scopeSkill})
			if err != nil {
				httpx.Internal(w, r, err)
				return
			}
			res.Done = !added
		}
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if res.Final {
		nv, err := h.loadOwned(ctx, h.DB, u.ID, id, false)
		if err == nil {
			err = h.fillItems(ctx, h.DB, &nv)
		}
		if err != nil {
			httpx.Internal(w, r, err)
			return
		}
		res.Assessment = &nv
	}
	httpx.JSON(w, http.StatusOK, res)
}

type MissionProgress struct {
	ID        uuid.UUID `json:"id"`
	SkillName string    `json:"skill_name"`
	Completed int       `json:"completed"`
	Target    int       `json:"target"`
	Status    string    `json:"status"`
}

type SkillChange struct {
	SkillID int      `json:"skill_id"`
	Name    string   `json:"name"`
	Before  *float64 `json:"before"`
	After   float64  `json:"after"`
}

func (h Handler) Complete(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UUIDParam(w, r, "assessmentID")
	if !ok {
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
	v, err := h.loadOwned(ctx, tx, u.ID, id, true)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if v.Status == "completed" {
		// Idempotent: completing twice returns the stored result, awards nothing new.
		_ = tx.Rollback(ctx)
		httpx.JSON(w, http.StatusOK, map[string]any{"result": v.Result, "unlocks": []rewards.Unlock{}, "skills": []SkillChange{}, "already_completed": true})
		return
	}
	if v.Status != "in_progress" {
		httpx.Error(w, http.StatusConflict, "not_in_progress", "Challenge ini tidak aktif.")
		return
	}

	rows, err := tx.Query(ctx, `SELECT first_try_correct, final_correct, hint_shown, skill_id, question_id FROM assessment_items
		WHERE assessment_id = $1 AND finalized_at IS NOT NULL ORDER BY position`, id)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var items []ItemResult
	skillSet := map[int]bool{}
	questionSet := map[uuid.UUID]bool{}
	for rows.Next() {
		var it ItemResult
		var sid int
		var qid uuid.UUID
		if err := rows.Scan(&it.FirstTryCorrect, &it.FinalCorrect, &it.HintUsed, &sid, &qid); err != nil {
			rows.Close()
			httpx.Internal(w, r, err)
			return
		}
		items = append(items, it)
		skillSet[sid] = true
		questionSet[qid] = true
	}
	rows.Close()
	if len(items) == 0 {
		httpx.Error(w, http.StatusConflict, "nothing_answered", "Jawab minimal satu soal dulu ya.")
		return
	}
	score := ScoreSession(items)
	skillIDs := make([]int, 0, len(skillSet))
	for sid := range skillSet {
		skillIDs = append(skillIDs, sid)
	}
	questionIDs := make([]uuid.UUID, 0, len(questionSet))
	for qid := range questionSet {
		questionIDs = append(questionIDs, qid)
	}

	before := map[int]float64{}
	brows, err := tx.Query(ctx, `SELECT skill_id, mastery::float8 FROM skill_progress WHERE child_id = $1 AND skill_id = ANY($2)`, v.ChildID, skillIDs)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	for brows.Next() {
		var sid int
		var m float64
		_ = brows.Scan(&sid, &m)
		before[sid] = m
	}
	brows.Close()

	if _, err := tx.Exec(ctx, `UPDATE assessments SET status = 'completed', completed_at = now(), score = $2, accuracy = $3, stars = $4, xp_earned = $5
		WHERE id = $1`, id, score.Score, score.Accuracy, score.Stars, score.XP); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if err := progress.RecomputeSkills(ctx, tx, v.ChildID, skillIDs); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	// Creator attribution rollup (§39). Best-effort: a failure here must never
	// block a child's session, and `eduplay recompute-attribution` can rebuild.
	if err := creators.RecomputeQuestions(ctx, tx, questionIDs); err != nil {
		slog.ErrorContext(ctx, "attribution recompute failed", "err", err, "assessment_id", id)
	}
	today := time.Now().In(h.TZ).Format("2006-01-02")
	if _, err := tx.Exec(ctx, `INSERT INTO daily_progress (child_id, day, challenges_completed, questions_answered, correct_answers, xp)
		VALUES ($1, $2, 1, $3, $4, $5)
		ON CONFLICT (child_id, day) DO UPDATE SET
			challenges_completed = daily_progress.challenges_completed + 1,
			questions_answered = daily_progress.questions_answered + EXCLUDED.questions_answered,
			correct_answers = daily_progress.correct_answers + EXCLUDED.correct_answers,
			xp = daily_progress.xp + EXCLUDED.xp`, v.ChildID, today, score.Total, score.Correct, score.XP); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	// Mission progress: a completed focused session counts once toward its mission.
	var mission *MissionProgress
	var mp MissionProgress
	err = tx.QueryRow(ctx, `
		UPDATE child_missions m SET completed = m.completed + 1,
			status = CASE WHEN m.completed + 1 >= m.target THEN 'completed' ELSE m.status END,
			completed_at = CASE WHEN m.completed + 1 >= m.target THEN now() END
		FROM assessments a, skills k
		WHERE a.id = $1 AND m.id = a.mission_id AND m.status = 'active' AND k.id = m.skill_id
		RETURNING m.id, k.name, m.completed, m.target, m.status`, id).Scan(&mp.ID, &mp.SkillName, &mp.Completed, &mp.Target, &mp.Status)
	if err == nil {
		mission = &mp
	} else if !errors.Is(err, pgx.ErrNoRows) {
		httpx.Internal(w, r, err)
		return
	}

	newXP, unlocks, err := rewards.Grant(ctx, tx, v.ChildID, score.XP, rewards.SessionFacts{Score: score.Score, MissionCompleted: mission != nil && mission.Status == "completed"})
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}

	changes := []SkillChange{}
	crows, err := tx.Query(ctx, `SELECT sp.skill_id, k.name, sp.mastery::float8 FROM skill_progress sp JOIN skills k ON k.id = sp.skill_id
		WHERE sp.child_id = $1 AND sp.skill_id = ANY($2) ORDER BY k.sort`, v.ChildID, skillIDs)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	for crows.Next() {
		var c SkillChange
		if err := crows.Scan(&c.SkillID, &c.Name, &c.After); err != nil {
			crows.Close()
			httpx.Internal(w, r, err)
			return
		}
		if b, ok := before[c.SkillID]; ok {
			c.Before = &b
		}
		changes = append(changes, c)
	}
	crows.Close()

	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	sessionsCompleted.Inc()
	analytics.Track(ctx, h.DB, "challenge_completed", u.ID, v.ChildID, map[string]any{
		"subject_id": v.SubjectID, "questions": score.Total, "accuracy": score.Accuracy})
	if unlocks == nil {
		unlocks = []rewards.Unlock{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"result": score, "unlocks": unlocks, "skills": changes, "mission": mission,
		"child": map[string]int{"xp": newXP, "level": children.LevelForXP(newXP)},
	})
}
