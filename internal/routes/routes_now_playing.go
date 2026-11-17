package routes

import (
	"net/http"
	"sync"
	"time"

	"charm.land/log/v2"
	"github.com/labstack/echo/v5"

	"github.com/damongolding/immich-kiosk/internal/config"
	"github.com/damongolding/immich-kiosk/internal/nowplaying"
	"github.com/damongolding/immich-kiosk/internal/templates/partials"
)

// nowPlayingLastErr makes HA errors visible in the log without repeating them every poll.
var (
	nowPlayingMu      sync.Mutex
	nowPlayingLastErr string
)

func logNowPlayingErr(err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	nowPlayingMu.Lock()
	defer nowPlayingMu.Unlock()
	if msg == nowPlayingLastErr {
		return
	}
	if msg == "" {
		log.Info("now playing: Home Assistant reachable again")
	} else {
		log.Warn("now playing: cannot read Home Assistant entity", "err", msg)
	}
	nowPlayingLastErr = msg
}

// NowPlaying renders the now-playing corner. It always answers 200 (empty body when
// there is nothing to show) because htmx does not swap on 204, which would leave a
// stale title on screen.
func NowPlaying(baseConfig *config.Config) echo.HandlerFunc {
	return func(c *echo.Context) error {
		requestData, err := InitializeRequestData(c, baseConfig)
		if err != nil {
			return err
		}
		if requestData == nil {
			return nil
		}

		np := requestData.RequestConfig.NowPlaying
		if !np.Enabled {
			return Render(c, http.StatusOK, partials.NowPlaying(nil, "", nil))
		}

		info, fetchErr := nowplaying.Fetch(c.Request().Context(), np.HAURL, np.Token, np.Entity, time.Now())
		logNowPlayingErr(fetchErr)
		if fetchErr != nil {
			info = nil
		}

		return Render(c, http.StatusOK, partials.NowPlaying(info, requestData.RequestConfig.TimeFormat, np.Location))
	}
}
