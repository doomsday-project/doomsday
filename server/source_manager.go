package server

import (
	"fmt"
	"sort"
	"time"

	"github.com/doomsday-project/doomsday/client/doomsday"
	"github.com/doomsday-project/doomsday/server/logger"
)

const (
	MinAuthInterval      = 5 * time.Second
	ExpiredRetryInterval = 5 * time.Minute
)

const AdHocThrottle = 30 * time.Second

type SourceManager struct {
	sources []Source
	queue   *taskQueue
	log     *logger.Logger
	global  *Cache
}

func NewSourceManager(sources []Source, log *logger.Logger) *SourceManager {
	if log == nil {
		panic("No logger was given")
	}

	globalCache := NewCache()
	queue := newTaskQueue(globalCache)

	return &SourceManager{
		sources: sources,
		queue:   queue,
		log:     log,
		global:  globalCache,
	}
}

func (s *SourceManager) BackgroundScheduler() error {
	now := time.Now()
	for i := range s.sources {
		s.sources[i].Auth(s.log)
		if s.sources[i].authStatus.LastErr != nil {
			return fmt.Errorf("Error performing initial auth for backend `%s': %s",
				s.sources[i].Core.Name,
				s.sources[i].authStatus.LastErr)
		}
	}

	for i := range s.sources {
		s.queue.enqueue(managerTask{
			kind:    queueTaskKindRefresh,
			source:  &s.sources[i],
			runTime: now.Add(1 * time.Second),
			reason:  runReasonSchedule,
		})
	}

	s.queue.start()
	return nil
}

func (s *SourceManager) Data() doomsday.CacheItems {
	items := []doomsday.CacheItem{}
	for _, v := range s.global.Map() {
		paths := []doomsday.CacheItemPath{}
		for _, path := range v.Paths {
			paths = append(paths, doomsday.CacheItemPath{
				Backend:  path.Source,
				Location: path.Location,
			})
		}
		items = append(items, doomsday.CacheItem{
			Paths:      paths,
			CommonName: v.Subject.CommonName,
			NotAfter:   v.NotAfter.Unix(),
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].NotAfter < items[j].NotAfter })
	return items
}

func (s *SourceManager) RefreshAll() {
	now := time.Now()
	cutoff := now.Add(-AdHocThrottle)
	for _, current := range s.sources {
		if current.refreshStatus.LastRun.StartedAt.IsZero() || //it was never run?
			(!current.refreshStatus.LastRun.FinishedAt.IsZero() && //if FinishedAt is zero, its currently in progress
				current.refreshStatus.LastRun.FinishedAt.Before(cutoff)) {
			s.queue.enqueue(managerTask{
				source:  &current,
				kind:    queueTaskKindRefresh,
				runTime: now,
				reason:  runReasonAdhoc,
			})
		}
	}
}

func (s *SourceManager) SchedulerState() SchedulerState {
	return s.queue.dumpState()
}
