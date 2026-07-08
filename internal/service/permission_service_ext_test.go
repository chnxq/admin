package service

import (
	"context"
	"strings"
	"testing"

	permissionv1 "admin/api/gen/permission/v1"
	resourcev1 "admin/api/gen/resource/v1"

	crudviewer "github.com/chnxq/x-crud/viewer"
	"github.com/chnxq/xkitmod/log"
)

type permissionServiceTestViewer struct {
	platform bool
	tenant   bool
	tenantID uint64
}

func (v permissionServiceTestViewer) UserID() uint64                    { return 1 }
func (v permissionServiceTestViewer) TenantID() uint64                  { return v.tenantID }
func (v permissionServiceTestViewer) OrgUnitID() uint64                 { return 0 }
func (v permissionServiceTestViewer) Permissions() []string             { return []string{"*"} }
func (v permissionServiceTestViewer) Roles() []string                   { return []string{"system"} }
func (v permissionServiceTestViewer) DataScope() []crudviewer.DataScope { return nil }
func (v permissionServiceTestViewer) TraceID() string                   { return "" }
func (v permissionServiceTestViewer) HasPermission(string, string) bool { return true }
func (v permissionServiceTestViewer) IsPlatformContext() bool           { return v.platform }
func (v permissionServiceTestViewer) IsTenantContext() bool             { return v.tenant }
func (v permissionServiceTestViewer) IsSystemContext() bool             { return v.platform && !v.tenant }
func (v permissionServiceTestViewer) ShouldAudit() bool                 { return false }

func TestAPIActionFromMethod_RecognizesNonCRUDActions(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   string
	}{
		{method: "POST", path: "/admin/v1/tasks/{id}:start", want: "start"},
		{method: "POST", path: "/admin/v1/tasks/{id}:stop", want: "stop"},
		{method: "POST", path: "/admin/v1/tasks/{id}:run-once", want: "run-once"},
		{method: "POST", path: "/admin/v1/task-groups/{id}:run-once", want: "run-once"},
		{method: "POST", path: "/admin/v1/internal-message/send", want: "send"},
		{method: "POST", path: "/admin/v1/internal-message/revoke", want: "revoke"},
		{method: "POST", path: "/admin/v1/users", want: "create"},
		{method: "PUT", path: "/admin/v1/users/{id}", want: "edit"},
	}

	for _, tc := range cases {
		if got := apiActionFromMethod(tc.method, tc.path); got != tc.want {
			t.Fatalf("method=%q path=%q expected %q, got %q", tc.method, tc.path, tc.want, got)
		}
	}
}

func TestAPIPermissionCode_UsesNormalizedPermissionSyntax(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   string
	}{
		{method: "POST", path: "/admin/v1/tasks/{id}:start", want: "tasks:start"},
		{method: "POST", path: "/admin/v1/tasks/{id}:stop", want: "tasks:stop"},
		{method: "POST", path: "/admin/v1/tasks/{id}:run-once", want: "tasks:run-once"},
		{method: "POST", path: "/admin/v1/task-groups/{id}:start", want: "task-groups:start"},
		{method: "POST", path: "/admin/v1/task-groups/{id}:stop", want: "task-groups:stop"},
		{method: "POST", path: "/admin/v1/task-groups/{id}:run-once", want: "task-groups:run-once"},
		{method: "POST", path: "/admin/v1/internal-message/send", want: "internal-message:send"},
		{method: "POST", path: "/admin/v1/internal-message/revoke", want: "internal-message:revoke"},
		{method: "GET", path: "/admin/v1/internal-message/messages", want: "internal-message/messages:view"},
		{method: "POST", path: "/admin/v1/internal-message/messages", want: "internal-message/messages:create"},
		{method: "GET", path: "/admin/v1/dict/categories", want: "dict/categories:view"},
		{method: "POST", path: "/admin/v1/dict/labels", want: "dict/labels:create"},
		{method: "GET", path: "/admin/v1/org-units", want: "org-units:view"},
		{method: "GET", path: "/admin/v1/task-logs", want: "task-logs:view"},
	}

	for _, tc := range cases {
		if got := apiPermissionCode(tc.method, tc.path); got != tc.want {
			t.Fatalf("method=%q path=%q expected %q, got %q", tc.method, tc.path, tc.want, got)
		}
	}
}

func TestPermissionCodeHelpers_UseResourceActionSplit(t *testing.T) {
	if got := codeAction("tasks:run-once"); got != "run-once" {
		t.Fatalf("expected run-once, got %q", got)
	}
	if got := permissionCodeBase("internal-message/messages:view"); got != "internal-message/messages" {
		t.Fatalf("expected internal-message/messages, got %q", got)
	}
	if got := permissionCodeNamespace("internal-message/messages:view"); got != "internal-message" {
		t.Fatalf("expected internal-message namespace, got %q", got)
	}
	if got := permissionCodeNamespace("dict-labels:view"); got != "dict-labels" {
		t.Fatalf("expected dict-labels namespace, got %q", got)
	}
	if got := normalizePermissionGroupModule("internal-message"); got != "internalmessage" {
		t.Fatalf("expected internalmessage group module, got %q", got)
	}
}

func TestAPIResourceFromPath_UsesExplicitSecondaryResourceRules(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{path: "/admin/v1/internal-message/messages", want: "internal-message/messages"},
		{path: "/admin/v1/internal-message/send", want: "internal-message"},
		{path: "/admin/v1/dict/categories", want: "dict/categories"},
		{path: "/admin/v1/dict/labels", want: "dict/labels"},
		{path: "/admin/v1/task-groups/{id}:run-once", want: "task-groups"},
		{path: "/admin/v1/permissions/sync:perms", want: "permissions"},
	}

	for _, tc := range cases {
		if got := apiResourceFromPath(tc.path); got != tc.want {
			t.Fatalf("path=%q expected resource %q, got %q", tc.path, tc.want, got)
		}
	}
}

func TestServicePermissionCode_IncludesServiceModule(t *testing.T) {
	got := servicePermissionCode("permission:view:service:dictlabelservice", "dict-labels:view")
	if got != "service:dictlabelservice:dict-labels:view" {
		t.Fatalf("unexpected service permission code: %q", got)
	}

	other := servicePermissionCode("permission:view:service:languageservice", "dict-labels:view")
	if other != "service:languageservice:dict-labels:view" {
		t.Fatalf("unexpected service permission code for second module: %q", other)
	}
	if got == other {
		t.Fatalf("expected different service permission codes for different services")
	}
}

func TestPermissionServiceSyncPermissions_RejectsTenantContext(t *testing.T) {
	svc := &PermissionService{
		log: log.NewHelper(log.NewStdLogger(permissionTestingWriter{t: t})),
	}
	ctx := crudviewer.WithContext(context.Background(), permissionServiceTestViewer{
		platform: true,
		tenant:   true,
		tenantID: 101,
	})

	err := svc.syncPermissions(ctx)
	if !permissionv1.IsForbidden(err) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}

func TestPermissionServiceSyncPermissions_AllowsPlatformContextPastTenantGate(t *testing.T) {
	svc := &PermissionService{
		log: log.NewHelper(log.NewStdLogger(permissionTestingWriter{t: t})),
	}
	ctx := crudviewer.WithContext(context.Background(), permissionServiceTestViewer{
		platform: true,
		tenant:   false,
		tenantID: 0,
	})

	err := svc.syncPermissions(ctx)
	if err == nil {
		t.Fatalf("expected incomplete dependency error")
	}
	if permissionv1.IsForbidden(err) {
		t.Fatalf("expected platform context to pass tenant gate, got forbidden: %v", err)
	}
	if !strings.Contains(err.Error(), "dependencies are incomplete") {
		t.Fatalf("expected dependency error, got %v", err)
	}
}

func TestServiceGroupIdentityFromAPI_AssignsExpectedGroups(t *testing.T) {
	userAPI := &resourcev1.Api{
		Method:            stringPtr("GET"),
		Path:              stringPtr("/admin/v1/users"),
		Module:            stringPtr("user"),
		ModuleDescription: stringPtr("User Management Service"),
		Description:       stringPtr("List users"),
		Status:            resourcev1.Api_ON.Enum(),
	}
	exportAPI := &resourcev1.Api{
		Method:            stringPtr("GET"),
		Path:              stringPtr("/admin/v1/users/export"),
		Module:            stringPtr("user"),
		ModuleDescription: stringPtr("User Management"),
		Description:       stringPtr("Export users"),
		Status:            resourcev1.Api_ON.Enum(),
	}
	miscAPI := &resourcev1.Api{
		Method:      stringPtr("POST"),
		Path:        stringPtr("/"),
		Description: stringPtr("Unknown API"),
		Status:      resourcev1.Api_ON.Enum(),
	}

	module, _ := serviceGroupIdentityFromAPI(userAPI)
	if module != "permission:view:service:user" {
		t.Fatalf("unexpected user group identity: module=%q", module)
	}

	module, _ = serviceGroupIdentityFromAPI(exportAPI)
	if module != permissionGroupModuleExport {
		t.Fatalf("unexpected export group identity: module=%q", module)
	}

	module, _ = serviceGroupIdentityFromAPI(miscAPI)
	if module != permissionGroupModuleMisc {
		t.Fatalf("unexpected misc group identity: module=%q", module)
	}
}
