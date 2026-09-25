package subscriptions

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"
)

var ErrBadSignature = errors.New("invalid webhook signature")

type Checkout struct {
	Ref       string
	Plan      string
	Amount    int64
	Currency  string
	Email     string
	Name      string
	ItemName  string
	FinishURL string // where the provider sends the browser back
}

// Event is the provider-neutral shape the subscription service consumes.
type Event struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // payment.succeeded | payment.failed | payment.pending
	CheckoutRef string    `json:"checkout_ref"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// Provider adapts one payment gateway (Midtrans, Xendit, Stripe, ...).
type Provider interface {
	Name() string
	CreateCheckout(ctx context.Context, c Checkout) (redirectURL string, err error)
	// ParseWebhook must authenticate the request before returning an Event.
	ParseWebhook(r *http.Request, body []byte) (Event, error)
}

// MockProvider is a development/staging gateway. It signs webhooks with
// HMAC-SHA256(PAYMENT_SECRET, body) exactly like a real provider would, so the
// verification + idempotency path is exercised end to end.
type MockProvider struct {
	Secret    []byte
	PublicURL string
}

const MockSignatureHeader = "X-Mock-Signature"

func (m MockProvider) Name() string { return "mock" }

func (m MockProvider) CreateCheckout(_ context.Context, c Checkout) (string, error) {
	return m.PublicURL + "/app/checkout/mock?ref=" + url.QueryEscape(c.Ref), nil
}

func (m MockProvider) Sign(body []byte) string {
	h := hmac.New(sha256.New, m.Secret)
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func (m MockProvider) ParseWebhook(r *http.Request, body []byte) (Event, error) {
	sig, err := hex.DecodeString(r.Header.Get(MockSignatureHeader))
	if err != nil {
		return Event{}, ErrBadSignature
	}
	h := hmac.New(sha256.New, m.Secret)
	h.Write(body)
	if !hmac.Equal(sig, h.Sum(nil)) {
		return Event{}, ErrBadSignature
	}
	var ev Event
	if err := json.Unmarshal(body, &ev); err != nil {
		return Event{}, err
	}
	if ev.ID == "" || ev.CheckoutRef == "" || ev.Type == "" {
		return Event{}, errors.New("incomplete event")
	}
	return ev, nil
}
