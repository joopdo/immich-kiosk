package partials

import (
	"testing"
	"time"

	"github.com/damongolding/immich-kiosk/internal/config"
)

func TestDayOffset(t *testing.T) {
	for days, want := range map[int]string{0: "", 1: "+1", -1: "-1", 2: "+2"} {
		if got := dayOffset(days); got != want {
			t.Errorf("dayOffset(%d) = %q, want %q", days, got, want)
		}
	}
}

func TestSecondaryClockTime(t *testing.T) {
	c := config.Config{TimeFormat: "24", SecondaryClockTimezone: "Asia/Seoul"}
	got, day := secondaryClockTime(c)

	loc, _ := time.LoadLocation("Asia/Seoul")
	now := time.Now()
	want := now.In(loc).Format("15:04")
	// allow a minute boundary between the two time.Now() calls
	if got != want && got != now.Add(time.Minute).In(loc).Format("15:04") {
		t.Errorf("secondaryClockTime = %q, want %q", got, want)
	}
	if day != "" && day != "+1" && day != "-1" {
		t.Errorf("unexpected day offset %q", day)
	}

	c.SecondaryClockTimezone = "Not/AZone"
	if got, day := secondaryClockTime(c); got != "" || day != "" {
		t.Errorf("invalid timezone should give empty result, got %q %q", got, day)
	}
}
