package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"shellsage/internal/agent"
)

// TaskStatus defines task lifecycle state
type TaskStatus string

const (
	StatusPending   TaskStatus = "PENDING"
	StatusRunning   TaskStatus = "RUNNING"
	StatusCompleted TaskStatus = "COMPLETED"
	StatusFailed    TaskStatus = "FAILED"
)

// Task represents a unit of agentic work in the queue
type Task struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	Result      string     `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

// TaskQueue manages ordered execution of tasks
type TaskQueue struct {
	tasks []*Task
	mu    sync.Mutex
	agent *agent.Agent
}

// NewTaskQueue initializes a task queue
func NewTaskQueue(ag *agent.Agent) *TaskQueue {
	return &TaskQueue{
		tasks: make([]*Task, 0),
		agent: ag,
	}
}

// Add appends a new task to the queue
func (q *TaskQueue) Add(description string) *Task {
	q.mu.Lock()
	defer q.mu.Unlock()

	task := &Task{
		ID:          fmt.Sprintf("task_%d", time.Now().UnixNano()%1000000),
		Description: description,
		Status:      StatusPending,
		CreatedAt:   time.Now(),
	}
	q.tasks = append(q.tasks, task)
	return task
}

// List returns a snapshot of all tasks
func (q *TaskQueue) List() []*Task {
	q.mu.Lock()
	defer q.mu.Unlock()

	result := make([]*Task, len(q.tasks))
	copy(result, q.tasks)
	return result
}

// ExecuteNext processes the first pending task in the queue
func (q *TaskQueue) ExecuteNext(ctx context.Context, listener agent.AgentListener) (*Task, error) {
	q.mu.Lock()
	var next *Task
	for _, t := range q.tasks {
		if t.Status == StatusPending {
			next = t
			break
		}
	}
	if next == nil {
		q.mu.Unlock()
		return nil, fmt.Errorf("no pending tasks in queue")
	}

	next.Status = StatusRunning
	now := time.Now()
	next.StartedAt = &now
	q.mu.Unlock()

	// Execute with agent
	ans, err := q.agent.Run(ctx, next.Description, listener)

	q.mu.Lock()
	defer q.mu.Unlock()
	finish := time.Now()
	next.FinishedAt = &finish

	if err != nil {
		next.Status = StatusFailed
		next.Error = err.Error()
		return next, err
	}

	next.Status = StatusCompleted
	next.Result = ans
	return next, nil
}
