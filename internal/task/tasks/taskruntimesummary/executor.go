package taskruntimesummary

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	taskruntime "admin/internal/task/runtime"
)

type Input struct {
	TenantScope string `json:"tenantScope"`
}

type Result struct {
	TenantScope string         `json:"tenantScope"`
	Total       int            `json:"total"`
	ByStatus    map[string]int `json:"byStatus"`
	ByType      map[string]int `json:"byType"`
	Tasks       []ResultItem   `json:"tasks,omitempty"`
}

type ResultItem struct {
	ID     uint64 `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"`
	Group  uint64 `json:"groupId"`
}

type Executor struct {
	provider Provider
}

func NewExecutor(provider Provider) *Executor {
	return &Executor{provider: provider}
}

func (e *Executor) InvokeTarget() string {
	return TaskRuntimeSummaryInvokeTarget
}

func (e *Executor) Validate(_ context.Context, req taskruntime.ValidationRequest) error {
	_, err := ParseInput(req.Raw, req.Task)
	return err
}

func (e *Executor) Execute(ctx context.Context, req taskruntime.ExecuteRequest) (string, error) {
	if e.provider == nil {
		return "", fmt.Errorf("task summary provider is not configured")
	}
	input, err := ParseInput(req.Input, req.Task)
	if err != nil {
		return "", err
	}

	var tenantID *uint32
	if input.TenantScope != "global" && req.Task != nil && req.Task.TenantId != nil && req.Task.GetTenantId() > 0 {
		value := req.Task.GetTenantId()
		tenantID = &value
	}

	items, err := e.provider.ListTasksForRuntime(ctx, tenantID)
	if err != nil {
		return "", err
	}

	result := Result{
		TenantScope: input.TenantScope,
		ByStatus:    map[string]int{},
		ByType:      map[string]int{},
		Tasks:       make([]ResultItem, 0, len(items)),
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		status := normalizeLabel(item.GetStatus().String(), "UNKNOWN")
		taskType := normalizeLabel(item.GetTaskType().String(), "UNKNOWN")
		result.Total++
		result.ByStatus[status]++
		result.ByType[taskType]++
		result.Tasks = append(result.Tasks, ResultItem{
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

func ParseInput(raw string, taskItem interface{ GetArgs() string }) (*Input, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" && taskItem != nil {
		raw = strings.TrimSpace(taskItem.GetArgs())
	}
	if raw == "" {
		return &Input{TenantScope: "current"}, nil
	}

	var payload Input
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

func normalizeLabel(value string, fallback string) string {
	value = strings.TrimSpace(strings.ToUpper(value))
	if value == "" {
		return fallback
	}
	return value
}
