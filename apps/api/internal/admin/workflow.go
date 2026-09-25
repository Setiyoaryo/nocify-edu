// Package admin is the content CMS: curriculum management, question authoring
// with an editorial workflow and quality gate (§23–24), audit log and stats.
package admin

import (
	"errors"
	"fmt"

	"github.com/nocify/eduplay/api/internal/auth"
)

// QualityChecklist is the minimum review of §24. Every item must be ticked to approve.
var QualityChecklist = []struct{ Key, Label string }{
	{"academic_answer", "Jawaban benar sudah diverifikasi"},
	{"academic_explanation", "Penjelasan sudah diverifikasi"},
	{"academic_curriculum", "Sesuai kurikulum"},
	{"academic_difficulty", "Tingkat kesulitan wajar"},
	{"ux_instruction", "Instruksi dapat dipahami anak"},
	{"ux_readable", "Teks mudah dibaca"},
	{"ux_touch", "Target sentuh cukup besar"},
	{"ux_visual", "Visual tidak ambigu"},
	{"a11y_color", "Makna tidak hanya disampaikan lewat warna"},
	{"a11y_alternatives", "Alternatif audio/teks tersedia bila perlu"},
	{"a11y_contrast", "Kontras cukup"},
}

type TransitionCtx struct {
	From         string
	To           string
	Actor        auth.User
	AuthorOfHead string // user id that created the current version
	Checklist    map[string]bool
	MediaReady   bool // all linked media assets are published
}

// CheckQuestionTransition enforces the editorial workflow. Pure for testing.
//
//	draft → review → approved → published → retired → draft
//	review → draft (changes requested), approved → draft
func CheckQuestionTransition(t TransitionCtx) error {
	reviewer := t.Actor.Is(auth.RoleReviewer, auth.RoleAdmin)
	switch {
	case t.From == "draft" && t.To == "review":
		return nil
	case t.From == "review" && t.To == "draft", t.From == "approved" && t.To == "draft":
		if !reviewer {
			return errors.New("only reviewers can request changes")
		}
		return nil
	case t.From == "review" && t.To == "approved":
		if !reviewer {
			return errors.New("only reviewers can approve")
		}
		if t.AuthorOfHead == t.Actor.ID.String() && t.Actor.Role != auth.RoleAdmin {
			return errors.New("the author of this version cannot approve it; ask another reviewer")
		}
		for _, item := range QualityChecklist {
			if !t.Checklist[item.Key] {
				return fmt.Errorf("quality gate: %q must be checked", item.Label)
			}
		}
		return nil
	case t.From == "approved" && t.To == "published":
		if !reviewer {
			return errors.New("only reviewers can publish")
		}
		if !t.MediaReady {
			return errors.New("all linked media must be published assets (brand asset policy)")
		}
		return nil
	case t.From == "published" && t.To == "retired":
		if !reviewer {
			return errors.New("only reviewers can retire")
		}
		return nil
	case t.From == "retired" && t.To == "draft":
		return nil
	}
	return fmt.Errorf("cannot move from %s to %s", t.From, t.To)
}
