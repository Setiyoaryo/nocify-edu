package subscriptions

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/nocify/eduplay/api/internal/analytics"
	"github.com/nocify/eduplay/api/internal/audit"
	"github.com/nocify/eduplay/api/internal/auth"
	"github.com/nocify/eduplay/api/internal/httpx"
)

var webhookResults = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "eduplay_payment_webhooks_total",
	Help: "Payment webhooks by outcome (processed, duplicate, rejected, failed).",
}, []string{"outcome"})

type Handler struct {
	DB        *pgxpool.Pool
	Policy    Policy
	Provider  Provider
	PublicURL string
}

func (h Handler) Get(w http.ResponseWriter, r *http.Request) {
	e, err := h.Policy.Get(r.Context(), h.DB, auth.MustUser(r.Context()).ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"subscription": e,
		"offer":        map[string]any{"plan": PlanMonthly, "amount": h.Policy.MonthlyPriceIDR, "currency": "IDR", "interval": "month"},
	})
}

func newRef(prefix string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func (h Handler) Checkout(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Plan string `json:"plan"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if in.Plan != PlanMonthly {
		httpx.Invalid(w, map[string]string{"plan": "Paket tidak tersedia."})
		return
	}
	u := auth.MustUser(r.Context())
	ref := newRef("chk_")
	c := Checkout{Ref: ref, Plan: in.Plan, Amount: h.Policy.MonthlyPriceIDR, Currency: "IDR",
		Email: u.Email, Name: u.DisplayName, ItemName: "EduPlay Langganan Bulanan",
		FinishURL: h.PublicURL + "/app/parent/subscription?checkout=" + ref}
	if _, err := h.DB.Exec(r.Context(), `INSERT INTO payment_checkouts (user_id, plan, amount, currency, provider, provider_ref) VALUES ($1,$2,$3,$4,$5,$6)`,
		u.ID, c.Plan, c.Amount, c.Currency, h.Provider.Name(), c.Ref); err != nil {
		httpx.Internal(w, r, err)
		return
	}
	redirect, err := h.Provider.CreateCheckout(r.Context(), c)
	if errors.Is(err, ErrNotConfigured) {
		httpx.Error(w, http.StatusServiceUnavailable, "payments_unavailable", "Pembayaran belum dapat diproses. Silakan coba lagi nanti.")
		return
	}
	if err != nil {
		_, _ = h.DB.Exec(r.Context(), `UPDATE payment_checkouts SET status = 'failed' WHERE provider_ref = $1`, ref)
		httpx.Internal(w, r, err)
		return
	}
	analytics.Track(r.Context(), h.DB, "checkout_started", u.ID, uuid.Nil, map[string]any{"plan": c.Plan})
	// The browser is only redirected; activation happens exclusively via webhook (§22).
	httpx.JSON(w, http.StatusCreated, map[string]any{"checkout_ref": c.Ref, "redirect_url": redirect})
}

// CheckoutStatus lets the return page poll until the webhook has landed.
func (h Handler) CheckoutStatus(w http.ResponseWriter, r *http.Request) {
	var status string
	err := h.DB.QueryRow(r.Context(), `SELECT status FROM payment_checkouts WHERE provider_ref = $1 AND user_id = $2`,
		chi.URLParam(r, "ref"), auth.MustUser(r.Context()).ID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": status})
}

func (h Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	u := auth.MustUser(r.Context())
	tag, err := h.DB.Exec(r.Context(), `UPDATE subscriptions SET status = 'cancelled', cancel_at_period_end = true, updated_at = now()
		WHERE user_id = $1 AND status IN ('active','trial')`, u.ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, http.StatusConflict, "not_active", "Tidak ada langganan aktif.")
		return
	}
	audit.Log(r.Context(), h.DB, u.ID, "subscription.cancel", "subscription", u.ID.String(), nil)
	analytics.Track(r.Context(), h.DB, "subscription_cancelled", u.ID, uuid.Nil, nil)
	h.Get(w, r)
}

func (h Handler) Resume(w http.ResponseWriter, r *http.Request) {
	u := auth.MustUser(r.Context())
	tag, err := h.DB.Exec(r.Context(), `UPDATE subscriptions SET status = 'active', cancel_at_period_end = false, updated_at = now()
		WHERE user_id = $1 AND status = 'cancelled' AND current_period_end > now()`, u.ID)
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, http.StatusConflict, "not_resumable", "Langganan tidak dapat dilanjutkan.")
		return
	}
	audit.Log(r.Context(), h.DB, u.ID, "subscription.resume", "subscription", u.ID.String(), nil)
	h.Get(w, r)
}

// Webhook is mounted without session auth or CSRF checks; it authenticates by
// provider signature and is idempotent on (provider, event id).
func (h Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "unreadable body")
		return
	}
	status, msg := h.handleWebhook(r, body)
	httpx.JSON(w, status, map[string]string{"result": msg})
}

func (h Handler) handleWebhook(r *http.Request, body []byte) (int, string) {
	ctx := r.Context()
	ev, err := h.Provider.ParseWebhook(r, body)
	if errors.Is(err, ErrNotConfigured) {
		return http.StatusServiceUnavailable, "not_configured"
	}
	if err != nil && !errors.Is(err, ErrBadSignature) {
		// Signature was fine but verifying with the provider failed: let it retry.
		webhookResults.WithLabelValues("failed").Inc()
		slog.ErrorContext(ctx, "payment webhook verification failed", "err", err)
		return http.StatusServiceUnavailable, "retry"
	}
	if err != nil {
		webhookResults.WithLabelValues("rejected").Inc()
		slog.WarnContext(ctx, "payment webhook rejected", "err", err, "ip", httpx.ClientIP(r))
		return http.StatusUnauthorized, "rejected"
	}
	outcome, err := h.ProcessEvent(ctx, ev, body)
	if err != nil {
		webhookResults.WithLabelValues("failed").Inc()
		slog.ErrorContext(ctx, "payment webhook failed", "err", err, "event_id", ev.ID)
		// Record the error outside the rolled-back tx so operators can see it;
		// a non-2xx lets the provider retry safely.
		_, _ = h.DB.Exec(ctx, `INSERT INTO payment_events (provider, event_id, event_type, payload, error) VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (provider, event_id) DO UPDATE SET error = EXCLUDED.error`, h.Provider.Name(), ev.ID, ev.Type, body, err.Error())
		return http.StatusInternalServerError, "error"
	}
	webhookResults.WithLabelValues(outcome).Inc()
	slog.InfoContext(ctx, "payment webhook", "event_id", ev.ID, "type", ev.Type, "outcome", outcome)
	return http.StatusOK, outcome
}

// ProcessEvent applies a verified event exactly once.
func (h Handler) ProcessEvent(ctx context.Context, ev Event, raw []byte) (string, error) {
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var eventRowID int64
	var processedAt *time.Time
	err = tx.QueryRow(ctx, `INSERT INTO payment_events (provider, event_id, event_type, payload) VALUES ($1,$2,$3,$4)
		ON CONFLICT (provider, event_id) DO UPDATE SET provider = EXCLUDED.provider
		RETURNING id, processed_at`, h.Provider.Name(), ev.ID, ev.Type, raw).Scan(&eventRowID, &processedAt)
	if err != nil {
		return "", err
	}
	if processedAt != nil {
		return "duplicate", tx.Commit(ctx)
	}
	// The upsert above holds the inbox row lock until commit, so concurrent
	// deliveries of the same event serialise here and the later one sees processed_at.

	var userID uuid.UUID
	var ckStatus, plan string
	var amount int64
	var currency string
	err = tx.QueryRow(ctx, `SELECT user_id, status, plan, amount, currency FROM payment_checkouts WHERE provider_ref = $1 AND provider = $2 FOR UPDATE`,
		ev.CheckoutRef, h.Provider.Name()).Scan(&userID, &ckStatus, &plan, &amount, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("unknown checkout %q", ev.CheckoutRef)
	}
	if err != nil {
		return "", err
	}

	outcome := "processed"
	switch ev.Type {
	case "payment.succeeded":
		if ev.Amount != amount || ev.Currency != currency {
			return "", fmt.Errorf("amount mismatch: got %d %s want %d %s", ev.Amount, ev.Currency, amount, currency)
		}
		if ckStatus == "paid" {
			outcome = "already_paid"
			break
		}
		if _, err := tx.Exec(ctx, `UPDATE payment_checkouts SET status = 'paid', paid_at = now() WHERE provider_ref = $1`, ev.CheckoutRef); err != nil {
			return "", err
		}
		// New period starts now, or extends an unexpired one (early renewal).
		if _, err := tx.Exec(ctx, `
			INSERT INTO subscriptions (user_id, plan, status, current_period_start, current_period_end, provider, provider_ref)
			VALUES ($1, $2, 'active', now(), now() + interval '1 month', $3, $4)
			ON CONFLICT (user_id) DO UPDATE SET
				plan = EXCLUDED.plan,
				status = 'active',
				cancel_at_period_end = false,
				current_period_start = CASE WHEN subscriptions.current_period_end > now() THEN subscriptions.current_period_start ELSE now() END,
				current_period_end   = GREATEST(subscriptions.current_period_end, now()) + interval '1 month',
				provider = EXCLUDED.provider, provider_ref = EXCLUDED.provider_ref, updated_at = now()`,
			userID, plan, h.Provider.Name(), ev.CheckoutRef); err != nil {
			return "", err
		}
		audit.Log(ctx, tx, uuid.Nil, "subscription.activated", "subscription", userID.String(), map[string]any{"checkout_ref": ev.CheckoutRef, "event_id": ev.ID})
		analytics.Track(ctx, tx, "payment_success", userID, uuid.Nil, map[string]any{"plan": plan})
	case "payment.failed":
		if ckStatus == "pending" {
			if _, err := tx.Exec(ctx, `UPDATE payment_checkouts SET status = 'failed' WHERE provider_ref = $1`, ev.CheckoutRef); err != nil {
				return "", err
			}
		}
		audit.Log(ctx, tx, uuid.Nil, "payment.failed", "payment_checkout", ev.CheckoutRef, map[string]any{"event_id": ev.ID})
		analytics.Track(ctx, tx, "payment_failed", userID, uuid.Nil, nil)
	case "payment.pending":
		outcome = "pending"
	default:
		outcome = "ignored"
	}
	if _, err := tx.Exec(ctx, `UPDATE payment_events SET processed_at = now(), error = NULL WHERE id = $1`, eventRowID); err != nil {
		return "", err
	}
	return outcome, tx.Commit(ctx)
}

// MockPay lets a signed-in parent complete their own mock checkout. It builds a
// signed webhook and feeds it through the same verification path a real
// provider would hit. Mounted only when PAYMENT_PROVIDER=mock outside production.
func (h Handler) MockPay(w http.ResponseWriter, r *http.Request) {
	mock, ok := h.Provider.(MockProvider)
	if !ok {
		httpx.NotFound(w)
		return
	}
	var in struct {
		Outcome string `json:"outcome"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	ref := chi.URLParam(r, "ref")
	var amount int64
	var currency string
	err := h.DB.QueryRow(r.Context(), `SELECT amount, currency FROM payment_checkouts WHERE provider_ref = $1 AND user_id = $2`,
		ref, auth.MustUser(r.Context()).ID).Scan(&amount, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.NotFound(w)
		return
	}
	if err != nil {
		httpx.Internal(w, r, err)
		return
	}
	typ := "payment.succeeded"
	if in.Outcome == "failed" {
		typ = "payment.failed"
	}
	body, _ := json.Marshal(Event{ID: newRef("evt_"), Type: typ, CheckoutRef: ref, Amount: amount, Currency: currency, OccurredAt: time.Now().UTC()})
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, "/webhooks/payment", bytes.NewReader(body))
	req.Header.Set(MockSignatureHeader, mock.Sign(body))
	status, msg := h.handleWebhook(req, body)
	httpx.JSON(w, status, map[string]string{"result": msg})
}
