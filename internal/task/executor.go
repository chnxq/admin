package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	taskv1 "admin/api/gen/task/v1"
)

const CleanupAuditLogsInvokeTarget = "system:cleanup:audit-logs"

type ApiAuditLogCleaner interface {
	CleanupApiAuditLogsBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error)
}

type LoginAuditLogCleaner interface {
	CleanupLoginAuditLogsBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error)
}

type PermissionAuditLogCleaner interface {
	CleanupPermissionAuditLogsBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error)
}

type RuntimeDeps struct {
	ApiAuditLogCleaner        ApiAuditLogCleaner
	LoginAuditLogCleaner      LoginAuditLogCleaner
	PermissionAuditLogCleaner PermissionAuditLogCleaner
}

type ExecuteRequest struct {
	Task  *taskv1.Task
	Input string
}

type ValidationRequest struct {
	Task *taskv1.Task
	Raw  string
}

// Executor defines a task invoke-target handler.
// Validate should reject malformed input before scheduling or execution.
// Execute should return a compact result string suitable for persistence into task_log.output.
type Executor interface {
	InvokeTarget() string
	Validate(context.Context, ValidationRequest) error
	Execute(context.Context, ExecuteRequest) (string, error)
}

// Registry maps invoke_target to concrete executor implementations.
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

// Get returns the executor for the given invoke target.
func (r *Registry) Get(target string) (Executor, bool) {
	if r == nil {
		return nil, false
	}
	executor, ok := r.executors[strings.TrimSpace(target)]
	return executor, ok
}

// Validate delegates task argument validation to the matched executor.
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

// Execute delegates task execution to the matched executor.
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
