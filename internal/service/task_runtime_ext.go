package service

import (
	"context"
	"fmt"

	taskv1 "admin/api/gen/task/v1"
	"admin/internal/data/repo"
	taskruntime "admin/internal/task/runtime"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func BindTaskServices(
	taskService *TaskService,
	taskGroupService *TaskGroupService,
	taskRepo repo.TaskRepo,
	taskGroupRepo repo.TaskGroupRepo,
	runner *taskruntime.Runner,
	scheduler *taskruntime.Scheduler,
) error {
	if runner == nil {
		if taskService != nil && taskService.log != nil {
			taskService.log.Errorf("bind task services failed: task runner is not configured")
		}
		return fmt.Errorf("task runner is not configured")
	}
	if scheduler == nil {
		if taskService != nil && taskService.log != nil {
			taskService.log.Errorf("bind task services failed: task scheduler is not configured")
		}
		return fmt.Errorf("task scheduler is not configured")
	}
	if taskService != nil {
		taskService.taskGroupRepo = taskGroupRepo
		taskService.runtimeRunner = runner
		taskService.scheduler = scheduler
	}
	if taskGroupService != nil {
		taskGroupService.taskRepo = taskRepo
		taskGroupService.runtimeRunner = runner
		taskGroupService.scheduler = scheduler
	}
	return nil
}

func (s *TaskService) start(ctx context.Context, req *taskv1.StartTaskRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	taskItem, err := loadTask(ctx, s.taskRepo, req.GetId())
	if err != nil {
		s.log.Errorf("start task failed: load task id=%d failed: %s", req.GetId(), err.Error())
		return nil, err
	}
	status := taskv1.Task_RUNNING
	taskItem.Status = &status
	if err := validateCronExpression(taskItem.GetCronExpression(), taskItem.GetStatus()); err != nil {
		s.log.Errorf("start task failed: validate cron task_id=%d cron=%q: %s", taskItem.GetId(), taskItem.GetCronExpression(), err.Error())
		return nil, err
	}
	if err := s.syncTaskSchedule(ctx, taskItem); err != nil {
		s.log.Errorf("start task failed: sync schedule task_id=%d: %s", taskItem.GetId(), err.Error())
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (s *TaskService) stop(ctx context.Context, req *taskv1.StopTaskRequest) (*emptypb.Empty, error) {
	if req == nil || req.GetId() == 0 {
		return nil, fmt.Errorf("invalid parameter")
	}
	if err := s.stopScheduledTask(ctx, req.GetId()); err != nil {
		s.log.Errorf("stop task failed: stop schedule task_id=%d: %s", req.GetId(), err.Error())
		return nil, err
	}
	status := taskv1.Task_STOPPED
	entryID := uint32(0)
	if err := updateTaskRuntimeState(ctx, s.taskRepo, req.GetId(), status, &entryID); err != nil {
		s.log.Errorf("stop task failed: update runtime state task_id=%d: %s", req.GetId(), err.Error())
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
		s.log.Errorf("run task once failed: load task id=%d failed: %s", req.GetId(), err.Error())
		return nil, err
	}
	if s.scheduler != nil {
		if err := s.scheduler.RunTaskNow(ctx, taskItem, req.GetInput()); err != nil {
			s.log.Errorf("run task once failed: scheduler run task_id=%d: %s", taskItem.GetId(), err.Error())
			return nil, err
		}
		return &emptypb.Empty{}, nil
	}
	if s.runtimeRunner == nil {
		s.log.Errorf("run task once failed: task runner is not configured task_id=%d", taskItem.GetId())
		return nil, fmt.Errorf("task runner is not configured")
	}
	if err := s.runtimeRunner.RunTask(ctx, taskItem, req.GetInput()); err != nil {
		s.log.Errorf("run task once failed: runtime runner task_id=%d: %s", taskItem.GetId(), err.Error())
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
		s.log.Errorf("start task group failed: list tasks group_id=%d: %s", req.GetId(), err.Error())
		return nil, err
	}
	for _, item := range items {
		status := taskv1.Task_RUNNING
		item.Status = &status
		if err := s.syncTaskSchedule(ctx, item); err != nil {
			s.log.Errorf("start task group failed: sync schedule group_id=%d task_id=%d: %s", req.GetId(), item.GetId(), err.Error())
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
		s.log.Errorf("stop task group failed: list tasks group_id=%d: %s", req.GetId(), err.Error())
		return nil, err
	}
	for _, item := range items {
		if err := s.stopScheduledTask(ctx, item.GetId()); err != nil {
			s.log.Errorf("stop task group failed: stop schedule group_id=%d task_id=%d: %s", req.GetId(), item.GetId(), err.Error())
			return nil, err
		}
		status := taskv1.Task_STOPPED
		entryID := uint32(0)
		if err := updateTaskRuntimeState(ctx, s.taskRepo, item.GetId(), status, &entryID); err != nil {
			s.log.Errorf("stop task group failed: update runtime state group_id=%d task_id=%d: %s", req.GetId(), item.GetId(), err.Error())
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
		s.log.Errorf("run task group once failed: list tasks group_id=%d: %s", req.GetId(), err.Error())
		return nil, err
	}
	for _, item := range items {
		if s.scheduler != nil {
			if err := s.scheduler.RunTaskNow(ctx, item, req.GetInput()); err != nil {
				s.log.Errorf("run task group once failed: scheduler run group_id=%d task_id=%d: %s", req.GetId(), item.GetId(), err.Error())
				return nil, err
			}
			continue
		}
		if s.runtimeRunner == nil {
			s.log.Errorf("run task group once failed: task runner is not configured group_id=%d", req.GetId())
			return nil, fmt.Errorf("task runner is not configured")
		}
		if err := s.runtimeRunner.RunTask(ctx, item, req.GetInput()); err != nil {
			s.log.Errorf("run task group once failed: runtime runner group_id=%d task_id=%d: %s", req.GetId(), item.GetId(), err.Error())
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
