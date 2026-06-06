package taskruntime

import (
	"context"
	"errors"
	"strings"
	"testing"

	taskv1 "admin/api/gen/task/v1"
)

type fakeRuntimeStore struct {
	getTaskFunc       func(context.Context, uint64) (*taskv1.Task, error)
	listRunnableFunc  func(context.Context) ([]*taskv1.Task, error)
	updateRuntimeFunc func(context.Context, uint64, taskv1.Task_Status, *uint32) error
	updated           []runtimeStateUpdate
}

type runtimeStateUpdate struct {
	taskID  uint64
	status  taskv1.Task_Status
	entryID uint32
}

func (s *fakeRuntimeStore) GetTask(ctx context.Context, taskID uint64) (*taskv1.Task, error) {
	if s.getTaskFunc != nil {
		return s.getTaskFunc(ctx, taskID)
	}
	return nil, nil
}

func (s *fakeRuntimeStore) ListRunnableTasks(ctx context.Context) ([]*taskv1.Task, error) {
	if s.listRunnableFunc != nil {
		return s.listRunnableFunc(ctx)
	}
	return nil, nil
}

func (s *fakeRuntimeStore) UpdateTaskRuntimeState(ctx context.Context, taskID uint64, status taskv1.Task_Status, entryID *uint32) error {
	if s.updateRuntimeFunc != nil {
		return s.updateRuntimeFunc(ctx, taskID, status, entryID)
	}
	value := uint32(0)
	if entryID != nil {
		value = *entryID
	}
	s.updated = append(s.updated, runtimeStateUpdate{taskID: taskID, status: status, entryID: value})
	return nil
}

func TestScheduler_SyncTaskStoppedTaskClearsEntry(t *testing.T) {
	store := &fakeRuntimeStore{}
	runner := NewRunner(MustNewRegistry(testExecutor{target: "demo:task"}), nil)
	scheduler := NewScheduler(store, runner)

	taskID := uint64(2001)
	status := taskv1.Task_STOPPED
	err := scheduler.SyncTask(context.Background(), &taskv1.Task{
		Id:     &taskID,
		Status: &status,
	})
	if err != nil {
		t.Fatalf("SyncTask failed: %v", err)
	}
	if len(store.updated) != 1 {
		t.Fatalf("expected 1 runtime update, got %d", len(store.updated))
	}
	if store.updated[0].entryID != 0 {
		t.Fatalf("expected cleared entry id, got %d", store.updated[0].entryID)
	}
}

func TestScheduler_SyncTaskRejectsInvalidCron(t *testing.T) {
	store := &fakeRuntimeStore{}
	runner := NewRunner(MustNewRegistry(testExecutor{target: "demo:task"}), nil)
	scheduler := NewScheduler(store, runner)

	taskID := uint64(2002)
	status := taskv1.Task_RUNNING
	target := "demo:task"
	err := scheduler.SyncTask(context.Background(), &taskv1.Task{
		Id:             &taskID,
		Status:         &status,
		InvokeTarget:   &target,
		CronExpression: schedulerStringPtr("invalid cron"),
	})
	if err == nil || !strings.Contains(err.Error(), "invalid cron expression") {
		t.Fatalf("expected invalid cron error, got %v", err)
	}
}

func TestScheduler_RestoreTasksAggregatesFailures(t *testing.T) {
	status := taskv1.Task_RUNNING
	target := "demo:task"
	taskID1 := uint64(2003)
	taskID2 := uint64(2004)
	store := &fakeRuntimeStore{
		listRunnableFunc: func(context.Context) ([]*taskv1.Task, error) {
			return []*taskv1.Task{
				{Id: &taskID1, TaskName: schedulerStringPtr("task-1"), Status: &status, InvokeTarget: &target, CronExpression: schedulerStringPtr("0 * * * * *")},
				{Id: &taskID2, TaskName: schedulerStringPtr("task-2"), Status: &status, InvokeTarget: &target, CronExpression: schedulerStringPtr("bad cron")},
			}, nil
		},
	}
	runner := NewRunner(MustNewRegistry(testExecutor{target: "demo:task"}), nil)
	scheduler := NewScheduler(store, runner)

	err := scheduler.RestoreTasks(context.Background())
	restoreErr, ok := err.(*TaskRestoreError)
	if !ok {
		t.Fatalf("expected TaskRestoreError, got %T %v", err, err)
	}
	if len(restoreErr.Failed) != 1 {
		t.Fatalf("expected 1 failed task, got %d", len(restoreErr.Failed))
	}
	if restoreErr.Failed[0].TaskID != taskID2 {
		t.Fatalf("expected failed task %d, got %d", taskID2, restoreErr.Failed[0].TaskID)
	}
}

func TestScheduler_RunTaskNowRequiresRunner(t *testing.T) {
	scheduler := NewScheduler(&fakeRuntimeStore{}, nil)
	taskID := uint64(2005)
	err := scheduler.RunTaskNow(context.Background(), &taskv1.Task{Id: &taskID}, "")
	if err == nil || !strings.Contains(err.Error(), "task runner is not configured") {
		t.Fatalf("expected runner missing error, got %v", err)
	}
}

func TestTaskRestoreError_Error(t *testing.T) {
	err := (&TaskRestoreError{Failed: []*TaskSyncError{{TaskID: 1, Cause: errors.New("boom")}}}).Error()
	if !strings.Contains(err, "restore 1 runnable tasks failed") {
		t.Fatalf("unexpected restore error text: %q", err)
	}
}

func schedulerStringPtr(value string) *string {
	return &value
}
