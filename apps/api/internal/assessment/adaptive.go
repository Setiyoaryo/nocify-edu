// Package assessment runs a child's challenge session: adaptive item selection,
// server-side grading, hints/retry, scoring and completion (§8, §10, §11, §42).
package assessment

import (
	"math/rand"
	"sort"
)

// Candidate is a published question that could be served.
type Candidate struct {
	QuestionID string
	SkillID    int
	Difficulty int
	RecentSeen bool // seen by this child in the last few sessions
}

// SkillState is the child's current standing on a skill.
type SkillState struct {
	SkillID     int
	Mastery     float64
	HasEvidence bool
}

// Outcome is one finalized item of the current session, oldest first.
type Outcome struct {
	SkillID         int
	Difficulty      int
	FirstTryCorrect bool
}

// Rules of §11, applied deterministically:
//   - start each skill at the difficulty its mastery band calls for;
//   - the first item of a session is a warm-up one step easier;
//   - 3 correct in a row → +1 difficulty; 2 wrong in a row → −1 (the client
//     also offers the hint right away on the next wrong answer).
const (
	upStreak   = 3
	downStreak = 2
)

// FocusSkills orders skills by learning need: weakest evidence first, untried
// skills treated as mid-need, mastered skills last ("challenge / next skill").
func FocusSkills(states []SkillState) []SkillState {
	out := append([]SkillState(nil), states...)
	need := func(s SkillState) float64 {
		if !s.HasEvidence {
			return 50
		}
		return s.Mastery
	}
	sort.SliceStable(out, func(i, j int) bool { return need(out[i]) < need(out[j]) })
	return out
}

// NextTarget decides skill and difficulty for position pos (0-based).
func NextTarget(pos int, states []SkillState, history []Outcome, base func(SkillState) int) (skillID, difficulty int) {
	focus := FocusSkills(states)
	if len(focus) == 0 {
		return 0, 2
	}
	// Rotate through the (up to) three most-needed skills for variety.
	n := len(focus)
	if n > 3 {
		n = 3
	}
	s := focus[pos%n]
	d := base(s)
	if pos == 0 {
		d--
	}
	// Streak adjustment over the session so far (any skill).
	streakC, streakW := 0, 0
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].FirstTryCorrect {
			if streakW > 0 {
				break
			}
			streakC++
		} else {
			if streakC > 0 {
				break
			}
			streakW++
		}
	}
	if streakC >= upStreak {
		d += streakC / upStreak
	}
	if streakW >= downStreak {
		d -= streakW / downStreak
	}
	if d < 1 {
		d = 1
	}
	if d > 5 {
		d = 5
	}
	return s.SkillID, d
}

// Pick chooses a question for the target, preferring the target skill, then
// closest difficulty, then questions not seen recently. used excludes items
// already in this session. Returns "" when the pool is exhausted.
func Pick(pool []Candidate, used map[string]bool, skillID, difficulty int, rng *rand.Rand) string {
	best := ""
	bestScore := 1 << 30
	ties := 0
	for _, c := range pool {
		if used[c.QuestionID] {
			continue
		}
		score := abs(c.Difficulty-difficulty) * 10
		if c.SkillID != skillID {
			score += 100
		}
		if c.RecentSeen {
			score += 5
		}
		switch {
		case score < bestScore:
			best, bestScore, ties = c.QuestionID, score, 1
		case score == bestScore:
			// Reservoir sampling keeps ties uniformly random.
			ties++
			if rng.Intn(ties) == 0 {
				best = c.QuestionID
			}
		}
	}
	return best
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Score summarises a finished session.
type Score struct {
	Score    int     `json:"score"`    // 0–100
	Accuracy float64 `json:"accuracy"` // first-try accuracy, 0–100
	Stars    int     `json:"stars"`    // 1–3, never 0: effort is always acknowledged
	XP       int     `json:"xp"`
	Correct  int     `json:"correct"`
	Total    int     `json:"total"`
}

type ItemResult struct {
	FirstTryCorrect bool
	FinalCorrect    bool
	HintUsed        bool
}

func ScoreSession(items []ItemResult) Score {
	var s Score
	s.Total = len(items)
	if s.Total == 0 {
		return Score{Stars: 1}
	}
	points := 0.0
	first := 0
	for _, it := range items {
		switch {
		case it.FirstTryCorrect && !it.HintUsed:
			points += 1
			s.XP += 10
		case it.FinalCorrect:
			points += 0.5
			s.XP += 5
		default:
			s.XP += 2 // effort
		}
		if it.FirstTryCorrect {
			first++
		}
		if it.FinalCorrect {
			s.Correct++
		}
	}
	s.XP += 10 // completion bonus
	s.Score = int(points/float64(s.Total)*100 + 0.5)
	s.Accuracy = float64(int(float64(first)/float64(s.Total)*1000+0.5)) / 10
	switch {
	case s.Score >= 90:
		s.Stars = 3
	case s.Score >= 60:
		s.Stars = 2
	default:
		s.Stars = 1
	}
	return s
}
