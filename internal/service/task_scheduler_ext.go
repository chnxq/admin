package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	taskv1 "admin/api/gen/task/v1"
	"admin/internal/data/repo"
	taskruntime "admin/internal/task"
	"github.com/robfig/cron/v3"
)

type taskScheduler struct {
	taskRepo               repo.TaskRepo
	taskLogRepo            repo.TaskLogRepo
	apiAuditLogRepo        repo.ApiAuditLogRepo
	loginAuditLogRepo      repo.LoginAuditLogRepo
	permissionAuditLogRepo repo.PermissionAuditLogRepo
	executorRegistry       *taskruntime.Registry

	parser  cron.Parser
	runner  *cron.Cron
	mu      sync.RWMutex
	entries map[uint64]cron.EntryID
}

type taskSyncError struct {
	TaskID   uint64
	TaskName string
	Cause    error
}

func (e *taskSyncError) Error() string {
	if e == nil || e.Cause == nil {
		return "task sync error"
	}
	if e.TaskName != "" {
		return fmt.Sprintf("task %d (%s): %v", e.TaskID, e.TaskName, e.Cause)
	}
	return fmt.Sprintf("task %d: %v", e.TaskID, e.Cause)
}

func newTaskScheduler(
	taskRepo repo.TaskRepo,
	taskLogRepo repo.TaskLogRepo,
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
	executorRegistry *taskruntime.Registry,
) *taskScheduler {
	return &taskScheduler{
		taskRepo:               taskRepo,
		taskLogRepo:            taskLogRepo,
		apiAuditLogRepo:        apiAuditLogRepo,
		loginAuditLogRepo:      loginAuditLogRepo,
		permissionAuditLogRepo: permissionAuditLogRepo,
		executorRegistry:       executorRegistry,
		parser:                 cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
		runner:                 cron.New(cron.WithSeconds()),
		entries:                make(map[uint64]cron.EntryID),
	}
}

func (s *taskScheduler) Start() {
	if s == nil || s.runner == nil {
		return
	}
	s.runner.Start()
}

func (s *taskScheduler) Stop() context.Context {
	if s == nil || s.runner == nil {
		return context.Background()
	}
	return s.runner.Stop()
}

func (s *taskScheduler) SyncTask(ctx context.Context, taskItem *taskv1.Task) error {
	if s == nil {
		return fmt.Errorf("task scheduler is not configured")
	}
	if taskItem == nil || taskItem.GetId() == 0 {
		return fmt.Errorf("task not found")
	}
	s.StopTask(ctx, taskItem.GetId())

	entryID := uint32(0)
	if taskItem.GetStatus() == taskv1.Task_RUNNING {
		if _, err := s.parser.Parse(taskItem.GetCronExpression()); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
		cronEntryID, err := s.runner.AddFunc(taskItem.GetCronExpression(), func() {
			runCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()

			reloadedTask, err := loadTask(runCtx, s.taskRepo, taskItem.GetId())
			if err != nil || reloadedTask == nil || reloadedTask.GetStatus() != taskv1.Task_RUNNING {
				return
			}
			_ = executeTaskOnce(
				runCtx,
				reloadedTask,
				"",
				s.taskLogRepo,
				s.apiAuditLogRepo,
				s.loginAuditLogRepo,
				s.permissionAuditLogRepo,
				s.executorRegistry,
			)
		})
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.entries[taskItem.GetId()] = cronEntryID
		s.mu.Unlock()
		entryID = uint32(cronEntryID)
	}

	return updateTaskRuntimeState(ctx, s.taskRepo, taskItem.GetId(), taskItem.GetStatus(), &entryID)
}

func (s *taskScheduler) StopTask(_ context.Context, taskID uint64) error {
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
		s.runner.Remove(cronEntryID)
	}
	return nil
}

func (s *taskScheduler) RunTaskNow(ctx context.Context, taskItem *taskv1.Task, overrideInput string) error {
	if taskItem == nil || taskItem.GetId() == 0 {
		return fmt.Errorf("task not found")
	}
	return executeTaskOnce(
		ctx,
		taskItem,
		overrideInput,
		s.taskLogRepo,
		s.apiAuditLogRepo,
		s.loginAuditLogRepo,
		s.permissionAuditLogRepo,
		s.executorRegistry,
	)
}

func (s *taskScheduler) LoadAllRunnableTasks(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("task scheduler is not configured")
	}
	runtimeRepo, ok := s.taskRepo.(repo.TaskRuntimeRepo)
	if !ok {
		return fmt.Errorf("task runtime repo is not configured")
	}
	items, err := runtimeRepo.ListRunnableTasks(ctx)
	if err != nil {
		return err
	}
	var failed []*taskSyncError
	for _, item := range items {
		if err := s.SyncTask(ctx, item); err != nil {
			failed = append(failed, &taskSyncError{
				TaskID:   item.GetId(),
				TaskName: item.GetTaskName(),
				Cause:    err,
			})
		}
	}
	if len(failed) == 0 {
		return nil
	}
	return &taskRuntimeRestoreError{Failed: failed}
}

type taskRuntimeRestoreError struct {
	Failed []*taskSyncError
}

func (e *taskRuntimeRestoreError) Error() string {
	if e == nil || len(e.Failed) == 0 {
		return ""
	}
	return fmt.Sprintf("restore %d runnable tasks failed", len(e.Failed))
}
