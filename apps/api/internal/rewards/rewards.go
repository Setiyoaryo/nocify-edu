// Package rewards grants XP, stars, badges and unlockable avatars/stickers/scenes
// (§12). No leaderboards, lootboxes or randomised rewards: every unlock is a
// predictable consequence of learning activity.
package rewards

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/children"
	"github.com/nocify/eduplay/api/internal/db"
	"github.com/nocify/eduplay/api/internal/httpx"
)

type Unlock struct {
	Kind string `json:"kind"` // badge | avatar | sticker | scene
	Code string `json:"code"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// SessionFacts is what the badge rules look at after a completed session.
type SessionFacts struct {
	Score            int
	MissionCompleted bool
}

// Grant adds XP and awards anything newly earned. Must run inside the
// completion transaction so a retry cannot double-award.
func Grant(ctx context.Context, tx pgx.Tx, childID uuid.UUID, xp int, facts SessionFacts) (newXP int, unlocks []Unlock, err error) {
	if err = tx.QueryRow(ctx, `UPDATE children SET xp = xp + $2, updated_at = now() WHERE id = $1 RETURNING xp`, childID, xp).Scan(&newXP); err != nil {
		return
	}
	rows, err := tx.Query(ctx, `
		INSERT INTO child_inventory (child_id, item_kind, item_code)
		SELECT $1, 'reward', code FROM rewards WHERE xp_required <= $2
		ON CONFLICT DO NOTHING RETURNING item_code`, childID, newXP)
	if err != nil {
		return
	}
	codes, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return
	}
	for _, c := range codes {
		var u Unlock
		if err = tx.QueryRow(ctx, `SELECT kind, code, name, icon FROM rewards WHERE code = $1`, c).Scan(&u.Kind, &u.Code, &u.Name, &u.Icon); err != nil {
			return
		}
		if u.Kind == "avatar" && freeAvatars[u.Code] {
			continue // starter avatars are not "new"
		}
		unlocks = append(unlocks, u)
	}

	var completed, activeDays int
	var skillMaster bool
	if err = tx.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM assessments WHERE child_id = $1 AND status = 'completed')::int,
		       (SELECT count(*) FROM daily_progress WHERE child_id = $1 AND challenges_completed > 0 AND day > current_date - 3)::int,
		       EXISTS (SELECT 1 FROM skill_progress WHERE child_id = $1 AND mastery > 85 AND evidence_count >= 5)`,
		childID).Scan(&completed, &activeDays, &skillMaster); err != nil {
		return
	}
	earned := map[string]bool{
		"first_challenge": completed >= 1,
		"perfect_score":   facts.Score == 100,
		"streak_3":        activeDays >= 3,
		"ten_challenges":  completed >= 10,
		"skill_master":    skillMaster,
		"mission_done":    facts.MissionCompleted,
	}
	for code, ok := range earned {
		if !ok {
			continue
		}
		var u Unlock
		err = tx.QueryRow(ctx, `
			WITH ins AS (
				INSERT INTO child_inventory (child_id, item_kind, item_code) VALUES ($1, 'badge', $2)
				ON CONFLICT DO NOTHING RETURNING item_code)
			SELECT 'badge', b.code, b.name, b.icon FROM ins JOIN badges b ON b.code = ins.item_code`, childID, code).
			Scan(&u.Kind, &u.Code, &u.Name, &u.Icon)
		if err == pgx.ErrNoRows {
			err = nil
			continue
		}
		if err != nil {
			return
		}
		unlocks = append(unlocks, u)
	}
	return
}

var freeAvatars = map[string]bool{}

// LoadFreeAvatars caches which avatars need no XP (so they are not announced as unlocks).
func LoadFreeAvatars(ctx context.Context, q db.DBTX) error {
	rows, err := q.Query(ctx, `SELECT code FROM rewards WHERE kind = 'avatar' AND xp_required = 0`)
	if err != nil {
		return err
	}
	codes, err := pgx.CollectRows(rows, pgx.RowTo[string])
	for _, c := range codes {
		freeAvatars[c] = true
	}
	return err
}

type Handler struct{ DB *pgxpool.Pool }

type item struct {
	Kind        string `json:"kind"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon"`
	XPRequired  int    `json:"xp_required,omitempty"`
	Owned       bool   `json:"owned"`
}

func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	child, ok := children.Owned(w, r, h.DB, "childID")
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(), `
		SELECT 'badge', b.code, b.name, b.description, b.icon, 0,
		       EXISTS (SELECT 1 FROM child_inventory i WHERE i.child_id = $1 AND i.item_kind = 'badge' AND i.item_code = b.code)
		FROM badges b
		UNION ALL
		SELECT r.kind, r.code, r.name, '', r.icon, r.xp_required, r.xp_required <= $2
		FROM rewards r
		ORDER BY 1, 6, 2`, child.ID, child.XP)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (item, error) {
		var it item
		err := row.Scan(&it.Kind, &it.Code, &it.Name, &it.Description, &it.Icon, &it.XPRequired, &it.Owned)
		return it, err
	})
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var stars int
	if err := h.DB.QueryRow(r.Context(), `SELECT COALESCE(sum(stars), 0)::int FROM assessments WHERE child_id = $1 AND status = 'completed'`, child.ID).Scan(&stars); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"child": child, "stars": stars, "items": items})
}
