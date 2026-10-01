package candidates

import (
	"strings"
	"testing"
	"time"

	"github.com/nocify/eduplay/api/internal/admin"
)

func fullChecklist() map[string]bool {
	c := map[string]bool{}
	for _, item := range admin.QualityChecklist {
		c[item.Key] = true
	}
	return c
}

func TestValidateDecision(t *testing.T) {
	tests := []struct {
		name      string
		kind      string
		decision  string
		checklist map[string]bool
		wantErr   bool
	}{
		{"question approve with full checklist", KindQuestion, DecisionApproved, fullChecklist(), false},
		{"question approve with empty checklist", KindQuestion, DecisionApproved, map[string]bool{}, true},
		{"question approve with partial checklist", KindQuestion, DecisionApproved, map[string]bool{"academic_answer": true}, true},
		{"material approve with full checklist", KindMaterial, DecisionApproved, fullChecklist(), false},
		{"material approve with empty checklist", KindMaterial, DecisionApproved, nil, true},
		{"reject needs no checklist", KindQuestion, DecisionRejected, nil, false},
		{"changes requested needs no checklist", KindMaterial, DecisionChangesRequested, nil, false},
		{"unknown decision", KindQuestion, "maybe", fullChecklist(), true},
		{"unknown kind", "blogpost", DecisionRejected, nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDecision(tc.kind, tc.decision, tc.checklist)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateDecision() err = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestDecisionStatus(t *testing.T) {
	tests := []struct {
		kind        string
		decision    string
		wantStatus  string
		publishable bool
	}{
		{KindQuestion, DecisionApproved, StatusQueued, true},
		{KindMaterial, DecisionApproved, StatusNotApplicable, false},
		{KindQuestion, DecisionRejected, StatusRejected, false},
		{KindMaterial, DecisionRejected, StatusRejected, false},
		{KindQuestion, DecisionChangesRequested, StatusChangesRequested, false},
	}
	for _, tc := range tests {
		status, publishable := DecisionStatus(tc.kind, tc.decision)
		if status != tc.wantStatus || publishable != tc.publishable {
			t.Errorf("DecisionStatus(%q,%q) = (%q,%v), want (%q,%v)",
				tc.kind, tc.decision, status, publishable, tc.wantStatus, tc.publishable)
		}
	}
}

func TestBuildSourceString(t *testing.T) {
	cp := 7
	p := Provenance{
		CPID:        &cp,
		SkillID:     3,
		SourceCode:  "kemendikdasmen-kurikulum",
		SourceTitle: "Permendikdasmen 13/2025",
		OriginalURL: "https://example.go.id/dokumen",
		RetrievedAt: time.Date(2026, 9, 30, 14, 0, 0, 0, time.UTC),
		ContentHash: "abcdef0123456789",
	}
	got := BuildSourceString(p)
	for _, want := range []string{"Permendikdasmen 13/2025", "https://example.go.id/dokumen", "diambil 2026-09-30", "chunk:abcdef012345"} {
		if !strings.Contains(got, want) {
			t.Errorf("BuildSourceString() = %q, missing %q", got, want)
		}
	}
	if strings.Contains(got, "abcdef0123456789") {
		t.Errorf("BuildSourceString() should truncate the hash: %q", got)
	}
	if BuildSourceString(Provenance{SourceCode: "only-code"}) != "only-code" {
		t.Errorf("unexpected fallback: %q", BuildSourceString(Provenance{SourceCode: "only-code"}))
	}
}

func TestCandidatePayloadQuestionInput(t *testing.T) {
	p := CandidatePayload{
		Code: "Q-TEST-1", Skill: "SD-B-01", Type: "short_answer", Difficulty: 3,
		LearningObjective: "Peserta didik dapat perkalian.",
		Content:           []byte(`{"prompt":"x"}`),
		Answer:            []byte(`{"accepted":["y"]}`),
		Explanation:       "karena",
	}
	in := p.QuestionInput(42, "sumber")
	if in.Code != p.Code || in.SkillID != 42 || in.Type != p.Type || in.Difficulty != 3 || in.Source != "sumber" {
		t.Fatalf("QuestionInput() = %+v", in)
	}
	if string(in.Content) != `{"prompt":"x"}` || string(in.Answer) != `{"accepted":["y"]}` || in.Explanation != "karena" {
		t.Fatalf("QuestionInput() payload mismatch: %+v", in)
	}
}

func TestValidKind(t *testing.T) {
	if !ValidKind(KindMaterial) || !ValidKind(KindQuestion) || ValidKind("other") {
		t.Fatal("ValidKind() wrong")
	}
}
