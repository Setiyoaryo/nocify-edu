package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/nocify/eduplay/api/internal/httpx"
)

const (
	googleAuthEndpoint  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenEndpoint = "https://oauth2.googleapis.com/token"
	googleJWKSURL       = "https://www.googleapis.com/oauth2/v3/certs"
	googleFlowTTL       = 10 * time.Minute
)

// GoogleOAuthHandler supports parent OIDC login and explicit account linking.
// The existing WhatsApp OTP and email/password handlers are left unchanged.
type GoogleOAuthHandler struct {
	Auth        Handler
	ClientID    string
	Secret      string
	RedirectURI string
	AuthURL     string
	TokenURL    string
	JWKSURL     string
	HTTPClient  *http.Client
}

func (h GoogleOAuthHandler) enabled() bool {
	return h.ClientID != "" && h.Secret != "" && h.RedirectURI != ""
}

func (h GoogleOAuthHandler) client() *http.Client {
	if h.HTTPClient != nil {
		return h.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (h GoogleOAuthHandler) endpoint(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func googleRandom() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func googleHash(value string) string {
	b := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func (h GoogleOAuthHandler) cookie(w http.ResponseWriter, name, value, path string, ttl int) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: value, Path: path, HttpOnly: true,
		Secure: h.Auth.S.CookieSecure, SameSite: http.SameSiteLaxMode,
		MaxAge: ttl,
	})
}

func (h GoogleOAuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"enabled": h.enabled()})
}

func (h GoogleOAuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	if !h.enabled() {
		httpx.Error(w, http.StatusServiceUnavailable, "google_disabled", "Login Google belum tersedia.")
		return
	}
	state, err := googleRandom()
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	nonce, err := googleRandom()
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	verifier, err := googleRandom()
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	var linkID *uuid.UUID
	if r.URL.Query().Get("link") == "1" {
		u, ok := UserFrom(r.Context())
		if !ok || u.Role != RoleParent {
			httpx.Error(w, http.StatusUnauthorized, "unauthenticated", "Masuk sebagai orang tua sebelum menautkan akun Google.")
			return
		}
		linkID = &u.ID
	}
	_, err = h.Auth.S.DB.Exec(r.Context(),
		"INSERT INTO google_oauth_flows (state_hash, nonce, code_verifier, link_user_id, expires_at) VALUES ($1, $2, $3, $4, now() + interval '10 minutes')",
		googleHash(state), nonce, verifier, linkID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	h.cookie(w, "eduplay_google_state", state, "/api/v1/auth/google", int(googleFlowTTL.Seconds()))
	q := url.Values{
		"client_id":             {h.ClientID},
		"redirect_uri":          {h.RedirectURI},
		"response_type":         {"code"},
		"scope":                 {"openid email profile"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {googleHash(verifier)},
		"code_challenge_method": {"S256"},
	}
	http.Redirect(w, r, h.endpoint(h.AuthURL, googleAuthEndpoint)+"?"+q.Encode(), http.StatusFound)
}

type googleClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Iss           string `json:"iss"`
	Aud           string `json:"aud"`
	Nonce         string `json:"nonce"`
	Exp           int64  `json:"exp"`
	Iat           int64  `json:"iat"`
}

type jwkSet struct {
	Keys []struct {
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Alg string `json:"alg"`
		Use string `json:"use"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

// verifyIDToken checks an RS256 signature with Google's public JWKs and the OIDC
// issuer, audience, nonce and expiry. Google tokens never become app sessions.
func (h GoogleOAuthHandler) verifyIDToken(ctx context.Context, raw, nonce string) (googleClaims, error) {
	var claims googleClaims
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return claims, errors.New("invalid JWT format")
	}
	headBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return claims, err
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headBytes, &header); err != nil || header.Alg != "RS256" || header.Kid == "" {
		return claims, errors.New("unsupported JWT signing header")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.endpoint(h.JWKSURL, googleJWKSURL), nil)
	if err != nil {
		return claims, err
	}
	res, err := h.client().Do(req)
	if err != nil {
		return claims, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return claims, fmt.Errorf("JWK provider returned status %d", res.StatusCode)
	}
	var keys jwkSet
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&keys); err != nil {
		return claims, err
	}
	var found bool
	for _, key := range keys.Keys {
		if key.Kid != header.Kid || key.Kty != "RSA" || (key.Use != "" && key.Use != "sig") || (key.Alg != "" && key.Alg != "RS256") {
			continue
		}
		nBytes, nErr := base64.RawURLEncoding.DecodeString(key.N)
		eBytes, eErr := base64.RawURLEncoding.DecodeString(key.E)
		if nErr != nil || eErr != nil || len(eBytes) == 0 || len(eBytes) > 4 {
			return claims, errors.New("invalid JWK")
		}
		exponent := 0
		for _, b := range eBytes {
			exponent = exponent<<8 | int(b)
		}
		if exponent < 3 {
			return claims, errors.New("invalid JWK exponent")
		}
		pub := &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: exponent}
		if pub.N.BitLen() < 2048 {
			return claims, errors.New("JWK key too short")
		}
		sig, err := base64.RawURLEncoding.DecodeString(parts[2])
		if err != nil {
			return claims, err
		}
		digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
		if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig); err != nil {
			return claims, errors.New("invalid token signature")
		}
		found = true
		break
	}
	if !found {
		return claims, errors.New("unknown JWT signing key")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, err
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return claims, err
	}
	now := time.Now().Unix()
	if (claims.Iss != "https://accounts.google.com" && claims.Iss != "accounts.google.com") ||
		claims.Aud != h.ClientID || claims.Exp <= now || claims.Iat > now+120 ||
		claims.Sub == "" || len(claims.Sub) > 255 || !claims.EmailVerified ||
		subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(nonce)) != 1 {
		return googleClaims{}, errors.New("OIDC claim validation failed")
	}
	claims.Email = normEmail(claims.Email)
	addr, err := mail.ParseAddress(claims.Email)
	if err != nil || addr.Address != claims.Email || len(claims.Email) > 254 {
		return googleClaims{}, errors.New("invalid identity email")
	}
	return claims, nil
}

func (h GoogleOAuthHandler) exchange(ctx context.Context, code, verifier string) (string, error) {
	form := url.Values{
		"client_id":     {h.ClientID},
		"client_secret": {h.Secret},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {h.RedirectURI},
		"grant_type":    {"authorization_code"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		h.endpoint(h.TokenURL, googleTokenEndpoint), strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := h.client().Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token exchange failed (status %d)", res.StatusCode)
	}
	var token struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&token); err != nil {
		return "", err
	}
	if token.IDToken == "" {
		return "", errors.New("missing id_token")
	}
	return token.IDToken, nil
}

func oauthRedirect(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, "/login?role=parent&oauth_error="+url.QueryEscape(code), http.StatusSeeOther)
}

func (h GoogleOAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("eduplay_google_state")
	state := r.URL.Query().Get("state")
	h.cookie(w, "eduplay_google_state", "", "/api/v1/auth/google", -1)
	if err != nil || state == "" || len(state) > 128 ||
		subtle.ConstantTimeCompare([]byte(state), []byte(cookie.Value)) != 1 {
		oauthRedirect(w, r, "state")
		return
	}
	var nonce, verifier string
	var linkID *uuid.UUID
	err = h.Auth.S.DB.QueryRow(r.Context(),
		"DELETE FROM google_oauth_flows WHERE state_hash = $1 AND expires_at > now() RETURNING nonce, code_verifier, link_user_id",
		googleHash(state)).Scan(&nonce, &verifier, &linkID)
	if err != nil {
		oauthRedirect(w, r, "expired")
		return
	}
	if r.URL.Query().Get("error") != "" {
		oauthRedirect(w, r, "cancelled")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		oauthRedirect(w, r, "provider")
		return
	}
	idToken, err := h.exchange(r.Context(), code, verifier)
	if err != nil {
		oauthRedirect(w, r, "provider")
		return
	}
	claims, err := h.verifyIDToken(r.Context(), idToken, nonce)
	if err != nil {
		oauthRedirect(w, r, "identity")
		return
	}
	if linkID != nil {
		u, ok := UserFrom(r.Context())
		if !ok || u.ID != *linkID || u.Role != RoleParent {
			oauthRedirect(w, r, "session")
			return
		}
		_, err = h.Auth.S.DB.Exec(r.Context(),
			"INSERT INTO auth_identities (user_id, provider, provider_subject, email_at_link) VALUES ($1, 'google', $2, $3)",
			u.ID, claims.Sub, claims.Email)
		if err != nil {
			oauthRedirect(w, r, "already_linked")
			return
		}
		http.Redirect(w, r, "/app/parent/settings?google=linked", http.StatusSeeOther)
		return
	}
	var user User
	err = h.Auth.S.DB.QueryRow(r.Context(),
		"SELECT u.id, u.email, u.display_name, u.role, u.created_at FROM auth_identities ai JOIN users u ON u.id = ai.user_id WHERE ai.provider = 'google' AND ai.provider_subject = $1",
		claims.Sub).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Role, &user.CreatedAt)
	if err == nil {
		if user.Role != RoleParent {
			oauthRedirect(w, r, "role")
			return
		}
		if err := h.Auth.S.CreateSession(r.Context(), w, user.ID); err != nil {
			httpx.Internal(w, r, err)
			return
		}
		http.Redirect(w, r, "/app/parent", http.StatusSeeOther)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		httpx.Internal(w, r, err)
		return
	}
	var exists bool
	if err := h.Auth.S.DB.QueryRow(r.Context(),
		"SELECT EXISTS(SELECT 1 FROM users WHERE lower(email) = $1)", claims.Email).Scan(&exists); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if exists {
		oauthRedirect(w, r, "account_exists")
		return
	}
	ticket, err := googleRandom()
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	name := strings.TrimSpace(claims.Name)
	if name == "" {
		name = "Orang Tua"
	}
	if len([]rune(name)) > 60 {
		name = string([]rune(name)[:60])
	}
	_, err = h.Auth.S.DB.Exec(r.Context(),
		"INSERT INTO google_signup_flows (ticket_hash, google_sub, email, display_name, expires_at) VALUES ($1, $2, $3, $4, now() + interval '10 minutes')",
		googleHash(ticket), claims.Sub, claims.Email, name)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	h.cookie(w, "eduplay_google_signup", ticket, "/api/v1/auth/google", int(googleFlowTTL.Seconds()))
	http.Redirect(w, r, "/register/google", http.StatusSeeOther)
}

func (h GoogleOAuthHandler) Pending(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("eduplay_google_signup")
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "signup_expired", "Pendaftaran Google telah kedaluwarsa.")
		return
	}
	var email, name string
	err = h.Auth.S.DB.QueryRow(r.Context(),
		"SELECT email, display_name FROM google_signup_flows WHERE ticket_hash = $1 AND expires_at > now()",
		googleHash(c.Value)).Scan(&email, &name)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "signup_expired", "Ulangi login Google.")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"email": email, "display_name": name})
}

func (h GoogleOAuthHandler) Complete(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("eduplay_google_signup")
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "signup_expired", "Ulangi login Google.")
		return
	}
	var in struct {
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
		Consent     bool   `json:"consent"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	fields := map[string]string{}
	if n := len([]rune(in.DisplayName)); n == 0 || n > 60 {
		fields["display_name"] = "Nama wajib diisi (maks. 60 karakter)."
	}
	if message := validatePassword(in.Password); message != "" {
		fields["password"] = message
	}
	if !in.Consent {
		fields["consent"] = "Persetujuan orang tua/wali diperlukan."
	}
	if len(fields) != 0 {
		httpx.Invalid(w, fields)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	tx, err := h.Auth.S.DB.Begin(r.Context())
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck
	var sub, email string
	err = tx.QueryRow(r.Context(),
		"DELETE FROM google_signup_flows WHERE ticket_hash = $1 AND expires_at > now() RETURNING google_sub, email",
		googleHash(c.Value)).Scan(&sub, &email)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "signup_expired", "Ulangi login Google.")
		return
	}
	var user User
	err = tx.QueryRow(r.Context(),
		"INSERT INTO users (email, password_hash, display_name, role, consent_version, consent_at) VALUES ($1, $2, $3, 'parent', $4, now()) RETURNING id, email, display_name, role, created_at",
		email, string(hash), in.DisplayName, ConsentVersion).Scan(&user.ID, &user.Email, &user.DisplayName, &user.Role, &user.CreatedAt)
	if err == nil {
		_, err = tx.Exec(r.Context(),
			"INSERT INTO auth_identities (user_id, provider, provider_subject, email_at_link) VALUES ($1, 'google', $2, $3)",
			user.ID, sub, email)
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpx.Error(w, http.StatusConflict, "account_exists", "Akun sudah terdaftar. Masuk menggunakan metode sebelumnya.")
		} else {
			httpx.Internal(w, r, err)
		}
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	h.cookie(w, "eduplay_google_signup", "", "/api/v1/auth/google", -1)
	if h.Auth.OnRegister != nil {
		h.Auth.OnRegister(r.Context(), user)
	}
	if err := h.Auth.S.CreateSession(r.Context(), w, user.ID); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"user": user})
}
