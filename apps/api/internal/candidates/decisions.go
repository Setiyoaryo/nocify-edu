// Package candidates is the reviewer CMS backend for content-pipeline
// candidates (§36–38, §41): it lists the material/question candidates the
// Python pipeline produced, shows a reviewer the source they came from, records
// the review decision, and publishes approved questions into canonical tables.
//
// Go only ever READS `content_staging`; the decision and publish ledger live in
// canonical `candidate_publications` (roadmap D2/D3).
package candidates

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nocify/eduplay/api/internal/admin"
)

const (
	KindMaterial = "material"
	KindQuestion = "question"
)

const (
	DecisionApproved         = "approved"
	DecisionRejected         = "rejected"
	DecisionChangesRequested = "changes_requested"
)

// candidate_publications.status values.
const (
	StatusQueued           = "queued"
	StatusPublishing       = "publishing"
	StatusPublished        = "published"
	StatusFailed           = "failed"
	StatusNotApplicable    = "not_applicable"
	StatusRejected         = "rejected"
	StatusChangesRequested = "changes_requested"
)

func ValidKind(kind string) bool { return kind == KindMaterial || kind == KindQuestion }

// DecisionStatus maps a review decision to the ledger status. publishable is
// true only when the row should enter the publish queue: questions do, materials
// are review-only in M5 (no canonical destination yet).
func DecisionStatus(kind, decision string) (status string, publishable bool) {
	switch decision {
	case DecisionApproved:
		if kind == KindQuestion {
			return StatusQueued, true
		}
		return StatusNotApplicable, false
	case DecisionRejected:
		return StatusRejected, false
	case DecisionChangesRequested:
		return StatusChangesRequested, false
	default:
		return "", false
	}
}

// ValidateDecision checks the review decision and, for approvals, the quality
// gate. Approving requires the same 11-item checklist as canonical questions
// (§24); rejecting or requesting changes does not.
func ValidateDecision(kind, decision string, checklist map[string]bool) error {
	if !ValidKind(kind) {
		return fmt.Errorf("unknown candidate kind %q", kind)
	}
	switch decision {
	case DecisionApproved:
		return admin.ChecklistComplete(checklist)
	case DecisionRejected, DecisionChangesRequested:
		return nil
	default:
		return fmt.Errorf("unknown decision %q", decision)
	}
}

// Provenance is the canonical copy of where a question came from (D4, §38).
type Provenance struct {
	CPID             *int
	SkillID          int
	SourceCode       string
	SourceDocumentID int64
	SourceChunkID    int64
	OriginalURL      string
	SourceTitle      string
	RetrievedAt      time.Time
	ContentHash      string
}

// BuildSourceString renders the human-readable provenance kept on
// questions.source (the canonical, free-text provenance field).
func BuildSourceString(p Provenance) string {
	var parts []string
	if p.SourceTitle != "" {
		parts = append(parts, p.SourceTitle)
	} else if p.SourceCode != "" {
		parts = append(parts, p.SourceCode)
	}
	if p.OriginalURL != "" {
		parts = append(parts, p.OriginalURL)
	}
	if !p.RetrievedAt.IsZero() {
		parts = append(parts, "diambil "+p.RetrievedAt.UTC().Format("2006-01-02"))
	}
	if p.ContentHash != "" {
		hash := p.ContentHash
		if len(hash) > 12 {
			hash = hash[:12]
		}
		parts = append(parts, "chunk:"+hash)
	}
	return strings.Join(parts, " · ")
}

// CandidatePayload is the question envelope stored in
// question_candidates.payload (packages/contracts/question-candidate.schema.json).
type CandidatePayload struct {
	Code              string          `json:"code"`
	Skill             string          `json:"skill"`
	Type              string          `json:"type"`
	Difficulty        int             `json:"difficulty"`
	LearningObjective string          `json:"learning_objective"`
	Content           json.RawMessage `json:"content"`
	Answer            json.RawMessage `json:"answer"`
	Explanation       string          `json:"explanation"`
	Hint              string          `json:"hint"`
}

// QuestionInput adapts the candidate envelope to the canonical authoring input.
func (p CandidatePayload) QuestionInput(skillID int, source string) admin.QuestionInput {
	return admin.QuestionInput{
		Code:              p.Code,
		SkillID:           skillID,
		Type:              p.Type,
		Difficulty:        p.Difficulty,
		LearningObjective: p.LearningObjective,
		Source:            source,
		Content:           p.Content,
		Answer:            p.Answer,
		Explanation:       p.Explanation,
		Hint:              p.Hint,
	}
}
