package taskruntime

import (
	"context"
	"fmt"
	"sync"
	"time"

	taskv1 "admin/api/gen/task/v1"

	"github.com/robfig/cron/v3"
)

const defaultTaskTimeout = 10 * time.Minute

type TaskRuntimeStore interface {
	GetTask(ctx context.Context, taskID uint64) (*taskv1.Task, error)
	ListRunnableTasks(ctx context.Context) ([]*taskv1.Task, error)
	UpdateTaskRuntimeState(ctx context.Context, taskID uint64, status taskv1.Task_Status, entryID *uint32) error
}

type Scheduler struct {
	store       TaskRuntimeStore
	runner      *Runner
	parser      cron.Parser
	cronRunner  *cron.Cron
	taskTimeout time.Duration

	mu      sync.RWMutex
	entries map[uint64]cron.EntryID
}

type SchedulerOption func(*Scheduler)

func WithTaskTimeout(timeout time.Duration) SchedulerOption {
	return func(s *Scheduler) {
		if timeout > 0 {
			s.taskTimeout = timeout
		}
	}
}

func NewScheduler(store TaskRuntimeStore, runner *Runner, opts ...SchedulerOption) *Scheduler {
	scheduler := &Scheduler{
		store:       store,
		runner:      runner,
		parser:      cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
		cronRunner:  cron.New(cron.WithSeconds()),
		taskTimeout: defaultTaskTimeout,
		entries:     make(map[uint64]cron.EntryID),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(scheduler)
		}
	}
	return scheduler
}

func (s *Scheduler) Start() {
	if s == nil || s.cronRunner == nil {
		return
	}
	s.cronRunner.Start()
}

func (s *Scheduler) Stop() context.Context {
	if s == nil || s.cronRunner == nil {
		return context.Background()
	}
	return s.cronRunner.Stop()
}

func (s *Scheduler) SyncTask(ctx context.Context, taskItem *taskv1.Task) error {
	if s == nil {
		return fmt.Errorf("task scheduler is not configured")
	}
	if taskItem == nil || taskItem.GetId() == 0 {
		return fmt.Errorf("task not found")
	}
	_ = s.StopTask(ctx, taskItem.GetId())

	entryID := uint32(0)
	if taskItem.GetStatus() == taskv1.Task_RUNNING {
		if _, err := s.parser.Parse(taskItem.GetCronExpression()); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
		cronEntryID, err := s.cronRunner.AddFunc(taskItem.GetCronExpression(), func() {
			runCtx, cancel := context.WithTimeout(context.Background(), s.taskTimeout)
			defer cancel()

			reloadedTask, err := s.store.GetTask(runCtx, taskItem.GetId())
			if err != nil || reloadedTask == nil || reloadedTask.GetStatus() != taskv1.Task_RUNNING {
				return
			}
			_ = s.runner.RunTask(runCtx, reloadedTask, "")
		})
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.entries[taskItem.GetId()] = cronEntryID
		s.mu.Unlock()
		entryID = uint32(cronEntryID)
	}

	return s.store.UpdateTaskRuntimeState(ctx, taskItem.GetId(), taskItem.GetStatus(), &entryID)
}

func (s *Scheduler) StopTask(_ context.Context, taskID uint64) error {
	if s == nil || taskID == 0 {
		return nil
	}
	s.mu.Lock()
	cronEntryID, ok := s.entries[taskID]
	if ok {
		delete(s.entries, taskID)
	}
	s.mu.Unlock()
	if ok {
		s.cronRunner.Remove(cronEntryID)
	}
	return nil
}

func (s *Scheduler) RunTaskNow(ctx context.Context, taskItem *taskv1.Task, overrideInput string) error {
	if taskItem == nil || taskItem.GetId() == 0 {
		return fmt.Errorf("task not found")
	}
	if s == nil || s.runner == nil {
		return fmt.Errorf("task runner is not configured")
	}
	return s.runner.RunTask(ctx, taskItem, overrideInput)
}

func (s *Scheduler) RestoreTasks(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("task scheduler is not configured")
	}
	if s.store == nil {
		return fmt.Errorf("task runtime store is not configured")
	}

	items, err := s.store.ListRunnableTasks(ctx)
	if err != nil {
		return err
	}

	var failed []*TaskSyncError
	for _, item := range items {
		if err := s.SyncTask(ctx, item); err != nil {
			failed = append(failed, &TaskSyncError{
				TaskID:   item.GetId(),
				TaskName: item.GetTaskName(),
				Cause:    err,
			})
		}
	}
	if len(failed) == 0 {
		return nil
	}
	return &TaskRestoreError{Failed: failed}
}

type TaskSyncError struct {
	TaskID   uint64
	TaskName string
	Cause    error
}

func (e *TaskSyncError) Error() string {
	if e == nil || e.Cause == nil {
		return "task sync error"
	}
	if e.TaskName != "" {
		return fmt.Sprintf("task %d (%s): %v", e.TaskID, e.TaskName, e.Cause)
	}
	return fmt.Sprintf("task %d: %v", e.TaskID, e.Cause)
}

type TaskRestoreError struct {
	Failed []*TaskSyncError
}

func (e *TaskRestoreError) Error() string {
	if e == nil || len(e.Failed) == 0 {
		return ""
	}
	return fmt.Sprintf("restore %d runnable tasks failed", len(e.Failed))
}
