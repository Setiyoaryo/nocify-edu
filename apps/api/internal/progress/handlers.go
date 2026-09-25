package progress

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/children"
	"github.com/nocify/eduplay/api/internal/httpx"
	"github.com/nocify/eduplay/api/internal/subscriptions"
)

type Handler struct {
	DB     *pgxpool.Pool
	Policy subscriptions.Policy
	TZ     *time.Location
}

type SkillView struct {
	SkillID     int     `json:"skill_id"`
	Name        string  `json:"name"`
	ChapterID   int     `json:"chapter_id"`
	ChapterName string  `json:"chapter_name"`
	SubjectID   int     `json:"subject_id"`
	SubjectName string  `json:"subject_name"`
	Mastery     float64 `json:"mastery"`
	Evidence    int     `json:"evidence"`
	Band        Band    `json:"band"`
	recentWrong int
}

type ChapterView struct {
	ChapterID       int      `json:"chapter_id"`
	Name            string   `json:"name"`
	SubjectID       int      `json:"subject_id"`
	Mastery         *float64 `json:"mastery"`
	SkillsPracticed int      `json:"skills_practiced"`
	SkillsTotal     int      `json:"skills_total"`
}

// Chapters aggregates skill mastery per chapter (Child → Subject → Chapter → Skill, §11).
func Chapters(skills []SkillView) []ChapterView {
	var out []ChapterView
	idx := map[int]int{}
	sums := map[int]float64{}
	for _, s := range skills {
		i, ok := idx[s.ChapterID]
		if !ok {
			i = len(out)
			idx[s.ChapterID] = i
			out = append(out, ChapterView{ChapterID: s.ChapterID, Name: s.ChapterName, SubjectID: s.SubjectID})
		}
		out[i].SkillsTotal++
		if s.Evidence > 0 {
			out[i].SkillsPracticed++
			sums[s.ChapterID] += s.Mastery
		}
	}
	for i := range out {
		if out[i].SkillsPracticed > 0 {
			m := round1(sums[out[i].ChapterID] / float64(out[i].SkillsPracticed))
			out[i].Mastery = &m
		}
	}
	return out
}

type SubjectView struct {
	SubjectID       int      `json:"subject_id"`
	Code            string   `json:"code"`
	Name            string   `json:"name"`
	Mastery         *float64 `json:"mastery"` // nil until there is evidence
	SkillsPracticed int      `json:"skills_practiced"`
	SkillsTotal     int      `json:"skills_total"`
}

// snapshot loads every skill of the child's grade with its progress.
func (h Handler) snapshot(ctx context.Context, child children.Child) ([]SkillView, []SubjectView, error) {
	rows, err := h.DB.Query(ctx, `
		SELECT k.id, k.name, c.id, c.name, s.id, s.code, s.name,
		       COALESCE(sp.mastery, 0)::float8, COALESCE(sp.evidence_count, 0),
		       (SELECT count(*) FROM (
		           SELECT is_correct FROM question_attempts qa
		           WHERE qa.child_id = $1 AND qa.skill_id = k.id AND qa.attempt_no = 1
		           ORDER BY created_at DESC LIMIT 3) last3 WHERE NOT is_correct)::int
		FROM skills k
		JOIN chapters c ON c.id = k.chapter_id
		JOIN curriculum_versions cv ON cv.id = c.curriculum_version_id AND cv.is_active
		JOIN subjects s ON s.id = c.subject_id
		LEFT JOIN skill_progress sp ON sp.skill_id = k.id AND sp.child_id = $1
		WHERE c.grade_id = $2
		ORDER BY s.sort, c.sort, k.sort`, child.ID, child.GradeID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var skills []SkillView
	subjects := map[int]*SubjectView{}
	var subjectOrder []int
	sums := map[int]float64{}
	for rows.Next() {
		var sv SkillView
		var code string
		if err := rows.Scan(&sv.SkillID, &sv.Name, &sv.ChapterID, &sv.ChapterName, &sv.SubjectID, &code, &sv.SubjectName, &sv.Mastery, &sv.Evidence, &sv.recentWrong); err != nil {
			return nil, nil, err
		}
		sv.Band = BandOf(sv.Mastery)
		skills = append(skills, sv)
		sub := subjects[sv.SubjectID]
		if sub == nil {
			sub = &SubjectView{SubjectID: sv.SubjectID, Code: code, Name: sv.SubjectName}
			subjects[sv.SubjectID] = sub
			subjectOrder = append(subjectOrder, sv.SubjectID)
		}
		sub.SkillsTotal++
		if sv.Evidence > 0 {
			sub.SkillsPracticed++
			sums[sv.SubjectID] += sv.Mastery
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	out := make([]SubjectView, 0, len(subjectOrder))
	for _, id := range subjectOrder {
		s := subjects[id]
		if s.SkillsPracticed > 0 {
			m := round1(sums[id] / float64(s.SkillsPracticed))
			s.Mastery = &m
		}
		out = append(out, *s)
	}
	return skills, out, nil
}

func round1(f float64) float64 { return float64(int(f*10+0.5)) / 10 }

func (h Handler) Progress(w http.ResponseWriter, r *http.Request) {
	child, ok := children.Owned(w, r, h.DB, "childID")
	if !ok {
		return
	}
	ctx := r.Context()
	ent, err := h.Policy.Get(ctx, h.DB, auth.MustUser(ctx).ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	skills, subjects, err := h.snapshot(ctx, child)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}

	var overall *float64
	var sum float64
	n := 0
	for _, s := range subjects {
		if s.Mastery != nil {
			sum += *s.Mastery
			n++
		}
	}
	if n > 0 {
		o := round1(sum / float64(n))
		overall = &o
	}

	groups := map[string][]SkillView{"strong": {}, "developing": {}, "needs_practice": {}}
	insufficient := 0
	for _, s := range skills {
		switch {
		case s.Evidence == 0:
		case s.Evidence < MinEvidence:
			insufficient++
		case s.Mastery >= 70:
			groups["strong"] = append(groups["strong"], s)
		case s.Mastery >= 40:
			groups["developing"] = append(groups["developing"], s)
		default:
			groups["needs_practice"] = append(groups["needs_practice"], s)
		}
	}
	sort.SliceStable(groups["strong"], func(i, j int) bool { return groups["strong"][i].Mastery > groups["strong"][j].Mastery })
	sort.SliceStable(groups["needs_practice"], func(i, j int) bool { return groups["needs_practice"][i].Mastery < groups["needs_practice"][j].Mastery })

	var activity struct {
		Challenges7d int      `json:"challenges_7d"`
		Questions7d  int      `json:"questions_7d"`
		Accuracy7d   *float64 `json:"accuracy_7d"`
		Started30d   int      `json:"started_30d"`
		Completed30d int      `json:"completed_30d"`
	}
	if err := h.DB.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM assessments WHERE child_id = $1 AND status = 'completed' AND completed_at > now() - interval '7 days')::int,
		  (SELECT count(*) FROM question_attempts WHERE child_id = $1 AND attempt_no = 1 AND created_at > now() - interval '7 days')::int,
		  (SELECT round(100.0 * avg(CASE WHEN is_correct THEN 1 ELSE 0 END), 1)::float8 FROM question_attempts WHERE child_id = $1 AND attempt_no = 1 AND created_at > now() - interval '7 days'),
		  (SELECT count(*) FROM assessments WHERE child_id = $1 AND started_at > now() - interval '30 days')::int,
		  (SELECT count(*) FROM assessments WHERE child_id = $1 AND status = 'completed' AND started_at > now() - interval '30 days')::int`,
		child.ID).Scan(&activity.Challenges7d, &activity.Questions7d, &activity.Accuracy7d, &activity.Started30d, &activity.Completed30d); err != nil {
		httpx.Internal(w, r, err)
		return
	}

	streak, err := Streak(ctx, h.DB, child.ID, h.TZ)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	resp := map[string]any{
		"streak":       streak,
		"chapters":     Chapters(skills),
		"child":        child,
		"premium":      ent.Premium,
		"overall":      overall,
		"subjects":     subjects,
		"activity":     activity,
		"insufficient": insufficient,
		"locked":       map[string]bool{"skills": !ent.Premium, "trend": !ent.Premium},
	}
	if ent.Premium {
		resp["skills"] = groups
		trend, err := h.trend(ctx, child.ID, 28)
		if err != nil {
			httpx.Internal(w, r, err)
			return
		}
		resp["trend"] = trend
	} else {
		// Free tier: one strong and one weak skill — enough to show the value of
		// skill-level insight (§4.3 critical conversion moment).
		teaser := map[string][]SkillView{"strong": {}, "developing": {}, "needs_practice": {}}
		if len(groups["strong"]) > 0 {
			teaser["strong"] = groups["strong"][:1]
		}
		if len(groups["needs_practice"]) > 0 {
			teaser["needs_practice"] = groups["needs_practice"][:1]
		} else if len(groups["developing"]) > 0 {
			teaser["developing"] = groups["developing"][:1]
		}
		resp["skills"] = teaser
		resp["hidden_skills"] = len(groups["strong"]) + len(groups["developing"]) + len(groups["needs_practice"]) -
			len(teaser["strong"]) - len(teaser["developing"]) - len(teaser["needs_practice"])
	}
	httpx.JSON(w, http.StatusOK, resp)
}

type TrendPoint struct {
	Day       string   `json:"day"`
	Questions int      `json:"questions"`
	Accuracy  *float64 `json:"accuracy"`
}

// trend returns weekly accuracy for the last `days` days (weeks start Monday, local time).
func (h Handler) trend(ctx context.Context, childID uuid.UUID, days int) ([]TrendPoint, error) {
	rows, err := h.DB.Query(ctx, `
		SELECT to_char(date_trunc('week', created_at AT TIME ZONE $3), 'YYYY-MM-DD'),
		       count(*)::int, round(100.0 * avg(CASE WHEN is_correct THEN 1 ELSE 0 END), 1)::float8
		FROM question_attempts
		WHERE child_id = $1 AND attempt_no = 1 AND created_at > now() - make_interval(days => $2)
		GROUP BY 1 ORDER BY 1`, childID, days, h.TZ.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TrendPoint{}
	for rows.Next() {
		var p TrendPoint
		if err := rows.Scan(&p.Day, &p.Questions, &p.Accuracy); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (h Handler) History(w http.ResponseWriter, r *http.Request) {
	child, ok := children.Owned(w, r, h.DB, "childID")
	if !ok {
		return
	}
	ent, err := h.Policy.Get(r.Context(), h.DB, auth.MustUser(r.Context()).ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	limit := httpx.QueryInt(r, "limit", 20)
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if !ent.Premium && limit > 3 {
		limit = 3
	}
	rows, err := h.DB.Query(r.Context(), `
		SELECT a.id, s.name, a.status, a.started_at, a.completed_at, a.score, a.accuracy::float8, a.stars, a.planned_count,
		       (SELECT count(*) FROM assessment_items i WHERE i.assessment_id = a.id AND i.finalized_at IS NOT NULL)::int
		FROM assessments a JOIN subjects s ON s.id = a.subject_id
		WHERE a.child_id = $1 AND a.status = 'completed'
		ORDER BY a.started_at DESC LIMIT $2`, child.ID, limit)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer rows.Close()
	type entry struct {
		ID          uuid.UUID  `json:"id"`
		Subject     string     `json:"subject"`
		Status      string     `json:"status"`
		StartedAt   time.Time  `json:"started_at"`
		CompletedAt *time.Time `json:"completed_at"`
		Score       *int       `json:"score"`
		Accuracy    *float64   `json:"accuracy"`
		Stars       *int       `json:"stars"`
		Planned     int        `json:"planned"`
		Answered    int        `json:"answered"`
	}
	out := []entry{}
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.ID, &e.Subject, &e.Status, &e.StartedAt, &e.CompletedAt, &e.Score, &e.Accuracy, &e.Stars, &e.Planned, &e.Answered); err != nil {
			httpx.Internal(w, r, err)
			return
		}
		out = append(out, e)
	}
	var total int
	if err := h.DB.QueryRow(r.Context(), `SELECT count(*) FROM assessments WHERE child_id = $1 AND status = 'completed'`, child.ID).Scan(&total); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"history": out, "total": total, "locked": !ent.Premium && total > len(out)})
}

type Recommendation struct {
	Kind      string `json:"kind"` // practice | remediate | continue | explore | challenge | habit
	Priority  int    `json:"priority"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	SubjectID int    `json:"subject_id,omitempty"`
	SkillID   int    `json:"skill_id,omitempty"`
}

// BuildRecommendations applies the deterministic rules in docs/progress-engine.md.
func BuildRecommendations(nickname string, skills []SkillView, daysSinceActive int) []Recommendation {
	var out []Recommendation
	explore := 0
	masteredBySubject := map[int]string{}
	for _, s := range skills {
		switch {
		case s.Evidence >= MinEvidence && s.recentWrong >= 3:
			out = append(out, Recommendation{Kind: "remediate", Priority: 1, SubjectID: s.SubjectID, SkillID: s.SkillID,
				Title:  fmt.Sprintf("Ulangi dasar %s bersama %s", s.Name, nickname),
				Detail: "3 jawaban terakhir di skill ini belum tepat. Coba bahas satu contoh bersama sebelum latihan lagi."})
		case s.Evidence >= MinEvidence && s.Band == BandNeedsPractice:
			out = append(out, Recommendation{Kind: "practice", Priority: 2, SubjectID: s.SubjectID, SkillID: s.SkillID,
				Title:  fmt.Sprintf("Latih 5 challenge tentang %s minggu ini", s.Name),
				Detail: fmt.Sprintf("Penguasaan saat ini %.0f%%. Latihan singkat dan rutin paling efektif.", s.Mastery)})
		case s.Evidence >= MinEvidence && s.Band == BandDeveloping:
			out = append(out, Recommendation{Kind: "continue", Priority: 3, SubjectID: s.SubjectID, SkillID: s.SkillID,
				Title:  fmt.Sprintf("Lanjutkan latihan %s", s.Name),
				Detail: fmt.Sprintf("Sudah %.0f%% — sedikit lagi menuju mahir.", s.Mastery)})
		case s.Evidence >= 5 && s.Band == BandMastered:
			masteredBySubject[s.SubjectID] = s.SubjectName
		case s.Evidence == 0 && explore < 2:
			explore++
			out = append(out, Recommendation{Kind: "explore", Priority: 4, SubjectID: s.SubjectID, SkillID: s.SkillID,
				Title:  fmt.Sprintf("Coba topik baru: %s", s.Name),
				Detail: fmt.Sprintf("%s belum pernah mencoba skill ini.", nickname)})
		}
	}
	for id, name := range masteredBySubject {
		out = append(out, Recommendation{Kind: "challenge", Priority: 5, SubjectID: id,
			Title:  fmt.Sprintf("%s siap tantangan berikutnya di %s", nickname, name),
			Detail: "Beberapa skill sudah dikuasai. Challenge berikutnya akan otomatis lebih menantang."})
	}
	if daysSinceActive >= 3 {
		out = append(out, Recommendation{Kind: "habit", Priority: 0,
			Title:  fmt.Sprintf("Ajak %s main 1 challenge singkat hari ini", nickname),
			Detail: "Belajar 5–10 menit secara rutin lebih efektif daripada sesi panjang sesekali."})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Priority < out[j].Priority })
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

func (h Handler) Recommendations(w http.ResponseWriter, r *http.Request) {
	child, ok := children.Owned(w, r, h.DB, "childID")
	if !ok {
		return
	}
	ent, err := h.Policy.Get(r.Context(), h.DB, auth.MustUser(r.Context()).ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	skills, _, err := h.snapshot(r.Context(), child)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	days := 0
	if child.LastActivity != nil {
		days = int(time.Since(*child.LastActivity).Hours() / 24)
	}
	recs := BuildRecommendations(child.Nickname, skills, days)
	total := len(recs)
	if !ent.Premium && len(recs) > 1 {
		recs = recs[:1]
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"recommendations": recs, "total": total, "locked": !ent.Premium && total > len(recs)})
}
