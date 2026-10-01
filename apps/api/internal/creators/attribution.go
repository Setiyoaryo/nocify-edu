package creators

import (
	"context"
	"math"
	"strings"

	"github.com/google/uuid"

	"github.com/nocify/eduplay/api/internal/db"
)

// Stats is the raw evidence behind a question's attribution rollup (§39).
//
//	Usage      — assessment_items rows: the question was served in a session.
//	Completion — items answered to a final state (finalized_at IS NOT NULL).
//	Correct    — final_correct.
//	FirstTry   — first_try_correct.
//	Hint       — items where a hint was shown.
type Stats struct {
	Usage      int
	Completion int
	Correct    int
	FirstTry   int
	Hint       int
}

// QualityScore is a deterministic 0–100 quality signal for a question, derived
// from how children actually did on it. It is a pure function so it can be unit
// tested without a database and so the stored rollup has one source of truth.
//
// Weights: 50% final accuracy, 30% first-try accuracy, 20% hint-free rate
// (hints are counted per served item). A question that was served but never
// answered has no quality signal yet, so it scores 0.
func QualityScore(s Stats) float64 {
	if s.Usage <= 0 || s.Completion <= 0 {
		return 0
	}
	completion := float64(s.Completion)
	accuracy := float64(s.Correct) / completion
	firstTry := float64(s.FirstTry) / completion
	hintFree := 1 - float64(s.Hint)/float64(s.Usage)
	if hintFree < 0 {
		hintFree = 0
	}
	v := 100 * (0.5*accuracy + 0.3*firstTry + 0.2*hintFree)
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	return math.Round(v*100) / 100
}

type questionStats struct {
	questionID uuid.UUID
	stats      Stats
}

// RecomputeQuestions rebuilds the rollup for specific questions. Safe to call
// inside the session-complete transaction (mirrors progress.RecomputeSkills).
func RecomputeQuestions(ctx context.Context, q db.DBTX, questionIDs []uuid.UUID) error {
	if len(questionIDs) == 0 {
		return nil
	}
	return recompute(ctx, q, questionIDs)
}

// RecomputeAll rebuilds every question's rollup from raw evidence and drops
// rollups whose evidence is gone (e.g. after a child erasure cascade). This is
// the `eduplay recompute-attribution` safety net.
func RecomputeAll(ctx context.Context, q db.DBTX) error {
	if _, err := q.Exec(ctx, `
		DELETE FROM question_attribution a
		WHERE NOT EXISTS (SELECT 1 FROM assessment_items i WHERE i.question_id = a.question_id)`); err != nil {
		return err
	}
	return recompute(ctx, q, nil)
}

// aggregateSQL sums raw evidence per question. `%s` is replaced by an optional
// filter (a constant, never user input) so the full and subset rebuilds share
// one definition of "usage" and "completion".
const aggregateSQL = `
	SELECT i.question_id,
	       count(*)::int,
	       count(*) FILTER (WHERE i.finalized_at IS NOT NULL)::int,
	       count(*) FILTER (WHERE i.final_correct)::int,
	       count(*) FILTER (WHERE i.first_try_correct)::int,
	       count(*) FILTER (WHERE i.hint_shown)::int
	FROM assessment_items i
	%s
	GROUP BY i.question_id`

// recompute aggregates raw assessment evidence per question, computes the
// quality score in Go, and upserts one rollup row per question. Passing a nil
// slice rebuilds every question with evidence.
func recompute(ctx context.Context, q db.DBTX, questionIDs []uuid.UUID) error {
	query := strings.Replace(aggregateSQL, "%s", "", 1)
	var args []any
	if questionIDs != nil {
		query = strings.Replace(aggregateSQL, "%s", "WHERE i.question_id = ANY($1)", 1)
		args = append(args, questionIDs)
	}

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	var pending []questionStats
	for rows.Next() {
		var p questionStats
		if err := rows.Scan(&p.questionID, &p.stats.Usage, &p.stats.Completion,
			&p.stats.Correct, &p.stats.FirstTry, &p.stats.Hint); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, p := range pending {
		// creator_id and published_at are read from the owning question so the
		// denormalised rollup always follows a reassignment or a (re)publish.
		if _, err := q.Exec(ctx, `
			INSERT INTO question_attribution
				(question_id, creator_id, usage_count, completion_count, correct_count,
				 first_try_count, hint_count, quality_score, published_at, computed_at)
			SELECT q.id, q.creator_id, $2, $3, $4, $5, $6, $7, q.published_at, now()
			FROM questions q WHERE q.id = $1
			ON CONFLICT (question_id) DO UPDATE SET
				creator_id = EXCLUDED.creator_id,
				usage_count = EXCLUDED.usage_count,
				completion_count = EXCLUDED.completion_count,
				correct_count = EXCLUDED.correct_count,
				first_try_count = EXCLUDED.first_try_count,
				hint_count = EXCLUDED.hint_count,
				quality_score = EXCLUDED.quality_score,
				published_at = EXCLUDED.published_at,
				computed_at = now()`,
			p.questionID, p.stats.Usage, p.stats.Completion, p.stats.Correct,
			p.stats.FirstTry, p.stats.Hint, QualityScore(p.stats)); err != nil {
			return err
		}
	}
	return nil
}
