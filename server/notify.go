package server

import (
	"fmt"
	"time"

	"github.com/thomasmmitchell/doomsday/client/doomsday"
	"github.com/thomasmmitchell/doomsday/server/logger"
	"github.com/thomasmmitchell/doomsday/server/notify"
	"github.com/thomasmmitchell/doomsday/server/notify/backend"
	"github.com/thomasmmitchell/doomsday/server/notify/schedule"
)

type notifier struct {
	s schedule.Schedule
	b backend.Backend
}

func NotifyFrom(conf notify.Config, m *SourceManager, l *logger.Logger) error {
	var n notifier
	var err error

	if conf.DoomsdayURL == "" {
		return fmt.Errorf("Please provide doomsday_url")
	}
	n.s, err = schedule.New(conf.Schedule.Type, conf.Schedule.Properties)
	if err != nil {
		return fmt.Errorf("Error creating schedule: %s", err)
	}

	uni := backend.BackendUniversalConfig{
		DoomsdayURL: conf.DoomsdayURL,
		Logger:      l,
	}
	n.b, err = backend.New(conf.Backend, uni)
	if err != nil {
		return fmt.Errorf("Error creating backend: %s", err)
	}

	n.s.Start()
	go func() {
		for range n.s.Channel() {
			l.WriteF("Triggering notification check")
			const (
				StateOK = iota
				StateExpired
				StateSoon
			)

			d := m.Data()
			state := StateOK
			expiredThreshold := time.Duration(0)
			expiringSoonThreshold := time.Hour * 24 * 7 * 4
			if len(d.Filter(doomsday.CacheItemFilter{Within: &expiredThreshold})) > 0 {
				state = StateExpired
			} else if len(d.Filter(doomsday.CacheItemFilter{Within: &expiringSoonThreshold})) > 0 {
				state = StateSoon
			}

			var sendErr error
			switch state {
			case StateOK:
				l.WriteF("No expiring certs")
				sendErr = n.b.OK()
			case StateSoon:
				l.WriteF("Certs expiring soon")
				sendErr = n.b.Soon()
			case StateExpired:
				l.WriteF("Certs expired")
				sendErr = n.b.Expired()
			}
			if sendErr != nil {
				l.WriteF("Could not send notification: %s", sendErr)
			}
		}
	}()

	return nil
}
