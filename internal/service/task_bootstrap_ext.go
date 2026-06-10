package service

import (
	"context"
	"fmt"
	"strings"

	taskruntime "admin/internal/task/runtime"
)

func RegisterTaskScheduler(
	ctx context.Context,
	taskService *TaskService,
	taskGroupService *TaskGroupService,
) (func(), error) {
	scheduler, err := resolveTaskScheduler(taskService, taskGroupService)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	scheduler.Start()
	if err := restoreTaskScheduler(ctx, taskService, scheduler); err != nil {
		taskService.log.Errorf("register task scheduler failed: restore tasks: %s", err.Error())
		stopTaskScheduler(scheduler)
		return nil, err
	}
	return func() {
		stopTaskScheduler(scheduler)
	}, nil
}

func resolveTaskScheduler(
	taskService *TaskService,
	taskGroupService *TaskGroupService,
) (*taskruntime.Scheduler, error) {
	if taskService == nil || taskGroupService == nil {
		return nil, fmt.Errorf("task services are not configured")
	}
	if taskService.scheduler == nil {
		taskService.log.Errorf("resolve task scheduler failed: task scheduler is not configured")
		return nil, fmt.Errorf("task scheduler is not configured")
	}
	if taskGroupService.scheduler == nil {
		taskService.log.Errorf("resolve task scheduler failed: task group scheduler is not configured")
		return nil, fmt.Errorf("task group scheduler is not configured")
	}
	if taskService.scheduler != taskGroupService.scheduler {
		taskService.log.Errorf("resolve task scheduler failed: task schedulers are inconsistent")
		return nil, fmt.Errorf("task schedulers are inconsistent")
	}
	return taskService.scheduler, nil
}

func restoreTaskScheduler(
	ctx context.Context,
	taskService *TaskService,
	scheduler *taskruntime.Scheduler,
) error {
	if err := scheduler.RestoreTasks(ctx); err != nil {
		restoreErr, ok := err.(*taskruntime.TaskRestoreError)
		if !ok {
			return err
		}
		logTaskRestoreErrors(taskService, restoreErr)
	}
	return nil
}

func logTaskRestoreErrors(taskService *TaskService, restoreErr *taskruntime.TaskRestoreError) {
	if taskService == nil || taskService.log == nil || restoreErr == nil {
		return
	}
	for _, item := range restoreErr.Failed {
		if item == nil || item.Cause == nil {
			continue
		}
		taskService.log.Errorf("skip restoring task %d (%s): %v", item.TaskID, strings.TrimSpace(item.TaskName), item.Cause)
	}
}

func stopTaskScheduler(scheduler *taskruntime.Scheduler) {
	if scheduler == nil {
		return
	}
	stopCtx := scheduler.Stop()
	select {
	case <-stopCtx.Done():
	default:
	}
}
