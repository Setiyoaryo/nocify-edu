package subscriptions

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// roundTrip lets the provider's HTTP client hit a fake Midtrans.
type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func fakeMidtrans(t *testing.T, status map[string]string) *http.Client {
	return &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		user, _, _ := r.BasicAuth()
		if user != "SB-Mid-server-test" {
			t.Errorf("missing basic auth")
		}
		var body []byte
		switch {
		case strings.HasSuffix(r.URL.Path, "/snap/v1/transactions"):
			body = []byte(`{"token":"tok","redirect_url":"https://app.sandbox.midtrans.com/snap/v3/redirection/tok"}`)
		case strings.HasSuffix(r.URL.Path, "/status"):
			body, _ = json.Marshal(status)
		default:
			t.Fatalf("unexpected %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Header: http.Header{}}, nil
	})}
}

func TestMidtransCheckoutAndWebhook(t *testing.T) {
	status := map[string]string{"order_id": "chk_1", "transaction_id": "tx-9", "transaction_status": "settlement",
		"gross_amount": "49000.00", "status_code": "200", "currency": "IDR", "transaction_time": "2026-09-25 10:00:00"}
	m := MidtransProvider{ServerKey: "SB-Mid-server-test", HTTP: fakeMidtrans(t, status)}

	url, err := m.CreateCheckout(httptest.NewRequest("GET", "/", nil).Context(), Checkout{Ref: "chk_1", Amount: 49000, Plan: "monthly"})
	if err != nil || !strings.Contains(url, "/snap/v3/redirection/") {
		t.Fatalf("checkout: %q %v", url, err)
	}

	n := map[string]string{"order_id": "chk_1", "status_code": "200", "gross_amount": "49000.00", "transaction_status": "settlement", "transaction_id": "tx-9"}
	n["signature_key"] = m.Signature(n["order_id"], n["status_code"], n["gross_amount"])
	body, _ := json.Marshal(n)
	ev, err := m.ParseWebhook(httptest.NewRequest("POST", "/", nil), body)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != "payment.succeeded" || ev.Amount != 49000 || ev.CheckoutRef != "chk_1" || ev.ID != "tx-9:settlement" {
		t.Fatalf("unexpected event %+v", ev)
	}

	n["signature_key"] = strings.Repeat("0", 128)
	body, _ = json.Marshal(n)
	if _, err := m.ParseWebhook(httptest.NewRequest("POST", "/", nil), body); err != ErrBadSignature {
		t.Fatalf("forged signature must be rejected, got %v", err)
	}
}

func TestMidtransStatusMapping(t *testing.T) {
	cases := map[[2]string]string{
		{"settlement", ""}:       "payment.succeeded",
		{"capture", "accept"}:    "payment.succeeded",
		{"capture", "challenge"}: "payment.pending",
		{"pending", ""}:          "payment.pending",
		{"expire", ""}:           "payment.failed",
		{"deny", ""}:             "payment.failed",
		{"cancel", ""}:           "payment.failed",
		{"refund", ""}:           "payment.pending", // not acted on automatically
	}
	for in, want := range cases {
		ev, err := MidtransEvent(midtransNotification{TransactionStatus: in[0], FraudStatus: in[1], GrossAmount: "49000.00", OrderID: "o", TransactionID: "t"})
		if err != nil || ev.Type != want {
			t.Errorf("%v → %s (%v), want %s", in, ev.Type, err, want)
		}
	}
}
