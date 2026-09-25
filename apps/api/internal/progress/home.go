package progress

import (
	"net/http"

	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/children"
	"github.com/nocify/eduplay/api/internal/curriculum"
	"github.com/nocify/eduplay/api/internal/httpx"
	"github.com/nocify/eduplay/api/internal/missions"
	"github.com/nocify/eduplay/api/internal/subscriptions"
)

// ChildHome is the child-facing summary: what to do next and why it is fun
// (§58). It deliberately has no percentages — children see stars and levels,
// parents see mastery.
func (h Handler) ChildHome(w http.ResponseWriter, r *http.Request) {
	child, ok := children.Owned(w, r, h.DB, "childID")
	if !ok {
		return
	}
	ctx := r.Context()
	rows, err := h.DB.Query(ctx, curriculum.SubjectsForGradeSQL, child.GradeID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	subjects, err := curriculum.CollectSubjects(rows)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	type subjectCard struct {
		curriculum.Subject
		Stars     int  `json:"stars"`
		Suggested bool `json:"suggested"`
	}
	cards := make([]subjectCard, 0, len(subjects))
	weakest, weakestIdx := 101.0, -1
	for i, s := range subjects {
		var stars int
		var avg *float64
		if err := h.DB.QueryRow(ctx, `
			SELECT COALESCE((SELECT sum(stars) FROM assessments WHERE child_id = $1 AND subject_id = $2 AND status = 'completed'), 0)::int,
			       (SELECT avg(sp.mastery)::float8 FROM skill_progress sp JOIN skills k ON k.id = sp.skill_id
			        JOIN chapters c ON c.id = k.chapter_id WHERE sp.child_id = $1 AND c.subject_id = $2 AND c.grade_id = $3)`,
			child.ID, s.ID, child.GradeID).Scan(&stars, &avg); err != nil {
			httpx.Internal(w, r, err)
			return
		}
		m := 50.0 // untried subjects are suggested before strong ones
		if avg != nil {
			m = *avg
		}
		if m < weakest {
			weakest, weakestIdx = m, i
		}
		cards = append(cards, subjectCard{Subject: s, Stars: stars})
	}
	if weakestIdx >= 0 {
		cards[weakestIdx].Suggested = true
	}

	ent, err := h.Policy.Get(ctx, h.DB, auth.MustUser(ctx).ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var remaining *int
	if ent.Limits.DailyChallenges > 0 {
		var used int
		if err := h.DB.QueryRow(ctx, `SELECT count(*) FROM assessments a JOIN children c ON c.id = a.child_id
			WHERE c.parent_id = $1 AND a.started_at >= $2`,
			auth.MustUser(ctx).ID, subscriptions.StartOfDay(h.TZ)).Scan(&used); err != nil {
			httpx.Internal(w, r, err)
			return
		}
		left := max(0, ent.Limits.DailyChallenges-used)
		remaining = &left
	}
	var stars, badges int
	if err := h.DB.QueryRow(ctx, `SELECT
		COALESCE((SELECT sum(stars) FROM assessments WHERE child_id = $1 AND status = 'completed'), 0)::int,
		(SELECT count(*) FROM child_inventory WHERE child_id = $1 AND item_kind = 'badge')::int`, child.ID).Scan(&stars, &badges); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var next *struct {
		Name string `json:"name"`
		Icon string `json:"icon"`
		XP   int    `json:"xp_required"`
	}
	var n struct {
		Name string `json:"name"`
		Icon string `json:"icon"`
		XP   int    `json:"xp_required"`
	}
	if err := h.DB.QueryRow(ctx, `SELECT name, icon, xp_required FROM rewards WHERE xp_required > $1 ORDER BY xp_required LIMIT 1`, child.XP).
		Scan(&n.Name, &n.Icon, &n.XP); err == nil {
		next = &n
	}
	active, err := missions.List(ctx, h.DB, child.ID, true)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	streak, err := Streak(ctx, h.DB, child.ID, h.TZ)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var today int
	_ = h.DB.QueryRow(ctx, `SELECT COALESCE(challenges_completed, 0) FROM daily_progress WHERE child_id = $1 AND day = $2`,
		child.ID, subscriptions.StartOfDay(h.TZ).Format("2006-01-02")).Scan(&today)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"missions": active, "streak": streak, "completed_today": today,
		"child": child, "subjects": cards, "stars": stars, "badges": badges,
		"challenges_left_today": remaining, "next_reward": next,
	})
}
