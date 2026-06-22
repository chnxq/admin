package service

import (
	"context"
	"strings"
	"testing"

	taskv1 "admin/api/gen/task/v1"
	taskpkg "admin/internal/task"
	taskruntime "admin/internal/task/runtime"
	"github.com/chnxq/xkitmod/log"
)

type bootstrapTestExecutor struct {
	target string
}

func (e bootstrapTestExecutor) InvokeTarget() string {
	return e.target
}

func (e bootstrapTestExecutor) Validate(context.Context, taskruntime.ValidationRequest) error {
	return nil
}

func (e bootstrapTestExecutor) Execute(_ context.Context, req taskruntime.ExecuteRequest) (string, error) {
	return req.Task.GetTaskName(), nil
}

type fakeSchedulerStore struct {
	listRunnableFunc func(context.Context) ([]*taskv1.Task, error)
}

func (s *fakeSchedulerStore) GetTask(context.Context, uint64) (*taskv1.Task, error) {
	return nil, nil
}

func (s *fakeSchedulerStore) ListRunnableTasks(ctx context.Context) ([]*taskv1.Task, error) {
	if s.listRunnableFunc != nil {
		return s.listRunnableFunc(ctx)
	}
	return nil, nil
}

func (s *fakeSchedulerStore) UpdateTaskRuntimeState(context.Context, uint64, taskv1.Task_Status, *uint32) error {
	return nil
}

func TestResolveTaskSchedulerRejectsInconsistentSchedulers(t *testing.T) {
	logger := log.NewHelper(log.NewStdLogger(taskTestingWriter{t: t}))
	_, err := taskpkg.ResolveScheduler(
		logger,
		taskruntime.NewScheduler(&fakeSchedulerStore{}, nil),
		taskruntime.NewScheduler(&fakeSchedulerStore{}, nil),
	)
	if err == nil || !strings.Contains(err.Error(), "inconsistent") {
		t.Fatalf("expected inconsistent scheduler error, got %v", err)
	}
}

func TestRegisterTaskSchedulerAllowsPartialRestoreFailures(t *testing.T) {
	status := taskv1.Task_RUNNING
	taskID := uint64(4001)
	taskName := "broken"
	target := "demo:task"
	runner := taskruntime.NewRunner(taskruntime.MustNewRegistry(bootstrapTestExecutor{target: "demo:task"}), nil)
	scheduler := taskruntime.NewScheduler(&fakeSchedulerStore{
		listRunnableFunc: func(context.Context) ([]*taskv1.Task, error) {
			return []*taskv1.Task{{
				Id:             &taskID,
				TaskName:       bootstrapStringPtr(taskName),
				Status:         &status,
				InvokeTarget:   &target,
				CronExpression: bootstrapStringPtr("bad cron"),
			}}, nil
		},
	}, runner)

	logger := log.NewHelper(log.NewStdLogger(taskTestingWriter{t: t}))
	cleanup, err := taskpkg.RegisterScheduler(context.Background(), logger, scheduler, scheduler)
	if err != nil {
		t.Fatalf("RegisterScheduler failed: %v", err)
	}
	if cleanup == nil {
		t.Fatalf("expected cleanup func")
	}
	cleanup()
}

func bootstrapStringPtr(value string) *string {
	return &value
}

type taskTestingWriter struct {
	t *testing.T
}

func (w taskTestingWriter) Write(p []byte) (int, error) {
	if w.t != nil {
		w.t.Log(string(p))
	}
	return len(p), nil
}
