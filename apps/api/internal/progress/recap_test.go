package progress

import (
	"testing"

	"github.com/nocify/eduplay/api/internal/children"
)

func TestDetermineArchetype(t *testing.T) {
	cases := []struct {
		name     string
		child    children.Child
		stats    RecapStats
		wantTier string
		wantCode string
	}{
		{
			name:     "new starter",
			child:    children.Child{Nickname: "Budi", Level: 1},
			stats:    RecapStats{TotalChallenges: 1, Streak: 0},
			wantTier: "starter",
			wantCode: "pahlawan_baru",
		},
		{
			name:     "streak master fire knight",
			child:    children.Child{Nickname: "Siti", Level: 2},
			stats:    RecapStats{TotalChallenges: 4, Streak: 5},
			wantTier: "epic",
			wantCode: "ksatria_api",
		},
		{
			name:     "eagle eye detective high accuracy",
			child:    children.Child{Nickname: "Rian", Level: 3},
			stats:    RecapStats{TotalChallenges: 6, TotalQuestions: 15, Accuracy: 90.0, Streak: 1},
			wantTier: "epic",
			wantCode: "detektif_elang",
		},
		{
			name:     "math wizard",
			child:    children.Child{Nickname: "Adi", Level: 2},
			stats:    RecapStats{TotalChallenges: 4, FavoriteSubject: "Matematika", TotalQuestions: 20},
			wantTier: "rare",
			wantCode: "penyihir_angka",
		},
		{
			name:     "legendary tier",
			child:    children.Child{Nickname: "Sultan", Level: 5},
			stats:    RecapStats{TotalChallenges: 25, Streak: 7, Stars: 50},
			wantTier: "legendary",
			wantCode: "ksatria_api",
		},
	}

	for _, c := range cases {
		got := DetermineArchetype(c.child, c.stats)
		if got.Tier != c.wantTier {
			t.Errorf("%s: got tier %s, want %s", c.name, got.Tier, c.wantTier)
		}
		if got.Code != c.wantCode {
			t.Errorf("%s: got code %s, want %s", c.name, got.Code, c.wantCode)
		}
		if got.Title == "" || got.Praise == "" {
			t.Errorf("%s: missing title or praise", c.name)
		}
	}
}
