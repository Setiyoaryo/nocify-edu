package questions

import (
	"encoding/json"
	"testing"
)

func raw(s string) json.RawMessage { return json.RawMessage(s) }

func TestValidateAndGrade(t *testing.T) {
	cases := []struct {
		typ, content, answer string
		right, wrong         string
	}{
		{"single_choice", `{"prompt":"2+2?","options":[{"id":"a","label":"3"},{"id":"b","label":"4"}]}`, `{"option_id":"b"}`,
			`{"option_id":"b"}`, `{"option_id":"a"}`},
		{"image_choice", `{"prompt":"Mana lebih banyak?","options":[{"id":"a","label":"","figures":[{"icon":"orange","count":3}]},{"id":"b","label":"","figures":[{"icon":"orange","count":5}]}]}`, `{"option_id":"b"}`,
			`{"option_id":"b"}`, `{"option_id":"z"}`},
		{"tap_count", `{"prompt":"Berapa jeruk?","symbol":"orange","count":6,"choices":[5,6,7]}`, `{"value":6}`,
			`{"value":6}`, `{"value":5}`},
		{"number_input", `{"prompt":"7+5="}`, `{"value":12}`, `{"value":12}`, `{"value":11}`},
		{"short_answer", `{"prompt":"Hewan yang mengeong?"}`, `{"accepted":["kucing"]}`, `{"text":"  Kucing "}`, `{"text":""}`},
		{"sequence", `{"prompt":"Urutkan","items":[{"id":"1","label":"satu"},{"id":"2","label":"dua"},{"id":"3","label":"tiga"}]}`, `{"order":["1","2","3"]}`,
			`{"order":["1","2","3"]}`, `{"order":["2","1","3"]}`},
		{"matching", `{"prompt":"Pasangkan","left":[{"id":"l1","label":"Sapi","icon":"cow"},{"id":"l2","label":"Burung","icon":"bird"}],"right":[{"id":"r1","label":"susu"},{"id":"r2","label":"telur"}]}`, `{"pairs":{"l1":"r1","l2":"r2"}}`,
			`{"pairs":{"l1":"r1","l2":"r2"}}`, `{"pairs":{"l1":"r2","l2":"r1"}}`},
	}
	for _, c := range cases {
		if err := Validate(c.typ, raw(c.content), raw(c.answer)); err != nil {
			t.Fatalf("%s: Validate: %v", c.typ, err)
		}
		if ok, err := Grade(c.typ, raw(c.content), raw(c.answer), raw(c.right)); err != nil || !ok {
			t.Errorf("%s: right answer graded %v (%v)", c.typ, ok, err)
		}
		if ok, _ := Grade(c.typ, raw(c.content), raw(c.answer), raw(c.wrong)); ok {
			t.Errorf("%s: wrong answer graded correct", c.typ)
		}
		if ok, _ := Grade(c.typ, raw(c.content), raw(c.answer), raw(`"garbage"`)); ok {
			t.Errorf("%s: malformed response graded correct", c.typ)
		}
	}
}

func TestValidateRejects(t *testing.T) {
	bad := []struct{ typ, content, answer string }{
		{"nope", `{"prompt":"x"}`, `{}`},
		{"single_choice", `{"prompt":""}`, `{}`},
		{"single_choice", `{"prompt":"x","options":[{"id":"a","label":"1"},{"id":"a","label":"2"}]}`, `{"option_id":"a"}`},
		{"single_choice", `{"prompt":"x","options":[{"id":"a","label":"1"},{"id":"b","label":"2"}]}`, `{"option_id":"c"}`},
		{"image_choice", `{"prompt":"x","options":[{"id":"a","label":"1"},{"id":"b","label":"2"}]}`, `{"option_id":"a"}`},
		{"tap_count", `{"prompt":"x","symbol":"star","count":4,"choices":[1,2]}`, `{"value":4}`},
		{"tap_count", `{"prompt":"x","symbol":"🍎","count":2,"choices":[1,2]}`, `{"value":2}`}, // emoji are not icons
		{"image_choice", `{"prompt":"x","options":[{"id":"a","label":"","figures":[{"icon":"unicorn"}]},{"id":"b","label":"","figures":[{"icon":"star"}]}]}`, `{"option_id":"a"}`},
		{"single_choice", `{"prompt":"x","figures":[{"icon":"star","color":"neon"}],"options":[{"id":"a","label":"1"},{"id":"b","label":"2"}]}`, `{"option_id":"a"}`},
		{"sequence", `{"prompt":"x","items":[{"id":"1","label":"a"},{"id":"2","label":"b"}]}`, `{"order":["1","1"]}`},
		{"single_choice", `{"prompt":"x","unknown":1}`, `{}`},
		{"single_choice", `{"prompt":"x","image":"/m/a.svg","options":[{"id":"a","label":"1"},{"id":"b","label":"2"}]}`, `{"option_id":"a"}`},
	}
	for i, c := range bad {
		if err := Validate(c.typ, raw(c.content), raw(c.answer)); err == nil {
			t.Errorf("case %d (%s): expected validation error", i, c.typ)
		}
	}
}

func TestPublicContentShufflesDeterministically(t *testing.T) {
	content := raw(`{"prompt":"x","items":[{"id":"1","label":"a"},{"id":"2","label":"b"},{"id":"3","label":"c"},{"id":"4","label":"d"}]}`)
	a, _ := PublicContent("sequence", content, "item-1")
	b, _ := PublicContent("sequence", content, "item-1")
	if string(a) != string(b) {
		t.Fatal("shuffle must be stable for the same seed")
	}
	var c Content
	_ = json.Unmarshal(a, &c)
	if c.Items[0].ID == "1" && c.Items[1].ID == "2" && c.Items[2].ID == "3" {
		t.Fatal("public order must not equal the answer order")
	}
}
