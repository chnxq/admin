package task

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	taskv1 "admin/api/gen/task/v1"
)

const TaskRuntimeSummaryInvokeTarget = "system:task:runtime-summary"

type TaskRuntimeSummaryInput struct {
	TenantScope string `json:"tenantScope"`
}

type TaskRuntimeSummaryResult struct {
	TenantScope string                         `json:"tenantScope"`
	Total       int                            `json:"total"`
	ByStatus    map[string]int                 `json:"byStatus"`
	ByType      map[string]int                 `json:"byType"`
	Tasks       []TaskRuntimeSummaryResultItem `json:"tasks,omitempty"`
}

type TaskRuntimeSummaryResultItem struct {
	ID     uint64 `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"`
	Group  uint64 `json:"groupId"`
}

type TaskRuntimeSummaryExecutor struct {
	taskSummaryProvider TaskSummaryProvider
}

func NewTaskRuntimeSummaryExecutor(deps RuntimeDeps) *TaskRuntimeSummaryExecutor {
	return &TaskRuntimeSummaryExecutor{
		taskSummaryProvider: deps.TaskSummaryProvider,
	}
}

func (e *TaskRuntimeSummaryExecutor) InvokeTarget() string {
	return TaskRuntimeSummaryInvokeTarget
}

func (e *TaskRuntimeSummaryExecutor) Validate(_ context.Context, req ValidationRequest) error {
	_, err := parseTaskRuntimeSummaryInput(req.Raw, req.Task)
	return err
}

func (e *TaskRuntimeSummaryExecutor) Execute(ctx context.Context, req ExecuteRequest) (string, error) {
	if e.taskSummaryProvider == nil {
		return "", fmt.Errorf("task summary provider is not configured")
	}
	input, err := parseTaskRuntimeSummaryInput(req.Input, req.Task)
	if err != nil {
		return "", err
	}

	var tenantID *uint32
	if input.TenantScope != "global" && req.Task != nil && req.Task.TenantId != nil && req.Task.GetTenantId() > 0 {
		value := req.Task.GetTenantId()
		tenantID = &value
	}

	items, err := e.taskSummaryProvider.ListTasksForRuntime(ctx, tenantID)
	if err != nil {
		return "", err
	}

	result := TaskRuntimeSummaryResult{
		TenantScope: input.TenantScope,
		ByStatus:    map[string]int{},
		ByType:      map[string]int{},
		Tasks:       make([]TaskRuntimeSummaryResultItem, 0, len(items)),
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		status := normalizeSummaryLabel(item.GetStatus().String(), "UNKNOWN")
		taskType := normalizeSummaryLabel(item.GetTaskType().String(), "UNKNOWN")
		result.Total++
		result.ByStatus[status]++
		result.ByType[taskType]++
		result.Tasks = append(result.Tasks, TaskRuntimeSummaryResultItem{
			ID:     item.GetId(),
			Name:   strings.TrimSpace(item.GetTaskName()),
			Status: status,
			Type:   taskType,
			Group:  item.GetGroupId(),
		})
	}

	sort.Slice(result.Tasks, func(i, j int) bool {
		return result.Tasks[i].ID < result.Tasks[j].ID
	})

	raw, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func parseTaskRuntimeSummaryInput(raw string, taskItem *taskv1.Task) (*TaskRuntimeSummaryInput, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" && taskItem != nil {
		raw = strings.TrimSpace(taskItem.GetArgs())
	}

	if raw == "" {
		return &TaskRuntimeSummaryInput{TenantScope: "current"}, nil
	}

	var payload TaskRuntimeSummaryInput
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("invalid runtime summary args: %w", err)
	}

	scope := strings.ToLower(strings.TrimSpace(payload.TenantScope))
	switch scope {
	case "", "current":
		payload.TenantScope = "current"
	case "global":
		payload.TenantScope = "global"
	default:
		return nil, fmt.Errorf("unsupported tenantScope: %s", payload.TenantScope)
	}
	return &payload, nil
}

func normalizeSummaryLabel(value string, fallback string) string {
	value = strings.TrimSpace(strings.ToUpper(value))
	if value == "" {
		return fallback
	}
	return value
}
