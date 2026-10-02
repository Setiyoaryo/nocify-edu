package questions

import (
	"errors"
	"fmt"
)

//go:generate go run ./gen

// Figure is one pictured object in a question: an icon from the approved set,
// repeated Count times, optionally tinted. Questions carry no emoji or free
// images; the web client renders each name from its licensed icon set
// (apps/web/src/lib/figures.ts must cover every name here — a test enforces it).
type Figure struct {
	Icon  string `json:"icon"`
	Count int    `json:"count,omitempty"`
	Color string `json:"color,omitempty"`
}

// Icons is generated from packages/figures/figures.json (see figures_gen.go);
// that file is the single source of truth shared with the web client and the
// Python pipeline.

var Colors = []string{"red", "orange", "yellow", "green", "blue", "purple", "pink", "brown", "gray"}

var iconSet, colorSet = toSet(Icons), toSet(Colors)

func toSet(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func ValidIcon(name string) bool { return iconSet[name] }

func validateFigures(fs []Figure, where string) error {
	total := 0
	for _, f := range fs {
		if !iconSet[f.Icon] {
			return fmt.Errorf("%s: unknown icon %q", where, f.Icon)
		}
		if f.Color != "" && !colorSet[f.Color] {
			return fmt.Errorf("%s: unknown color %q", where, f.Color)
		}
		if f.Count < 0 || f.Count > 12 {
			return fmt.Errorf("%s: count must be 1–12", where)
		}
		total += max(f.Count, 1)
	}
	if total > 24 {
		return errors.New(where + ": at most 24 figures")
	}
	return nil
}
