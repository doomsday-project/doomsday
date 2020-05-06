package server

import (
	"sort"
	"sync"
	"time"

	"github.com/doomsday-project/doomsday/server/logger"
)

type taskKind uint

const (
	queueTaskKindAuth taskKind = iota
	queueTaskKindRefresh
)

func (t taskKind) String() string {
	if t == queueTaskKindAuth {
		return "authentication"
	}

	return "refresh"
}

type runReason uint

const (
	runReasonSchedule runReason = iota
	runReasonAdhoc
)

func (r runReason) String() string {
	if r == runReasonSchedule {
		return "scheduled"
	}

	return "adhoc"
}

type managerTask struct {
	id      uint
	kind    taskKind
	source  *Source
	runTime time.Time
	reason  runReason
	ready   bool
}

func (m *managerTask) durationUntil() time.Duration {
	return m.runTime.Sub(time.Now())
}

type taskQueue struct {
	data        []managerTask
	lock        *sync.Mutex
	cond        *sync.Cond
	log         *logger.Logger
	globalCache *Cache
	nextTaskID  uint
}

func newTaskQueue(cache *Cache, log *logger.Logger) *taskQueue {
	lock := &sync.Mutex{}
	return &taskQueue{
		lock:        lock,
		log:         log,
		cond:        sync.NewCond(lock),
		globalCache: cache,
	}
}

//next blocks until there is a task for this thread to handle. it then dequeues
// and returns that task.
func (t *taskQueue) next() managerTask {
	t.lock.Lock()

	for t.empty() || !t.data[0].ready {
		t.cond.Wait()
	}

	ret := t.dequeueNoLock()
	t.lock.Unlock()
	return ret
}

//enqueue puts a task into the queue, unique by the tuple source, taskType. If
//there already exists a task for this source/taskType, it will be removed and
//replaced with this new one atomically.
func (t *taskQueue) enqueue(task managerTask) {
	t.lock.Lock()
	t.removeExistingNoLock(task.source, task.kind)
	task.id = t.nextTaskID
	t.nextTaskID++
	t.data = append(t.data, task)
	t.sort()

	t.lock.Unlock()
	time.AfterFunc(time.Until(task.runTime), func() {
		t.lock.Lock()
		defer t.lock.Unlock()

		foundTask := t.findTaskWithIDNoLock(task.id)
		if foundTask != nil {
			foundTask.ready = true
			t.cond.Signal()
		}
	})
}

//If the queue order is shuffled in any way after a call to this function, the
// returned pointer is invalidated. Therefore, you should only call this and
// manipulate the returned object while you are holding the lock
func (t *taskQueue) findTaskWithIDNoLock(id uint) *managerTask {
	var ret *managerTask
	for i := range t.data {
		if t.data[i].id == id {
			ret = &t.data[i]
			break
		}
	}

	return ret
}

func (t *taskQueue) dequeueNoLock() managerTask {
	ret := t.data[0]
	t.data[0] = t.data[len(t.data)-1]
	t.data = t.data[:len(t.data)-1]
	t.sort()

	return ret
}

func (t *taskQueue) sort() {
	sort.Slice(t.data, func(i, j int) bool {
		return t.data[i].runTime.Before(t.data[j].runTime)
	})
}

func (t *taskQueue) removeExistingNoLock(source *Source, taskType taskKind) {
	//A source is considered equal if it has the same pointer to a core
	for i := range t.data {
		if t.data[i].source.Core == source.Core &&
			t.data[i].kind == taskType {

			t.data[i] = t.data[len(t.data)-1]
			t.data = t.data[:len(t.data)-1]
			t.sort()
		}
	}
}

func (t *taskQueue) empty() bool {
	return len(t.data) == 0
}

func (t *taskQueue) start() {
	go func() {
		for {
			next := t.next()
			t.log.WriteF("Scheduler running %s %s of `%s'", next.reason, next.kind, next.source.Core.Name)
			t.run(next)
		}
	}()
}

func (t *taskQueue) run(task managerTask) {
	var nextTime time.Time
	var skipSched bool
	switch task.kind {
	case queueTaskKindAuth:
		task.source.Auth(t.log)
		nextTime, skipSched = task.source.CalcNextAuth()

	case queueTaskKindRefresh:
		task.source.Refresh(t.globalCache, t.log)
		nextTime = task.source.CalcNextRefresh()
	}

	if skipSched {
		t.log.WriteF("Skipping further scheduling of `%s' for `%s'", task.kind.String(), task.source.Core.Name)
		return
	}

	task.runTime = nextTime
	task.reason = runReasonSchedule

	t.enqueue(managerTask{
		source:  task.source,
		runTime: nextTime,
		reason:  runReasonSchedule,
		kind:    task.kind,
	})
}

type SchedulerState struct {
	Tasks []SchedulerTask
}

type SchedulerTask struct {
	At     time.Time
	Reason string
	Kind   string
	Ready  bool
}

func (t *taskQueue) dumpState() SchedulerState {
	ret := SchedulerState{
		Tasks: []SchedulerTask{},
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	for _, task := range t.data {
		ret.Tasks = append(ret.Tasks, SchedulerTask{
			At:     task.runTime,
			Reason: task.reason.String(),
			Kind:   task.kind.String(),
			Ready:  task.ready,
		})
	}

	return ret
}
