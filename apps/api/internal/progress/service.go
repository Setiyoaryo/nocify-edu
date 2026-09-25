package progress

import (
	"context"

	"github.com/google/uuid"

	"github.com/nocify/eduplay/api/internal/db"
)

// RecomputeSkills rebuilds skill_progress rows for the given skills from raw
// first attempts. Called after every completed session; safe to call anytime.
func RecomputeSkills(ctx context.Context, q db.DBTX, childID uuid.UUID, skillIDs []int) error {
	for _, skillID := range skillIDs {
		rows, err := q.Query(ctx, `
			SELECT is_correct, hint_used FROM question_attempts
			WHERE child_id = $1 AND skill_id = $2 AND attempt_no = 1
			ORDER BY created_at DESC, id DESC`, childID, skillID)
		if err != nil {
			return err
		}
		var ev []Evidence
		correct := 0
		for rows.Next() {
			var e Evidence
			if err := rows.Scan(&e.Correct, &e.HintUsed); err != nil {
				rows.Close()
				return err
			}
			if e.Correct {
				correct++
			}
			ev = append(ev, e)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if len(ev) == 0 {
			continue
		}
		if _, err := q.Exec(ctx, `
			INSERT INTO skill_progress (child_id, skill_id, mastery, evidence_count, correct_count, last_attempt_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, (SELECT max(created_at) FROM question_attempts WHERE child_id = $1 AND skill_id = $2), now())
			ON CONFLICT (child_id, skill_id) DO UPDATE SET
				mastery = EXCLUDED.mastery, evidence_count = EXCLUDED.evidence_count,
				correct_count = EXCLUDED.correct_count, last_attempt_at = EXCLUDED.last_attempt_at, updated_at = now()`,
			childID, skillID, Mastery(ev), len(ev), correct); err != nil {
			return err
		}
	}
	return nil
}

// RecomputeChild rebuilds every skill the child has evidence for (used after an
// algorithm change: `eduplay recompute`).
func RecomputeChild(ctx context.Context, q db.DBTX, childID uuid.UUID) error {
	rows, err := q.Query(ctx, `SELECT DISTINCT skill_id FROM question_attempts WHERE child_id = $1`, childID)
	if err != nil {
		return err
	}
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	return RecomputeSkills(ctx, q, childID, ids)
}
