package service

import (
	"testing"

	resourcev1 "admin/api/gen/resource/v1"
)

func TestCollectFeaturePermissions_UsesFirstExplicitViewAsSingleMenuEntry(t *testing.T) {
	menus := []*resourcev1.Menu{
		{
			Id:       uint32Ptr(30),
			Name:     stringPtr("SystemDeviceCenter"),
			Path:     stringPtr("/xdev/device-center"),
			Type:     resourcev1.Menu_MENU.Enum(),
			Status:   resourcev1.Menu_ON.Enum(),
			ParentId: uint32Ptr(1),
			Meta: &resourcev1.MenuMeta{
				Title: stringPtr("Device Center"),
				Authority: []string{
					"xdev:device-group-device:view",
					"xdev:device-group-device:create",
					"xdev:device-group-device:edit",
					"xdev:device-group-device:delete",
					"xdev:device-group-device:export",
					"xdev:device-group-org-unit:view",
					"xdev:device-group-org-unit:create",
					"xdev:device-group-org-unit:edit",
					"xdev:device-group-org-unit:delete",
					"xdev:device-group-user:view",
					"xdev:device-group-user:create",
					"xdev:device-group-user:edit",
					"xdev:device-group-user:delete",
				},
			},
		},
	}

	apis := []*resourcev1.Api{
		{
			Id:          uint32Ptr(180),
			Method:      stringPtr("GET"),
			Path:        stringPtr("/xdev/v1/device-group-devices"),
			Description: stringPtr("List device center devices"),
			Status:      resourcev1.Api_ON.Enum(),
		},
		{
			Id:          uint32Ptr(186),
			Method:      stringPtr("GET"),
			Path:        stringPtr("/xdev/v1/device-group-org-units"),
			Description: stringPtr("List device center org units"),
			Status:      resourcev1.Api_ON.Enum(),
		},
		{
			Id:          uint32Ptr(192),
			Method:      stringPtr("GET"),
			Path:        stringPtr("/xdev/v1/device-group-users"),
			Description: stringPtr("List device center users"),
			Status:      resourcev1.Api_ON.Enum(),
		},
		{
			Id:          uint32Ptr(193),
			Method:      stringPtr("POST"),
			Path:        stringPtr("/xdev/v1/device-group-users"),
			Description: stringPtr("Create device center user"),
			Status:      resourcev1.Api_ON.Enum(),
		},
	}

	items := collectFeaturePermissions(menus, apis)

	var primary *desiredPermission
	for i := range items {
		if items[i].code == "xdev:device-group-device:view" {
			primary = &items[i]
		}
		switch items[i].code {
		case "xdev:device-group-org-unit:view",
			"xdev:device-group-org-unit:create",
			"xdev:device-group-org-unit:edit",
			"xdev:device-group-org-unit:delete",
			"xdev:device-group-user:view",
			"xdev:device-group-user:create",
			"xdev:device-group-user:edit",
			"xdev:device-group-user:delete":
			t.Fatalf("secondary explicit code should not create a standalone permission: %q", items[i].code)
		}
	}
	if primary == nil {
		t.Fatalf("expected primary explicit view permission to be generated")
	}
	if len(primary.menuIDs) != 1 || primary.menuIDs[0] != 30 {
		t.Fatalf("expected primary permission to bind menu 30, got %#v", primary.menuIDs)
	}
	if len(primary.apiIDs) != 4 || primary.apiIDs[0] != 180 || primary.apiIDs[1] != 186 || primary.apiIDs[2] != 192 || primary.apiIDs[3] != 193 {
		t.Fatalf("expected primary permission to keep its own API bindings and absorb non-primary explicit API bindings, got %#v", primary.apiIDs)
	}
}

func TestCollectFeaturePermissions_CompositeCenterKeepsPrimaryActions(t *testing.T) {
	menus := []*resourcev1.Menu{
		{
			Id:     uint32Ptr(31),
			Name:   stringPtr("XdevParameterCenter"),
			Path:   stringPtr("/xdev/parameter-center"),
			Type:   resourcev1.Menu_MENU.Enum(),
			Status: resourcev1.Menu_ON.Enum(),
			Meta: &resourcev1.MenuMeta{
				Title: stringPtr("Parameter Center"),
				Authority: []string{
					"xdev:device-parameter-group:view",
					"xdev:device-parameter-group:create",
					"xdev:device-parameter-group:edit",
					"xdev:device-parameter-group:delete",
					"xdev:device-parameter-group:export",
					"xdev:device-parameter-item:view",
					"xdev:device-parameter-item:create",
					"xdev:device-parameter-item:edit",
					"xdev:device-parameter-item:delete",
				},
			},
		},
		{
			Id:     uint32Ptr(32),
			Name:   stringPtr("XdevDeviceParameterGroup"),
			Path:   stringPtr("/xdev/device-parameter-group"),
			Type:   resourcev1.Menu_MENU.Enum(),
			Status: resourcev1.Menu_ON.Enum(),
			Meta: &resourcev1.MenuMeta{
				Title: stringPtr("Device Parameter Group"),
				Authority: []string{
					"xdev:device-parameter-group:view",
					"xdev:device-parameter-group:create",
					"xdev:device-parameter-group:edit",
					"xdev:device-parameter-group:delete",
					"xdev:device-parameter-group:export",
				},
			},
		},
	}

	apis := []*resourcev1.Api{
		{Id: uint32Ptr(1), Method: stringPtr("GET"), Path: stringPtr("/xdev/v1/device-parameter-groups"), Status: resourcev1.Api_ON.Enum()},
		{Id: uint32Ptr(2), Method: stringPtr("POST"), Path: stringPtr("/xdev/v1/device-parameter-groups"), Status: resourcev1.Api_ON.Enum()},
		{Id: uint32Ptr(3), Method: stringPtr("PUT"), Path: stringPtr("/xdev/v1/device-parameter-groups/{id}"), Status: resourcev1.Api_ON.Enum()},
		{Id: uint32Ptr(4), Method: stringPtr("DELETE"), Path: stringPtr("/xdev/v1/device-parameter-groups/{id}"), Status: resourcev1.Api_ON.Enum()},
		{Id: uint32Ptr(5), Method: stringPtr("GET"), Path: stringPtr("/xdev/v1/device-parameter-items"), Status: resourcev1.Api_ON.Enum()},
		{Id: uint32Ptr(6), Method: stringPtr("POST"), Path: stringPtr("/xdev/v1/device-parameter-items"), Status: resourcev1.Api_ON.Enum()},
	}

	items := collectFeaturePermissions(menus, apis)
	byCode := make(map[string]desiredPermission, len(items))
	for _, item := range items {
		byCode[item.code] = item
	}

	for _, code := range []string{
		"xdev:device-parameter-group:view",
		"xdev:device-parameter-group:create",
		"xdev:device-parameter-group:edit",
		"xdev:device-parameter-group:delete",
		"xdev:device-parameter-group:export",
	} {
		if _, ok := byCode[code]; !ok {
			t.Fatalf("expected primary permission %q", code)
		}
	}
	for _, code := range []string{
		"xdev:device-parameter-item:view",
		"xdev:device-parameter-item:create",
		"xdev:device-parameter-item:edit",
		"xdev:device-parameter-item:delete",
	} {
		if _, ok := byCode[code]; ok {
			t.Fatalf("secondary permission should be collapsed: %q", code)
		}
	}

	primary := byCode["xdev:device-parameter-group:view"]
	if primary.name != "[菜单]Parameter Center" {
		t.Fatalf("expected composite menu to own the shared permission name, got %q", primary.name)
	}
	if len(primary.menuIDs) != 2 || primary.menuIDs[0] != 31 || primary.menuIDs[1] != 32 {
		t.Fatalf("expected both parameter menus to share the permission, got %#v", primary.menuIDs)
	}
	if len(primary.apiIDs) != 3 || primary.apiIDs[0] != 1 || primary.apiIDs[1] != 5 || primary.apiIDs[2] != 6 {
		t.Fatalf("expected primary view to absorb secondary APIs, got %#v", primary.apiIDs)
	}
	if create := byCode["xdev:device-parameter-group:create"]; len(create.apiIDs) != 1 || create.apiIDs[0] != 2 {
		t.Fatalf("expected primary create API binding, got %#v", create.apiIDs)
	}
}

func TestCollectFeaturePermissions_KeepsSecondaryActionsWithoutSecondaryViewEntry(t *testing.T) {
	menus := []*resourcev1.Menu{
		{
			Id:       uint32Ptr(40),
			Name:     stringPtr("TaskManagement"),
			Path:     stringPtr("/task/index"),
			Type:     resourcev1.Menu_MENU.Enum(),
			Status:   resourcev1.Menu_ON.Enum(),
			ParentId: uint32Ptr(2),
			Meta: &resourcev1.MenuMeta{
				Title: stringPtr("Task Management"),
				Authority: []string{
					"tasks:view",
					"tasks:create",
					"tasks:edit",
					"tasks:delete",
					"tasks:export",
					"tasks:start",
					"tasks:stop",
					"tasks:run-once",
					"task-groups:create",
					"task-groups:edit",
					"task-groups:delete",
					"task-groups:start",
					"task-groups:stop",
					"task-groups:run-once",
				},
			},
		},
	}

	apis := []*resourcev1.Api{
		{
			Id:          uint32Ptr(1),
			Method:      stringPtr("POST"),
			Path:        stringPtr("/admin/v1/tasks/{id}:run-once"),
			Description: stringPtr("Run task once"),
			Status:      resourcev1.Api_ON.Enum(),
		},
		{
			Id:          uint32Ptr(2),
			Method:      stringPtr("POST"),
			Path:        stringPtr("/admin/v1/task-groups"),
			Description: stringPtr("Create task group"),
			Status:      resourcev1.Api_ON.Enum(),
		},
		{
			Id:          uint32Ptr(3),
			Method:      stringPtr("POST"),
			Path:        stringPtr("/admin/v1/task-groups/{id}:run-once"),
			Description: stringPtr("Run task group once"),
			Status:      resourcev1.Api_ON.Enum(),
		},
	}

	items := collectFeaturePermissions(menus, apis)
	byCode := map[string]desiredPermission{}
	for _, item := range items {
		byCode[item.code] = item
	}

	if _, ok := byCode["task-groups:view"]; ok {
		t.Fatalf("did not expect synthetic task-groups:view permission when no explicit secondary view exists")
	}
	if item, ok := byCode["tasks:run-once"]; !ok {
		t.Fatalf("expected tasks:run-once permission to be preserved")
	} else if len(item.apiIDs) != 1 || item.apiIDs[0] != 1 {
		t.Fatalf("expected tasks:run-once to bind API 1, got %#v", item.apiIDs)
	}
	if item, ok := byCode["task-groups:create"]; !ok {
		t.Fatalf("expected task-groups:create permission to be preserved")
	} else if len(item.apiIDs) != 1 || item.apiIDs[0] != 2 {
		t.Fatalf("expected task-groups:create to bind API 2, got %#v", item.apiIDs)
	}
	if item, ok := byCode["task-groups:run-once"]; !ok {
		t.Fatalf("expected task-groups:run-once permission to be preserved")
	} else if len(item.apiIDs) != 1 || item.apiIDs[0] != 3 {
		t.Fatalf("expected task-groups:run-once to bind API 3, got %#v", item.apiIDs)
	}
}
