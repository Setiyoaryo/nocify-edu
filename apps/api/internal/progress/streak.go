package progress

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/nocify/eduplay/api/internal/db"
)

// Streak counts consecutive local days with at least one completed challenge,
// ending today or yesterday (so it does not break before the child has had a
// chance to play today).
func Streak(ctx context.Context, q db.DBTX, childID uuid.UUID, tz *time.Location) (int, error) {
	rows, err := q.Query(ctx, `SELECT day FROM daily_progress WHERE child_id = $1 AND challenges_completed > 0 ORDER BY day DESC LIMIT 400`, childID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var days []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return 0, err
		}
		days = append(days, d)
	}
	return StreakFrom(days, time.Now().In(tz)), rows.Err()
}

// StreakFrom is the pure rule, days sorted newest first (dates only).
func StreakFrom(days []time.Time, now time.Time) int {
	day := func(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }
	cur := day(now)
	if len(days) == 0 {
		return 0
	}
	first := day(days[0])
	if first.Before(cur.AddDate(0, 0, -1)) {
		return 0
	}
	n := 0
	expect := first
	for _, d := range days {
		if !day(d).Equal(expect) {
			break
		}
		n++
		expect = expect.AddDate(0, 0, -1)
	}
	return n
}
