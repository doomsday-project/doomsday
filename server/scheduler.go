package server

import (
	"bytes"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/doomsday-project/doomsday/server/logger"
)

type taskKind uint

const (
	//right now, the order of these actually matters, because it is used for
	// prioritization when sorting the queue
	queueTaskKindAuth taskKind = iota
	queueTaskKindRefresh
)

func (t taskKind) String() string {
	if t == queueTaskKindAuth {
		return "auth"
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

type managerTasks []managerTask

type taskQueue struct {
	data        managerTasks
	running     managerTasks
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
	task.id = t.nextTaskID
	t.nextTaskID++
	t.log.WriteF("Enqueuing new %s %s task for backend `%s' with id %d", task.reason, task.kind, task.source.Core.Name, task.id)
	t.data = append(t.data, task)
	t.data.sort()
	t.lock.Unlock()

	time.AfterFunc(time.Until(task.runTime), func() {
		t.lock.Lock()
		defer t.lock.Unlock()

		foundTask := t.data.findTaskWithID(task.id)
		if foundTask == nil {
			t.log.WriteF("Skipping marking task as ready because it has been removed pre-emptively (id %d)",
				task.id)
			return
		}

		t.log.WriteF("Marking %s %s task for backend `%s' as ready (id %d)",
			foundTask.reason, foundTask.kind, foundTask.source.Core.Name, task.id)
		foundTask.ready = true
		t.data.sort()

		t.cond.Signal()
	})
}

func (t managerTasks) idxWithID(id uint) int {
	var ret int = -1
	for i := range t {
		if t[i].id == id {
			ret = i
			break
		}
	}

	return ret
}

//sort priority:
// 1. readiness
// 2. auth before refresh if both are ready
// 3. scheduled time
// 4. id
func (t managerTasks) sort() {
	sort.Slice(t, func(i, j int) bool {
		if t[i].ready && !t[j].ready {
			return true
		}

		if t[j].ready && !t[i].ready {
			return false
		}

		if t[i].ready && t[j].ready {
			if t[i].kind != t[j].kind {
				return t[i].kind < t[j].kind
			}
		}

		if t[i].runTime.Equal(t[j].runTime) {
			return t[i].id < t[j].id
		}
		return t[i].runTime.Before(t[j].runTime)
	})
}

//If the queue order is shuffled in any way after a call to this function, the
// returned pointer is invalidated. Therefore, you should only call this and
// manipulate the returned object while you are holding the lock
func (t managerTasks) findTaskWithID(id uint) *managerTask {
	var ret *managerTask
	if idx := t.idxWithID(id); idx >= 0 {
		ret = &t[idx]
	}

	return ret
}

func (t managerTasks) idxWithSourceAndKind(sourceName string, taskType taskKind) int {
	var ret int = -1
	for i := range t {
		if t[i].source.Core.Name == sourceName && t[i].kind == taskType {
			ret = i
			break
		}
	}

	return ret
}

func (t *managerTasks) deleteTaskWithID(id uint) {
	if idx := t.idxWithID(id); idx >= 0 {
		(*t)[idx] = (*t)[len(*t)-1]
		*t = (*t)[:len(*t)-1]
		(*t).sort()
	}
}

func (t *taskQueue) dequeueNoLock() managerTask {
	ret := t.data[0]
	t.data[0] = t.data[len(t.data)-1]
	t.data = t.data[:len(t.data)-1]
	t.data.sort()

	return ret
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
			t.scheduleNextRunOf(next)
		}
	}()
}

func (t *taskQueue) run(task managerTask) {

	t.lock.Lock()

	currentIdx := t.running.idxWithSourceAndKind(task.source.Core.Name, task.kind)
	if currentIdx >= 0 {
		t.log.WriteF("Skipping %s %s task run of `%s' because same task already in progress", task.reason, task.kind, task.source.Core.Name)

		t.lock.Unlock()
		return
	}
	t.running = append(t.running, task)

	t.lock.Unlock()

	defer func() {
		t.lock.Lock()
		t.running.deleteTaskWithID(task.id)
		t.lock.Unlock()
	}()

	switch task.kind {
	case queueTaskKindAuth:
		task.source.Auth(t.log)

	case queueTaskKindRefresh:
		task.source.Refresh(t.globalCache, t.log)
	}
}

func (t *taskQueue) scheduleNextRunOf(task managerTask) {
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
		t.log.WriteF("Skipping further scheduling of `%s' for `%s'", task.kind.String(), task.source.Core.Name)
		return
	}

	t.enqueue(managerTask{
		source:  task.source,
		runTime: nextTime,
		reason:  runReasonSchedule,
		kind:    task.kind,
	})
}

type SchedulerState struct {
	Running []SchedulerTask `json:"running"`
	Pending []SchedulerTask `json:"pending"`
}

func (s SchedulerState) String() string {
	b, _ := json.Marshal(s)
	bOut := bytes.Buffer{}
	json.Indent(&bOut, b, "", "  ")
	return bOut.String()
}

type SchedulerTask struct {
	ID      uint      `json:"id"`
	At      time.Time `json:"at"`
	Backend string    `json:"backend"`
	Reason  string    `json:"reason"`
	Kind    string    `json:"kind"`
	Ready   bool      `json:"ready"`
}

func (t *taskQueue) dumpState() SchedulerState {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.dumpStateNoLock()
}

func (t *taskQueue) dumpStateNoLock() SchedulerState {
	ret := SchedulerState{
		Running: []SchedulerTask{},
		Pending: []SchedulerTask{},
	}

	for _, task := range t.running {
		ret.Running = append(ret.Running, SchedulerTask{
			ID:      task.id,
			At:      task.runTime,
			Backend: task.source.Core.Name,
			Reason:  task.reason.String(),
			Kind:    task.kind.String(),
			Ready:   task.ready,
		})
	}

	for _, task := range t.data {
		ret.Pending = append(ret.Pending, SchedulerTask{
			ID:      task.id,
			At:      task.runTime,
			Backend: task.source.Core.Name,
			Reason:  task.reason.String(),
			Kind:    task.kind.String(),
			Ready:   task.ready,
		})
	}

	return ret
}
