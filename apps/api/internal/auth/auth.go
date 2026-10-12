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
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	"sync/atomic"
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
	otps         sync.Map
}

func (s *Service) hashToken(token string) string {
	m := hmac.New(sha256.New, s.Secret)
	m.Write([]byte(token))
	return hex.EncodeToString(m.Sum(nil))
}

type otpEntry struct {
	code      string
	expiresAt time.Time
	attempts  atomic.Int32
}

func (s *Service) SetOTP(phone, code string, exp time.Time) {
	s.otps.Store(phone, &otpEntry{code: code, expiresAt: exp})
}

func (s *Service) CheckOTP(phone, code string) bool {
	v, ok := s.otps.Load(phone)
	if !ok {
		return false
	}
	entry := v.(*otpEntry)
	if time.Now().After(entry.expiresAt) {
		s.otps.Delete(phone)
		return false
	}
	if entry.code != code {
		if entry.attempts.Add(1) >= 5 {
			s.otps.Delete(phone)
		}
		return false
	}
	s.otps.Delete(phone)
	return true
}

func NormPhone(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("Nomor WhatsApp wajib diisi.")
	}

	hasPlus := strings.HasPrefix(raw, "+")
	if strings.Count(raw, "+") > 1 || (!hasPlus && strings.Contains(raw, "+")) {
		return "", errors.New("Nomor WhatsApp tidak valid (contoh: 081234567890).")
	}

	var digits []rune
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		} else if r != '+' && r != '-' && r != ' ' && r != '(' && r != ')' && r != '.' {
			return "", errors.New("Nomor WhatsApp tidak valid (contoh: 081234567890).")
		}
	}
	s := string(digits)

	if hasPlus && !strings.HasPrefix(s, "62") && !strings.HasPrefix(s, "0") && !strings.HasPrefix(s, "8") {
		return "", errors.New("Nomor WhatsApp tidak valid (contoh: 081234567890).")
	}

	if strings.HasPrefix(s, "0062") {
		s = strings.TrimPrefix(s, "0062")
	} else if strings.HasPrefix(s, "62") {
		s = strings.TrimPrefix(s, "62")
	}
	s = strings.TrimLeft(s, "0")
	if len(s) < 8 || len(s) > 15 {
		return "", errors.New("Nomor WhatsApp tidak valid (contoh: 081234567890).")
	}
	return "+62" + s, nil
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
	// Best-effort: a data-subject erasure (UU PDP) must not be denied by a
	// transient failure of a derived rollup. The delete commits regardless;
	// `eduplay recompute-attribution` reconciles any drift.
	if err := attribution.RecomputeBestEffort(ctx, tx, questionIDs); err != nil {
		slog.ErrorContext(ctx, "attribution recompute after account erasure failed", "err", err, "user_id", u.ID)
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	h.S.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

type sendOTPReq struct {
	Phone string `json:"phone"`
	Role  string `json:"role"`
}

// GenerateOTP creates a cryptographically secure 6-digit OTP code [100000, 999999].
// It returns an error if reading random entropy fails, never falling back to a static code.
func GenerateOTP() (string, error) {
	return GenerateOTPReader(rand.Reader)
}

func GenerateOTPReader(r io.Reader) (string, error) {
	bi, err := rand.Int(r, big.NewInt(900000))
	if err != nil {
		return "", fmt.Errorf("csprng failure: %w", err)
	}
	return fmt.Sprintf("%06d", 100000+bi.Int64()), nil
}

func (h Handler) SendOTP(w http.ResponseWriter, r *http.Request) {
	var in sendOTPReq
	if !httpx.Decode(w, r, &in) {
		return
	}
	phone, err := NormPhone(in.Phone)
	if err != nil {
		httpx.Invalid(w, map[string]string{"phone": err.Error()})
		return
	}

	if h.S.DB != nil && in.Role == "child" {
		var parentID uuid.UUID
		err := h.S.DB.QueryRow(r.Context(), `
			SELECT id FROM users WHERE phone = $1
			ORDER BY created_at ASC LIMIT 1`, phone).Scan(&parentID)
		if errors.Is(err, pgx.ErrNoRows) {
			digits := strings.TrimPrefix(phone, "+")
			synEmail := fmt.Sprintf("wa-%s@eduplay.local", digits)
			err = h.S.DB.QueryRow(r.Context(), `
				SELECT id FROM users WHERE lower(email) = $1
				ORDER BY created_at ASC LIMIT 1`, synEmail).Scan(&parentID)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "account_not_found", "Nomor WhatsApp belum terdaftar. Minta Ayah atau Bunda mendaftar dan membuat profil anak terlebih dahulu.")
			return
		} else if err != nil {
			httpx.Internal(w, r, err)
			return
		}

		var count int
		err = h.S.DB.QueryRow(r.Context(), `
			SELECT count(*) FROM children WHERE parent_id = $1`, parentID).Scan(&count)
		if err != nil {
			httpx.Internal(w, r, err)
			return
		}
		if count == 0 {
			httpx.Error(w, http.StatusConflict, "no_children", "Belum ada profil anak yang dibuat di akun ini. Minta Ayah atau Bunda masuk terlebih dahulu untuk membuat profil anak Anda.")
			return
		}
	}

	code, err := GenerateOTP()
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}

	h.S.SetOTP(phone, code, time.Now().Add(5*time.Minute))
	slog.Info("WhatsApp OTP dispatched", "phone", phone, "role", in.Role)

	httpx.JSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"phone":   phone,
		"message": "Kode OTP berhasil dikirim ke WhatsApp.",
	})
}

type verifyOTPReq struct {
	Phone       string `json:"phone"`
	Code        string `json:"code"`
	Role        string `json:"role"`
	DisplayName string `json:"display_name,omitempty"`
}

func (h Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var in verifyOTPReq
	if !httpx.Decode(w, r, &in) {
		return
	}
	phone, err := NormPhone(in.Phone)
	if err != nil {
		httpx.Invalid(w, map[string]string{"phone": err.Error()})
		return
	}
	code := strings.TrimSpace(in.Code)
	if code == "" {
		httpx.Invalid(w, map[string]string{"code": "Kode OTP wajib diisi."})
		return
	}

	if !h.S.CheckOTP(phone, code) {
		httpx.Error(w, http.StatusUnauthorized, "invalid_otp", "Kode OTP salah atau telah kadaluarsa.")
		return
	}

	if h.S.DB == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "db_unavailable", "Database tidak tersedia.")
		return
	}

	var u User
	var isNewUser bool
	err = h.S.DB.QueryRow(r.Context(), `
		SELECT id, email, display_name, role, created_at
		FROM users
		WHERE phone = $1
		ORDER BY created_at ASC LIMIT 1`, phone).Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		digits := strings.TrimPrefix(phone, "+")
		synEmail := fmt.Sprintf("wa-%s@eduplay.local", digits)
		err = h.S.DB.QueryRow(r.Context(), `
			SELECT id, email, display_name, role, created_at
			FROM users
			WHERE lower(email) = $1`, synEmail).Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.CreatedAt)

		if errors.Is(err, pgx.ErrNoRows) {
			if in.Role == "child" {
				httpx.Error(w, http.StatusNotFound, "account_not_found", "Nomor WhatsApp belum terdaftar. Minta Ayah atau Bunda mendaftar dan membuat profil anak terlebih dahulu.")
				return
			}
			displayName := strings.TrimSpace(in.DisplayName)
			if displayName == "" {
				displayName = "Orang Tua"
			}
			dummyHash, _ := bcrypt.GenerateFromPassword([]byte(uuid.NewString()), 10)
			err = h.S.DB.QueryRow(r.Context(), `
				INSERT INTO users (email, password_hash, display_name, phone, role, consent_version, consent_at)
				VALUES ($1, $2, $3, $4, 'parent', $5, now())
				RETURNING id, email, display_name, role, created_at`,
				synEmail, string(dummyHash), displayName, phone, ConsentVersion).Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.CreatedAt)
			if err != nil {
				httpx.Internal(w, r, err)
				return
			}
			isNewUser = true
			if h.OnRegister != nil {
				h.OnRegister(r.Context(), u)
			}
		} else if err != nil {
			httpx.Internal(w, r, err)
			return
		} else {
			_, _ = h.S.DB.Exec(r.Context(), `UPDATE users SET phone = $1 WHERE id = $2`, phone, u.ID)
			if strings.TrimSpace(in.DisplayName) != "" && (u.DisplayName == "Orang Tua" || u.DisplayName == "") {
				_, _ = h.S.DB.Exec(r.Context(), `UPDATE users SET display_name = $1 WHERE id = $2`, strings.TrimSpace(in.DisplayName), u.ID)
				u.DisplayName = strings.TrimSpace(in.DisplayName)
			}
		}
	} else if err != nil {
		httpx.Internal(w, r, err)
		return
	} else {
		if strings.TrimSpace(in.DisplayName) != "" && (u.DisplayName == "Orang Tua" || u.DisplayName == "") {
			_, _ = h.S.DB.Exec(r.Context(), `UPDATE users SET display_name = $1 WHERE id = $2`, strings.TrimSpace(in.DisplayName), u.ID)
			u.DisplayName = strings.TrimSpace(in.DisplayName)
		}
	}

	var childrenList []map[string]any
	if in.Role == "child" {
		rows, err := h.S.DB.Query(r.Context(), `
			SELECT c.id, c.nickname, c.grade_id, g.name, g.ux_band, c.avatar, c.pin_hash IS NOT NULL, c.xp, c.created_at,
			       (SELECT max(a.started_at) FROM assessments a WHERE a.child_id = c.id)
			FROM children c JOIN grades g ON g.id = c.grade_id
			WHERE c.parent_id = $1
			ORDER BY c.created_at ASC`, u.ID)
		if err != nil {
			httpx.Internal(w, r, err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var cid uuid.UUID
			var nickname, gradeName, uxBand, avatar string
			var gradeID, xp int
			var hasPIN bool
			var createdAt time.Time
			var lastActivity *time.Time
			if err := rows.Scan(&cid, &nickname, &gradeID, &gradeName, &uxBand, &avatar, &hasPIN, &xp, &createdAt, &lastActivity); err == nil {
				childrenList = append(childrenList, map[string]any{
					"id":            cid.String(),
					"nickname":      nickname,
					"grade_id":      gradeID,
					"grade_name":    gradeName,
					"ux_band":       uxBand,
					"avatar":        avatar,
					"has_pin":       hasPIN,
					"xp":            xp,
					"level":         1 + xp/100,
					"last_activity": lastActivity,
				})
			}
		}
		if len(childrenList) == 0 {
			httpx.Error(w, http.StatusConflict, "no_children", "Belum ada profil anak yang dibuat di akun ini. Minta Ayah atau Bunda masuk terlebih dahulu untuk membuat profil anak Anda.")
			return
		}
	}

	if err := h.S.CreateSession(r.Context(), w, u.ID); err != nil {
		httpx.Internal(w, r, err)
		return
	}

	resp := map[string]any{
		"user":        u,
		"role":        in.Role,
		"is_new_user": isNewUser,
	}
	if in.Role == "child" {
		resp["children"] = childrenList
	}

	httpx.JSON(w, http.StatusOK, resp)
}
