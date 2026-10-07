package httpx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "remote addr with port",
			remoteAddr: "192.168.1.50:12345",
			headers:    nil,
			want:       "192.168.1.50",
		},
		{
			name:       "remote addr without port",
			remoteAddr: "10.0.0.1",
			headers:    nil,
			want:       "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For takes precedence over spoofed X-Real-IP",
			remoteAddr: "192.168.1.50:12345",
			headers: map[string]string{
				"X-Real-IP":       "203.0.113.195",
				"X-Forwarded-For": "1.2.3.4, 198.51.100.99",
			},
			want: "198.51.100.99",
		},
		{
			name:       "X-Real-IP used when X-Forwarded-For absent",
			remoteAddr: "192.168.1.50:12345",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.195",
			},
			want: "203.0.113.195",
		},
		{
			name:       "X-Forwarded-For multiple hops picks rightmost valid IP (prevents spoofing)",
			remoteAddr: "192.168.1.50:12345",
			headers: map[string]string{
				"X-Forwarded-For": "1.1.1.1, 2.2.2.2, 203.0.113.50",
			},
			want: "203.0.113.50",
		},
		{
			name:       "X-Forwarded-For single valid IP",
			remoteAddr: "192.168.1.50:12345",
			headers: map[string]string{
				"X-Forwarded-For": "198.51.100.22",
			},
			want: "198.51.100.22",
		},
		{
			name:       "X-Forwarded-For with garbage falls back to RemoteAddr",
			remoteAddr: "192.168.1.50:12345",
			headers: map[string]string{
				"X-Forwarded-For": "invalid-ip-string",
			},
			want: "192.168.1.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/api", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			got := ClientIP(req)
			if got != tt.want {
				t.Fatalf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRateLimit(t *testing.T) {
	limiter := RateLimit(3, time.Minute)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}
	}

	// 4th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("request 4: expected 429, got %d", rec.Code)
	}

	// Different IP should still be allowed
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "10.0.0.2:1234"
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("request from another IP: expected 200, got %d", rec2.Code)
	}
}

func TestRateLimitCapacityCap(t *testing.T) {
	limiter := RateLimit(5, time.Minute)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Fill up to maxRateLimitBuckets (10,000)
	for i := 0; i < maxRateLimitBuckets; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		b1 := (i >> 8) & 0xff
		b2 := i & 0xff
		req.RemoteAddr = fmt.Sprintf("10.0.%d.%d:1234", b1, b2)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rec.Code)
		}
	}

	// 10,001st request from a new IP should be rejected with 429 (shedding)
	reqOver := httptest.NewRequest(http.MethodGet, "/test", nil)
	reqOver.RemoteAddr = "192.168.1.1:1234"
	recOver := httptest.NewRecorder()
	handler.ServeHTTP(recOver, reqOver)
	if recOver.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 when capacity is exceeded, got %d", recOver.Code)
	}

	// Existing IP that is already in buckets should still be allowed (not shed)
	reqExisting := httptest.NewRequest(http.MethodGet, "/test", nil)
	reqExisting.RemoteAddr = "10.0.0.0:1234"
	recExisting := httptest.NewRecorder()
	handler.ServeHTTP(recExisting, reqExisting)
	if recExisting.Code != http.StatusOK {
		t.Fatalf("existing IP: expected 200, got %d", recExisting.Code)
	}
}
