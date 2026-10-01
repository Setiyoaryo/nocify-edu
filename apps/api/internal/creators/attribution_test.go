package creators

import (
	"math"
	"testing"
)

func TestQualityScore(t *testing.T) {
	cases := []struct {
		name string
		in   Stats
		want float64
	}{
		{"no usage", Stats{}, 0},
		{"served but never answered", Stats{Usage: 5}, 0},
		{"perfect, no hints", Stats{Usage: 4, Completion: 4, Correct: 4, FirstTry: 4}, 100},
		{"all wrong, no hints", Stats{Usage: 4, Completion: 4}, 20},
		{"half correct, half first try, one hint of three",
			Stats{Usage: 3, Completion: 2, Correct: 1, FirstTry: 1, Hint: 1}, 53.33},
		{"hint-heavy clamps hint-free at zero",
			Stats{Usage: 2, Completion: 2, Correct: 2, FirstTry: 2, Hint: 5}, 80},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := QualityScore(tc.in)
			if math.Abs(got-tc.want) > 0.001 {
				t.Fatalf("QualityScore(%+v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
