package candidates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/admin"
	"github.com/nocify/eduplay/api/internal/audit"
	"github.com/nocify/eduplay/api/internal/creators"
	"github.com/nocify/eduplay/api/internal/questions"
)

// PublishOptions configures a publish run.
type PublishOptions struct {
	Limit       int  // rows per batch
	All         bool // keep draining the queue until empty
	DryRun      bool // validate only, write nothing
	RetryFailed bool // re-queue failed rows first
	MaxAttempts int  // with RetryFailed: only retry rows with attempts < MaxAttempts (0 = no limit)
}

// PublishReport summarises a run.
type PublishReport struct {
	Claimed   int      `json:"claimed"`
	Published int      `json:"published"`
	Linked    int      `json:"linked"`
	Failed    int      `json:"failed"`
	Skipped   int      `json:"skipped"`
	Remaining int      `json:"remaining"`
	DryRun    bool     `json:"dry_run"`
	Errors    []string `json:"errors,omitempty"`
}

// Publish claims queued, approved question candidates and writes them to the
// canonical questions tables with their provenance. Safe to re-run: an existing
// question with the same code and provenance is linked, not duplicated.
func Publish(ctx context.Context, pool *pgxpool.Pool, opt PublishOptions) (PublishReport, error) {
	rep := PublishReport{DryRun: opt.DryRun}
	if opt.Limit <= 0 {
		opt.Limit = 50
	}
	if err := reapStale(ctx, pool); err != nil {
		return rep, err
	}
	if opt.RetryFailed {
		if err := requeueFailed(ctx, pool, opt.MaxAttempts); err != nil {
			return rep, err
		}
	}
	for {
		ids, err := queuedIDs(ctx, pool, opt.Limit)
		if err != nil {
			return rep, err
		}
		if len(ids) == 0 {
			break
		}
		for _, id := range ids {
			rep.Claimed++
			outcome, err := publishOne(ctx, pool, id, opt.DryRun)
			if err != nil {
				return rep, err
			}
			switch outcome {
			case "published":
				rep.Published++
			case "linked":
				rep.Linked++
			case "skipped":
				rep.Skipped++
			case "failed":
				rep.Failed++
				rep.Errors = append(rep.Errors, fmt.Sprintf("publication %d failed (see last_error)", id))
			}
		}
		if !opt.All || opt.DryRun {
			break
		}
	}
	if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM candidate_publications WHERE kind = 'question' AND status = 'queued'`).Scan(&rep.Remaining); err != nil {
		return rep, err
	}
	return rep, nil
}

func reapStale(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `UPDATE candidate_publications SET status = 'queued', updated_at = now()
		WHERE kind = 'question' AND status = 'publishing' AND updated_at < now() - interval '15 minutes'`)
	return err
}

func requeueFailed(ctx context.Context, pool *pgxpool.Pool, maxAttempts int) error {
	_, err := pool.Exec(ctx, `UPDATE candidate_publications SET status = 'queued', updated_at = now()
		WHERE kind = 'question' AND status = 'failed' AND ($1 = 0 OR attempts < $1)`, maxAttempts)
	return err
}

func queuedIDs(ctx context.Context, pool *pgxpool.Pool, limit int) ([]int64, error) {
	rows, err := pool.Query(ctx, `SELECT id FROM candidate_publications
		WHERE kind = 'question' AND status = 'queued' ORDER BY updated_at, id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// publishOne processes a single queued publication in its own transaction.
func publishOne(ctx context.Context, pool *pgxpool.Pool, pubID int64, dryRun bool) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	var candidateID int64
	var decidedBy *uuid.UUID
	var checklist []byte
	var notes string
	err = tx.QueryRow(ctx, `SELECT candidate_id, decided_by, checklist, notes FROM candidate_publications
		WHERE id = $1 AND kind = 'question' AND status = 'queued' FOR UPDATE SKIP LOCKED`, pubID).
		Scan(&candidateID, &decidedBy, &checklist, &notes)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = tx.Rollback(ctx)
		return "skipped", nil
	}
	if err != nil {
		_ = tx.Rollback(ctx)
		return "", err
	}

	if dryRun {
		_, rerr := resolveQuestion(ctx, tx, candidateID)
		_ = tx.Rollback(ctx)
		if rerr != nil {
			return "failed", nil
		}
		return "published", nil
	}

	if _, err := tx.Exec(ctx, `UPDATE candidate_publications SET status = 'publishing', attempts = attempts + 1, updated_at = now() WHERE id = $1`, pubID); err != nil {
		_ = tx.Rollback(ctx)
		return "", err
	}
	qid, linked, perr := publishQuestion(ctx, tx, candidateID, decidedBy, checklist, notes)
	if perr != nil {
		_ = tx.Rollback(ctx)
		if _, ferr := pool.Exec(ctx, `UPDATE candidate_publications SET status = 'failed', last_error = $2, updated_at = now() WHERE id = $1`, pubID, perr.Error()); ferr != nil {
			return "", ferr
		}
		return "failed", nil
	}
	if _, err := tx.Exec(ctx, `UPDATE candidate_publications SET status = 'published', canonical_question_id = $2, published_at = now(), last_error = '', updated_at = now() WHERE id = $1`, pubID, qid); err != nil {
		_ = tx.Rollback(ctx)
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	if linked {
		return "linked", nil
	}
	return "published", nil
}

type resolved struct {
	payload     CandidatePayload
	prov        Provenance
	skillID     int
	cpID        *int
	aiGenerated bool
}

// resolveQuestion loads a candidate with its lineage, validates the payload and
// resolves the canonical skill. It never writes.
func resolveQuestion(ctx context.Context, tx pgx.Tx, candidateID int64) (resolved, error) {
	var r resolved
	var payloadBytes []byte
	var retrievedAt *time.Time
	err := tx.QueryRow(ctx, `
		SELECT q.payload, q.content_hash, q.ai_generated, s.code, d.title, d.canonical_url, d.retrieved_at, d.id, c.id
		FROM content_staging.question_candidates q
		JOIN content_staging.source_documents d ON d.id = q.document_id
		JOIN content_staging.document_chunks c ON c.id = q.chunk_id
		JOIN content_staging.source_registry s ON s.id = q.source_id
		WHERE q.id = $1`, candidateID).
		Scan(&payloadBytes, &r.prov.ContentHash, &r.aiGenerated, &r.prov.SourceCode, &r.prov.SourceTitle,
			&r.prov.OriginalURL, &retrievedAt, &r.prov.SourceDocumentID, &r.prov.SourceChunkID)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, errors.New("candidate no longer exists in staging")
	}
	if err != nil {
		return r, err
	}
	if retrievedAt != nil {
		r.prov.RetrievedAt = *retrievedAt
	}
	if err := json.Unmarshal(payloadBytes, &r.payload); err != nil {
		return r, fmt.Errorf("payload: %w", err)
	}
	if strings.TrimSpace(r.payload.Code) == "" {
		return r, errors.New("payload has no code")
	}
	if r.payload.Difficulty < 1 || r.payload.Difficulty > 5 {
		return r, fmt.Errorf("difficulty %d out of range 1–5", r.payload.Difficulty)
	}
	if err := questions.Validate(r.payload.Type, r.payload.Content, r.payload.Answer); err != nil {
		return r, err
	}
	if err := tx.QueryRow(ctx, `SELECT id, cp_id FROM skills WHERE code = $1`, r.payload.Skill).Scan(&r.skillID, &r.cpID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r, fmt.Errorf("skill %q not found", r.payload.Skill)
		}
		return r, err
	}
	r.prov.SkillID = r.skillID
	r.prov.CPID = r.cpID
	return r, nil
}

// publishQuestion creates or links the canonical question and records provenance.
func publishQuestion(ctx context.Context, tx pgx.Tx, candidateID int64, decidedBy *uuid.UUID, checklist []byte, notes string) (uuid.UUID, bool, error) {
	if decidedBy == nil {
		return uuid.Nil, false, errors.New("candidate has no reviewer (decided_by is NULL)")
	}
	r, err := resolveQuestion(ctx, tx, candidateID)
	if err != nil {
		return uuid.Nil, false, err
	}

	var existing uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM questions WHERE code = $1`, r.payload.Code).Scan(&existing)
	switch {
	case err == nil:
		var existingHash string
		_ = tx.QueryRow(ctx, `SELECT content_hash FROM question_provenance WHERE question_id = $1`, existing).Scan(&existingHash)
		if existingHash != r.prov.ContentHash {
			return uuid.Nil, false, fmt.Errorf("code collision: question %s already exists with different provenance", r.payload.Code)
		}
		if err := insertProvenance(ctx, tx, existing, r.prov); err != nil {
			return uuid.Nil, false, err
		}
		if err := attributeQuestion(ctx, tx, existing, *decidedBy, r.aiGenerated); err != nil {
			return uuid.Nil, false, err
		}
		audit.Log(ctx, tx, *decidedBy, "candidate.publish", "question", existing.String(),
			map[string]any{"candidate_id": candidateID, "linked": true})
		return existing, true, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return uuid.Nil, false, err
	}

	in := r.payload.QuestionInput(r.skillID, BuildSourceString(r.prov))
	qid, err := admin.CreateQuestion(ctx, tx, in, *decidedBy)
	if err != nil {
		return uuid.Nil, false, err
	}
	var vid uuid.UUID
	if err := tx.QueryRow(ctx, `UPDATE questions SET status = 'published', published_version_id = current_version_id,
		published_at = now(), reviewer_id = $2, updated_at = now() WHERE id = $1 RETURNING current_version_id`,
		qid, *decidedBy).Scan(&vid); err != nil {
		return uuid.Nil, false, err
	}
	if err := insertProvenance(ctx, tx, qid, r.prov); err != nil {
		return uuid.Nil, false, err
	}
	if err := attributeQuestion(ctx, tx, qid, *decidedBy, r.aiGenerated); err != nil {
		return uuid.Nil, false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO question_reviews
		(question_id, question_version_id, reviewer_id, from_status, to_status, checklist, notes)
		VALUES ($1, $2, $3, 'approved', 'published', $4, $5)`, qid, vid, *decidedBy, checklist, notes); err != nil {
		return uuid.Nil, false, err
	}
	audit.Log(ctx, tx, *decidedBy, "candidate.publish", "question", qid.String(),
		map[string]any{"candidate_id": candidateID, "linked": false})
	return qid, false, nil
}

// attributeQuestion records creator attribution and adaptation lineage for a
// pipeline-published question (§39). Pipeline content belongs to the system
// creator; the reviewer who approved it is recorded on the lineage. A creator
// already assigned by a human is never overwritten (COALESCE).
func attributeQuestion(ctx context.Context, tx pgx.Tx, questionID, reviewer uuid.UUID, aiGenerated bool) error {
	if _, err := tx.Exec(ctx,
		`UPDATE questions SET creator_id = COALESCE(creator_id, $2) WHERE id = $1`,
		questionID, creators.SystemCreatorID); err != nil {
		return err
	}
	adaptation := "original"
	if aiGenerated {
		adaptation = "ai_assisted"
	}
	_, err := tx.Exec(ctx,
		`UPDATE question_provenance SET adaptation_type = $2, reviewed_by = $3 WHERE question_id = $1`,
		questionID, adaptation, reviewer)
	return err
}

func insertProvenance(ctx context.Context, tx pgx.Tx, questionID uuid.UUID, p Provenance) error {
	_, err := tx.Exec(ctx, `INSERT INTO question_provenance
		(question_id, cp_id, skill_id, source_code, source_document_id, source_chunk_id,
		 original_url, source_title, retrieved_at, content_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (question_id) DO NOTHING`,
		questionID, p.CPID, p.SkillID, p.SourceCode, p.SourceDocumentID, p.SourceChunkID,
		p.OriginalURL, p.SourceTitle, nullableTime(p.RetrievedAt), p.ContentHash)
	return err
}

func nullableTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
