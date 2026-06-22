package server

import (
	"context"
	"testing"

	identityv1 "admin/api/gen/identity/v1"
	cachepkg "github.com/chnxq/xkitpkg/cache"
)

func TestTenantOptionsStoreRefreshAndLoad(t *testing.T) {
	cache := cachepkg.NewMemory()
	if err := cache.Connect(); err != nil {
		t.Fatalf("connect memory cache failed: %v", err)
	}
	store := &tenantOptionsStore{cache: cache}

	resp, err := store.Refresh(context.Background(), stubTenantRepo{
		tenantByCode: map[string]*identityv1.Tenant{
			"tenant-b": {
				Id:     ptr(uint32(2)),
				Code:   ptr("tenant-b"),
				Name:   ptr("Tenant B"),
				Status: identityv1.Tenant_ON.Enum(),
			},
			"tenant-a": {
				Id:     ptr(uint32(1)),
				Code:   ptr("tenant-a"),
				Name:   ptr("Tenant A"),
				Status: identityv1.Tenant_ON.Enum(),
			},
			"tenant-off": {
				Id:     ptr(uint32(3)),
				Code:   ptr("tenant-off"),
				Name:   ptr("Tenant Off"),
				Status: identityv1.Tenant_OFF.Enum(),
			},
		},
	})
	if err != nil {
		t.Fatalf("refresh tenant options failed: %v", err)
	}

	if got := len(resp.GetItems()); got != 2 {
		t.Fatalf("expected 2 tenant options, got %d", got)
	}
	if first := resp.GetItems()[0]; first == nil || first.GetId() != 1 {
		t.Fatalf("expected first tenant id 1, got %+v", first)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load tenant options failed: %v", err)
	}
	if got := loaded.GetTotal(); got != 2 {
		t.Fatalf("expected cached total 2, got %d", got)
	}
	if second := loaded.GetItems()[1]; second == nil || second.GetId() != 2 {
		t.Fatalf("expected second tenant id 2, got %+v", second)
	}
}
