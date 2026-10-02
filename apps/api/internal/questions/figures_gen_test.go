package questions

import (
	"encoding/json"
	"os"
	"testing"
)

// figuresJSON mirrors the generator's path: packages/figures/figures.json is the
// single source of truth, and figures_gen.go must match it exactly.
const figuresJSON = "../../../../packages/figures/figures.json"

type figureEntry struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// TestIconsMatchSharedVocabulary fails when figures_gen.go is stale. Regenerate
// with `go generate ./internal/questions/...` from apps/api.
func TestIconsMatchSharedVocabulary(t *testing.T) {
	raw, err := os.ReadFile(figuresJSON)
	if err != nil {
		t.Fatalf("read %s: %v", figuresJSON, err)
	}
	var want []figureEntry
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("parse %s: %v", figuresJSON, err)
	}
	if len(want) != len(Icons) {
		t.Fatalf("figures_gen.go has %d icons, figures.json has %d — run `go generate ./internal/questions/...`",
			len(Icons), len(want))
	}
	for i, f := range want {
		if Icons[i] != f.Name {
			t.Fatalf("icon %d: figures_gen.go has %q, figures.json has %q — run `go generate ./internal/questions/...`",
				i, Icons[i], f.Name)
		}
		if !colorSet[f.Color] {
			t.Errorf("figure %q: unknown color %q (not in Colors)", f.Name, f.Color)
		}
		if f.Label == "" {
			t.Errorf("figure %q: empty label", f.Name)
		}
	}
}
