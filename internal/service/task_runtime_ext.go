package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	taskv1 "admin/api/gen/task/v1"
	"admin/internal/data/repo"
	taskruntime "admin/internal/task"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *TaskService) RegisterRuntimeDeps(
	taskGroupRepo repo.TaskGroupRepo,
	taskLogRepo repo.TaskLogRepo,
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
) {
	s.taskGroupRepo = taskGroupRepo
	s.taskLogRepo = taskLogRepo
	s.apiAuditLogRepo = apiAuditLogRepo
	s.loginAuditLogRepo = loginAuditLogRepo
	s.permissionAuditLogRepo = permissionAuditLogRepo
	s.executorRegistry = newTaskExecutorRegistry(s.taskRepo, apiAuditLogRepo, loginAuditLogRepo, permissionAuditLogRepo)
}

func (s *TaskGroupService) RegisterRuntimeDeps(
	taskRepo repo.TaskRepo,
	taskLogRepo repo.TaskLogRepo,
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
) {
	s.taskRepo = taskRepo
	s.taskLogRepo = taskLogRepo
	s.apiAuditLogRepo = apiAuditLogRepo
	s.loginAuditLogRepo = loginAuditLogRepo
	s.permissionAuditLogRepo = permissionAuditLogRepo
	s.executorRegistry = newTaskExecutorRegistry(taskRepo, apiAuditLogRepo, loginAuditLogRepo, permissionAuditLogRepo)
}

func (s *TaskService) start(ctx context.Context, req *taskv1.StartTaskRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	taskItem, err := loadTask(ctx, s.taskRepo, req.GetId())
	if err != nil {
		return nil, err
	}
	status := taskv1.Task_RUNNING
	taskItem.Status = &status
	if err := validateCronExpression(taskItem.GetCronExpression(), taskItem.GetStatus()); err != nil {
		return nil, err
	}
	if err := s.syncTaskSchedule(ctx, taskItem); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *TaskService) stop(ctx context.Context, req *taskv1.StopTaskRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	if err := s.stopScheduledTask(ctx, req.GetId()); err != nil {
		return nil, err
	}
	status := taskv1.Task_STOPPED
	entryID := uint32(0)
	if err := updateTaskRuntimeState(ctx, s.taskRepo, req.GetId(), status, &entryID); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *TaskService) runOnce(ctx context.Context, req *taskv1.RunTaskOnceRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}

	taskItem, err := loadTask(ctx, s.taskRepo, req.GetId())
	if err != nil {
		return nil, err
	}
	if s.scheduler != nil {
		if err := s.scheduler.RunTaskNow(ctx, taskItem, req.GetInput()); err != nil {
			return nil, err
		}
		return &emptypb.Empty{}, nil
	}
	if err := executeTaskOnce(ctx, taskItem, req.GetInput(), s.taskLogRepo, s.apiAuditLogRepo, s.loginAuditLogRepo, s.permissionAuditLogRepo, s.executorRegistry); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *TaskGroupService) start(ctx context.Context, req *taskv1.StartTaskGroupRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	items, err := listTasksByGroup(ctx, s.taskRepo, req.GetId())
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		status := taskv1.Task_RUNNING
		item.Status = &status
		if err := s.syncTaskSchedule(ctx, item); err != nil {
			return nil, err
		}
	}
	return &emptypb.Empty{}, nil
}

func (s *TaskGroupService) stop(ctx context.Context, req *taskv1.StopTaskGroupRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	items, err := listTasksByGroup(ctx, s.taskRepo, req.GetId())
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if err := s.stopScheduledTask(ctx, item.GetId()); err != nil {
			return nil, err
		}
		status := taskv1.Task_STOPPED
		entryID := uint32(0)
		if err := updateTaskRuntimeState(ctx, s.taskRepo, item.GetId(), status, &entryID); err != nil {
			return nil, err
		}
	}
	return &emptypb.Empty{}, nil
}

func (s *TaskGroupService) runOnce(ctx context.Context, req *taskv1.RunTaskGroupOnceRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	items, err := listTasksByGroup(ctx, s.taskRepo, req.GetId())
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if s.scheduler != nil {
			if err := s.scheduler.RunTaskNow(ctx, item, req.GetInput()); err != nil {
				return nil, err
			}
			continue
		}
		if err := executeTaskOnce(ctx, item, req.GetInput(), s.taskLogRepo, s.apiAuditLogRepo, s.loginAuditLogRepo, s.permissionAuditLogRepo, s.executorRegistry); err != nil {
			return nil, err
		}
	}
	return &emptypb.Empty{}, nil
}

func loadTask(ctx context.Context, taskRepo repo.TaskRepo, taskID uint64) (*taskv1.Task, error) {
	if taskRepo == nil {
		return nil, fmt.Errorf("task repo is not configured")
	}
	return taskRepo.Get(ctx, &taskv1.GetTaskRequest{
		QueryBy: &taskv1.GetTaskRequest_Id{Id: taskID},
	})
}

func listTasksByGroup(ctx context.Context, taskRepo repo.TaskRepo, groupID uint64) ([]*taskv1.Task, error) {
	runtimeRepo, ok := taskRepo.(repo.TaskRuntimeRepo)
	if !ok {
		return nil, fmt.Errorf("task runtime repo is not configured")
	}
	return runtimeRepo.ListTasksByGroupID(ctx, groupID)
}

func updateTaskRuntimeState(ctx context.Context, taskRepo repo.TaskRepo, taskID uint64, status taskv1.Task_Status, entryID *uint32) error {
	runtimeRepo, ok := taskRepo.(repo.TaskRuntimeRepo)
	if !ok {
		return fmt.Errorf("task runtime repo is not configured")
	}
	return runtimeRepo.UpdateTaskRuntimeState(ctx, taskID, status, entryID)
}

func executeTaskOnce(
	ctx context.Context,
	taskItem *taskv1.Task,
	overrideInput string,
	taskLogRepo repo.TaskLogRepo,
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
	executorRegistry *taskruntime.Registry,
) error {
	if taskItem == nil || taskItem.GetId() == 0 {
		return fmt.Errorf("task not found")
	}

	startAt := time.Now()
	input := strings.TrimSpace(taskItem.GetArgs())
	if strings.TrimSpace(overrideInput) != "" {
		input = strings.TrimSpace(overrideInput)
	}

	resultText, execErr := dispatchTaskExecution(
		ctx,
		taskItem,
		input,
		executorRegistry,
	)

	writeErr := writeTaskExecutionLog(ctx, taskLogRepo, taskItem, input, resultText, execErr, startAt)
	if execErr != nil {
		if writeErr != nil {
			return fmt.Errorf("%w; write task log: %v", execErr, writeErr)
		}
		return execErr
	}
	if writeErr != nil {
		return writeErr
	}
	return nil
}

func dispatchTaskExecution(
	ctx context.Context,
	taskItem *taskv1.Task,
	input string,
	registry *taskruntime.Registry,
) (string, error) {
	if registry == nil {
		return "", fmt.Errorf("task executor registry is not configured")
	}
	return registry.Execute(ctx, taskItem, input)
}

func writeTaskExecutionLog(
	ctx context.Context,
	taskLogRepo repo.TaskLogRepo,
	taskItem *taskv1.Task,
	input string,
	output string,
	execErr error,
	startAt time.Time,
) error {
	if taskLogRepo == nil {
		return nil
	}
	writer, ok := taskLogRepo.(repo.TaskLogWriter)
	if !ok {
		return fmt.Errorf("task log writer is not configured")
	}

	processTime := uint32(time.Since(startAt).Milliseconds())
	logItem := &taskv1.TaskLog{
		TaskId:      uint64Ptr(taskItem.GetId()),
		Input:       stringValuePtr(input),
		ProcessTime: &processTime,
		ExecuteTime: timestamppb.New(startAt),
	}
	if taskItem.TenantId != nil {
		tenantID := taskItem.GetTenantId()
		logItem.TenantId = &tenantID
	}
	if execErr != nil {
		status := taskv1.TaskLog_FAILURE
		text := execErr.Error()
		logItem.Status = &status
		logItem.Error = &text
		if strings.TrimSpace(output) != "" {
			logItem.Output = &output
		}
	} else {
		status := taskv1.TaskLog_SUCCESS
		logItem.Status = &status
		logItem.Output = &output
	}
	return writer.WriteTaskLog(ctx, logItem)
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}

func stringValuePtr(value string) *string {
	return &value
}
