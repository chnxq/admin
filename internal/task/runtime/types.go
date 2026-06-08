package taskruntime

import (
	"context"
	"fmt"
	"strings"

	taskv1 "admin/api/gen/task/v1"
)

type TaskLogWriter interface {
	WriteTaskLog(ctx context.Context, data *taskv1.TaskLog) error
}

type ExecuteRequest struct {
	Task  *taskv1.Task
	Input string
}

type ValidationRequest struct {
	Task *taskv1.Task
	Raw  string
}

type Executor interface {
	InvokeTarget() string
	Validate(context.Context, ValidationRequest) error
	Execute(context.Context, ExecuteRequest) (string, error)
}

type Registry struct {
	executors map[string]Executor
}

func NewRegistry(executors ...Executor) (*Registry, error) {
	registry := &Registry{executors: make(map[string]Executor, len(executors))}
	for _, executor := range executors {
		if executor == nil {
			continue
		}
		key := strings.TrimSpace(executor.InvokeTarget())
		if key == "" {
			return nil, fmt.Errorf("task executor invoke target is required")
		}
		if _, exists := registry.executors[key]; exists {
			return nil, fmt.Errorf("duplicate task executor: %s", key)
		}
		registry.executors[key] = executor
	}
	return registry, nil
}

func MustNewRegistry(executors ...Executor) *Registry {
	registry, err := NewRegistry(executors...)
	if err != nil {
		panic(err)
	}
	return registry
}

func (r *Registry) Get(target string) (Executor, bool) {
	if r == nil {
		return nil, false
	}
	executor, ok := r.executors[strings.TrimSpace(target)]
	return executor, ok
}

func (r *Registry) Validate(ctx context.Context, taskItem *taskv1.Task, raw string) error {
	executor, ok := r.Get(taskInvokeTarget(taskItem))
	if !ok {
		return fmt.Errorf("unsupported task invoke target: %s", taskInvokeTarget(taskItem))
	}
	return executor.Validate(ctx, ValidationRequest{
		Task: taskItem,
		Raw:  raw,
	})
}

func (r *Registry) Execute(ctx context.Context, taskItem *taskv1.Task, raw string) (string, error) {
	executor, ok := r.Get(taskInvokeTarget(taskItem))
	if !ok {
		return "", fmt.Errorf("unsupported task invoke target: %s", taskInvokeTarget(taskItem))
	}
	return executor.Execute(ctx, ExecuteRequest{
		Task:  taskItem,
		Input: raw,
	})
}

func taskInvokeTarget(taskItem *taskv1.Task) string {
	if taskItem == nil {
		return ""
	}
	return strings.TrimSpace(taskItem.GetInvokeTarget())
}
