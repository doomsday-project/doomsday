package server

import (
	"sort"
	"sync"
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
	//sources is a static view of the queue, such that the get calls don't need
	// to sync with the populate calls. Otherwise, we would need to read-lock on
	// get calls so that the order of the queue doesn't shift from underneath us
	// as the async scheduler is running
	sources []Source
	queue   *taskQueue
	lock    sync.RWMutex
	log     *logger.Logger
	global  *Cache
}

func NewSourceManager(sources []Source, log *logger.Logger) *SourceManager {
	if log == nil {
		panic("No logger was given")
	}

	queue := newTaskQueue()
	now := time.Now()
	for i := range sources {
		queue.enqueue(managerTask{
			kind:    queueTaskKindAuth,
			source:  &sources[i],
			runTime: now,
			reason:  runReasonSchedule,
		})
	}

	return &SourceManager{
		sources: sources,
		queue:   queue,
		log:     log,
		global:  NewCache(),
	}
}

func (s *SourceManager) BackgroundScheduler() {
	//TODO: Seed the queue

	go func() {
		for {
			current := s.queue.next()
			//TODO: Remember to push off schedule if an ad-hoc has been run in the meantime
			current.source.Refresh(s.global, "scheduled", s.log)
		}
	}()
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
