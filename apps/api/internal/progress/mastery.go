// Package progress turns raw question attempts into skill mastery, subject
// progress and recommendations (§11, §43). Everything here is recomputable
// from question_attempts, so the algorithm can change without losing history.
package progress

import "math"

const (
	// MasteryWindow is how many recent first attempts count toward a skill.
	MasteryWindow = 20
	// recencyDecay weights newer evidence more (1.0, 0.9, 0.81, ...).
	recencyDecay = 0.9
	// MinEvidence below which the parent sees "belum cukup data".
	MinEvidence = 3
)

// Evidence is one first attempt at an item, newest first.
type Evidence struct {
	Correct  bool
	HintUsed bool
}

// Mastery returns 0–100. Correct-with-hint earns half credit.
func Mastery(ev []Evidence) float64 {
	if len(ev) == 0 {
		return 0
	}
	if len(ev) > MasteryWindow {
		ev = ev[:MasteryWindow]
	}
	var num, den float64
	w := 1.0
	for _, e := range ev {
		credit := 0.0
		if e.Correct {
			credit = 1
			if e.HintUsed {
				credit = 0.5
			}
		}
		num += w * credit
		den += w
		w *= recencyDecay
	}
	return math.Round(num/den*1000) / 10
}

type Band string

const (
	BandNeedsPractice Band = "needs_practice" // < 40
	BandDeveloping    Band = "developing"     // 40–70
	BandProficient    Band = "proficient"     // 70–85
	BandMastered      Band = "mastered"       // > 85
)

// BandOf maps mastery to the deterministic bands of §11.
func BandOf(m float64) Band {
	switch {
	case m < 40:
		return BandNeedsPractice
	case m < 70:
		return BandDeveloping
	case m <= 85:
		return BandProficient
	}
	return BandMastered
}

// TargetDifficulty is the starting difficulty (1–5) for a skill at a mastery level:
// <40 easy/reinforcement, 40–70 normal, 70–85 medium-hard, >85 challenge.
func TargetDifficulty(m float64, hasEvidence bool) int {
	if !hasEvidence {
		return 2
	}
	switch BandOf(m) {
	case BandNeedsPractice:
		return 1
	case BandDeveloping:
		return 2
	case BandProficient:
		return 3
	}
	return 4
}
