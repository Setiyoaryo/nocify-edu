package subscriptions

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"
)

// MidtransProvider implements Snap checkout and HTTP notifications.
// https://docs.midtrans.com/docs/https-notification-webhooks
//
// A notification is accepted only if (1) its signature_key equals
// SHA512(order_id + status_code + gross_amount + ServerKey) and (2) the
// transaction status re-read from Midtrans' Status API agrees. The status API
// response is authoritative for the status and amount we act on.
type MidtransProvider struct {
	ServerKey  string
	Production bool
	HTTP       *http.Client
}

func (m MidtransProvider) Name() string { return "midtrans" }

func (m MidtransProvider) snapURL() string {
	if m.Production {
		return "https://app.midtrans.com/snap/v1/transactions"
	}
	return "https://app.sandbox.midtrans.com/snap/v1/transactions"
}

func (m MidtransProvider) apiBase() string {
	if m.Production {
		return "https://api.midtrans.com"
	}
	return "https://api.sandbox.midtrans.com"
}

func (m MidtransProvider) client() *http.Client {
	if m.HTTP != nil {
		return m.HTTP
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (m MidtransProvider) do(ctx context.Context, method, url string, body any, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rd)
	if err != nil {
		return err
	}
	req.SetBasicAuth(m.ServerKey, "")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	res, err := m.client().Do(req)
	if err != nil {
		return fmt.Errorf("midtrans: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		return fmt.Errorf("midtrans %s %s: HTTP %d: %s", method, url, res.StatusCode, truncate(raw, 300))
	}
	return json.Unmarshal(raw, out)
}

func truncate(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "…"
	}
	return string(b)
}

func (m MidtransProvider) CreateCheckout(ctx context.Context, c Checkout) (string, error) {
	if m.ServerKey == "" {
		return "", ErrNotConfigured
	}
	body := map[string]any{
		"transaction_details": map[string]any{"order_id": c.Ref, "gross_amount": c.Amount},
		"item_details":        []map[string]any{{"id": c.Plan, "price": c.Amount, "quantity": 1, "name": c.ItemName}},
		"customer_details":    map[string]any{"first_name": c.Name, "email": c.Email},
		"callbacks":           map[string]any{"finish": c.FinishURL},
		"expiry":              map[string]any{"unit": "hours", "duration": 24},
	}
	var out struct {
		Token       string `json:"token"`
		RedirectURL string `json:"redirect_url"`
	}
	if err := m.do(ctx, http.MethodPost, m.snapURL(), body, &out); err != nil {
		return "", err
	}
	if out.RedirectURL == "" {
		return "", errors.New("midtrans: empty redirect_url")
	}
	return out.RedirectURL, nil
}

type midtransNotification struct {
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
	TransactionID     string `json:"transaction_id"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	Currency          string `json:"currency"`
	TransactionTime   string `json:"transaction_time"`
}

// Signature returns SHA512(order_id + status_code + gross_amount + ServerKey) in hex.
func (m MidtransProvider) Signature(orderID, statusCode, grossAmount string) string {
	sum := sha512.Sum512([]byte(orderID + statusCode + grossAmount + m.ServerKey))
	return hex.EncodeToString(sum[:])
}

func (m MidtransProvider) ParseWebhook(r *http.Request, body []byte) (Event, error) {
	if m.ServerKey == "" {
		return Event{}, ErrNotConfigured
	}
	var n midtransNotification
	if err := json.Unmarshal(body, &n); err != nil {
		return Event{}, ErrBadSignature
	}
	want := m.Signature(n.OrderID, n.StatusCode, n.GrossAmount)
	if n.OrderID == "" || subtle.ConstantTimeCompare([]byte(want), []byte(n.SignatureKey)) != 1 {
		return Event{}, ErrBadSignature
	}
	// Defence in depth: re-read the transaction from Midtrans.
	var st midtransNotification
	if err := m.do(r.Context(), http.MethodGet, m.apiBase()+"/v2/"+n.OrderID+"/status", nil, &st); err != nil {
		return Event{}, err
	}
	return MidtransEvent(st)
}

// MidtransEvent maps a Midtrans transaction to a provider-neutral event.
func MidtransEvent(n midtransNotification) (Event, error) {
	amount, err := strconv.ParseFloat(n.GrossAmount, 64)
	if err != nil {
		return Event{}, fmt.Errorf("midtrans: bad gross_amount %q", n.GrossAmount)
	}
	typ := "payment.pending"
	switch n.TransactionStatus {
	case "settlement":
		typ = "payment.succeeded"
	case "capture":
		if n.FraudStatus == "" || n.FraudStatus == "accept" {
			typ = "payment.succeeded"
		}
	case "deny", "cancel", "expire", "failure":
		typ = "payment.failed"
	}
	currency := n.Currency
	if currency == "" {
		currency = "IDR"
	}
	at, _ := time.ParseInLocation("2006-01-02 15:04:05", n.TransactionTime, time.FixedZone("WIB", 7*3600))
	return Event{
		// One order produces several notifications (pending → settlement);
		// each status is its own idempotent event.
		ID:          n.TransactionID + ":" + n.TransactionStatus,
		Type:        typ,
		CheckoutRef: n.OrderID,
		Amount:      int64(math.Round(amount)),
		Currency:    currency,
		OccurredAt:  at,
	}, nil
}

var ErrNotConfigured = errors.New("payment provider is not configured")
