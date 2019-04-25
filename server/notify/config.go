package notify

import (
	"github.com/doomsday-project/doomsday/server/notify/backend"
	"github.com/doomsday-project/doomsday/server/notify/schedule"
)

type Config struct {
	Backend     backend.Config  `yaml:"backend"`
	Schedule    schedule.Config `yaml:"schedule"`
	DoomsdayURL string          `yaml:"doomsday_url"`
}
