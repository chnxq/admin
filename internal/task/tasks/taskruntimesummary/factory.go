package taskruntimesummary

import (
	"context"

	taskv1 "admin/api/gen/task/v1"
	"admin/internal/data/repo"
	taskruntime "admin/internal/task/runtime"
)

func NewTaskRuntimeSummaryExecutor(taskRepo repo.TaskRepo) taskruntime.Executor {
	var provider Provider
	if value, ok := taskRepo.(interface {
		ListTasksForRuntime(ctx context.Context, tenantID *uint32) ([]*taskv1.Task, error)
	}); ok {
		provider = value
	}
	return NewExecutor(provider)
}
