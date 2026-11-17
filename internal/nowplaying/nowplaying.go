// Package nowplaying reads a "now playing" summary from a Home Assistant entity
// so the kiosk can show what is playing on a nearby TV.
//
// The HA entity does the media-player logic (which player, on/off toggle, when to
// hide). Kiosk only needs this contract:
//
//	state:      "playing" | "paused" (anything else = hide)
//	attributes: title, subtitle (strings), duration, position (seconds),
//	            position_updated_at (ISO 8601, when position was sampled)
package nowplaying

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	StatePlaying = "playing"
	StatePaused  = "paused"
)

// Info is what the kiosk renders. A nil *Info means "show nothing".
type Info struct {
	State    string
	Title    string
	Subtitle string
	// Duration and Position in seconds. Duration 0 = unknown length (live TV, stream).
	Duration float64
	Position float64
	// EndsAt is when playback finishes if it keeps playing from now. Zero if Duration is unknown.
	EndsAt time.Time
}

// Paused reports whether playback is paused.
func (i *Info) Paused() bool { return i != nil && i.State == StatePaused }

// Progress returns 0..100.
func (i *Info) Progress() float64 {
	if i == nil || i.Duration <= 0 {
		return 0
	}
	p := i.Position / i.Duration * 100
	return max(0, min(100, p))
}

type haState struct {
	State      string `json:"state"`
	Attributes struct {
		Title             string   `json:"title"`
		Subtitle          string   `json:"subtitle"`
		Duration          *float64 `json:"duration"`
		Position          *float64 `json:"position"`
		PositionUpdatedAt string   `json:"position_updated_at"`
	} `json:"attributes"`
}

var client = &http.Client{Timeout: 5 * time.Second}

// Fetch reads the entity from the HA REST API. It returns (nil, nil) when there is
// nothing to show.
func Fetch(ctx context.Context, haURL, token, entity string, now time.Time) (*Info, error) {
	if haURL == "" || token == "" || entity == "" {
		return nil, errors.New("now_playing not configured")
	}

	endpoint, err := url.JoinPath(haURL, "api", "states", entity)
	if err != nil {
		return nil, fmt.Errorf("building HA url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HA returned %s", resp.Status)
	}

	var s haState
	if err := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, 64<<10)).Decode(&s); err != nil {
		return nil, fmt.Errorf("decoding HA state: %w", err)
	}

	return fromState(s, now), nil
}

func fromState(s haState, now time.Time) *Info {
	if s.State != StatePlaying && s.State != StatePaused {
		return nil
	}
	a := s.Attributes
	title := strings.TrimSpace(a.Title)
	if title == "" {
		return nil
	}

	info := &Info{State: s.State, Title: title, Subtitle: strings.TrimSpace(a.Subtitle)}
	if a.Duration != nil && *a.Duration > 0 {
		info.Duration = *a.Duration
	}
	if a.Position != nil && *a.Position > 0 {
		info.Position = *a.Position
	}

	// HA samples the position now and then; move it forward to "now" while playing.
	if info.State == StatePlaying && a.PositionUpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, a.PositionUpdatedAt); err == nil {
			if d := now.Sub(t).Seconds(); d > 0 {
				info.Position += d
			}
		}
	}

	if info.Duration > 0 {
		info.Position = min(info.Position, info.Duration)
		info.EndsAt = now.Add(time.Duration((info.Duration - info.Position) * float64(time.Second)))
	}

	return info
}

// FormatDuration renders seconds as h:mm (e.g. 1:12, 0:45). Minutes only: the
// widget refreshes every few seconds, so seconds would just flicker.
func FormatDuration(seconds float64) string {
	m := int(seconds) / 60
	return fmt.Sprintf("%d:%02d", m/60, m%60)
}
