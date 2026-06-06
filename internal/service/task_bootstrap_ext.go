package service

import (
	"context"
	"fmt"
	"strings"

	"admin/internal/data/repo"
)

func RegisterTaskScheduler(
	ctx context.Context,
	taskService *TaskService,
	taskGroupService *TaskGroupService,
	taskRepo repo.TaskRepo,
	taskLogRepo repo.TaskLogRepo,
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
) (func(), error) {
	if taskService == nil || taskGroupService == nil {
		return nil, fmt.Errorf("task services are not configured")
	}

	scheduler := newTaskScheduler(
		taskRepo,
		taskLogRepo,
		apiAuditLogRepo,
		loginAuditLogRepo,
		permissionAuditLogRepo,
		newTaskExecutorRegistry(apiAuditLogRepo, loginAuditLogRepo, permissionAuditLogRepo),
	)

	taskService.scheduler = scheduler
	taskGroupService.scheduler = scheduler

	if ctx == nil {
		ctx = context.Background()
	}
	scheduler.Start()
	if err := scheduler.LoadAllRunnableTasks(ctx); err != nil {
		if restoreErr, ok := err.(*taskRuntimeRestoreError); ok {
			if taskService.log != nil {
				for _, item := range restoreErr.Failed {
					if item == nil || item.Cause == nil {
						continue
					}
					taskService.log.Errorf("skip restoring task %d (%s): %v", item.TaskID, strings.TrimSpace(item.TaskName), item.Cause)
				}
			}
		} else {
			stopCtx := scheduler.Stop()
			select {
			case <-stopCtx.Done():
			default:
			}
			return nil, err
		}
	}
	return func() {
		stopCtx := scheduler.Stop()
		select {
		case <-stopCtx.Done():
		default:
		}
	}, nil
}
