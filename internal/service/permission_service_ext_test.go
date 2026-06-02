package service

import (
	"context"
	"strings"
	"testing"

	permissionv1 "admin/api/gen/permission/v1"
	resourcev1 "admin/api/gen/resource/v1"

	crudviewer "github.com/chnxq/x-crud/viewer"
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

func TestPermissionServiceSyncPermissions_RejectsTenantContext(t *testing.T) {
	svc := &PermissionService{}
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
	svc := &PermissionService{}
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
		ModuleDescription: stringPtr("用户管理服务"),
		Description:       stringPtr("查询用户"),
		Status:            resourcev1.Api_ON.Enum(),
	}
	exportAPI := &resourcev1.Api{
		Method:            stringPtr("GET"),
		Path:              stringPtr("/admin/v1/users/export"),
		Module:            stringPtr("user"),
		ModuleDescription: stringPtr("用户管理"),
		Description:       stringPtr("导出用户"),
		Status:            resourcev1.Api_ON.Enum(),
	}
	miscAPI := &resourcev1.Api{
		Method:      stringPtr("POST"),
		Path:        stringPtr("/"),
		Description: stringPtr("未知接口"),
		Status:      resourcev1.Api_ON.Enum(),
	}

	module, name := serviceGroupIdentityFromAPI(userAPI)
	if module != "permission:view:service:user" || name != "用户管理" {
		t.Fatalf("unexpected user group identity: module=%q name=%q", module, name)
	}

	module, name = serviceGroupIdentityFromAPI(exportAPI)
	if module != permissionGroupModuleExport || name != "数据导出" {
		t.Fatalf("unexpected export group identity: module=%q name=%q", module, name)
	}

	module, name = serviceGroupIdentityFromAPI(miscAPI)
	if module != permissionGroupModuleMisc || name != "未分类" {
		t.Fatalf("unexpected misc group identity: module=%q name=%q", module, name)
	}
}

func TestExportPermissionNameFromAPI_RewritesQueryPrefix(t *testing.T) {
	api := &resourcev1.Api{
		Method:      stringPtr("GET"),
		Path:        stringPtr("/admin/v1/users"),
		Description: stringPtr("查询用户"),
		Status:      resourcev1.Api_ON.Enum(),
	}
	if got := exportPermissionNameFromAPI(api); got != "导出用户" {
		t.Fatalf("expected 导出用户, got %q", got)
	}

	api.Description = stringPtr("下载用户")
	if got := exportPermissionNameFromAPI(api); got != "导出下载用户" {
		t.Fatalf("expected 导出下载用户, got %q", got)
	}
}

func TestResolveMenuTitleKey_UsesReadableDisplayName(t *testing.T) {
	menu := &resourcev1.Menu{
		Meta: &resourcev1.MenuMeta{
			Title: stringPtr("menu.system.api"),
		},
	}
	if got := displayMenuTitle(menu); got != "API管理" {
		t.Fatalf("expected API管理, got %q", got)
	}
}

func TestFeaturePermissionDisplayName_UsesMenuReadableName(t *testing.T) {
	if got := featurePermissionDisplayName("API管理", "view"); got != "[菜单]API管理" {
		t.Fatalf("expected [菜单]API管理, got %q", got)
	}
	if got := featurePermissionDisplayName("API管理", "edit"); got != "更新API" {
		t.Fatalf("expected 更新API, got %q", got)
	}
	if got := featurePermissionDisplayName("API管理", "create"); got != "新增API" {
		t.Fatalf("expected 新增API, got %q", got)
	}
	if got := featurePermissionDisplayName("用户管理", "edit"); got != "更新用户" {
		t.Fatalf("expected 更新用户, got %q", got)
	}
}

func TestExplicitFeaturePermissionDisplayName(t *testing.T) {
	cases := map[string]string{
		"permissions:view":              "[菜单]权限点管理",
		"permissions:create":            "新增权限点",
		"permissions:edit":              "更新权限点",
		"permissions:delete":            "删除权限点",
		"permissions:export":            "导出权限点",
		"permissions:sync:perms:create": "同步权限点",
		"permission:groups:create":      "新增权限组",
		"permission:groups:edit":        "更新权限组",
		"permission:groups:delete":      "删除权限组",
	}
	for code, expected := range cases {
		if got := explicitFeaturePermissionDisplayName("权限点管理", code); got != expected {
			t.Fatalf("code %q expected %q, got %q", code, expected, got)
		}
	}
}

func TestPermissionCodeNamespace(t *testing.T) {
	if got := permissionCodeNamespace("permissions:sync:perms:create"); got != "permissions" {
		t.Fatalf("expected permissions, got %q", got)
	}
	if got := permissionCodeNamespace("dict:labels:view"); got != "dict" {
		t.Fatalf("expected dict, got %q", got)
	}
	if got := permissionCodeNamespace(""); got != "" {
		t.Fatalf("expected empty namespace, got %q", got)
	}
}

func TestServicePermissionCode_IncludesServiceModule(t *testing.T) {
	got := servicePermissionCode("permission:view:service:dictlabelservice", "dict:labels:view")
	if got != "service:dictlabelservice:dict:labels:view" {
		t.Fatalf("unexpected service permission code: %q", got)
	}

	other := servicePermissionCode("permission:view:service:languageservice", "dict:labels:view")
	if other != "service:languageservice:dict:labels:view" {
		t.Fatalf("unexpected service permission code for second module: %q", other)
	}
	if got == other {
		t.Fatalf("expected different service permission codes for different services")
	}
}

func TestExplicitFeaturePermissionDisplayName_GroupAndPermissionSplit(t *testing.T) {
	if got := explicitFeaturePermissionDisplayName("权限点管理", "permissions:create"); got != "新增权限点" {
		t.Fatalf("expected 新增权限点, got %q", got)
	}
	if got := explicitFeaturePermissionDisplayName("权限点管理", "permission:groups:create"); got != "新增权限组" {
		t.Fatalf("expected 新增权限组, got %q", got)
	}
}
