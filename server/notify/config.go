package notify

import (
	"github.com/thomasmmitchell/doomsday/server/notify/backend"
	"github.com/thomasmmitchell/doomsday/server/notify/schedule"
)

type Config struct {
	Backend     backend.Config  `yaml:"backend"`
	Schedule    schedule.Config `yaml:"schedule"`
	DoomsdayURL string          `yaml:"doomsday_url"`
}
