package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestNormPhone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"standard 08...", "081234567890", "+6281234567890", false},
		{"standard +628...", "+6281234567890", "+6281234567890", false},
		{"standard 628...", "6281234567890", "+6281234567890", false},
		{"bugfix +620...", "+62081234567890", "+6281234567890", false},
		{"bugfix 620...", "62081234567890", "+6281234567890", false},
		{"bugfix +6200...", "+620081234567890", "+6281234567890", false},
		{"bugfix 0062...", "006281234567890", "+6281234567890", false},
		{"bare national 8...", "81234567890", "+6281234567890", false},
		{"hyphenated 08...", "0812-3456-7890", "+6281234567890", false},
		{"parenthesized (08...)", "(0812) 3456 7890", "+6281234567890", false},
		{"formatted +62 (08...)", "+62 (0812) 3456-7890", "+6281234567890", false},
		{"spaced +62 8...", "+62 812 3456 7890", "+6281234567890", false},
		{"leading and trailing whitespace", "   081234567890   ", "+6281234567890", false},
		{"dot separated 08...", "0812.3456.7890", "+6281234567890", false},
		{"too short digits", "123", "", true},
		{"empty string", "", "", true},
		{"only whitespace", "    ", "", true},
		{"only prefix +62", "+62", "", true},
		{"prefix +620", "+620", "", true},
		{"only zeros 00000000", "00000000", "", true},
		{"prefix +62 with only zeros", "+6200000", "", true},
		{"too long digits (> 15)", "081234567890123456", "", true},
		{"alphabetic input", "abcdefghijk", "", true},
		{"alphanumeric mixed suffix", "081234567890abc", "", true},
		{"alphanumeric mixed prefix", "abc081234567890", "", true},
		{"special symbols @#$", "0812@3456#7890", "", true},
		{"plus in middle", "0812+34567890", "", true},
		{"multiple plus signs", "++6281234567890", "", true},
		{"foreign plus US +1", "+1 202 555 0123", "", true},
		{"foreign plus SG +65", "+65 9123 4567", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormPhone(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormPhone(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("NormPhone(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

type errReader struct{}

var _ io.Reader = (*errReader)(nil)

func (e *errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("entropy source exhausted")
}

func TestGenerateOTP(t *testing.T) {
	// 1. Success case: generates 6 digits in range [100000, 999999]
	code, err := GenerateOTP()
	if err != nil {
		t.Fatalf("GenerateOTP() failed: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6 digits, got %q", code)
	}
	n, err := strconv.Atoi(code)
	if err != nil {
		t.Fatalf("code is not numeric: %q", code)
	}
	if n < 100000 || n > 999999 {
		t.Fatalf("code %d out of range [100000, 999999]", n)
	}

	// 2. Failure case: CSPRNG error must return an error and NOT fallback to static "123456"
	badCode, err := GenerateOTPReader(&errReader{})
	if err == nil {
		t.Fatalf("expected CSPRNG error, got nil with code: %q", badCode)
	}
	if badCode != "" {
		t.Fatalf("expected empty code on CSPRNG error, got %q", badCode)
	}
}

func TestOTPStore(t *testing.T) {
	s := &Service{}
	phone := "+6281234567890"
	code := "654321"

	// Nonexistent OTP check fails
	if s.CheckOTP(phone, code) {
		t.Fatalf("expected CheckOTP to return false for nonexistent OTP")
	}

	// Set valid OTP
	s.SetOTP(phone, code, time.Now().Add(1*time.Minute))
	if !s.CheckOTP(phone, code) {
		t.Fatalf("expected OTP to be valid")
	}

	// Second check fails (one-time consumption)
	if s.CheckOTP(phone, code) {
		t.Fatalf("expected OTP to be consumed after first successful check")
	}

	// Expired OTP fails
	s.SetOTP(phone, code, time.Now().Add(-1*time.Second))
	if s.CheckOTP(phone, code) {
		t.Fatalf("expected expired OTP to fail")
	}

	// Brute-force protection: invalidates after 5 wrong attempts
	s.SetOTP(phone, code, time.Now().Add(1*time.Minute))
	for i := 0; i < 4; i++ {
		if s.CheckOTP(phone, "000000") {
			t.Fatalf("wrong code should not succeed")
		}
	}
	// 5th wrong attempt triggers deletion
	if s.CheckOTP(phone, "000000") {
		t.Fatalf("5th wrong attempt should fail")
	}
	// 6th attempt with correct code fails because OTP was deleted
	if s.CheckOTP(phone, code) {
		t.Fatalf("expected OTP to be invalidated after 5 failed attempts")
	}
}

func TestSendOTPHandler(t *testing.T) {
	s := &Service{}
	h := Handler{S: s}

	// 1. Malformed JSON body
	req := httptest.NewRequest("POST", "/auth/otp/send", bytes.NewReader([]byte("{invalid-json")))
	w := httptest.NewRecorder()
	h.SendOTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for malformed json, got %d", w.Code)
	}

	// 2. Invalid phone
	body, _ := json.Marshal(map[string]any{"phone": "invalid"})
	req = httptest.NewRequest("POST", "/auth/otp/send", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.SendOTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 for invalid phone, got %d", w.Code)
	}

	// 3. Valid phone
	body, _ = json.Marshal(map[string]any{"phone": "+62081234567890", "role": "parent"})
	req = httptest.NewRequest("POST", "/auth/otp/send", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.SendOTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["status"] != "ok" {
		t.Errorf("expected status ok, got %v", res["status"])
	}
	if res["phone"] != "+6281234567890" {
		t.Errorf("expected normalized phone +6281234567890, got %v", res["phone"])
	}

	// 4. CRITICAL: Ensure dev_code is NOT leaked in JSON response!
	if devCode, exists := res["dev_code"]; exists {
		t.Fatalf("SECURITY VIOLATION: dev_code was leaked in response: %v", devCode)
	}

	// Verify that an OTP was stored in the service
	phone := "+6281234567890"
	val, ok := s.otps.Load(phone)
	if !ok {
		t.Fatalf("expected OTP entry to be stored in Service.otps for phone %s", phone)
	}
	entry := val.(*otpEntry)
	if len(entry.code) != 6 {
		t.Fatalf("expected stored OTP to be 6 digits, got %q", entry.code)
	}
}

func TestVerifyOTPHandler_NoBackdoor(t *testing.T) {
	s := &Service{}
	h := Handler{S: s}

	phone := "+6281234567890"
	correctCode := "789123"
	s.SetOTP(phone, correctCode, time.Now().Add(5*time.Minute))

	// 1. Attempt backdoor bypass "123456" - MUST BE REJECTED (HTTP 401)
	body, _ := json.Marshal(map[string]any{
		"phone": phone,
		"code":  "123456",
		"role":  "parent",
	})
	req := httptest.NewRequest("POST", "/auth/otp/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.VerifyOTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("SECURITY VIOLATION: backdoor '123456' was accepted! Got status %d, expected 401", w.Code)
	}

	// 2. Attempt backdoor bypass on another phone that never requested OTP - MUST BE REJECTED (HTTP 401)
	body, _ = json.Marshal(map[string]any{
		"phone": "089999999999",
		"code":  "123456",
		"role":  "parent",
	})
	req = httptest.NewRequest("POST", "/auth/otp/verify", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.VerifyOTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("SECURITY VIOLATION: backdoor '123456' on unrequested phone was accepted! Got status %d, expected 401", w.Code)
	}

	// 3. Empty OTP code validation (HTTP 422)
	body, _ = json.Marshal(map[string]any{
		"phone": phone,
		"code":  "",
		"role":  "parent",
	})
	req = httptest.NewRequest("POST", "/auth/otp/verify", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.VerifyOTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 for empty code, got %d", w.Code)
	}

	// 4. Invalid phone validation (HTTP 422)
	body, _ = json.Marshal(map[string]any{
		"phone": "invalid",
		"code":  "123456",
		"role":  "parent",
	})
	req = httptest.NewRequest("POST", "/auth/otp/verify", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.VerifyOTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 for invalid phone, got %d", w.Code)
	}

	// 5. Malformed JSON (HTTP 400)
	req = httptest.NewRequest("POST", "/auth/otp/verify", bytes.NewReader([]byte("{bad-json")))
	w = httptest.NewRecorder()
	h.VerifyOTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for bad json, got %d", w.Code)
	}

	// 6. Child role with invalid phone (HTTP 422)
	body, _ = json.Marshal(map[string]any{
		"phone": "0000",
		"code":  "123456",
		"role":  "child",
	})
	req = httptest.NewRequest("POST", "/auth/otp/verify", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.VerifyOTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 for invalid child phone, got %d", w.Code)
	}

	// 7. Valid OTP with nil DB returns 503 instead of crashing
	body, _ = json.Marshal(map[string]any{
		"phone":        phone,
		"code":         correctCode,
		"role":         "parent",
		"display_name": "Bunda Hebat",
	})
	req = httptest.NewRequest("POST", "/auth/otp/verify", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.VerifyOTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 for nil DB, got %d", w.Code)
	}
}
