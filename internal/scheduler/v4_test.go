package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDailyRecurringRearms(t *testing.T) {
	// point config dir at temp so persistence does not touch the real home
	tmp := t.TempDir()
	t.Setenv("SHELLSAGE_HOME", tmp)

	ag := agentForMock(t)
	s := NewScheduler(ag)

	// target 20 minutes from now (recurring loop advances by day, so arm a past minute to test re-arm quickly)
	now := time.Now()
	job, err := s.ScheduleDaily(now.Hour(), (now.Minute()+1)%60, "standup notes")
	if err != nil {
		t.Fatal(err)
	}
	if !job.Recurring {
		t.Error("daily flag missing")
	}

	// simulate fireDue immediately by mutating the target time
	s.mu.Lock()
	job.Executed = false
	job.TargetTime = time.Now().Add(-time.Second)
	s.mu.Unlock()

	fired := make(chan *ScheduledJob, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx, func(j *ScheduledJob) { fired <- j })
	defer s.Stop()

	select {
	case j := <-fired:
		if !j.Recurring {
			t.Error("fired job should still be recurring")
		}
		s.mu.Lock()
		after := j.TargetTime
		s.mu.Unlock()
		if after.Before(time.Now()) {
			t.Errorf("recurring job must re-arm to a future time, got %v", after)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("daily job did not fire")
	}
}

func TestCancelPendingJob(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SHELLSAGE_HOME", tmp)
	s := NewScheduler(agentForMock(t))
	job, err := s.ScheduleIn(time.Hour, "never runs")
	if err != nil {
		t.Fatal(err)
	}
	if !s.Cancel(job.ID) {
		t.Error("cancel should succeed")
	}
	if len(s.ListJobs()) != 0 {
		t.Error("job should be removed")
	}
	if s.Cancel(job.ID) {
		t.Error("double cancel should fail")
	}
}

func TestSchedulePersistenceFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SHELLSAGE_HOME", tmp)
	s := NewScheduler(agentForMock(t))
	if _, err := s.ScheduleIn(time.Hour, "persist me"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(tmp, "schedule.json"))
	if err != nil {
		t.Fatalf("schedule not persisted: %v", err)
	}
	if !strings.Contains(string(data), "persist me") {
		t.Error("persisted schedule missing goal")
	}
}
