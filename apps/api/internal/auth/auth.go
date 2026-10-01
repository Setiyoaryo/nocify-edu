// Package auth implements parent/staff accounts, cookie sessions and RBAC.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/nocify/eduplay/api/internal/attribution"
	"github.com/nocify/eduplay/api/internal/httpx"
)

const (
	CookieName = "eduplay_session"
	// ConsentVersion identifies the privacy notice / parental consent text the
	// parent accepted at registration (UU PDP No. 27/2022, child data).
	ConsentVersion = "2026-09-v1"
)

type Role string

const (
	RoleParent   Role = "parent"
	RoleEditor   Role = "content_editor"
	RoleReviewer Role = "reviewer"
	RoleAdmin    Role = "admin"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        Role      `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

func (u User) Is(roles ...Role) bool {
	for _, r := range roles {
		if u.Role == r {
			return true
		}
	}
	return false
}

type ctxKey struct{}

func UserFrom(ctx context.Context) (User, bool) {
	u, ok := ctx.Value(ctxKey{}).(User)
	return u, ok
}

// MustUser is for handlers mounted behind RequireUser.
func MustUser(ctx context.Context) User {
	u, _ := UserFrom(ctx)
	return u
}

type Service struct {
	DB           *pgxpool.Pool
	Secret       []byte
	TTL          time.Duration
	CookieSecure bool
}

func (s *Service) hashToken(token string) string {
	m := hmac.New(sha256.New, s.Secret)
	m.Write([]byte(token))
	return hex.EncodeToString(m.Sum(nil))
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(b), err
}

// CreateSession issues a new random token, stores only its HMAC, and sets the cookie.
func (s *Service) CreateSession(ctx context.Context, w http.ResponseWriter, userID uuid.UUID) error {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expires := time.Now().Add(s.TTL)
	if _, err := s.DB.Exec(ctx, `INSERT INTO sessions (id, user_id, expires_at) VALUES ($1,$2,$3)`,
		s.hashToken(token), userID, expires); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: token, Path: "/", Expires: expires,
		HttpOnly: true, Secure: s.CookieSecure, SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (s *Service) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: s.CookieSecure, SameSite: http.SameSiteLaxMode})
}

// Middleware attaches the session user (if any) to the request context.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(CookieName)
		if err != nil || c.Value == "" {
			next.ServeHTTP(w, r)
			return
		}
		var u User
		err = s.DB.QueryRow(r.Context(), `
			UPDATE sessions SET last_seen_at = now()
			FROM users WHERE sessions.id = $1 AND users.id = sessions.user_id AND sessions.expires_at > now()
			RETURNING users.id, users.email, users.display_name, users.role, users.created_at`,
			s.hashToken(c.Value)).Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.CreatedAt)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				httpx.Internal(w, r, err)
				return
			}
			s.clearCookie(w)
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
	})
}

func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFrom(r.Context()); !ok {
			httpx.Error(w, http.StatusUnauthorized, "unauthenticated", "Silakan masuk terlebih dahulu.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole enforces admin RBAC server-side. Admin always passes.
func RequireRole(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := UserFrom(r.Context())
			if !ok {
				httpx.Error(w, http.StatusUnauthorized, "unauthenticated", "Silakan masuk terlebih dahulu.")
				return
			}
			if !u.Is(append(roles, RoleAdmin)...) {
				httpx.Forbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ---------------------------------------------------------------- handlers

type OnRegister func(ctx context.Context, u User)

type Handler struct {
	S          *Service
	OnRegister OnRegister
}

type credentials struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Consent     bool   `json:"consent"`
}

func normEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

func validatePassword(pw string) string {
	if utf8.RuneCountInString(pw) < 8 {
		return "Minimal 8 karakter."
	}
	if len(pw) > 72 { // bcrypt limit
		return "Maksimal 72 karakter."
	}
	return ""
}

func (h Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.Email = normEmail(in.Email)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	fields := map[string]string{}
	if _, err := mail.ParseAddress(in.Email); err != nil || len(in.Email) > 254 {
		fields["email"] = "Email tidak valid."
	}
	if msg := validatePassword(in.Password); msg != "" {
		fields["password"] = msg
	}
	if n := utf8.RuneCountInString(in.DisplayName); n < 1 || n > 60 {
		fields["display_name"] = "Nama wajib diisi (maks. 60 karakter)."
	}
	if !in.Consent {
		fields["consent"] = "Persetujuan orang tua/wali diperlukan."
	}
	if len(fields) > 0 {
		httpx.Invalid(w, fields)
		return
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var u User
	err = h.S.DB.QueryRow(r.Context(), `
		INSERT INTO users (email, password_hash, display_name, consent_version, consent_at)
		VALUES ($1,$2,$3,$4,now()) RETURNING id, email, display_name, role, created_at`,
		in.Email, hash, in.DisplayName, ConsentVersion).Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpx.Invalid(w, map[string]string{"email": "Email sudah terdaftar."})
			return
		}
		httpx.Internal(w, r, err)
		return
	}
	if err := h.S.CreateSession(r.Context(), w, u.ID); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if h.OnRegister != nil {
		h.OnRegister(r.Context(), u)
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"user": u})
}

// dummyHash keeps login timing uniform whether or not the email exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equaliser"), 12)

func (h Handler) Login(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !httpx.Decode(w, r, &in) {
		return
	}
	var u User
	var hash string
	err := h.S.DB.QueryRow(r.Context(), `SELECT id, email, display_name, role, created_at, password_hash FROM users WHERE lower(email) = $1`,
		normEmail(in.Email)).Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.CreatedAt, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(in.Password))
		httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "Email atau kata sandi salah.")
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "Email atau kata sandi salah.")
		return
	}
	if err := h.S.CreateSession(r.Context(), w, u.ID); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"user": u})
}

func (h Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(CookieName); err == nil {
		_, _ = h.S.DB.Exec(r.Context(), `DELETE FROM sessions WHERE id = $1`, h.S.hashToken(c.Value))
	}
	h.S.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) Me(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"user": MustUser(r.Context())})
}

func (h Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DisplayName string `json:"display_name"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if n := utf8.RuneCountInString(in.DisplayName); n < 1 || n > 60 {
		httpx.Invalid(w, map[string]string{"display_name": "Nama wajib diisi (maks. 60 karakter)."})
		return
	}
	u := MustUser(r.Context())
	if _, err := h.S.DB.Exec(r.Context(), `UPDATE users SET display_name = $2, updated_at = now() WHERE id = $1`, u.ID, in.DisplayName); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	u.DisplayName = in.DisplayName
	httpx.JSON(w, http.StatusOK, map[string]any{"user": u})
}

func (h Handler) checkPassword(ctx context.Context, id uuid.UUID, pw string) (bool, error) {
	var hash string
	if err := h.S.DB.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, id).Scan(&hash); err != nil {
		return false, err
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil, nil
}

func (h Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	u := MustUser(r.Context())
	ok, err := h.checkPassword(r.Context(), u.ID, in.Current)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if !ok {
		httpx.Invalid(w, map[string]string{"current_password": "Kata sandi saat ini salah."})
		return
	}
	if msg := validatePassword(in.New); msg != "" {
		httpx.Invalid(w, map[string]string{"new_password": msg})
		return
	}
	hash, err := HashPassword(in.New)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	// Changing the password revokes every other session.
	tx, err := h.S.DB.Begin(r.Context())
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(r.Context())
	if _, err := tx.Exec(r.Context(), `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, u.ID, hash); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if _, err := tx.Exec(r.Context(), `DELETE FROM sessions WHERE user_id = $1`, u.ID); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if err := h.S.CreateSession(r.Context(), w, u.ID); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteAccount erases the parent and (by cascade) every child profile and all
// learning evidence. Required for data-subject rights under UU PDP.
func (h Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	u := MustUser(r.Context())
	ok, err := h.checkPassword(r.Context(), u.ID, in.Password)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if !ok {
		httpx.Invalid(w, map[string]string{"password": "Kata sandi salah."})
		return
	}
	ctx := r.Context()
	tx, err := h.S.DB.Begin(ctx)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	// Erasure cascades every child profile and its assessment_items away, so
	// capture the affected questions first and refresh the attribution rollup in
	// the same transaction (§39). Otherwise the rollup keeps stale usage/quality.
	questionIDs, err := attribution.QuestionsForParent(ctx, tx, u.ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, u.ID); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if err := attribution.RecomputeQuestions(ctx, tx, questionIDs); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	h.S.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
