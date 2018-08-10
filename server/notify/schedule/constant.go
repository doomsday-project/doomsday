package schedule

import (
	"fmt"
	"time"
)

type Constant struct {
	interval time.Duration
	c        chan bool
}

type ConstantConfig struct {
	Interval int `yaml:"interval"`
}

func newConstantSchedule(conf ConstantConfig) (*Constant, error) {
	if conf.Interval <= 0 {
		return nil, fmt.Errorf("Interval must be greater than 0")
	}

	return &Constant{
		interval: time.Duration(conf.Interval) * time.Minute,
		c:        make(chan bool),
	}, nil
}

func (c *Constant) Start() {
	go func() {
		for range time.Tick(c.interval) {
			c.c <- true
		}
	}()
}

func (c *Constant) Channel() chan bool {
	return c.c
}
