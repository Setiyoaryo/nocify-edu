package httpx

import (
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "eduplay_http_request_duration_seconds",
		Help:    "API latency by route.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status"})
)

// Observe logs one structured line per request and records latency by route
// pattern (not raw path, to keep label cardinality bounded and ids out of metrics).
func Observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		elapsed := time.Since(start)
		httpDuration.WithLabelValues(r.Method, route, strconv.Itoa(status)).Observe(elapsed.Seconds())
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		}
		slog.Log(r.Context(), level, "http",
			"method", r.Method, "route", route, "status", status,
			"ms", elapsed.Milliseconds(), "req_id", middleware.GetReqID(r.Context()))
	})
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// SameOrigin is the CSRF defence for cookie-authenticated, state-changing
// requests: SameSite=Lax cookies plus a strict Origin/Referer allow-list.
// Webhooks are mounted outside this middleware (they authenticate by signature).
func SameOrigin(allowed []string) func(http.Handler) http.Handler {
	set := map[string]bool{}
	for _, o := range allowed {
		set[strings.TrimRight(o, "/")] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}
			origin := r.Header.Get("Origin")
			if origin == "" {
				if u, err := url.Parse(r.Header.Get("Referer")); err == nil && u.Host != "" {
					origin = u.Scheme + "://" + u.Host
				}
			}
			// Non-browser clients (no Origin, no Referer) cannot carry a victim's
			// cookies cross-site, so they are allowed through.
			if origin != "" && !set[origin] {
				Error(w, http.StatusForbidden, "bad_origin", "Permintaan ditolak.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit is a fixed-window, per-client-IP limiter. It is in-memory, which is
// correct for the single-binary MVP; move to Redis only if the API is scaled out.
func RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	type bucket struct {
		start time.Time
		n     int
	}
	var mu sync.Mutex
	buckets := map[string]*bucket{}
	go func() {
		for range time.Tick(window) {
			mu.Lock()
			for k, b := range buckets {
				if time.Since(b.start) > window {
					delete(buckets, k)
				}
			}
			mu.Unlock()
		}
	}()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := ClientIP(r)
			mu.Lock()
			b := buckets[key]
			now := time.Now()
			if b == nil || now.Sub(b.start) > window {
				b = &bucket{start: now}
				buckets[key] = b
			}
			b.n++
			over := b.n > limit
			mu.Unlock()
			if over {
				w.Header().Set("Retry-After", strconv.Itoa(int(window.Seconds())))
				Error(w, http.StatusTooManyRequests, "rate_limited", "Terlalu banyak percobaan. Coba lagi sebentar lagi.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP trusts X-Forwarded-For's first hop only because the API is only
// reachable through Traefik; RealIP middleware is not used for that reason.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
