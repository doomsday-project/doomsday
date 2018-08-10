package notify

import (
	"fmt"
	"time"

	"github.com/thomasmmitchell/doomsday"
	"github.com/thomasmmitchell/doomsday/server/logger"
	"github.com/thomasmmitchell/doomsday/server/manager"
	"github.com/thomasmmitchell/doomsday/server/notify/backend"
	"github.com/thomasmmitchell/doomsday/server/notify/schedule"
)

type Config struct {
	Backend     BackendConfig  `yaml:"backend"`
	Schedule    ScheduleConfig `yaml:"schedule"`
	DoomsdayURL string         `yaml:"doomsday_url"`
	ShowOK      bool           `yaml:"notify_if_ok"`
}

type BackendConfig struct {
	Type       string                 `yaml:"type"`
	Properties map[string]interface{} `yaml:"properties"`
}

type ScheduleConfig struct {
	Type       string                 `yaml:"type"`
	Properties map[string]interface{} `yaml:"properties"`
}

type notifier struct {
	s schedule.Schedule
	b backend.Backend
}

func NotifyFrom(conf Config, m *manager.SourceManager, l *logger.Logger) error {
	var n notifier
	var err error

	if conf.DoomsdayURL == "" {
		return fmt.Errorf("Please provide doomsday_url")
	}
	n.s, err = schedule.New(conf.Schedule.Type, conf.Schedule.Properties)
	if err != nil {
		return fmt.Errorf("Error creating schedule: %s", err)
	}

	n.b, err = backend.New(conf.Backend.Type, conf.Backend.Properties)
	if err != nil {
		return fmt.Errorf("Error creating backend: %s", err)
	}

	n.s.Start()
	go func() {
		for range n.s.Channel() {
			l.Write("Triggering notification check")
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
				l.Write("No expiring certs")
				if conf.ShowOK {
					sendErr = n.b.Send(backend.Message{
						backend.MText{Text: "No tracked certs are expiring soon. For detailed information, check out the doomsday at "},
						backend.MLink{Link: conf.DoomsdayURL},
					})
				}
			case StateSoon:
				l.Write("Certs expiring soon")
				sendErr = n.b.Send(backend.Message{
					backend.MText{Text: "WARNING: You have certs expiring soon! For detailed information, check out the doomsday at "},
					backend.MLink{Link: conf.DoomsdayURL},
				})
			case StateExpired:
				l.Write("Certs expired")
				sendErr = n.b.Send(backend.Message{
					backend.MText{Text: "AHHH! You have expired certs! For detailed information, check out the doomsday at "},
					backend.MLink{Link: conf.DoomsdayURL},
				})
			}
			if sendErr != nil {
				l.Write("Could not send notification: %s", sendErr)
			}
		}
	}()

	return nil
}
