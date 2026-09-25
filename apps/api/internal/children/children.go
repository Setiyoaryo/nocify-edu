// Package children manages child learning profiles. The parent account is the
// boundary: every query is scoped by parent_id (§20). Only the minimum data
// needed for learning is stored: nickname, grade, avatar, optional PIN (§40).
package children

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/nocify/eduplay/api/internal/analytics"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/db"
	"github.com/nocify/eduplay/api/internal/httpx"
	"github.com/nocify/eduplay/api/internal/subscriptions"
)

type Child struct {
	ID           uuid.UUID  `json:"id"`
	Nickname     string     `json:"nickname"`
	GradeID      int        `json:"grade_id"`
	GradeName    string     `json:"grade_name"`
	UXBand       string     `json:"ux_band"`
	Avatar       string     `json:"avatar"`
	HasPIN       bool       `json:"has_pin"`
	XP           int        `json:"xp"`
	Level        int        `json:"level"`
	LastActivity *time.Time `json:"last_activity"`
	CreatedAt    time.Time  `json:"created_at"`
}

// LevelForXP: every 100 XP is a level. Simple and explainable to parents.
func LevelForXP(xp int) int { return 1 + xp/100 }

const selectChild = `
	SELECT c.id, c.nickname, c.grade_id, g.name, g.ux_band, c.avatar, c.pin_hash IS NOT NULL, c.xp, c.created_at,
	       (SELECT max(a.started_at) FROM assessments a WHERE a.child_id = c.id)
	FROM children c JOIN grades g ON g.id = c.grade_id`

func scan(row pgx.Row) (Child, error) {
	var c Child
	err := row.Scan(&c.ID, &c.Nickname, &c.GradeID, &c.GradeName, &c.UXBand, &c.Avatar, &c.HasPIN, &c.XP, &c.CreatedAt, &c.LastActivity)
	c.Level = LevelForXP(c.XP)
	return c, err
}

var ErrNotFound = errors.New("child not found")

// LoadOwned returns the child only if it belongs to parentID. Every module that
// takes a child id from the client must go through this.
func LoadOwned(ctx context.Context, q db.DBTX, parentID, childID uuid.UUID) (Child, error) {
	c, err := scan(q.QueryRow(ctx, selectChild+` WHERE c.id = $1 AND c.parent_id = $2`, childID, parentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

type Handler struct {
	DB     *pgxpool.Pool
	Policy subscriptions.Policy
}

// Owned is a helper for handlers: resolves {childID} and writes 404 on failure.
func Owned(w http.ResponseWriter, r *http.Request, q db.DBTX, param string) (Child, bool) {
	id, ok := httpx.UUIDParam(w, r, param)
	if !ok {
		return Child{}, false
	}
	c, err := LoadOwned(r.Context(), q, auth.MustUser(r.Context()).ID, id)
	if errors.Is(err, ErrNotFound) {
		httpx.NotFound(w)
		return c, false
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return c, false
	}
	return c, true
}

func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), selectChild+` WHERE c.parent_id = $1 ORDER BY c.created_at`, auth.MustUser(r.Context()).ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Child, error) { return scan(row) })
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"children": out})
}

type input struct {
	Nickname *string `json:"nickname"`
	GradeID  *int    `json:"grade_id"`
	Avatar   *string `json:"avatar"`
	PIN      *string `json:"pin"` // 4 digits; "" removes the PIN
}

var pinRe = regexp.MustCompile(`^[0-9]{4}$`)

func (h Handler) validate(ctx context.Context, in *input, xp int, creating bool) (map[string]string, error) {
	f := map[string]string{}
	if in.Nickname != nil {
		*in.Nickname = strings.TrimSpace(*in.Nickname)
		if n := utf8.RuneCountInString(*in.Nickname); n < 1 || n > 30 {
			f["nickname"] = "Nama panggilan 1–30 karakter."
		}
	} else if creating {
		f["nickname"] = "Nama panggilan wajib diisi."
	}
	if in.GradeID != nil {
		var ok bool
		if err := h.DB.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM grades WHERE id = $1)`, *in.GradeID).Scan(&ok); err != nil {
			return nil, err
		}
		if !ok {
			f["grade_id"] = "Kelas tidak valid."
		}
	} else if creating {
		f["grade_id"] = "Kelas wajib dipilih."
	}
	if in.Avatar != nil {
		var ok bool
		if err := h.DB.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM rewards WHERE kind = 'avatar' AND code = $1 AND xp_required <= $2)`, *in.Avatar, xp).Scan(&ok); err != nil {
			return nil, err
		}
		if !ok {
			f["avatar"] = "Avatar belum terbuka."
		}
	}
	if in.PIN != nil && *in.PIN != "" && !pinRe.MatchString(*in.PIN) {
		f["pin"] = "PIN harus 4 angka."
	}
	return f, nil
}

func pinHash(pin *string) (any, error) {
	if pin == nil || *pin == "" {
		return nil, nil
	}
	b, err := bcrypt.GenerateFromPassword([]byte(*pin), 10)
	return string(b), err
}

func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in input
	if !httpx.Decode(w, r, &in) {
		return
	}
	u := auth.MustUser(r.Context())
	if in.Avatar == nil {
		def := "cat"
		in.Avatar = &def
	}
	fields, err := h.validate(r.Context(), &in, 0, true)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if len(fields) > 0 {
		httpx.Invalid(w, fields)
		return
	}
	ent, err := h.Policy.Get(r.Context(), h.DB, u.ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var count int
	if err := h.DB.QueryRow(r.Context(), `SELECT count(*) FROM children WHERE parent_id = $1`, u.ID).Scan(&count); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if count >= ent.Limits.MaxChildren {
		code, msg := "child_limit", "Batas jumlah profil anak tercapai."
		if !ent.Premium {
			code, msg = "upgrade_required", "Paket gratis mendukung 1 profil anak. Berlangganan untuk menambah profil."
		}
		httpx.Error(w, http.StatusPaymentRequired, code, msg)
		return
	}
	hash, err := pinHash(in.PIN)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var id uuid.UUID
	if err := h.DB.QueryRow(r.Context(), `INSERT INTO children (parent_id, nickname, grade_id, avatar, pin_hash) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		u.ID, *in.Nickname, *in.GradeID, *in.Avatar, hash).Scan(&id); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	analytics.Track(r.Context(), h.DB, "child_created", u.ID, id, nil)
	c, err := LoadOwned(r.Context(), h.DB, u.ID, id)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"child": c})
}

func (h Handler) Get(w http.ResponseWriter, r *http.Request) {
	c, ok := Owned(w, r, h.DB, "childID")
	if !ok {
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"child": c})
}

func (h Handler) Update(w http.ResponseWriter, r *http.Request) {
	c, ok := Owned(w, r, h.DB, "childID")
	if !ok {
		return
	}
	var in input
	if !httpx.Decode(w, r, &in) {
		return
	}
	fields, err := h.validate(r.Context(), &in, c.XP, false)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if len(fields) > 0 {
		httpx.Invalid(w, fields)
		return
	}
	hash, err := pinHash(in.PIN)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	_, err = h.DB.Exec(r.Context(), `UPDATE children SET
		nickname = COALESCE($2, nickname),
		grade_id = COALESCE($3, grade_id),
		avatar   = COALESCE($4, avatar),
		pin_hash = CASE WHEN $5 THEN $6 ELSE pin_hash END,
		updated_at = now()
		WHERE id = $1`, c.ID, in.Nickname, in.GradeID, in.Avatar, in.PIN != nil, hash)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	h.Get(w, r)
}

// Delete permanently removes the profile and all its learning evidence.
func (h Handler) Delete(w http.ResponseWriter, r *http.Request) {
	c, ok := Owned(w, r, h.DB, "childID")
	if !ok {
		return
	}
	if _, err := h.DB.Exec(r.Context(), `DELETE FROM children WHERE id = $1`, c.ID); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) VerifyPIN(w http.ResponseWriter, r *http.Request) {
	c, ok := Owned(w, r, h.DB, "childID")
	if !ok {
		return
	}
	var in struct {
		PIN string `json:"pin"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	var hash *string
	if err := h.DB.QueryRow(r.Context(), `SELECT pin_hash FROM children WHERE id = $1`, c.ID).Scan(&hash); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if hash != nil && bcrypt.CompareHashAndPassword([]byte(*hash), []byte(in.PIN)) != nil {
		httpx.Error(w, http.StatusForbidden, "wrong_pin", "PIN belum tepat. Coba lagi ya.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
