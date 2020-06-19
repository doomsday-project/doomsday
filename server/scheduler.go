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
	//the order of these actually matters, because it is used for
	// prioritization when sorting the queue
	queueTaskKindAuth taskKind = iota
	queueTaskKindRefresh
)

func (t taskKind) String() string {
	switch t {
	case queueTaskKindAuth:
		return "auth"
	case queueTaskKindRefresh:
		return "refresh"
	default:
		return "unknown"
	}
}

type runReason uint

const (
	runReasonSchedule runReason = iota
	runReasonAdhoc
)

func (r runReason) String() string {
	switch r {
	case runReasonSchedule:
		return "scheduled"
	case runReasonAdhoc:
		return "adhoc"
	default:
		return "unknown"
	}
}

type taskState uint

const (
	//the order of these actually matters, because it is used for
	// prioritization when sorting the queue
	queueTaskStatePending = iota
	queueTaskStateReady
	queueTaskStateSkip
)

func (s taskState) String() string {
	switch s {
	case queueTaskStatePending:
		return "pending"
	case queueTaskStateReady:
		return "ready"
	case queueTaskStateSkip:
		return "skipping"
	default:
		return "unknown"
	}
}

type managerTask struct {
	id             uint
	kind           taskKind
	source         *Source
	runTime        time.Time
	reason         runReason
	state          taskState
	assignedWorker *taskWorker
}

func (m *managerTask) durationUntil() time.Duration {
	return m.runTime.Sub(time.Now())
}

func (m *managerTask) run(cache *Cache, log *logger.Logger) {
	switch m.kind {
	case queueTaskKindAuth:
		m.source.Auth(log)

	case queueTaskKindRefresh:
		m.source.Refresh(cache, log)
	}
}

type managerTasks []managerTask

type taskQueue struct {
	data        managerTasks
	running     managerTasks
	lock        *sync.Mutex
	cond        *sync.Cond
	log         *logger.Logger
	globalCache *Cache
	numWorkers  uint
	workers     []*taskWorker
	nextTaskID  uint
}

func newTaskQueue(cache *Cache, numWorkers uint, log *logger.Logger) *taskQueue {
	lock := &sync.Mutex{}
	return &taskQueue{
		lock:        lock,
		log:         log,
		cond:        sync.NewCond(lock),
		globalCache: cache,
		numWorkers:  numWorkers,
	}
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

		if t.data.sameTaskExistsAndReady(foundTask) || t.running.sameTaskExistsAndReady(foundTask) {
			t.log.WriteF("Marking %s %s task for backend `%s' as to skip (id %d)",
				foundTask.reason, foundTask.kind, foundTask.source.Core.Name, task.id)
			foundTask.state = queueTaskStateSkip
		} else {
			t.log.WriteF("Marking %s %s task for backend `%s' as ready (id %d)",
				foundTask.reason, foundTask.kind, foundTask.source.Core.Name, task.id)
			foundTask.state = queueTaskStateReady
		}

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
// 1. skip<ready<pending
// 2. auth before refresh if both are ready
// 3. scheduled time
// 4. id
func (t managerTasks) sort() {
	sort.Slice(t, func(i, j int) bool {
		if t[i].state != t[j].state {
			return t[i].state > t[j].state
		}

		if t[i].kind != t[j].kind {
			return t[i].kind < t[j].kind
		}

		if !t[i].runTime.Equal(t[j].runTime) {
			return t[i].runTime.Before(t[j].runTime)
		}

		return t[i].id < t[j].id
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

//no lock
//considered the same task if the associated source core has the same name, and
// the kind of task is the same. Considered ready if the ready member of the task
// is true.
func (t managerTasks) sameTaskExistsAndReady(task *managerTask) bool {
	for i := range t {
		if t[i].source.Core.Name == task.source.Core.Name &&
			t[i].kind == task.kind &&
			t[i].state == queueTaskStateReady {
			return true
		}
	}

	return false
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
	workerFactory := newTaskWorkerFactory(t, t.globalCache, t.log)
	for i := uint(0); i < t.numWorkers; i++ {
		t.workers = append(t.workers, workerFactory.newWorker())
		t.workers[i].consumeScheduler()
	}
}

type SchedulerState struct {
	Running []SchedulerTask `json:"running"`
	Pending []SchedulerTask `json:"pending"`
	Workers []WorkerDump    `json:"workers"`
}

func (s SchedulerState) String() string {
	b, _ := json.Marshal(s)
	bOut := bytes.Buffer{}
	json.Indent(&bOut, b, "", "  ")
	return bOut.String()
}

type SchedulerTask struct {
	ID       uint      `json:"id"`
	At       time.Time `json:"at"`
	Backend  string    `json:"backend"`
	Reason   string    `json:"reason"`
	Kind     string    `json:"kind"`
	State    string    `json:"state"`
	WorkerID int       `json:"worker"`
}

type WorkerDump struct {
	ID      uint      `json:"id"`
	State   string    `json:"state"`
	StateAt time.Time `json:"state_at"`
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
			ID:       task.id,
			At:       task.runTime,
			Backend:  task.source.Core.Name,
			Reason:   task.reason.String(),
			Kind:     task.kind.String(),
			State:    task.state.String(),
			WorkerID: int(task.assignedWorker.id),
		})
	}

	for _, task := range t.data {
		ret.Pending = append(ret.Pending, SchedulerTask{
			ID:       task.id,
			At:       task.runTime,
			Backend:  task.source.Core.Name,
			Reason:   task.reason.String(),
			Kind:     task.kind.String(),
			State:    task.state.String(),
			WorkerID: -1,
		})
	}

	for _, worker := range t.workers {
		state, stateAt := worker.State()
		ret.Workers = append(ret.Workers, WorkerDump{
			ID:      worker.id,
			State:   state.String(),
			StateAt: stateAt,
		})
	}

	return ret
}
