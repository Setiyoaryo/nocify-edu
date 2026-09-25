// Package questions defines question types, validates authored content and
// grades responses. Each type is a (validator, grader) pair so new types plug in
// without touching the assessment engine (§9, §15). Grading is server-side only:
// answers never leave the API before a response is submitted.
package questions

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"strings"
)

type Option struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Figures []Figure `json:"figures,omitempty"`
	Image   string   `json:"image,omitempty"` // media URL of a published asset
	Alt     string   `json:"alt,omitempty"`
}

type Item struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Icon  string `json:"icon,omitempty"`
	Color string `json:"color,omitempty"`
}

// Content is the union of all type payloads. Unused fields stay empty.
type Content struct {
	Prompt    string   `json:"prompt"`
	AudioText string   `json:"audio_text,omitempty"` // read aloud via speech synthesis when set
	Figures   []Figure `json:"figures,omitempty"`    // pictured stimulus, e.g. 3 oranges
	Image     string   `json:"image,omitempty"`
	ImageAlt  string   `json:"image_alt,omitempty"`
	Options   []Option `json:"options,omitempty"`
	Symbol    string   `json:"symbol,omitempty"` // tap_count: icon of the thing to count
	Color     string   `json:"color,omitempty"`  // tap_count: its tint
	Count     int      `json:"count,omitempty"`  // tap_count: how many are shown
	Choices   []int    `json:"choices,omitempty"`
	Unit      string   `json:"unit,omitempty"`
	Items     []Item   `json:"items,omitempty"` // sequence
	Left      []Item   `json:"left,omitempty"`  // matching
	Right     []Item   `json:"right,omitempty"`
}

// Answer / Response share a shape; which fields matter depends on the type.
type Answer struct {
	OptionID string            `json:"option_id,omitempty"`
	Value    *float64          `json:"value,omitempty"`
	Accepted []string          `json:"accepted,omitempty"`
	Text     string            `json:"text,omitempty"`
	Order    []string          `json:"order,omitempty"`
	Pairs    map[string]string `json:"pairs,omitempty"`
}

type Type struct {
	Code     string   `json:"code"`
	Label    string   `json:"label"`
	Bands    []string `json:"bands"` // UX bands the type suits
	validate func(c Content, a Answer) error
	grade    func(c Content, a Answer, resp Answer) bool
}

var registry = map[string]Type{}
var order []string

func register(t Type) {
	registry[t.Code] = t
	order = append(order, t.Code)
}

func Types() []Type {
	out := make([]Type, 0, len(order))
	for _, c := range order {
		out = append(out, registry[c])
	}
	return out
}

func Known(code string) bool { _, ok := registry[code]; return ok }

func init() {
	choiceValidate := func(needVisual bool) func(Content, Answer) error {
		return func(c Content, a Answer) error {
			if len(c.Options) < 2 || len(c.Options) > 6 {
				return errors.New("options: 2–6 required")
			}
			seen := map[string]bool{}
			for _, o := range c.Options {
				if o.ID == "" || seen[o.ID] {
					return errors.New("options: ids must be unique and non-empty")
				}
				seen[o.ID] = true
				if needVisual && len(o.Figures) == 0 && o.Image == "" {
					return fmt.Errorf("option %s: figures or image required", o.ID)
				}
				if err := validateFigures(o.Figures, "option "+o.ID); err != nil {
					return err
				}
				if !needVisual && o.Label == "" {
					return fmt.Errorf("option %s: label required", o.ID)
				}
				if o.Image != "" && o.Alt == "" {
					return fmt.Errorf("option %s: alt text required for image", o.ID)
				}
			}
			if !seen[a.OptionID] {
				return errors.New("answer.option_id must match an option")
			}
			return nil
		}
	}
	choiceGrade := func(_ Content, a, r Answer) bool { return r.OptionID != "" && r.OptionID == a.OptionID }
	numberGrade := func(_ Content, a, r Answer) bool {
		return a.Value != nil && r.Value != nil && math.Abs(*a.Value-*r.Value) < 1e-9
	}

	register(Type{Code: "single_choice", Label: "Pilihan ganda", Bands: []string{"lower", "upper", "secondary"},
		validate: choiceValidate(false), grade: choiceGrade})
	register(Type{Code: "image_choice", Label: "Pilih gambar", Bands: []string{"early", "lower"},
		validate: choiceValidate(true), grade: choiceGrade})
	register(Type{Code: "tap_count", Label: "Hitung dengan ketuk", Bands: []string{"early", "lower"},
		validate: func(c Content, a Answer) error {
			if !ValidIcon(c.Symbol) || c.Count < 1 || c.Count > 20 {
				return errors.New("tap_count: symbol (approved icon) and count (1–20) required")
			}
			if c.Color != "" && !colorSet[c.Color] {
				return fmt.Errorf("tap_count: unknown color %q", c.Color)
			}
			if len(c.Choices) < 2 || len(c.Choices) > 5 {
				return errors.New("tap_count: 2–5 choices required")
			}
			if a.Value == nil || *a.Value != float64(c.Count) {
				return errors.New("tap_count: answer.value must equal count")
			}
			found := false
			for _, ch := range c.Choices {
				found = found || ch == c.Count
			}
			if !found {
				return errors.New("tap_count: choices must include the count")
			}
			return nil
		}, grade: numberGrade})
	register(Type{Code: "number_input", Label: "Isian angka", Bands: []string{"lower", "upper", "secondary"},
		validate: func(c Content, a Answer) error {
			if a.Value == nil {
				return errors.New("answer.value required")
			}
			return nil
		}, grade: numberGrade})
	register(Type{Code: "short_answer", Label: "Isian singkat", Bands: []string{"upper", "secondary"},
		validate: func(c Content, a Answer) error {
			if len(a.Accepted) == 0 {
				return errors.New("answer.accepted: at least one accepted answer")
			}
			return nil
		}, grade: func(_ Content, a, r Answer) bool {
			got := normText(r.Text)
			for _, acc := range a.Accepted {
				if got != "" && got == normText(acc) {
					return true
				}
			}
			return false
		}})
	register(Type{Code: "sequence", Label: "Urutkan", Bands: []string{"early", "lower", "upper", "secondary"},
		validate: func(c Content, a Answer) error {
			if len(c.Items) < 2 || len(c.Items) > 6 {
				return errors.New("items: 2–6 required")
			}
			if err := sameIDs(c.Items, a.Order); err != nil {
				return fmt.Errorf("answer.order: %w", err)
			}
			return nil
		}, grade: func(_ Content, a, r Answer) bool {
			if len(r.Order) != len(a.Order) {
				return false
			}
			for i := range a.Order {
				if a.Order[i] != r.Order[i] {
					return false
				}
			}
			return true
		}})
	register(Type{Code: "matching", Label: "Pasangkan", Bands: []string{"lower", "upper", "secondary"},
		validate: func(c Content, a Answer) error {
			if len(c.Left) < 2 || len(c.Left) > 5 || len(c.Right) != len(c.Left) {
				return errors.New("left/right: 2–5 items each, same length")
			}
			rights := map[string]bool{}
			for _, it := range c.Right {
				rights[it.ID] = true
			}
			if len(a.Pairs) != len(c.Left) {
				return errors.New("answer.pairs must pair every left item")
			}
			used := map[string]bool{}
			for _, l := range c.Left {
				rid, ok := a.Pairs[l.ID]
				if !ok || !rights[rid] || used[rid] {
					return fmt.Errorf("answer.pairs[%s] invalid", l.ID)
				}
				used[rid] = true
			}
			return nil
		}, grade: func(_ Content, a, r Answer) bool {
			if len(r.Pairs) != len(a.Pairs) {
				return false
			}
			for k, v := range a.Pairs {
				if r.Pairs[k] != v {
					return false
				}
			}
			return true
		}})
}

func sameIDs(items []Item, ids []string) error {
	if len(items) != len(ids) {
		return errors.New("must list every item exactly once")
	}
	want := map[string]bool{}
	for _, it := range items {
		if it.ID == "" || it.Label == "" || want[it.ID] {
			return errors.New("items need unique id and label")
		}
		want[it.ID] = true
	}
	for _, id := range ids {
		if !want[id] {
			return errors.New("must list every item exactly once")
		}
		delete(want, id)
	}
	return nil
}

func normText(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}

// Validate checks authored content against its type. Used by the CMS and seeder.
func Validate(typ string, rawContent, rawAnswer json.RawMessage) error {
	t, ok := registry[typ]
	if !ok {
		return fmt.Errorf("unknown question type %q", typ)
	}
	var c Content
	if err := strictUnmarshal(rawContent, &c); err != nil {
		return fmt.Errorf("content: %w", err)
	}
	if strings.TrimSpace(c.Prompt) == "" {
		return errors.New("content.prompt required")
	}
	if c.Image != "" && c.ImageAlt == "" {
		return errors.New("content.image_alt required when image is set")
	}
	if err := validateFigures(c.Figures, "content.figures"); err != nil {
		return err
	}
	for _, list := range [][]Item{c.Items, c.Left, c.Right} {
		for _, it := range list {
			if it.Icon != "" && !ValidIcon(it.Icon) {
				return fmt.Errorf("item %s: unknown icon %q", it.ID, it.Icon)
			}
			if it.Color != "" && !colorSet[it.Color] {
				return fmt.Errorf("item %s: unknown color %q", it.ID, it.Color)
			}
		}
	}
	var a Answer
	if err := strictUnmarshal(rawAnswer, &a); err != nil {
		return fmt.Errorf("answer: %w", err)
	}
	return t.validate(c, a)
}

func strictUnmarshal(raw json.RawMessage, v any) error {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// Grade returns whether resp is correct. Malformed responses are simply wrong.
func Grade(typ string, rawContent, rawAnswer, rawResp json.RawMessage) (bool, error) {
	t, ok := registry[typ]
	if !ok {
		return false, fmt.Errorf("unknown question type %q", typ)
	}
	var c Content
	var a, r Answer
	if err := json.Unmarshal(rawContent, &c); err != nil {
		return false, err
	}
	if err := json.Unmarshal(rawAnswer, &a); err != nil {
		return false, err
	}
	if err := json.Unmarshal(rawResp, &r); err != nil {
		return false, nil
	}
	return t.grade(c, a, r), nil
}

// PublicContent prepares content for the child client: sequence items and
// matching right-hand items are shuffled deterministically per assessment item
// (so a reload shows the same order, but never the answer order).
func PublicContent(typ string, raw json.RawMessage, seed string) (json.RawMessage, error) {
	if typ != "sequence" && typ != "matching" {
		return raw, nil
	}
	var c Content
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	h := fnv.New64a()
	h.Write([]byte(seed))
	rng := rand.New(rand.NewSource(int64(h.Sum64())))
	shuffle := func(items []Item) {
		orig := append([]Item(nil), items...)
		for tries := 0; tries < 5; tries++ {
			rng.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
			if !sameOrder(orig, items) {
				return
			}
		}
	}
	if typ == "sequence" {
		shuffle(c.Items)
	} else {
		shuffle(c.Right)
	}
	return json.Marshal(c)
}

func sameOrder(a, b []Item) bool {
	for i := range a {
		if a[i].ID != b[i].ID {
			return false
		}
	}
	return true
}
