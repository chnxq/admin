package task

import (
	"context"
	"fmt"
	"strings"

	taskv1 "admin/api/gen/task/v1"
	"admin/internal/data/repo"
	taskruntime "admin/internal/task/runtime"
	crudviewer "github.com/chnxq/x-crud/viewer"
	"github.com/chnxq/xkitmod/log"
	"github.com/robfig/cron/v3"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type ServiceLogger interface {
	Errorf(format string, args ...any)
}

type RuntimeBindingTarget interface {
	SetTaskRuntimeDeps(taskRepo repo.TaskRepo, taskGroupRepo repo.TaskGroupRepo, runner *taskruntime.Runner, scheduler *taskruntime.Scheduler)
}

func ConfigureServices(
	taskService RuntimeBindingTarget,
	taskGroupService RuntimeBindingTarget,
	taskRepo repo.TaskRepo,
	taskGroupRepo repo.TaskGroupRepo,
	taskLogRepo repo.TaskLogRepo,
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
) error {
	bundle, err := NewRuntimeBundleFromRepos(
		taskRepo,
		taskLogRepo,
		apiAuditLogRepo,
		loginAuditLogRepo,
		permissionAuditLogRepo,
	)
	if err != nil {
		return err
	}
	return BindServices(
		nil,
		taskService,
		taskGroupService,
		taskRepo,
		taskGroupRepo,
		bundle.Runner,
		bundle.Scheduler,
	)
}

func RegisterServices(
	ctx context.Context,
	taskServiceScheduler *taskruntime.Scheduler,
	taskGroupServiceScheduler *taskruntime.Scheduler,
	logger ServiceLogger,
) (func(), error) {
	return RegisterScheduler(ctx, logger, taskServiceScheduler, taskGroupServiceScheduler)
}

func BindServices(
	logger ServiceLogger,
	taskService RuntimeBindingTarget,
	taskGroupService RuntimeBindingTarget,
	taskRepo repo.TaskRepo,
	taskGroupRepo repo.TaskGroupRepo,
	runner *taskruntime.Runner,
	scheduler *taskruntime.Scheduler,
) error {
	if runner == nil {
		if logger != nil {
			logger.Errorf("bind task services failed: task runner is not configured")
		}
		return fmt.Errorf("task runner is not configured")
	}
	if scheduler == nil {
		if logger != nil {
			logger.Errorf("bind task services failed: task scheduler is not configured")
		}
		return fmt.Errorf("task scheduler is not configured")
	}
	if taskService != nil {
		taskService.SetTaskRuntimeDeps(taskRepo, taskGroupRepo, runner, scheduler)
	}
	if taskGroupService != nil {
		taskGroupService.SetTaskRuntimeDeps(taskRepo, taskGroupRepo, runner, scheduler)
	}
	return nil
}

func RegisterScheduler(
	ctx context.Context,
	logger ServiceLogger,
	taskServiceScheduler *taskruntime.Scheduler,
	taskGroupServiceScheduler *taskruntime.Scheduler,
) (func(), error) {
	scheduler, err := ResolveScheduler(logger, taskServiceScheduler, taskGroupServiceScheduler)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	scheduler.Start()
	if err := RestoreScheduler(ctx, logger, scheduler); err != nil {
		if logger != nil {
			logger.Errorf("register task scheduler failed: restore tasks: %s", err.Error())
		}
		StopScheduler(scheduler)
		return nil, err
	}
	return func() {
		StopScheduler(scheduler)
	}, nil
}

func ResolveScheduler(logger ServiceLogger, taskServiceScheduler *taskruntime.Scheduler, taskGroupServiceScheduler *taskruntime.Scheduler) (*taskruntime.Scheduler, error) {
	if taskServiceScheduler == nil || taskGroupServiceScheduler == nil {
		if logger != nil {
			if taskServiceScheduler == nil {
				logger.Errorf("resolve task scheduler failed: task scheduler is not configured")
			} else {
				logger.Errorf("resolve task scheduler failed: task group scheduler is not configured")
			}
		}
		return nil, fmt.Errorf("task scheduler is not configured")
	}
	if taskServiceScheduler != taskGroupServiceScheduler {
		if logger != nil {
			logger.Errorf("resolve task scheduler failed: task schedulers are inconsistent")
		}
		return nil, fmt.Errorf("task schedulers are inconsistent")
	}
	return taskServiceScheduler, nil
}

func RestoreScheduler(ctx context.Context, logger ServiceLogger, scheduler *taskruntime.Scheduler) error {
	if err := scheduler.RestoreTasks(ctx); err != nil {
		restoreErr, ok := err.(*taskruntime.TaskRestoreError)
		if !ok {
			return err
		}
		LogTaskRestoreErrors(logger, restoreErr)
	}
	return nil
}

func LogTaskRestoreErrors(logger ServiceLogger, restoreErr *taskruntime.TaskRestoreError) {
	if logger == nil || restoreErr == nil {
		return
	}
	for _, item := range restoreErr.Failed {
		if item == nil || item.Cause == nil {
			continue
		}
		logger.Errorf("skip restoring task %d (%s): %v", item.TaskID, strings.TrimSpace(item.TaskName), item.Cause)
	}
}

func StopScheduler(scheduler *taskruntime.Scheduler) {
	if scheduler == nil {
		return
	}
	stopCtx := scheduler.Stop()
	select {
	case <-stopCtx.Done():
	default:
	}
}

func LoadTask(ctx context.Context, taskRepo repo.TaskRepo, taskID uint64) (*taskv1.Task, error) {
	if taskRepo == nil {
		return nil, fmt.Errorf("task repo is not configured")
	}
	return taskRepo.Get(ctx, &taskv1.GetTaskRequest{
		QueryBy: &taskv1.GetTaskRequest_Id{Id: taskID},
	})
}

func ListTasksByGroup(ctx context.Context, taskRepo repo.TaskRepo, groupID uint64) ([]*taskv1.Task, error) {
	runtimeRepo, ok := taskRepo.(repo.TaskRuntimeRepo)
	if !ok {
		return nil, fmt.Errorf("task runtime repo is not configured")
	}
	return runtimeRepo.ListTasksByGroupID(ctx, groupID)
}

func UpdateTaskRuntimeState(ctx context.Context, taskRepo repo.TaskRepo, taskID uint64, status taskv1.Task_Status, entryID *uint32) error {
	runtimeRepo, ok := taskRepo.(repo.TaskRuntimeRepo)
	if !ok {
		return fmt.Errorf("task runtime repo is not configured")
	}
	return runtimeRepo.UpdateTaskRuntimeState(ctx, taskID, status, entryID)
}

func EnsureTaskGroupExists(ctx context.Context, taskGroupRepo repo.TaskGroupRepo, groupID uint64) error {
	if taskGroupRepo == nil {
		return fmt.Errorf("task group repo is not configured")
	}
	_, err := taskGroupRepo.Get(ctx, &taskv1.GetTaskGroupRequest{
		QueryBy: &taskv1.GetTaskGroupRequest_Id{Id: groupID},
	})
	return err
}

func LoadTaskByName(ctx context.Context, taskRepo repo.TaskRepo, taskName string) (*taskv1.Task, error) {
	if taskRepo == nil {
		return nil, fmt.Errorf("task repo finder is not configured")
	}
	finder, ok := taskRepo.(interface {
		FindTaskByName(context.Context, string) (*taskv1.Task, error)
	})
	if !ok {
		return nil, fmt.Errorf("task repo finder is not configured")
	}
	return finder.FindTaskByName(ctx, taskName)
}

func SyncTaskSchedule(ctx context.Context, scheduler *taskruntime.Scheduler, taskRepo repo.TaskRepo, taskItem *taskv1.Task) error {
	if scheduler == nil {
		return UpdateTaskRuntimeState(ctx, taskRepo, taskItem.GetId(), taskItem.GetStatus(), uint32Ptr(0))
	}
	return scheduler.SyncTask(ctx, taskItem)
}

func StopScheduledTask(ctx context.Context, scheduler *taskruntime.Scheduler, taskID uint64) error {
	if scheduler == nil {
		return nil
	}
	return scheduler.StopTask(ctx, taskID)
}

func ValidateInvokeTarget(target string, runner *taskruntime.Runner) error {
	if runner == nil {
		return fmt.Errorf("task runner is not configured")
	}
	if !runner.SupportsInvokeTarget(target) {
		return fmt.Errorf("unsupported task invoke target: %s", target)
	}
	return nil
}

func ValidateCronExpression(expr string, status taskv1.Task_Status) error {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		if status == taskv1.Task_RUNNING {
			return fmt.Errorf("cron expression is required when task is running")
		}
		return nil
	}
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(expr); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	return nil
}

func ValidateTaskArgs(ctx context.Context, taskItem *taskv1.Task, raw string, runner *taskruntime.Runner) error {
	if runner == nil {
		return fmt.Errorf("task runner is not configured")
	}
	return runner.ValidateTask(ctx, taskItem, raw)
}

func ValidateTaskInput(
	ctx context.Context,
	data *taskv1.Task,
	creating bool,
	taskGroupRepo repo.TaskGroupRepo,
	runner *taskruntime.Runner,
) error {
	if data == nil {
		return fmt.Errorf("task data is required")
	}
	if strings.TrimSpace(data.GetTaskName()) == "" {
		return fmt.Errorf("task name is required")
	}
	if data.GetGroupId() == 0 {
		return fmt.Errorf("group id is required")
	}
	if err := EnsureTaskGroupExists(ctx, taskGroupRepo, data.GetGroupId()); err != nil {
		return err
	}
	if data.GetTaskType() == taskv1.Task_TASK_TYPE_UNSPECIFIED {
		return fmt.Errorf("task type is required")
	}
	if data.GetRetry() > 5 {
		return fmt.Errorf("retry must be between 0 and 5")
	}
	if data.GetStatus() == taskv1.Task_STATUS_UNSPECIFIED {
		return fmt.Errorf("task status is required")
	}
	if err := ValidateInvokeTarget(data.GetInvokeTarget(), runner); err != nil {
		return err
	}
	if err := ValidateCronExpression(data.GetCronExpression(), data.GetStatus()); err != nil {
		return err
	}
	if err := ValidateTaskArgs(ctx, data, data.GetArgs(), runner); err != nil {
		return err
	}
	if creating && data.TenantId == nil {
		if viewer, ok := crudviewer.FromContext(ctx); ok && viewer != nil && viewer.IsTenantContext() {
			tenantID := uint32(viewer.TenantID())
			data.TenantId = &tenantID
		}
	}
	return nil
}

func ValidateTaskGroupInput(ctx context.Context, data *taskv1.TaskGroup, creating bool) error {
	if data == nil {
		return fmt.Errorf("task group data is required")
	}
	if strings.TrimSpace(data.GetGroupName()) == "" {
		return fmt.Errorf("group name is required")
	}
	if creating && data.TenantId == nil {
		if viewer, ok := crudviewer.FromContext(ctx); ok && viewer != nil && viewer.IsTenantContext() {
			tenantID := uint32(viewer.TenantID())
			data.TenantId = &tenantID
		}
	}
	return nil
}

func CreateTask(
	ctx context.Context,
	logger ServiceLogger,
	taskRepo repo.TaskRepo,
	taskGroupRepo repo.TaskGroupRepo,
	runner *taskruntime.Runner,
	scheduler *taskruntime.Scheduler,
	req *taskv1.CreateTaskRequest,
) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("invalid parameter")
	}
	if err := ValidateTaskInput(ctx, req.Data, true, taskGroupRepo, runner); err != nil {
		if logger != nil {
			logger.Errorf("create task failed: validate input task_name=%q group_id=%d: %s", req.Data.GetTaskName(), req.Data.GetGroupId(), err.Error())
		}
		return nil, err
	}
	if _, err := taskRepo.Create(ctx, req); err != nil {
		return nil, err
	}
	taskItem, err := LoadTaskByName(ctx, taskRepo, req.Data.GetTaskName())
	if err != nil {
		return nil, err
	}
	if err := SyncTaskSchedule(ctx, scheduler, taskRepo, taskItem); err != nil {
		if logger != nil {
			logger.Errorf("create task failed: sync schedule task_id=%d: %s", taskItem.GetId(), err.Error())
		}
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func UpdateTask(
	ctx context.Context,
	logger ServiceLogger,
	taskRepo repo.TaskRepo,
	taskGroupRepo repo.TaskGroupRepo,
	runner *taskruntime.Runner,
	scheduler *taskruntime.Scheduler,
	req *taskv1.UpdateTaskRequest,
) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	if err := ValidateTaskInput(ctx, req.Data, false, taskGroupRepo, runner); err != nil {
		if logger != nil {
			logger.Errorf("update task failed: validate input task_id=%d task_name=%q: %s", req.GetId(), req.Data.GetTaskName(), err.Error())
		}
		return nil, err
	}
	if _, err := taskRepo.Update(ctx, req); err != nil {
		return nil, err
	}
	taskItem, err := LoadTask(ctx, taskRepo, req.GetId())
	if err != nil {
		return nil, err
	}
	if err := SyncTaskSchedule(ctx, scheduler, taskRepo, taskItem); err != nil {
		if logger != nil {
			logger.Errorf("update task failed: sync schedule task_id=%d: %s", taskItem.GetId(), err.Error())
		}
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func DeleteTask(
	ctx context.Context,
	logger ServiceLogger,
	taskRepo repo.TaskRepo,
	scheduler *taskruntime.Scheduler,
	req *taskv1.DeleteTaskRequest,
) (*emptypb.Empty, error) {
	if req == nil || len(req.GetIds()) == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	for _, id := range req.GetIds() {
		if err := StopScheduledTask(ctx, scheduler, id); err != nil {
			if logger != nil {
				logger.Errorf("delete task failed: stop schedule task_id=%d: %s", id, err.Error())
			}
			return nil, err
		}
	}
	return taskRepo.Delete(ctx, req)
}

func StartTask(
	ctx context.Context,
	logger ServiceLogger,
	taskRepo repo.TaskRepo,
	scheduler *taskruntime.Scheduler,
	req *taskv1.StartTaskRequest,
) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	taskItem, err := LoadTask(ctx, taskRepo, req.GetId())
	if err != nil {
		if logger != nil {
			logger.Errorf("start task failed: load task id=%d failed: %s", req.GetId(), err.Error())
		}
		return nil, err
	}
	status := taskv1.Task_RUNNING
	taskItem.Status = &status
	if err := ValidateCronExpression(taskItem.GetCronExpression(), taskItem.GetStatus()); err != nil {
		if logger != nil {
			logger.Errorf("start task failed: validate cron task_id=%d cron=%q: %s", taskItem.GetId(), taskItem.GetCronExpression(), err.Error())
		}
		return nil, err
	}
	if err := SyncTaskSchedule(ctx, scheduler, taskRepo, taskItem); err != nil {
		if logger != nil {
			logger.Errorf("start task failed: sync schedule task_id=%d: %s", taskItem.GetId(), err.Error())
		}
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func StopTask(
	ctx context.Context,
	logger ServiceLogger,
	taskRepo repo.TaskRepo,
	scheduler *taskruntime.Scheduler,
	req *taskv1.StopTaskRequest,
) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	if err := StopScheduledTask(ctx, scheduler, req.GetId()); err != nil {
		if logger != nil {
			logger.Errorf("stop task failed: stop schedule task_id=%d: %s", req.GetId(), err.Error())
		}
		return nil, err
	}
	status := taskv1.Task_STOPPED
	entryID := uint32(0)
	if err := UpdateTaskRuntimeState(ctx, taskRepo, req.GetId(), status, &entryID); err != nil {
		if logger != nil {
			logger.Errorf("stop task failed: update runtime state task_id=%d: %s", req.GetId(), err.Error())
		}
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func RunTaskOnce(
	ctx context.Context,
	logger ServiceLogger,
	taskRepo repo.TaskRepo,
	runner *taskruntime.Runner,
	scheduler *taskruntime.Scheduler,
	req *taskv1.RunTaskOnceRequest,
) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	taskItem, err := LoadTask(ctx, taskRepo, req.GetId())
	if err != nil {
		if logger != nil {
			logger.Errorf("run task once failed: load task id=%d failed: %s", req.GetId(), err.Error())
		}
		return nil, err
	}
	if scheduler != nil {
		if err := scheduler.RunTaskNow(ctx, taskItem, req.GetInput()); err != nil {
			if logger != nil {
				logger.Errorf("run task once failed: scheduler run task_id=%d: %s", taskItem.GetId(), err.Error())
			}
			return nil, err
		}
		return &emptypb.Empty{}, nil
	}
	if runner == nil {
		if logger != nil {
			logger.Errorf("run task once failed: task runner is not configured task_id=%d", taskItem.GetId())
		}
		return nil, fmt.Errorf("task runner is not configured")
	}
	if err := runner.RunTask(ctx, taskItem, req.GetInput()); err != nil {
		if logger != nil {
			logger.Errorf("run task once failed: runtime runner task_id=%d: %s", taskItem.GetId(), err.Error())
		}
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func CreateTaskGroup(ctx context.Context, logger ServiceLogger, taskGroupRepo repo.TaskGroupRepo, req *taskv1.CreateTaskGroupRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("invalid parameter")
	}
	if err := ValidateTaskGroupInput(ctx, req.Data, true); err != nil {
		if logger != nil {
			logger.Errorf("create task group failed: validate input group_name=%q: %s", req.Data.GetGroupName(), err.Error())
		}
		return nil, err
	}
	return taskGroupRepo.Create(ctx, req)
}

func UpdateTaskGroup(ctx context.Context, logger ServiceLogger, taskGroupRepo repo.TaskGroupRepo, req *taskv1.UpdateTaskGroupRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	if err := ValidateTaskGroupInput(ctx, req.Data, false); err != nil {
		if logger != nil {
			logger.Errorf("update task group failed: validate input group_id=%d group_name=%q: %s", req.GetId(), req.Data.GetGroupName(), err.Error())
		}
		return nil, err
	}
	return taskGroupRepo.Update(ctx, req)
}

func DeleteTaskGroup(
	ctx context.Context,
	logger ServiceLogger,
	taskGroupRepo repo.TaskGroupRepo,
	taskRepo repo.TaskRepo,
	req *taskv1.DeleteTaskGroupRequest,
) (*emptypb.Empty, error) {
	if req == nil || len(req.GetIds()) == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	for _, groupID := range req.GetIds() {
		items, err := ListTasksByGroup(ctx, taskRepo, groupID)
		if err != nil {
			if logger != nil {
				logger.Errorf("delete task group failed: list tasks group_id=%d: %s", groupID, err.Error())
			}
			return nil, err
		}
		if len(items) > 0 {
			if logger != nil {
				logger.Errorf("delete task group failed: group_id=%d still has %d tasks", groupID, len(items))
			}
			return nil, fmt.Errorf("task group %d still has tasks", groupID)
		}
	}
	return taskGroupRepo.Delete(ctx, req)
}

func StartTaskGroup(
	ctx context.Context,
	logger ServiceLogger,
	taskRepo repo.TaskRepo,
	scheduler *taskruntime.Scheduler,
	req *taskv1.StartTaskGroupRequest,
) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	items, err := ListTasksByGroup(ctx, taskRepo, req.GetId())
	if err != nil {
		if logger != nil {
			logger.Errorf("start task group failed: list tasks group_id=%d: %s", req.GetId(), err.Error())
		}
		return nil, err
	}
	for _, item := range items {
		status := taskv1.Task_RUNNING
		item.Status = &status
		if err := SyncTaskSchedule(ctx, scheduler, taskRepo, item); err != nil {
			if logger != nil {
				logger.Errorf("start task group failed: sync schedule group_id=%d task_id=%d: %s", req.GetId(), item.GetId(), err.Error())
			}
			return nil, err
		}
	}
	return &emptypb.Empty{}, nil
}

func StopTaskGroup(
	ctx context.Context,
	logger ServiceLogger,
	taskRepo repo.TaskRepo,
	scheduler *taskruntime.Scheduler,
	req *taskv1.StopTaskGroupRequest,
) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	items, err := ListTasksByGroup(ctx, taskRepo, req.GetId())
	if err != nil {
		if logger != nil {
			logger.Errorf("stop task group failed: list tasks group_id=%d: %s", req.GetId(), err.Error())
		}
		return nil, err
	}
	for _, item := range items {
		if err := StopScheduledTask(ctx, scheduler, item.GetId()); err != nil {
			if logger != nil {
				logger.Errorf("stop task group failed: stop schedule group_id=%d task_id=%d: %s", req.GetId(), item.GetId(), err.Error())
			}
			return nil, err
		}
		status := taskv1.Task_STOPPED
		entryID := uint32(0)
		if err := UpdateTaskRuntimeState(ctx, taskRepo, item.GetId(), status, &entryID); err != nil {
			if logger != nil {
				logger.Errorf("stop task group failed: update runtime state group_id=%d task_id=%d: %s", req.GetId(), item.GetId(), err.Error())
			}
			return nil, err
		}
	}
	return &emptypb.Empty{}, nil
}

func RunTaskGroupOnce(
	ctx context.Context,
	logger ServiceLogger,
	taskRepo repo.TaskRepo,
	runner *taskruntime.Runner,
	scheduler *taskruntime.Scheduler,
	req *taskv1.RunTaskGroupOnceRequest,
) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	items, err := ListTasksByGroup(ctx, taskRepo, req.GetId())
	if err != nil {
		if logger != nil {
			logger.Errorf("run task group once failed: list tasks group_id=%d: %s", req.GetId(), err.Error())
		}
		return nil, err
	}
	for _, item := range items {
		if scheduler != nil {
			if err := scheduler.RunTaskNow(ctx, item, req.GetInput()); err != nil {
				if logger != nil {
					logger.Errorf("run task group once failed: scheduler run group_id=%d task_id=%d: %s", req.GetId(), item.GetId(), err.Error())
				}
				return nil, err
			}
			continue
		}
		if runner == nil {
			if logger != nil {
				logger.Errorf("run task group once failed: task runner is not configured group_id=%d", req.GetId())
			}
			return nil, fmt.Errorf("task runner is not configured")
		}
		if err := runner.RunTask(ctx, item, req.GetInput()); err != nil {
			if logger != nil {
				logger.Errorf("run task group once failed: runtime runner group_id=%d task_id=%d: %s", req.GetId(), item.GetId(), err.Error())
			}
			return nil, err
		}
	}
	return &emptypb.Empty{}, nil
}

func NewLoggerAdapter(helper *log.Helper) ServiceLogger {
	return helper
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}
