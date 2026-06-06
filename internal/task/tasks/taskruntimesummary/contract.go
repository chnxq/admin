package taskruntimesummary

import (
	"context"

	taskv1 "admin/api/gen/task/v1"
)

const (
	TaskRuntimeSummaryInvokeTarget = "system:task:runtime-summary"
)

type Provider interface {
	ListTasksForRuntime(ctx context.Context, tenantID *uint32) ([]*taskv1.Task, error)
}
