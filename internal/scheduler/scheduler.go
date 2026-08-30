package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"shellsage/internal/agent"
)

// ScheduledJob represents a time-bound task
type ScheduledJob struct {
	ID        string    `json:"id"`
	Goal      string    `json:"goal"`
	TargetTime time.Time `json:"target_time"`
	Executed  bool      `json:"executed"`
	Result    string    `json:"result,omitempty"`
	Error     string    `json:"error,omitempty"`
}

// Scheduler handles time-based background task execution
type Scheduler struct {
	jobs    []*ScheduledJob
	mu      sync.Mutex
	agent   *agent.Agent
	stopCh  chan struct{}
	running bool
}

// NewScheduler creates a scheduler instance
func NewScheduler(ag *agent.Agent) *Scheduler {
	return &Scheduler{
		jobs:   make([]*ScheduledJob, 0),
		agent:  ag,
		stopCh: make(chan struct{}),
	}
}

// ScheduleAt sets a job to trigger at a specific local time today (or tomorrow if past)
func (s *Scheduler) ScheduleAt(hour, minute int, goal string) (*ScheduledJob, error) {
	now := time.Now()
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	if target.Before(now) {
		// Schedule for next day if the time today has already passed
		target = target.Add(24 * time.Hour)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	job := &ScheduledJob{
		ID:         fmt.Sprintf("job_%d", time.Now().UnixNano()%1000000),
		Goal:       goal,
		TargetTime: target,
		Executed:   false,
	}

	s.jobs = append(s.jobs, job)
	return job, nil
}

// ScheduleIn sets a job to trigger after a relative duration (e.g. 10m, 1h)
func (s *Scheduler) ScheduleIn(duration time.Duration, goal string) (*ScheduledJob, error) {
	target := time.Now().Add(duration)

	s.mu.Lock()
	defer s.mu.Unlock()

	job := &ScheduledJob{
		ID:         fmt.Sprintf("job_%d", time.Now().UnixNano()%1000000),
		Goal:       goal,
		TargetTime: target,
		Executed:   false,
	}

	s.jobs = append(s.jobs, job)
	return job, nil
}

// ListJobs returns scheduled jobs
func (s *Scheduler) ListJobs() []*ScheduledJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := make([]*ScheduledJob, len(s.jobs))
	copy(res, s.jobs)
	return res
}

// Start runs the background ticker checking for due jobs
func (s *Scheduler) Start(ctx context.Context, notify func(job *ScheduledJob)) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.checkAndExecute(ctx, notify)
			}
		}
	}()
}

func (s *Scheduler) checkAndExecute(ctx context.Context, notify func(job *ScheduledJob)) {
	s.mu.Lock()
	now := time.Now()
	var dueJobs []*ScheduledJob

	for _, j := range s.jobs {
		if !j.Executed && now.After(j.TargetTime) {
			j.Executed = true
			dueJobs = append(dueJobs, j)
		}
	}
	s.mu.Unlock()

	for _, j := range dueJobs {
		res, err := s.agent.Run(ctx, j.Goal, nil)
		s.mu.Lock()
		if err != nil {
			j.Error = err.Error()
		} else {
			j.Result = res
		}
		s.mu.Unlock()

		if notify != nil {
			notify(j)
		}
	}
}

// Stop terminates background scheduling
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		close(s.stopCh)
		s.running = false
	}
}
