package schedule

import (
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type Schedule interface {
	Start()
	Channel() chan bool
}

const (
	typeUnknown int = iota
	typeConstant
	typeCron
)

func New(scheduleType string, conf map[string]interface{}) (Schedule, error) {
	properties, err := yaml.Marshal(&conf)
	if err != nil {
		panic("Could not re-marshal into YAML")
	}

	t := resolveType(strings.ToLower(scheduleType))
	if t == typeUnknown {
		return nil, fmt.Errorf("Unrecognized schedule type (%s)", scheduleType)
	}

	var c interface{}

	switch t {
	case typeConstant:
		c = &ConstantConfig{}
		err = yaml.Unmarshal(properties, c.(*ConstantConfig))
	case typeCron:
		c = &CronConfig{}
		err = yaml.Unmarshal(properties, c.(*CronConfig))
	}

	if err != nil {
		return nil, fmt.Errorf("Error when parsing backend config: %s", err)
	}

	var schedule Schedule
	switch t {
	case typeConstant:
		schedule, err = newConstantSchedule(*c.(*ConstantConfig))
	case typeCron:
		schedule, err = newCronSchedule(*c.(*CronConfig))
	}

	return schedule, err
}

func resolveType(t string) int {
	switch t {
	case "constant", "interval":
		return typeConstant
	case "cron", "cronspec":
		return typeCron
	default:
		return typeUnknown
	}
}
