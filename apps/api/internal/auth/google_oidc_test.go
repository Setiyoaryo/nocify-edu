package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGoogleRandomAndHash(t *testing.T) {
	a, err := googleRandom()
	if err != nil {
		t.Fatal(err)
	}
	b, err := googleRandom()
	if err != nil {
		t.Fatal(err)
	}
	if a == b || len(a) < 32 || googleHash(a) == a || googleHash(a) == googleHash(b) {
		t.Fatal("OAuth state and PKCE verifier must be random and hashed deterministically")
	}
	if googleHash(a) != googleHash(a) {
		t.Fatal("hash must be deterministic")
	}
}

func TestGoogleDisabledStatusAndStart(t *testing.T) {
	h := GoogleOAuthHandler{}
	status := httptest.NewRecorder()
	h.Status(status, httptest.NewRequest(http.MethodGet, "/auth/google/status", nil))
	if status.Code != http.StatusOK || !strings.Contains(status.Body.String(), "\"enabled\":false") {
		t.Fatalf("disabled status: %d: %s", status.Code, status.Body.String())
	}
	start := httptest.NewRecorder()
	h.Start(start, httptest.NewRequest(http.MethodGet, "/auth/google/start", nil))
	if start.Code != http.StatusServiceUnavailable {
		t.Fatalf("disabled start status = %d", start.Code)
	}
}

func TestGoogleOAuthUnsupportedJWT(t *testing.T) {
	h := GoogleOAuthHandler{ClientID: "test-client"}
	if _, err := h.verifyIDToken(t.Context(), "not-a-jwt", "nonce"); err == nil {
		t.Fatal("unsigned or malformed ID tokens must fail")
	}
}
