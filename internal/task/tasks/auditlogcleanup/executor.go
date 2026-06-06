package auditlogcleanup

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	taskruntime "admin/internal/task/runtime"
)

type Input struct {
	ExpireHours uint32   `json:"expireHours"`
	Targets     []string `json:"targets"`
}

type Result struct {
	ExpireHours       uint32 `json:"expireHours"`
	DeletedAPI        int    `json:"deletedApi"`
	DeletedLogin      int    `json:"deletedLogin"`
	DeletedPermission int    `json:"deletedPermission"`
	TotalDeleted      int    `json:"totalDeleted"`
}

type Executor struct {
	store Store
}

func NewExecutor(store Store) *Executor {
	return &Executor{store: store}
}

func (e *Executor) InvokeTarget() string {
	return CleanupAuditLogsInvokeTarget
}

func (e *Executor) Validate(_ context.Context, req taskruntime.ValidationRequest) error {
	raw := strings.TrimSpace(req.Raw)
	if raw == "" && req.Task != nil {
		raw = strings.TrimSpace(req.Task.GetArgs())
	}
	if raw == "" {
		return fmt.Errorf("task args are required")
	}

	payload, err := ParseInput(raw)
	if err != nil {
		return err
	}
	if payload.ExpireHours == 0 {
		return fmt.Errorf("expireHours must be greater than 0")
	}
	if len(payload.Targets) == 0 {
		return fmt.Errorf("targets must not be empty")
	}
	for _, target := range payload.Targets {
		switch normalizeTarget(target) {
		case "api", "login", "permission":
		default:
			return fmt.Errorf("unsupported cleanup target: %s", target)
		}
	}
	return nil
}

func (e *Executor) Execute(ctx context.Context, req taskruntime.ExecuteRequest) (string, error) {
	taskItem := req.Task
	if taskItem == nil {
		return "", fmt.Errorf("task not found")
	}
	raw := strings.TrimSpace(req.Input)
	if raw == "" {
		raw = strings.TrimSpace(taskItem.GetArgs())
	}

	payload, err := ParseInput(raw)
	if err != nil {
		return "", err
	}
	if err := e.Validate(ctx, taskruntime.ValidationRequest{Task: taskItem, Raw: raw}); err != nil {
		return "", err
	}

	before := time.Now().Add(-time.Duration(payload.ExpireHours) * time.Hour)
	result := Result{ExpireHours: payload.ExpireHours}

	var tenantID *uint32
	if taskItem.TenantId != nil && taskItem.GetTenantId() > 0 {
		value := taskItem.GetTenantId()
		tenantID = &value
	}

	for _, target := range payload.Targets {
		switch normalizeTarget(target) {
		case "api":
			if e.store == nil {
				return "", fmt.Errorf("audit log cleanup store is not configured")
			}
			affected, err := e.store.CleanupAPIBefore(ctx, tenantID, before)
			if err != nil {
				return "", err
			}
			result.DeletedAPI = affected
		case "login":
			if e.store == nil {
				return "", fmt.Errorf("audit log cleanup store is not configured")
			}
			affected, err := e.store.CleanupLoginBefore(ctx, tenantID, before)
			if err != nil {
				return "", err
			}
			result.DeletedLogin = affected
		case "permission":
			if e.store == nil {
				return "", fmt.Errorf("audit log cleanup store is not configured")
			}
			affected, err := e.store.CleanupPermissionBefore(ctx, tenantID, before)
			if err != nil {
				return "", err
			}
			result.DeletedPermission = affected
		}
	}

	result.TotalDeleted = result.DeletedAPI + result.DeletedLogin + result.DeletedPermission
	rawResult, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(rawResult), nil
}

func ParseInput(raw string) (*Input, error) {
	var payload Input
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return nil, fmt.Errorf("invalid cleanup args: %w", err)
	}
	return &payload, nil
}

func normalizeTarget(target string) string {
	return strings.ToLower(strings.TrimSpace(target))
}
