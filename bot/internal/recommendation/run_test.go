package recommendation

import (
	"testing"
	"time"
)

func TestDurationUntilNextHourUTC_beforeHour(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 20, 8, 30, 0, 0, time.UTC)
	d := DurationUntilNextHourUTC(9, now)
	if d != 30*time.Minute {
		t.Fatalf("got %v want 30m", d)
	}
}

func TestDurationUntilNextHourUTC_afterHour_nextDay(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	d := DurationUntilNextHourUTC(9, now)
	if d != 23*time.Hour {
		t.Fatalf("got %v want 23h", d)
	}
}
