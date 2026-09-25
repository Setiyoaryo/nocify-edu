package admin

import (
	"testing"

	"github.com/google/uuid"

	"github.com/nocify/eduplay/api/internal/auth"
)

func fullChecklist() map[string]bool {
	m := map[string]bool{}
	for _, c := range QualityChecklist {
		m[c.Key] = true
	}
	return m
}

func TestQuestionWorkflow(t *testing.T) {
	editor := auth.User{ID: uuid.New(), Role: auth.RoleEditor}
	reviewer := auth.User{ID: uuid.New(), Role: auth.RoleReviewer}
	admin := auth.User{ID: uuid.New(), Role: auth.RoleAdmin}

	ok := []TransitionCtx{
		{From: "draft", To: "review", Actor: editor},
		{From: "review", To: "approved", Actor: reviewer, AuthorOfHead: editor.ID.String(), Checklist: fullChecklist()},
		{From: "approved", To: "published", Actor: reviewer, MediaReady: true},
		{From: "published", To: "retired", Actor: reviewer},
		{From: "retired", To: "draft", Actor: editor},
		{From: "review", To: "draft", Actor: reviewer},
		// admins may approve their own work (small teams), still with the full checklist
		{From: "review", To: "approved", Actor: admin, AuthorOfHead: admin.ID.String(), Checklist: fullChecklist()},
	}
	for _, c := range ok {
		if err := CheckQuestionTransition(c); err != nil {
			t.Errorf("%s→%s by %s: %v", c.From, c.To, c.Actor.Role, err)
		}
	}

	partial := fullChecklist()
	partial["academic_answer"] = false
	bad := []TransitionCtx{
		{From: "draft", To: "published", Actor: admin},                                                                    // no skipping review
		{From: "review", To: "approved", Actor: editor, Checklist: fullChecklist()},                                       // editor can't approve
		{From: "review", To: "approved", Actor: reviewer, AuthorOfHead: reviewer.ID.String(), Checklist: fullChecklist()}, // self-approval
		{From: "review", To: "approved", Actor: reviewer, AuthorOfHead: editor.ID.String(), Checklist: partial},           // incomplete gate
		{From: "approved", To: "published", Actor: reviewer, MediaReady: false},                                           // unpublished asset
		{From: "approved", To: "published", Actor: editor, MediaReady: true},
	}
	for i, c := range bad {
		if err := CheckQuestionTransition(c); err == nil {
			t.Errorf("case %d %s→%s by %s should fail", i, c.From, c.To, c.Actor.Role)
		}
	}
}
