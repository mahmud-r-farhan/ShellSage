package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"shellsage/internal/agent"
	"shellsage/internal/config"
)

// ScheduledJob represents a time-bound task
type ScheduledJob struct {
	ID         string    `json:"id"`
	Goal       string    `json:"goal"`
	TargetTime time.Time `json:"target_time"`
	Executed   bool      `json:"executed"`
	Result     string    `json:"result,omitempty"`
	Error      string    `json:"error,omitempty"`
	Recurring  bool      `json:"recurring,omitempty"` // /daily jobs re-arm for the next day
}

// Scheduler handles time-based background task execution using a single
// next-due timer (no busy-polling — a laptop left idle won't burn CPU).
type Scheduler struct {
	jobs    []*ScheduledJob
	mu      sync.Mutex
	agent   *agent.Agent
	stopCh  chan struct{}
	running bool
	timer   *time.Timer
	notify  func(job *ScheduledJob)
}

func scheduleFilePath() string {
	return filepath.Join(config.GetUserConfigDir(), "schedule.json")
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
		target = target.Add(24 * time.Hour)
	}
	return s.add(target, goal, false)
}

// ScheduleIn sets a job to trigger after a relative duration (e.g. 10m, 1h)
func (s *Scheduler) ScheduleIn(duration time.Duration, goal string) (*ScheduledJob, error) {
	if duration <= 0 {
		return nil, fmt.Errorf("duration must be positive")
	}
	return s.add(time.Now().Add(duration), goal, false)
}

// ScheduleDaily sets a recurring job at HH:MM every day.
func (s *Scheduler) ScheduleDaily(hour, minute int, goal string) (*ScheduledJob, error) {
	now := time.Now()
	target := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if target.Before(now) {
		target = target.Add(24 * time.Hour)
	}
	return s.add(target, goal, true)
}

func (s *Scheduler) add(target time.Time, goal string, recurring bool) (*ScheduledJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job := &ScheduledJob{
		ID:         fmt.Sprintf("job_%d", time.Now().UnixNano()%1000000),
		Goal:       goal,
		TargetTime: target,
		Recurring:  recurring,
	}
	s.jobs = append(s.jobs, job)
	s.persistLocked()
	s.rearmLocked()
	return job, nil
}

// Cancel removes a pending job by id (already-executed jobs are not removable).
func (s *Scheduler) Cancel(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, j := range s.jobs {
		if j.ID == id && !j.Executed {
			s.jobs = append(s.jobs[:i], s.jobs[i+1:]...)
			s.persistLocked()
			s.rearmLocked()
			return true
		}
	}
	return false
}

// SetAgent swaps the underlying agent (after provider/model changes) while
// preserving pending jobs and the live timer.
func (s *Scheduler) SetAgent(ag *agent.Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agent = ag
}

// ListJobs returns scheduled jobs
func (s *Scheduler) ListJobs() []*ScheduledJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := make([]*ScheduledJob, len(s.jobs))
	copy(res, s.jobs)
	return res
}

// Start runs the background scheduler and restores persisted pending jobs.
func (s *Scheduler) Start(ctx context.Context, notify func(job *ScheduledJob)) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.notify = notify
	s.restoreLocked()
	s.rearmLocked()
	stop := s.stopCh // captured under lock; Stop() closes + replaces it
	s.mu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
		case <-stop:
		}
		s.mu.Lock()
		if s.timer != nil {
			s.timer.Stop()
		}
		s.mu.Unlock()
	}()
}

// restoreLocked loads persisted jobs; jobs whose time passed while offline are
// marked as missed instead of silently firing on next launch.
func (s *Scheduler) restoreLocked() {
	path := scheduleFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var saved []*ScheduledJob
	if err := json.Unmarshal(data, &saved); err != nil {
		return
	}
	now := time.Now()
	existing := map[string]bool{}
	for _, j := range s.jobs {
		existing[j.ID] = true
	}
	for _, j := range saved {
		if j == nil || existing[j.ID] {
			continue
		}
		if !j.Executed {
			if j.TargetTime.Before(now) {
				if j.Recurring {
					// re-arm to next occurrence
					for j.TargetTime.Before(now) {
						j.TargetTime = j.TargetTime.Add(24 * time.Hour)
					}
				} else {
					j.Executed = true
					j.Error = "missed: ShellSage was offline at the target time"
				}
			}
			s.jobs = append(s.jobs, j)
		}
	}
}

// rearmLocked schedules a single timer for the soonest pending job.
func (s *Scheduler) rearmLocked() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if !s.running {
		return
	}

	now := time.Now()
	var next *ScheduledJob
	for _, j := range s.jobs {
		if j.Executed {
			continue
		}
		if next == nil || j.TargetTime.Before(next.TargetTime) {
			next = j
		}
	}
	if next == nil {
		return
	}

	delay := next.TargetTime.Sub(now)
	if delay < 0 {
		delay = 0
	}
	s.timer = time.AfterFunc(delay, func() { s.fireDue() })
}

func (s *Scheduler) fireDue() {
	ctx := context.Background()

	s.mu.Lock()
	now := time.Now()
	var due []*ScheduledJob
	for _, j := range s.jobs {
		if !j.Executed && !now.Before(j.TargetTime) {
			due = append(due, j)
		}
	}
	notify := s.notify
	s.mu.Unlock()

	for _, j := range due {
		res, err := s.agent.Run(ctx, j.Goal, nil)
		s.mu.Lock()
		if err != nil {
			j.Error = err.Error()
		} else {
			j.Result = res
		}
		if j.Recurring {
			// re-arm for tomorrow instead of marking done
			j.Executed = false
			j.TargetTime = j.TargetTime.Add(24 * time.Hour)
			for j.TargetTime.Before(time.Now()) {
				j.TargetTime = j.TargetTime.Add(24 * time.Hour)
			}
		} else {
			j.Executed = true
		}
		s.persistLocked()
		s.mu.Unlock()

		if notify != nil {
			notify(j)
		}
	}

	s.mu.Lock()
	s.rearmLocked()
	s.mu.Unlock()
}

func (s *Scheduler) persistLocked() {
	type persisted struct {
		Jobs []*ScheduledJob `json:"jobs"`
	}
	var pending []*ScheduledJob
	for _, j := range s.jobs {
		if !j.Executed || j.Recurring {
			pending = append(pending, j)
		}
	}
	data, err := json.MarshalIndent(persisted{Jobs: pending}, "", "  ")
	if err != nil {
		return
	}
	path := scheduleFilePath()
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	_ = os.WriteFile(path, data, 0600)
}

// Stop terminates background scheduling
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		s.running = false
		if s.timer != nil {
			s.timer.Stop()
			s.timer = nil
		}
		close(s.stopCh) // wakes the watcher goroutine (it holds the old value)
		s.stopCh = make(chan struct{})
	}
}
