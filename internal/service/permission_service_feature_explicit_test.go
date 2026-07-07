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
