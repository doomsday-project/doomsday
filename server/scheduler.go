package server

import (
	"sort"
	"sync"
	"time"
)

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
	ready   bool
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
	t.data = append(t.data, task)
	t.sort()
	t.lock.Unlock()
	time.AfterFunc(time.Until(task.runTime), func() {
		t.lock.Lock()
		task.ready = true
		t.cond.Signal()
		t.lock.Unlock()
	})
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

func (t *taskQueue) removeExistingNoLock(source *Source, taskType uint) {
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
