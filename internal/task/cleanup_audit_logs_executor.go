package task

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type CleanupAuditLogInput struct {
	ExpireHours uint32   `json:"expireHours"`
	Targets     []string `json:"targets"`
}

type CleanupAuditLogResult struct {
	ExpireHours       uint32 `json:"expireHours"`
	DeletedAPI        int    `json:"deletedApi"`
	DeletedLogin      int    `json:"deletedLogin"`
	DeletedPermission int    `json:"deletedPermission"`
	TotalDeleted      int    `json:"totalDeleted"`
}

type CleanupAuditLogsExecutor struct {
	apiAuditLogCleaner        ApiAuditLogCleaner
	loginAuditLogCleaner      LoginAuditLogCleaner
	permissionAuditLogCleaner PermissionAuditLogCleaner
}

func NewCleanupAuditLogsExecutor(deps RuntimeDeps) *CleanupAuditLogsExecutor {
	return &CleanupAuditLogsExecutor{
		apiAuditLogCleaner:        deps.ApiAuditLogCleaner,
		loginAuditLogCleaner:      deps.LoginAuditLogCleaner,
		permissionAuditLogCleaner: deps.PermissionAuditLogCleaner,
	}
}

func (e *CleanupAuditLogsExecutor) InvokeTarget() string {
	return CleanupAuditLogsInvokeTarget
}

func (e *CleanupAuditLogsExecutor) Validate(_ context.Context, req ValidationRequest) error {
	raw := strings.TrimSpace(req.Raw)
	if raw == "" && req.Task != nil {
		raw = strings.TrimSpace(req.Task.GetArgs())
	}
	if raw == "" {
		return fmt.Errorf("task args are required")
	}

	payload, err := ParseCleanupAuditLogInput(raw)
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
		switch normalizeCleanupTarget(target) {
		case "api", "login", "permission":
		default:
			return fmt.Errorf("unsupported cleanup target: %s", target)
		}
	}
	return nil
}

func (e *CleanupAuditLogsExecutor) Execute(ctx context.Context, req ExecuteRequest) (string, error) {
	taskItem := req.Task
	if taskItem == nil {
		return "", fmt.Errorf("task not found")
	}
	raw := strings.TrimSpace(req.Input)
	if raw == "" {
		raw = strings.TrimSpace(taskItem.GetArgs())
	}

	payload, err := ParseCleanupAuditLogInput(raw)
	if err != nil {
		return "", err
	}
	if err := e.Validate(ctx, ValidationRequest{Task: taskItem, Raw: raw}); err != nil {
		return "", err
	}

	before := time.Now().Add(-time.Duration(payload.ExpireHours) * time.Hour)
	result := CleanupAuditLogResult{
		ExpireHours: payload.ExpireHours,
	}

	var tenantID *uint32
	if taskItem.TenantId != nil && taskItem.GetTenantId() > 0 {
		value := taskItem.GetTenantId()
		tenantID = &value
	}

	for _, target := range payload.Targets {
		switch normalizeCleanupTarget(target) {
		case "api":
			if e.apiAuditLogCleaner == nil {
				return "", fmt.Errorf("api audit log cleaner is not configured")
			}
			affected, err := e.apiAuditLogCleaner.CleanupApiAuditLogsBefore(ctx, tenantID, before)
			if err != nil {
				return "", err
			}
			result.DeletedAPI = affected
		case "login":
			if e.loginAuditLogCleaner == nil {
				return "", fmt.Errorf("login audit log cleaner is not configured")
			}
			affected, err := e.loginAuditLogCleaner.CleanupLoginAuditLogsBefore(ctx, tenantID, before)
			if err != nil {
				return "", err
			}
			result.DeletedLogin = affected
		case "permission":
			if e.permissionAuditLogCleaner == nil {
				return "", fmt.Errorf("permission audit log cleaner is not configured")
			}
			affected, err := e.permissionAuditLogCleaner.CleanupPermissionAuditLogsBefore(ctx, tenantID, before)
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

func ParseCleanupAuditLogInput(raw string) (*CleanupAuditLogInput, error) {
	var payload CleanupAuditLogInput
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return nil, fmt.Errorf("invalid cleanup args: %w", err)
	}
	return &payload, nil
}

func normalizeCleanupTarget(target string) string {
	return strings.ToLower(strings.TrimSpace(target))
}
