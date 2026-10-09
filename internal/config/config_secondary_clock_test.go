package config

import "testing"

func TestCheckSecondaryClock(t *testing.T) {
	tests := []struct {
		tz, label, wantTZ, wantLabel string
	}{
		{"", "", "", ""},
		{" Asia/Seoul ", "", "Asia/Seoul", "Seoul"},
		{"Asia/Seoul", "🇰🇷 대전", "Asia/Seoul", "🇰🇷 대전"},
		{"America/New_York", "", "America/New_York", "New York"},
		{"Not/AZone", "x", "", "x"},
	}
	for _, tt := range tests {
		c := &Config{SecondaryClockTimezone: tt.tz, SecondaryClockLabel: tt.label}
		c.checkSecondaryClock()
		if c.SecondaryClockTimezone != tt.wantTZ || c.SecondaryClockLabel != tt.wantLabel {
			t.Errorf("checkSecondaryClock(%q,%q) = (%q,%q), want (%q,%q)",
				tt.tz, tt.label, c.SecondaryClockTimezone, c.SecondaryClockLabel, tt.wantTZ, tt.wantLabel)
		}
	}
}
