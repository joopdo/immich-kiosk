package nowplaying

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func f(v float64) *float64 { return &v }

func TestFromState(t *testing.T) {
	now := time.Date(2026, 11, 17, 21, 0, 0, 0, time.UTC)

	var s haState
	s.State = "playing"
	s.Attributes.Title = "The Bear"
	s.Attributes.Subtitle = "S3 E4 · Violet"
	s.Attributes.Duration = f(3600)
	s.Attributes.Position = f(600)
	s.Attributes.PositionUpdatedAt = now.Add(-60 * time.Second).Format(time.RFC3339Nano)

	i := fromState(s, now)
	if i == nil {
		t.Fatal("want info")
	}
	if i.Position != 660 {
		t.Errorf("position extrapolated = %v, want 660", i.Position)
	}
	if want := now.Add(2940 * time.Second); !i.EndsAt.Equal(want) {
		t.Errorf("EndsAt = %v, want %v", i.EndsAt, want)
	}

	// paused: no extrapolation
	s.State = "paused"
	if i := fromState(s, now); i.Position != 600 || !i.Paused() {
		t.Errorf("paused position = %v", i.Position)
	}

	// position past the end is clamped
	s.State = "playing"
	s.Attributes.PositionUpdatedAt = now.Add(-2 * time.Hour).Format(time.RFC3339Nano)
	if i := fromState(s, now); i.Position != 3600 || i.Progress() != 100 {
		t.Errorf("clamp: position = %v progress = %v", i.Position, i.Progress())
	}

	// live stream: no duration -> no end time
	s.Attributes.Duration = nil
	if i := fromState(s, now); !i.EndsAt.IsZero() || i.Progress() != 0 {
		t.Errorf("no duration should have no end time")
	}

	for _, st := range []string{"off", "idle", "unavailable", ""} {
		s.State = st
		if fromState(s, now) != nil {
			t.Errorf("state %q should hide", st)
		}
	}

	s.State = "playing"
	s.Attributes.Title = "  "
	if fromState(s, now) != nil {
		t.Error("empty title should hide")
	}
}

func TestFormatDuration(t *testing.T) {
	for in, want := range map[float64]string{0: "0:00", 59: "0:00", 2700: "0:45", 4320: "1:12", 7440: "2:04"} {
		if got := FormatDuration(in); got != want {
			t.Errorf("FormatDuration(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/api/states/sensor.photoframe_now_playing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"state":"playing","attributes":{"title":"Dune","duration":9000,"position":null}}`))
	}))
	defer srv.Close()

	i, err := Fetch(t.Context(), srv.URL, "tok", "sensor.photoframe_now_playing", time.Now())
	if err != nil || i == nil || i.Title != "Dune" || i.Position != 0 {
		t.Fatalf("Fetch = %+v, %v", i, err)
	}
	if _, err := Fetch(t.Context(), srv.URL, "wrong", "sensor.photoframe_now_playing", time.Now()); err == nil {
		t.Error("want error on 401")
	}
	if _, err := Fetch(t.Context(), "", "tok", "x", time.Now()); err == nil {
		t.Error("want error when not configured")
	}
}

func TestArtTracker(t *testing.T) {
	var tr ArtTracker
	t0 := time.Date(2026, 11, 18, 20, 0, 0, 0, time.UTC)
	info := func(state string) *Info {
		return &Info{State: state, Title: "Dune", Picture: "/api/media_player_proxy/media_player.kodi?x=1"}
	}
	w := 60 * time.Second

	i := info(StatePlaying)
	tr.Update(i, t0, w)
	if !i.ShowArt {
		t.Error("new title should show art")
	}
	i = info(StatePlaying)
	tr.Update(i, t0.Add(61*time.Second), w)
	if i.ShowArt {
		t.Error("art should hide after the window")
	}
	i = info(StatePaused)
	tr.Update(i, t0.Add(90*time.Second), w)
	if i.ShowArt {
		t.Error("pausing should not bring the art back")
	}
	i = info(StatePlaying)
	tr.Update(i, t0.Add(120*time.Second), w)
	if !i.ShowArt {
		t.Error("resume should show art again")
	}
	i = info(StatePlaying)
	tr.Update(i, t0.Add(500*time.Second), -1)
	if !i.ShowArt {
		t.Error("window -1 = always")
	}
	i = info(StatePlaying)
	tr.Update(i, t0, 0)
	if i.ShowArt {
		t.Error("window 0 = never")
	}
	tr.Update(nil, t0, w)
	if tr.Picture() != "" {
		t.Error("nil should clear the picture")
	}
}

func TestArtPathAllowed(t *testing.T) {
	for p, want := range map[string]bool{
		"/api/media_player_proxy/media_player.kodirp5?token=a&cache=b": true,
		"/api/states/sensor.x":                 false,
		"/api/media_player_proxy/../config":    false,
		"http://evil/api/media_player_proxy/x": false,
		"":                                     false,
	} {
		if ArtPathAllowed(p) != want {
			t.Errorf("ArtPathAllowed(%q) != %v", p, want)
		}
	}
}
