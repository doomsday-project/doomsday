package server

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/doomsday-project/doomsday/client/doomsday"
	"github.com/doomsday-project/doomsday/server/logger"
	"github.com/doomsday-project/doomsday/storage"
)

type Source struct {
	Core        *Core
	Interval    time.Duration
	lock        sync.RWMutex
	authTTL     time.Duration
	lastRefresh *RunInfo
	lastAuth    *RunInfo
	shouldAuth  bool
}

type RunInfo struct {
	At  time.Time
	Err error
}

func (s *Source) Refresh(global *Cache, mode string, log *logger.Logger) {
	log.WriteF("Running %s populate of `%s'", mode, s.Core.Name)
	startedAt := time.Now()
	old := s.Core.Cache()
	if old == nil {
		old = NewCache()
	}
	results, err := s.Core.Populate()
	if err != nil {
		log.WriteF("Error populating info from backend `%s': %s", s.Core.Name, err)
		s.lock.Lock()
		s.lastRefresh = &RunInfo{
			At:  startedAt,
			Err: err,
		}
		s.lock.Unlock()
		return
	}
	global.ApplyDiff(old, s.Core.Cache())
	log.WriteF("Finished %s populate of `%s' after %s. %d/%d paths searched. %d certs found",
		mode, s.Core.Name, time.Since(startedAt), results.NumSuccess, results.NumPaths, results.NumCerts)
	s.lock.Lock()
	s.lastRefresh = &RunInfo{At: startedAt}
	s.lock.Unlock()
}

func (s *Source) CalcNextRefresh() time.Time {
	s.lock.RLock()
	ret := s.lastRefresh.At.Add(s.Interval)
	s.lock.RUnlock()
	return ret
}

func (s *Source) CalcNextAuth() (time.Time, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if !s.shouldAuth {
		return time.Time{}, fmt.Errorf("No further auth is possible")
	}
	if s.authTTL == storage.TTLInfinite {
		return time.Time{}, fmt.Errorf("No further auth is necessary")
	}
	return s.lastAuth.At.Add(s.authTTL), nil
}

func (s *Source) LastRefresh() *RunInfo {
	s.lock.RLock()
	ret := s.lastRefresh
	s.lock.RUnlock()
	return ret
}

const (
	queueTaskKindAuth uint = iota
	queueTaskKindRefresh
)

const (
	runReasonSchedule uint = iota
	runReasonAdhoc
)

type managerTask struct {
	kind    uint
	source  *Source
	runTime time.Time
	reason  uint
}

func (m *managerTask) ready() bool {
	return !m.runTime.Before(time.Now())
}

func (m *managerTask) durationUntil() time.Duration {
	return m.runTime.Sub(time.Now())
}

type taskQueue struct {
	data []managerTask
	lock *sync.Mutex
	cond *sync.Cond
}

func newTaskQueue() *taskQueue {
	lock := &sync.Mutex{}
	return &taskQueue{
		lock: lock,
		cond: sync.NewCond(lock),
	}
}

//next blocks until the next task is ready to be run
// only one caller should call next at a time.
func (t *taskQueue) next() managerTask {
	t.lock.Lock()
	for t.empty() {
		t.cond.Wait() //wait for something to be inserted
	}

	//At this point, taskQueue can't be empty again until we take stuff out...
	//So, now we need to wait until either the task at the front of the queue is
	// ready or until something else is inserted. If something else is inserted,
	// repeat this process, because the thing that was inserted may be scheduled
	// to occur before the thing we were originally waiting on, and while we're
	// waiting on that, something else might be inserted, and so on.
	for !t.data[0].ready() {
		timer := time.AfterFunc(t.data[0].durationUntil(), func() {
			//This can technically fire very late if the timing is right (wrong?). We
			//could hook up more sync to make it so that can't happen, but as it
			//stands, we're not catastrophically impacted by spurious wakes. As long
			//as it stays that way, this is fine.
			t.lock.Lock()
			t.cond.Broadcast()
			t.lock.Unlock()
		})
		t.cond.Wait()
		timer.Stop()
	}
	ret := t.dequeueNoLock()
	t.lock.Unlock()
	return ret
}

func (t *taskQueue) enqueue(task managerTask) {
	t.lock.Lock()
	t.data = append(t.data, task)
	t.sort()
	t.cond.Broadcast()
	t.lock.Unlock()
}

func (t *taskQueue) dequeueNoLock() managerTask {
	ret := t.data[0]
	t.data[0] = t.data[len(t.data)-1]
	t.data = t.data[:len(t.data)-1]
	return ret
}

func (t *taskQueue) sort() {
	sort.Slice(t.data, func(i, j int) bool {
		return t.data[i].runTime.Before(t.data[j].runTime)
	})
}

func (t *taskQueue) empty() bool {
	return len(t.data) == 0
}

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
	for _, current := range s.sources {
		s.queue.enqueue(managerTask{
			source:  &current,
			kind:    queueTaskKindRefresh,
			runTime: now,
			reason:  runReasonAdhoc,
		})
	}
}
