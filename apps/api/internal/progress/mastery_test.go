package progress

import (
	"testing"
	"time"
)

func TestMastery(t *testing.T) {
	cases := []struct {
		name string
		ev   []Evidence
		want float64
	}{
		{"empty", nil, 0},
		{"all correct", []Evidence{{true, false}, {true, false}}, 100},
		{"all wrong", []Evidence{{false, false}, {false, false}}, 0},
		{"hint halves credit", []Evidence{{true, true}}, 50},
		// newest correct (w=1), older wrong (w=.9): 1/1.9
		{"recency weighted", []Evidence{{true, false}, {false, false}}, 52.6},
		{"older correct weighs less", []Evidence{{false, false}, {true, false}}, 47.4},
	}
	for _, c := range cases {
		if got := Mastery(c.ev); got != c.want {
			t.Errorf("%s: Mastery = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestMasteryWindow(t *testing.T) {
	ev := make([]Evidence, MasteryWindow)
	for i := range ev {
		ev[i] = Evidence{Correct: true}
	}
	ev = append(ev, make([]Evidence, 50)...) // 50 old wrong answers fall outside the window
	if got := Mastery(ev); got != 100 {
		t.Fatalf("old evidence should be ignored, got %v", got)
	}
}

func TestBandsAndDifficulty(t *testing.T) {
	cases := []struct {
		m    float64
		band Band
		diff int
	}{
		{0, BandNeedsPractice, 1}, {39.9, BandNeedsPractice, 1},
		{40, BandDeveloping, 2}, {69.9, BandDeveloping, 2},
		{70, BandProficient, 3}, {85, BandProficient, 3},
		{85.1, BandMastered, 4}, {100, BandMastered, 4},
	}
	for _, c := range cases {
		if b := BandOf(c.m); b != c.band {
			t.Errorf("BandOf(%v) = %s, want %s", c.m, b, c.band)
		}
		if d := TargetDifficulty(c.m, true); d != c.diff {
			t.Errorf("TargetDifficulty(%v) = %d, want %d", c.m, d, c.diff)
		}
	}
	if d := TargetDifficulty(0, false); d != 2 {
		t.Errorf("no evidence should start at normal difficulty, got %d", d)
	}
}

func TestStreakFrom(t *testing.T) {
	now := time.Date(2026, 9, 25, 15, 0, 0, 0, time.UTC)
	d := func(day int) time.Time { return time.Date(2026, 9, day, 0, 0, 0, 0, time.UTC) }
	cases := []struct {
		days []time.Time
		want int
	}{
		{nil, 0},
		{[]time.Time{d(25), d(24), d(23)}, 3},
		{[]time.Time{d(24), d(23)}, 2},        // today not played yet: streak still alive
		{[]time.Time{d(23), d(22)}, 0},        // missed yesterday
		{[]time.Time{d(25), d(23), d(22)}, 1}, // gap
	}
	for i, c := range cases {
		if got := StreakFrom(c.days, now); got != c.want {
			t.Errorf("case %d: got %d want %d", i, got, c.want)
		}
	}
}
