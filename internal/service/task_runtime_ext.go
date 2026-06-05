package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	taskv1 "admin/api/gen/task/v1"
	"admin/internal/data/repo"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type cleanupAuditLogInput struct {
	ExpireHours uint32   `json:"expireHours"`
	Targets     []string `json:"targets"`
}

type cleanupAuditLogResult struct {
	ExpireHours       uint32 `json:"expireHours"`
	DeletedAPI        int    `json:"deletedApi"`
	DeletedLogin      int    `json:"deletedLogin"`
	DeletedPermission int    `json:"deletedPermission"`
	TotalDeleted      int    `json:"totalDeleted"`
}

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
}

func (s *TaskService) start(ctx context.Context, req *taskv1.StartTaskRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	if err := updateTaskRuntimeState(ctx, s.taskRepo, req.GetId(), taskv1.Task_RUNNING, nil); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *TaskService) stop(ctx context.Context, req *taskv1.StopTaskRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	if err := updateTaskRuntimeState(ctx, s.taskRepo, req.GetId(), taskv1.Task_STOPPED, nil); err != nil {
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
	if err := executeTaskOnce(
		ctx,
		taskItem,
		req.GetInput(),
		s.taskLogRepo,
		s.apiAuditLogRepo,
		s.loginAuditLogRepo,
		s.permissionAuditLogRepo,
	); err != nil {
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
		if err := updateTaskRuntimeState(ctx, s.taskRepo, item.GetId(), taskv1.Task_RUNNING, nil); err != nil {
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
		if err := updateTaskRuntimeState(ctx, s.taskRepo, item.GetId(), taskv1.Task_STOPPED, nil); err != nil {
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
		if err := executeTaskOnce(
			ctx,
			item,
			req.GetInput(),
			s.taskLogRepo,
			s.apiAuditLogRepo,
			s.loginAuditLogRepo,
			s.permissionAuditLogRepo,
		); err != nil {
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
		apiAuditLogRepo,
		loginAuditLogRepo,
		permissionAuditLogRepo,
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
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
) (string, error) {
	switch strings.TrimSpace(taskItem.GetInvokeTarget()) {
	case "system:cleanup:audit-logs":
		return executeCleanupAuditLogs(ctx, taskItem, input, apiAuditLogRepo, loginAuditLogRepo, permissionAuditLogRepo)
	default:
		return "", fmt.Errorf("unsupported task invoke target: %s", taskItem.GetInvokeTarget())
	}
}

func executeCleanupAuditLogs(
	ctx context.Context,
	taskItem *taskv1.Task,
	input string,
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
) (string, error) {
	var payload cleanupAuditLogInput
	if strings.TrimSpace(input) != "" {
		if err := json.Unmarshal([]byte(input), &payload); err != nil {
			return "", fmt.Errorf("parse task args failed: %w", err)
		}
	}
	if payload.ExpireHours == 0 {
		return "", fmt.Errorf("expireHours must be greater than 0")
	}
	if len(payload.Targets) == 0 {
		payload.Targets = []string{"api", "login", "permission"}
	}

	before := time.Now().Add(-time.Duration(payload.ExpireHours) * time.Hour)
	result := cleanupAuditLogResult{
		ExpireHours: payload.ExpireHours,
	}

	var tenantID *uint32
	if taskItem.TenantId != nil && taskItem.GetTenantId() > 0 {
		value := taskItem.GetTenantId()
		tenantID = &value
	}

	for _, target := range payload.Targets {
		switch strings.ToLower(strings.TrimSpace(target)) {
		case "api":
			cleaner, ok := apiAuditLogRepo.(repo.ApiAuditLogCleaner)
			if !ok {
				return "", fmt.Errorf("api audit log cleaner is not configured")
			}
			affected, err := cleaner.CleanupApiAuditLogsBefore(ctx, tenantID, before)
			if err != nil {
				return "", err
			}
			result.DeletedAPI = affected
		case "login":
			cleaner, ok := loginAuditLogRepo.(repo.LoginAuditLogCleaner)
			if !ok {
				return "", fmt.Errorf("login audit log cleaner is not configured")
			}
			affected, err := cleaner.CleanupLoginAuditLogsBefore(ctx, tenantID, before)
			if err != nil {
				return "", err
			}
			result.DeletedLogin = affected
		case "permission":
			cleaner, ok := permissionAuditLogRepo.(repo.PermissionAuditLogCleaner)
			if !ok {
				return "", fmt.Errorf("permission audit log cleaner is not configured")
			}
			affected, err := cleaner.CleanupPermissionAuditLogsBefore(ctx, tenantID, before)
			if err != nil {
				return "", err
			}
			result.DeletedPermission = affected
		default:
			return "", fmt.Errorf("unsupported cleanup target: %s", target)
		}
	}

	result.TotalDeleted = result.DeletedAPI + result.DeletedLogin + result.DeletedPermission
	raw, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(raw), nil
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
