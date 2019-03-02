package manager

import (
	"sort"
	"sync"
	"time"

	"github.com/thomasmmitchell/doomsday"
	"github.com/thomasmmitchell/doomsday/server/logger"
)

type Source struct {
	Core     *doomsday.Core
	Interval time.Duration
	nextRun  time.Time
}

//Bump sets nextRun to now + interval
func (s *Source) Bump() {
	s.nextRun = time.Now().Add(s.Interval)
}

func (s *Source) Refresh(global *doomsday.Cache, mode string, log *logger.Logger) {
	log.WriteF("Running %s populate of `%s'", mode, s.Core.Name)
	startedAt := time.Now()
	old := s.Core.Cache()
	if old == nil {
		old = doomsday.NewCache()
	}
	results, err := s.Core.Populate()
	if err != nil {
		log.WriteF("Error populating info from backend `%s': %s", s.Core.Name, err)
		return
	}
	global.ApplyDiff(old, s.Core.Cache())
	log.WriteF("Finished %s populate of `%s' after %s. %d/%d paths searched. %d certs found",
		mode, s.Core.Name, time.Since(startedAt), results.NumSuccess, results.NumPaths, results.NumCerts)
}

type SourceManager struct {
	//sources is a static view of the queue, such that the get calls don't need
	// to sync with the populate calls. Otherwise, we would need to read-lock on
	// get calls so that the order of the queue doesn't shift from underneath us
	// as the async scheduler is running
	sources []Source
	queue   []*Source
	lock    sync.RWMutex
	log     *logger.Logger
	global  *doomsday.Cache
}

func NewSourceManager(sources []Source, log *logger.Logger) *SourceManager {
	if log == nil {
		panic("No logger was given")
	}

	queue := make([]*Source, 0, len(sources))
	for i := range sources {
		queue = append(queue, &sources[i])
	}

	return &SourceManager{
		sources: sources,
		queue:   queue,
		log:     log,
		global:  doomsday.NewCache(),
	}
}

func (s *SourceManager) BackgroundScheduler() {
	if len(s.sources) == 0 {
		return
	}

	go func() {
		for {
			//Because ad-hoc refreshes can happen asynchronously, this thread may wake
			//before its time to populate again because the refresh has pushed it back.
			//In that case, just reevaluate your sleep time and come back later
			s.lock.Lock()
			current := s.queue[0]
			shouldRun := !(time.Now().Before(current.nextRun))
			if shouldRun {
				current.Bump()
				s.sortQueue()
			}
			s.lock.Unlock()

			if shouldRun {
				current.Refresh(s.global, "scheduled", s.log)
			}

			s.lock.RLock()
			nextTime := s.queue[0].nextRun
			s.lock.RUnlock()
			time.Sleep(time.Until(nextTime))
		}
	}()
}

func (s *SourceManager) sortQueue() {
	//This is a naive sort when only one thing can change order. Also, we should
	//be using a linked-list heap for performance. But considering that the number
	//of backends will be in the single digits... I really don't care about either
	//of those things. I've instead decided to fill the lines of code that I've
	//saved with that decision by writing this comment.
	sort.Slice(s.queue,
		func(i, j int) bool {
			return s.queue[i].nextRun.Before(s.queue[j].nextRun)
		},
	)
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
			NotBefore:  v.NotBefore.Unix(),
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].NotAfter < items[j].NotAfter })
	return items
}

func (s *SourceManager) RefreshAll() {
	//How long must have passed before we'll refresh a backend by request to avoid spamming
	const tooRecentThreshold time.Duration = time.Minute
	for _, current := range s.sources {
		s.lock.Lock()
		lastRun := current.nextRun.Add(-current.Interval)
		cutoffTime := time.Now().Add(-(tooRecentThreshold))
		shouldRun := lastRun.Before(cutoffTime)
		if shouldRun {
			current.Bump()
			s.sortQueue()
		}
		s.lock.Unlock()

		if shouldRun {
			current.Refresh(s.global, "ad-hoc", s.log)
		}
	}
}
