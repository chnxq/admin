package task

import (
	"context"
	"fmt"

	taskv1 "admin/api/gen/task/v1"
	"admin/internal/data/repo"
	taskruntime "admin/internal/task/runtime"
)

type runtimeStore struct {
	taskRepo    repo.TaskRepo
	runtimeRepo repo.TaskRuntimeRepo
}

func NewTaskRuntimeStore(taskRepo repo.TaskRepo) (taskruntime.TaskRuntimeStore, error) {
	if taskRepo == nil {
		return nil, fmt.Errorf("task repo is not configured")
	}
	runtimeRepo, ok := taskRepo.(repo.TaskRuntimeRepo)
	if !ok {
		return nil, fmt.Errorf("task runtime repo is not configured")
	}
	return &runtimeStore{
		taskRepo:    taskRepo,
		runtimeRepo: runtimeRepo,
	}, nil
}

func (s *runtimeStore) GetTask(ctx context.Context, taskID uint64) (*taskv1.Task, error) {
	if s == nil || s.taskRepo == nil {
		return nil, fmt.Errorf("task repo is not configured")
	}
	return s.taskRepo.Get(ctx, &taskv1.GetTaskRequest{
		QueryBy: &taskv1.GetTaskRequest_Id{Id: taskID},
	})
}

func (s *runtimeStore) ListRunnableTasks(ctx context.Context) ([]*taskv1.Task, error) {
	if s == nil || s.runtimeRepo == nil {
		return nil, fmt.Errorf("task runtime repo is not configured")
	}
	return s.runtimeRepo.ListRunnableTasks(ctx)
}

func (s *runtimeStore) UpdateTaskRuntimeState(
	ctx context.Context,
	taskID uint64,
	status taskv1.Task_Status,
	entryID *uint32,
) error {
	if s == nil || s.runtimeRepo == nil {
		return fmt.Errorf("task runtime repo is not configured")
	}
	return s.runtimeRepo.UpdateTaskRuntimeState(ctx, taskID, status, entryID)
}
