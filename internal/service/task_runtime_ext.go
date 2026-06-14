package service

import (
	"context"

	taskv1 "admin/api/gen/task/v1"
	"admin/internal/data/repo"
	taskpkg "admin/internal/task"
	taskruntime "admin/internal/task/runtime"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func (s *TaskService) SetTaskRuntimeDeps(_ repo.TaskRepo, taskGroupRepo repo.TaskGroupRepo, runner *taskruntime.Runner, scheduler *taskruntime.Scheduler) {
	if s == nil {
		return
	}
	s.taskGroupRepo = taskGroupRepo
	s.runtimeRunner = runner
	s.scheduler = scheduler
}

func (s *TaskService) start(ctx context.Context, req *taskv1.StartTaskRequest) (*emptypb.Empty, error) {
	return taskpkg.StartTask(ctx, s.log, s.taskRepo, s.scheduler, req)
}

func (s *TaskService) stop(ctx context.Context, req *taskv1.StopTaskRequest) (*emptypb.Empty, error) {
	return taskpkg.StopTask(ctx, s.log, s.taskRepo, s.scheduler, req)
}

func (s *TaskService) runOnce(ctx context.Context, req *taskv1.RunTaskOnceRequest) (*emptypb.Empty, error) {
	return taskpkg.RunTaskOnce(ctx, s.log, s.taskRepo, s.runtimeRunner, s.scheduler, req)
}

func (s *TaskGroupService) start(ctx context.Context, req *taskv1.StartTaskGroupRequest) (*emptypb.Empty, error) {
	return taskpkg.StartTaskGroup(ctx, s.log, s.taskRepo, s.scheduler, req)
}

func (s *TaskGroupService) stop(ctx context.Context, req *taskv1.StopTaskGroupRequest) (*emptypb.Empty, error) {
	return taskpkg.StopTaskGroup(ctx, s.log, s.taskRepo, s.scheduler, req)
}

func (s *TaskGroupService) runOnce(ctx context.Context, req *taskv1.RunTaskGroupOnceRequest) (*emptypb.Empty, error) {
	return taskpkg.RunTaskGroupOnce(ctx, s.log, s.taskRepo, s.runtimeRunner, s.scheduler, req)
}

func (s *TaskGroupService) SetTaskRuntimeDeps(taskRepo repo.TaskRepo, _ repo.TaskGroupRepo, runner *taskruntime.Runner, scheduler *taskruntime.Scheduler) {
	if s == nil {
		return
	}
	s.taskRepo = taskRepo
	s.runtimeRunner = runner
	s.scheduler = scheduler
}
