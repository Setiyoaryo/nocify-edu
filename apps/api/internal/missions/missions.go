// Package missions lets a parent turn a recommendation into a concrete goal
// for the child ("Latih 5 challenge tentang pembagian minggu ini", §13). The
// child sees active missions first on their home screen; each completed
// skill-focused session started from a mission counts toward it.
package missions

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nocify/eduplay/api/internal/analytics"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/children"
	"github.com/nocify/eduplay/api/internal/db"
	"github.com/nocify/eduplay/api/internal/httpx"
)

// MaxActive keeps missions meaningful: a child with ten goals has none.
const MaxActive = 3

type Mission struct {
	ID          uuid.UUID  `json:"id"`
	SkillID     int        `json:"skill_id"`
	SkillName   string     `json:"skill_name"`
	SubjectID   int        `json:"subject_id"`
	SubjectName string     `json:"subject_name"`
	SubjectCode string     `json:"subject_code"`
	Target      int        `json:"target"`
	Completed   int        `json:"completed"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	DueAt       *time.Time `json:"due_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// List returns active missions and those finished in the last 14 days.
func List(ctx context.Context, q db.DBTX, childID uuid.UUID, activeOnly bool) ([]Mission, error) {
	rows, err := q.Query(ctx, `
		SELECT m.id, k.id, k.name, s.id, s.name, s.code, m.target, m.completed, m.status, m.created_at, m.due_at, m.completed_at
		FROM child_missions m JOIN skills k ON k.id = m.skill_id JOIN chapters c ON c.id = k.chapter_id JOIN subjects s ON s.id = c.subject_id
		WHERE m.child_id = $1 AND (m.status = 'active' OR (NOT $2 AND m.status = 'completed' AND m.completed_at > now() - interval '14 days'))
		ORDER BY m.status = 'active' DESC, m.created_at DESC`, childID, activeOnly)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[Mission])
}

type Handler struct{ DB *pgxpool.Pool }

func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	child, ok := children.Owned(w, r, h.DB, "childID")
	if !ok {
		return
	}
	out, err := List(r.Context(), h.DB, child.ID, false)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"missions": out})
}

func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	child, ok := children.Owned(w, r, h.DB, "childID")
	if !ok {
		return
	}
	var in struct {
		SkillID int `json:"skill_id"`
		Target  int `json:"target"`
		Days    int `json:"days"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if in.Target == 0 {
		in.Target = 5
	}
	if in.Days == 0 {
		in.Days = 7
	}
	f := map[string]string{}
	if in.Target < 1 || in.Target > 20 {
		f["target"] = "Target 1–20 challenge."
	}
	if in.Days < 1 || in.Days > 30 {
		f["days"] = "Durasi 1–30 hari."
	}
	var live int
	err := h.DB.QueryRow(r.Context(), `SELECT count(q.id) FROM skills k JOIN chapters c ON c.id = k.chapter_id
		LEFT JOIN questions q ON q.skill_id = k.id AND q.published_version_id IS NOT NULL
		WHERE k.id = $1 AND c.grade_id = $2 GROUP BY k.id`, in.SkillID, child.GradeID).Scan(&live)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		f["skill_id"] = "Skill tidak tersedia untuk kelas anak ini."
	case err != nil:
		httpx.Internal(w, r, err)
		return
	case live == 0:
		f["skill_id"] = "Belum ada soal untuk skill ini."
	}
	if len(f) > 0 {
		httpx.Invalid(w, f)
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())

	var lockedID uuid.UUID
	if err := tx.QueryRow(r.Context(), `SELECT id FROM children WHERE id = $1 FOR UPDATE`, child.ID).Scan(&lockedID); err != nil {
		httpx.Internal(w, r, err)
		return
	}

	var active int
	if err := tx.QueryRow(r.Context(), `SELECT count(*) FROM child_missions WHERE child_id = $1 AND status = 'active'`, child.ID).Scan(&active); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if active >= MaxActive {
		httpx.Error(w, http.StatusConflict, "too_many_missions", "Maksimal 3 misi aktif agar anak tetap fokus. Selesaikan atau batalkan salah satu dulu.")
		return
	}
	u := auth.MustUser(r.Context())
	var id uuid.UUID
	err = tx.QueryRow(r.Context(), `INSERT INTO child_missions (child_id, skill_id, target, created_by, due_at)
		VALUES ($1, $2, $3, $4, now() + make_interval(days => $5)) RETURNING id`, child.ID, in.SkillID, in.Target, u.ID, in.Days).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		httpx.Error(w, http.StatusConflict, "mission_exists", "Skill ini sudah menjadi misi aktif.")
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	analytics.Track(r.Context(), h.DB, "mission_created", u.ID, child.ID, map[string]any{"skill_id": in.SkillID, "target": in.Target})
	h.List(w, r)
}

func (h Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	child, ok := children.Owned(w, r, h.DB, "childID")
	if !ok {
		return
	}
	mid, ok := httpx.UUIDParam(w, r, "missionID")
	if !ok {
		return
	}
	tag, err := h.DB.Exec(r.Context(), `UPDATE child_missions SET status = 'cancelled' WHERE id = $1 AND child_id = $2 AND status = 'active'`, mid, child.ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.NotFound(w)
		return
	}
	h.List(w, r)
}
