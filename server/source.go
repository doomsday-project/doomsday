package server

import (
	"sync"
	"time"

	"github.com/doomsday-project/doomsday/server/logger"
	"github.com/doomsday-project/doomsday/storage"
)

type Source struct {
	Core     *Core
	Interval time.Duration
	lock     sync.RWMutex
	authTTL  time.Duration

	refreshStatus RunInfo
	authStatus    RunInfo
	authMetadata  interface{}
}

type RunInfo struct {
	LastRun     RunTiming
	LastSuccess RunTiming
	LastErr     error
}

type RunTiming struct {
	StartedAt  time.Time
	FinishedAt time.Time
}

//TODO: clean up the mode argument
func (s *Source) Refresh(global *Cache, mode string, log *logger.Logger) {
	log.WriteF("Running %s populate of `%s'", mode, s.Core.Name)
	old := s.Core.Cache()
	if old == nil {
		old = NewCache()
	}

	s.lock.Lock()
	s.refreshStatus.LastRun = RunTiming{StartedAt: time.Now()}
	s.lock.Unlock()

	results, err := s.Core.Populate()

	s.lock.Lock()
	s.refreshStatus.LastRun.FinishedAt = time.Now()
	defer s.lock.Unlock()

	if err != nil {
		log.WriteF("Error populating info from backend `%s': %s", s.Core.Name, err)
		s.lock.Lock()
		s.refreshStatus.LastErr = err
		return
	}

	s.refreshStatus.LastErr = nil
	s.refreshStatus.LastSuccess = s.refreshStatus.LastRun

	global.ApplyDiff(old, s.Core.Cache())

	log.WriteF("Finished %s populate of `%s' after %s. %d/%d paths searched. %d certs found",
		mode, s.Core.Name, time.Since(s.refreshStatus.LastRun.StartedAt), results.NumSuccess, results.NumPaths, results.NumCerts)
}

func (s *Source) Auth(log *logger.Logger) {
	log.WriteF("Starting authentication for `%s'", s.Core.Name)

	s.lock.Lock()
	s.authStatus.LastRun = RunTiming{StartedAt: time.Now()}
	s.lock.Unlock()

	ttl, metadata, err := s.Core.Backend.Authenticate(s.authMetadata)

	s.lock.Lock()
	defer s.lock.Unlock()

	s.authStatus.LastRun.FinishedAt = time.Now()

	if err != nil {
		s.authStatus.LastErr = err
		log.WriteF("Failed auth for `%s' after %s: %s", s.Core.Name, time.Since(s.authStatus.LastRun.StartedAt), err)
	}

	s.authStatus.LastErr = nil
	s.authStatus.LastSuccess = s.authStatus.LastRun

	s.authTTL = ttl
	s.authMetadata = metadata

	log.WriteF("Finished auth for `%s' after %s", s.Core.Name, time.Since(s.authStatus.LastRun.StartedAt))
}

//CalcNextAuth returns the time of the next authentication to attempt.
// The second return value is true if this should never be scheduled again
func (s *Source) CalcNextAuth() (time.Time, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.authTTL == storage.TTLInfinite {
		return time.Time{}, true
	}

	expiryTime := s.authStatus.LastSuccess.StartedAt.Add(s.authTTL)
	authInterval := expiryTime.Sub(s.authStatus.LastRun.FinishedAt) / 2
	if authInterval < MinAuthInterval {
		authInterval = MinAuthInterval
	}

	nextAuth := s.authStatus.LastRun.FinishedAt.Add(authInterval)
	if nextAuth.After(expiryTime) {
		nextAuth = s.authStatus.LastRun.FinishedAt.Add(ExpiredRetryInterval)
	}

	return nextAuth, false
}

func (s *Source) CalcNextRefresh() time.Time {
	s.lock.RLock()
	ret := s.refreshStatus.LastRun.FinishedAt.Add(s.Interval)
	s.lock.RUnlock()
	return ret
}
