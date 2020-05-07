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
	queue := newTaskQueue(globalCache, log)

	return &SourceManager{
		sources: sources,
		queue:   queue,
		log:     log,
		global:  globalCache,
	}
}

func (s *SourceManager) BackgroundScheduler() error {
	for i := range s.sources {
		s.sources[i].Auth(s.log)
		if s.sources[i].authStatus.LastErr != nil {
			return fmt.Errorf("Error performing initial auth for backend `%s': %s",
				s.sources[i].Core.Name,
				s.sources[i].authStatus.LastErr)
		}
	}

	now := time.Now()

	for i := range s.sources {
		s.queue.enqueue(managerTask{
			kind:    queueTaskKindRefresh,
			source:  &s.sources[i],
			runTime: now,
			reason:  runReasonSchedule,
		})

		nextAuthTime, skipAuth := s.sources[i].CalcNextAuth()
		if !skipAuth {
			s.queue.enqueue(managerTask{
				kind:    queueTaskKindAuth,
				source:  &s.sources[i],
				runTime: nextAuthTime,
				reason:  runReasonSchedule,
			})
		}
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
	for i := range s.sources {
		s.queue.enqueue(managerTask{
			source:  &s.sources[i],
			kind:    queueTaskKindRefresh,
			runTime: now,
			reason:  runReasonAdhoc,
		})
	}
}

func (s *SourceManager) SchedulerState() SchedulerState {
	return s.queue.dumpState()
}
