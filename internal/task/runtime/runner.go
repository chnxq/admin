package taskruntime

import (
	"context"
	"fmt"
	"strings"
	"time"

	taskv1 "admin/api/gen/task/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type Runner struct {
	registry  *Registry
	logWriter TaskLogWriter
}

func NewRunner(registry *Registry, logWriter TaskLogWriter) *Runner {
	return &Runner{
		registry:  registry,
		logWriter: logWriter,
	}
}

func (r *Runner) Registry() *Registry {
	if r == nil {
		return nil
	}
	return r.registry
}

func (r *Runner) ValidateTask(ctx context.Context, taskItem *taskv1.Task, raw string) error {
	if r == nil || r.registry == nil {
		return fmt.Errorf("task executor registry is not configured")
	}
	return r.registry.Validate(ctx, taskItem, raw)
}

func (r *Runner) SupportsInvokeTarget(target string) bool {
	if r == nil || r.registry == nil {
		return false
	}
	_, ok := r.registry.Get(target)
	return ok
}

func (r *Runner) RunTask(ctx context.Context, taskItem *taskv1.Task, overrideInput string) error {
	if taskItem == nil || taskItem.GetId() == 0 {
		return fmt.Errorf("task not found")
	}
	if r == nil || r.registry == nil {
		return fmt.Errorf("task executor registry is not configured")
	}

	startAt := time.Now()
	input := strings.TrimSpace(taskItem.GetArgs())
	if strings.TrimSpace(overrideInput) != "" {
		input = strings.TrimSpace(overrideInput)
	}

	resultText, execErr := r.registry.Execute(ctx, taskItem, input)
	writeErr := r.writeTaskExecutionLog(ctx, taskItem, input, resultText, execErr, startAt)
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

func (r *Runner) writeTaskExecutionLog(
	ctx context.Context,
	taskItem *taskv1.Task,
	input string,
	output string,
	execErr error,
	startAt time.Time,
) error {
	if r == nil || r.logWriter == nil {
		return nil
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
	return r.logWriter.WriteTaskLog(ctx, logItem)
}

func uint64Ptr(value uint64) *uint64 {
	return &value
}

func stringValuePtr(value string) *string {
	return &value
}
