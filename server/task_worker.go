package server

import (
	"sync"
	"time"

	"github.com/doomsday-project/doomsday/server/logger"
)

type taskWorkerFactory struct {
	sched *taskQueue
	cache *Cache
	log   *logger.Logger
	curID uint
}

func newTaskWorkerFactory(sched *taskQueue, cache *Cache, log *logger.Logger) *taskWorkerFactory {
	return &taskWorkerFactory{
		sched: sched,
		cache: cache,
		log:   log,
	}
}

func (f *taskWorkerFactory) newWorker() *taskWorker {
	ret := &taskWorker{
		sched:   f.sched,
		cache:   f.cache,
		log:     f.log,
		id:      f.curID,
		state:   WorkerStateIdle,
		stateAt: time.Now(),
	}

	f.curID++
	return ret
}

type taskWorker struct {
	sched *taskQueue
	cache *Cache
	log   *logger.Logger
	id    uint
	//Never try to grab the scheduler lock while holding this lock.
	// You can try to grab this lock while holding the scheduler lock, but
	// not in the reverse order.
	stateLock sync.RWMutex
	state     WorkerState
	stateAt   time.Time
}

//WorkerState is the current thing the worker is doing. i.e. idle vs running
type WorkerState uint

const (
	//WorkerStateIdle means the worker is currently waiting for a task
	WorkerStateIdle = iota
	//WorkerStateRunning means the worker is currently running a task
	WorkerStateRunning
	//WorkerStateScheduling means the worker is currently scheduling a future task
	WorkerStateScheduling
)

func (w WorkerState) String() string {
	ret := "unknown"
	switch w {
	case WorkerStateIdle:
		ret = "idle"
	case WorkerStateRunning:
		ret = "running"
	case WorkerStateScheduling:
		ret = "scheduling"
	}

	return ret
}

type taskWorkerState struct {
	state *WorkerState
}

func (w *taskWorker) consumeScheduler() {
	go func() {
		for {
			next := w.runNext()
			w.scheduleNextRunOf(next)
		}
	}()
}

//runNext blocks until there is a task for this worker to handle. it then
//dequeues and runs that task if it is not marked to skip.
func (w *taskWorker) runNext() managerTask {
	w.sched.lock.Lock()

	for w.sched.empty() || w.sched.data[0].state == queueTaskStatePending {
		w.sched.cond.Wait()
	}

	ret := w.sched.dequeueNoLock()

	if ret.state == queueTaskStateSkip {
		w.sched.lock.Unlock()
		w.log.WriteF("Worker %d skipping %s %s of `%s'", w.id, ret.reason, ret.kind, ret.source.Core.Name)
		return ret
	}

	ret.assignedWorker = w
	w.sched.running = append(w.sched.running, ret)
	w.SetState(WorkerStateRunning)
	w.sched.lock.Unlock()

	w.log.WriteF("Worker %d running %s %s of `%s'", w.id, ret.reason, ret.kind, ret.source.Core.Name)

	ret.run(w.cache, w.log)

	w.SetState(WorkerStateScheduling)
	w.sched.lock.Lock()
	w.sched.running.deleteTaskWithID(ret.id)
	w.sched.lock.Unlock()
	w.SetState(WorkerStateIdle)

	return ret
}

func (w *taskWorker) scheduleNextRunOf(task managerTask) {
	if task.reason == runReasonAdhoc {
		return
	}

	var nextTime time.Time
	var skipSched bool

	switch task.kind {
	case queueTaskKindAuth:
		nextTime, skipSched = task.source.CalcNextAuth()

	case queueTaskKindRefresh:
		nextTime = task.source.CalcNextRefresh()
	}

	if skipSched {
		w.log.WriteF("Skipping further scheduling of `%s' for `%s'", task.kind.String(), task.source.Core.Name)
		return
	}

	w.sched.enqueue(managerTask{
		source:  task.source,
		runTime: nextTime,
		reason:  runReasonSchedule,
		kind:    task.kind,
	})
}

//State returns the current state this worker is in, and the time that it
// entered that state at
func (w *taskWorker) State() (WorkerState, time.Time) {
	w.stateLock.RLock()
	defer w.stateLock.RUnlock()
	return w.state, w.stateAt
}

func (w *taskWorker) SetState(state WorkerState) {
	w.stateLock.Lock()
	defer w.stateLock.Unlock()
	w.state = state
	w.stateAt = time.Now()
}
