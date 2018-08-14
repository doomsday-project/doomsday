package schedule

import (
	"fmt"
	"time"

	"github.com/robfig/cron"
)

type Cron struct {
	sched cron.Schedule
	c     chan bool
}

type CronConfig struct {
	Spec string `yaml:"spec"`
}

func newCronSchedule(conf CronConfig) (*Cron, error) {
	if conf.Spec == "" {
		return nil, fmt.Errorf("Cron spec must be given")
	}

	ret := &Cron{c: make(chan bool)}
	var err error

	ret.sched, err = cron.ParseStandard(conf.Spec)
	if err != nil {
		return nil, fmt.Errorf("Could not parse cron spec: %s", err)
	}

	return ret, nil
}

func (c *Cron) Start() {
	go func() {
		t := time.Now()
		for {
			t = c.sched.Next(t)
			time.Sleep(time.Until(t))
			c.c <- true
		}
	}()
}

func (c *Cron) Channel() chan bool {
	return c.c
}
