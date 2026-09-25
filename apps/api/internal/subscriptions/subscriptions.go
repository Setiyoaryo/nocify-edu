// Package subscriptions owns plans, entitlements and the payment webhook (§21–22).
// The payment gateway is behind Provider so the business domain never depends
// on a specific vendor.
package subscriptions

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nocify/eduplay/api/internal/db"
)

const (
	PlanFree    = "free"
	PlanMonthly = "monthly"
	// PastDueGrace keeps access while the provider retries a failed renewal.
	PastDueGrace = 3 * 24 * time.Hour
)

type Limits struct {
	MaxChildren     int `json:"max_children"`
	DailyChallenges int `json:"daily_challenges"` // 0 = unlimited
}

// Entitlement is the only thing the rest of the backend asks about billing.
// It is always derived from PostgreSQL, never from the client (§21).
type Entitlement struct {
	Plan              string     `json:"plan"`
	Premium           bool       `json:"premium"`
	Status            string     `json:"status,omitempty"`
	CurrentPeriodEnd  *time.Time `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd bool       `json:"cancel_at_period_end"`
	Limits            Limits     `json:"limits"`
}

type Policy struct {
	FreeDailyChallenges int
	FreeMaxChildren     int
	PaidMaxChildren     int
	MonthlyPriceIDR     int64
}

// IsPremium applies the status rules documented in docs/subscriptions.md.
func IsPremium(status string, periodEnd, now time.Time) bool {
	switch status {
	case "trial", "active", "cancelled":
		return now.Before(periodEnd)
	case "past_due":
		return now.Before(periodEnd.Add(PastDueGrace))
	}
	return false
}

func (p Policy) Get(ctx context.Context, q db.DBTX, userID uuid.UUID) (Entitlement, error) {
	var status, plan string
	var end time.Time
	var cancel bool
	err := q.QueryRow(ctx, `SELECT plan, status, current_period_end, cancel_at_period_end FROM subscriptions WHERE user_id = $1`, userID).
		Scan(&plan, &status, &end, &cancel)
	free := Entitlement{Plan: PlanFree, Limits: Limits{MaxChildren: p.FreeMaxChildren, DailyChallenges: p.FreeDailyChallenges}}
	if errors.Is(err, pgx.ErrNoRows) {
		return free, nil
	}
	if err != nil {
		return Entitlement{}, err
	}
	e := free
	e.Status = status
	e.CurrentPeriodEnd = &end
	e.CancelAtPeriodEnd = cancel
	if IsPremium(status, end, time.Now()) {
		e.Plan = plan
		e.Premium = true
		e.Limits = Limits{MaxChildren: p.PaidMaxChildren, DailyChallenges: 0}
	}
	return e, nil
}

// StartOfDay is local midnight; the free daily challenge quota resets then.
func StartOfDay(tz *time.Location) time.Time {
	now := time.Now().In(tz)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, tz)
}
