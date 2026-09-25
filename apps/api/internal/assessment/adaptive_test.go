package assessment

import (
	"math/rand"
	"testing"
)

func base(s SkillState) int {
	if !s.HasEvidence {
		return 2
	}
	switch {
	case s.Mastery < 40:
		return 1
	case s.Mastery < 70:
		return 2
	case s.Mastery <= 85:
		return 3
	}
	return 4
}

func TestFocusSkillsWeakestFirst(t *testing.T) {
	got := FocusSkills([]SkillState{
		{SkillID: 1, Mastery: 90, HasEvidence: true},
		{SkillID: 2, Mastery: 20, HasEvidence: true},
		{SkillID: 3},
	})
	if got[0].SkillID != 2 || got[1].SkillID != 3 || got[2].SkillID != 1 {
		t.Fatalf("unexpected order %+v", got)
	}
}

func TestNextTargetWarmupAndStreaks(t *testing.T) {
	states := []SkillState{{SkillID: 7, Mastery: 60, HasEvidence: true}} // base 2
	if _, d := NextTarget(0, states, nil, base); d != 1 {
		t.Fatalf("warm-up should be one easier, got %d", d)
	}
	if _, d := NextTarget(1, states, []Outcome{{FirstTryCorrect: true}}, base); d != 2 {
		t.Fatalf("no streak → base, got %d", d)
	}
	three := []Outcome{{FirstTryCorrect: true}, {FirstTryCorrect: true}, {FirstTryCorrect: true}}
	if _, d := NextTarget(3, states, three, base); d != 3 {
		t.Fatalf("3 correct → harder, got %d", d)
	}
	two := []Outcome{{FirstTryCorrect: true}, {FirstTryCorrect: false}, {FirstTryCorrect: false}}
	if _, d := NextTarget(3, states, two, base); d != 1 {
		t.Fatalf("2 wrong → easier, got %d", d)
	}
}

func TestPickPrefersSkillThenDifficulty(t *testing.T) {
	pool := []Candidate{
		{QuestionID: "other-skill-exact", SkillID: 2, Difficulty: 3},
		{QuestionID: "same-skill-off1", SkillID: 1, Difficulty: 2},
		{QuestionID: "same-skill-exact-seen", SkillID: 1, Difficulty: 3, RecentSeen: true},
		{QuestionID: "same-skill-exact", SkillID: 1, Difficulty: 3},
	}
	rng := rand.New(rand.NewSource(1))
	if got := Pick(pool, nil, 1, 3, rng); got != "same-skill-exact" {
		t.Fatalf("got %s", got)
	}
	used := map[string]bool{"same-skill-exact": true, "same-skill-exact-seen": true}
	if got := Pick(pool, used, 1, 3, rng); got != "same-skill-off1" {
		t.Fatalf("got %s", got)
	}
	used["same-skill-off1"] = true
	if got := Pick(pool, used, 1, 3, rng); got != "other-skill-exact" {
		t.Fatalf("fallback to other skill, got %s", got)
	}
	used["other-skill-exact"] = true
	if got := Pick(pool, used, 1, 3, rng); got != "" {
		t.Fatalf("exhausted pool should return empty, got %s", got)
	}
}

func TestScoreSession(t *testing.T) {
	s := ScoreSession([]ItemResult{
		{FirstTryCorrect: true, FinalCorrect: true},
		{FirstTryCorrect: false, FinalCorrect: true, HintUsed: true},
		{FirstTryCorrect: false, FinalCorrect: false},
		{FirstTryCorrect: true, FinalCorrect: true},
	})
	// points 1 + .5 + 0 + 1 = 2.5/4 = 62.5 → 63
	if s.Score != 63 || s.Accuracy != 50 || s.Stars != 2 || s.Correct != 3 || s.XP != 10+5+2+10+10 {
		t.Fatalf("unexpected score %+v", s)
	}
	if ScoreSession(nil).Stars != 1 {
		t.Fatal("empty session still earns a star")
	}
}
