package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNormPhone(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"081234567890", "+6281234567890", false},
		{"+6281234567890", "+6281234567890", false},
		{"6281234567890", "+6281234567890", false},
		{"81234567890", "+6281234567890", false},
		{"0812-3456-7890", "+6281234567890", false},
		{"(0812) 3456 7890", "+6281234567890", false},
		{"123", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		got, err := NormPhone(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("NormPhone(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("NormPhone(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestOTPStore(t *testing.T) {
	s := &Service{}
	phone := "+6281234567890"
	code := "654321"

	s.SetOTP(phone, code, time.Now().Add(1*time.Minute))
	if !s.CheckOTP(phone, code) {
		t.Errorf("expected OTP to be valid")
	}

	// second check should fail (one-time use)
	if s.CheckOTP(phone, code) {
		t.Errorf("expected OTP to be consumed after first check")
	}

	// expired OTP
	s.SetOTP(phone, code, time.Now().Add(-1*time.Second))
	if s.CheckOTP(phone, code) {
		t.Errorf("expected expired OTP to fail")
	}
}

func TestSendOTPHandlerValidation(t *testing.T) {
	s := &Service{}
	h := Handler{S: s}

	// Invalid phone
	body, _ := json.Marshal(map[string]any{"phone": "invalid"})
	req := httptest.NewRequest("POST", "/auth/otp/send", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.SendOTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d", w.Code)
	}

	// Valid phone
	body, _ = json.Marshal(map[string]any{"phone": "081234567890", "role": "parent"})
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
}
