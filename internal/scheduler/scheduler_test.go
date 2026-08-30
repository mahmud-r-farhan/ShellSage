package scheduler

import (
	"context"
	"testing"
	"time"

	"shellsage/internal/agent"
	"shellsage/internal/provider"
	"shellsage/internal/tools"
)

type mockTestProvider struct{}

func (m *mockTestProvider) Name() string { return "mock" }
func (m *mockTestProvider) ListAvailableModels() []string { return []string{"m"} }
func (m *mockTestProvider) Chat(ctx context.Context, req *provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{
		Content: "Task executed successfully",
		Usage:   provider.TokenUsage{TotalTokens: 5},
	}, nil
}
func (m *mockTestProvider) Stream(ctx context.Context, req *provider.ChatRequest, cb provider.StreamCallback) (*provider.ChatResponse, error) {
	return m.Chat(ctx, req)
}

func TestQueueAndScheduler(t *testing.T) {
	ag := agent.NewAgent(&mockTestProvider{}, tools.NewToolRegistry(), "m", 0.1)

	// Test Queue
	q := NewTaskQueue(ag)
	t1 := q.Add("Do something")
	if t1.Status != StatusPending {
		t.Errorf("expected task status PENDING")
	}

	completed, err := q.ExecuteNext(context.Background(), nil)
	if err != nil {
		t.Fatalf("failed to execute queued task: %v", err)
	}
	if completed.Status != StatusCompleted {
		t.Errorf("expected COMPLETED status, got %s", completed.Status)
	}

	// Test Scheduler In Duration
	sched := NewScheduler(ag)
	job, err := sched.ScheduleIn(50*time.Millisecond, "Quick check")
	if err != nil {
		t.Fatalf("failed to schedule job: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	notified := make(chan struct{})
	sched.Start(ctx, func(j *ScheduledJob) {
		close(notified)
	})
	defer sched.Stop()

	select {
	case <-notified:
		if !job.Executed {
			t.Errorf("job should be marked executed")
		}
	case <-time.After(5 * time.Second):
		t.Errorf("scheduler timed out without executing job")
	}
}
