package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	identityv1 "admin/api/gen/identity/v1"
	"admin/internal/data/repo"
	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
	"github.com/chnxq/xkitpkg/app"
	cachepkg "github.com/chnxq/xkitpkg/cache"
	conf "github.com/chnxq/xkitpkg/conf/v1"
)

const tenantOptionsCacheKey = "profile:tenant-options"

type tenantOptionsStore struct {
	cache cachepkg.AdapterCache
}

var (
	sharedTenantOptionsStoreMu   sync.Mutex
	sharedTenantOptionsStore     *tenantOptionsStore
	sharedTenantOptionsStoreErr  error
	sharedTenantOptionsStoreInit bool
)

func newTenantOptionsStore(dataCfg *conf.Data) (*tenantOptionsStore, error) {
	sharedTenantOptionsStoreMu.Lock()
	defer sharedTenantOptionsStoreMu.Unlock()

	if sharedTenantOptionsStoreInit {
		return sharedTenantOptionsStore, sharedTenantOptionsStoreErr
	}
	sharedTenantOptionsStore, sharedTenantOptionsStoreErr = newStandaloneTenantOptionsStore(dataCfg)
	sharedTenantOptionsStoreInit = true
	return sharedTenantOptionsStore, sharedTenantOptionsStoreErr
}

func newStandaloneTenantOptionsStore(dataCfg *conf.Data) (*tenantOptionsStore, error) {
	var (
		cache cachepkg.AdapterCache
		err   error
	)
	if dataCfg != nil {
		cache, err = cachepkg.NewCache(dataCfg)
		if err != nil {
			return nil, err
		}
	} else {
		cache = cachepkg.NewMemory()
		if err := cache.Connect(); err != nil {
			return nil, err
		}
	}
	return &tenantOptionsStore{cache: cache}, nil
}

func (s *tenantOptionsStore) Load() (*identityv1.ListTenantResponse, error) {
	if s == nil || s.cache == nil {
		return nil, fmt.Errorf("tenant options store is unavailable")
	}
	payload, err := s.cache.Get(tenantOptionsCacheKey)
	if err != nil {
		return nil, err
	}
	resp := &identityv1.ListTenantResponse{}
	if err := json.Unmarshal([]byte(payload), resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *tenantOptionsStore) Save(resp *identityv1.ListTenantResponse) error {
	if s == nil || s.cache == nil {
		return fmt.Errorf("tenant options store is unavailable")
	}
	if resp == nil {
		resp = &identityv1.ListTenantResponse{}
	}
	payload, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return s.cache.Set(tenantOptionsCacheKey, string(payload), 0)
}

func (s *tenantOptionsStore) Refresh(ctx context.Context, tenantRepo repo.TenantRepo) (*identityv1.ListTenantResponse, error) {
	result, err := loadTenantOptionsFromRepo(ctx, tenantRepo)
	if err != nil {
		return nil, err
	}
	if err := s.Save(result); err != nil {
		return nil, err
	}
	return result, nil
}

func loadTenantOptionsFromRepo(ctx context.Context, tenantRepo repo.TenantRepo) (*identityv1.ListTenantResponse, error) {
	if tenantRepo == nil {
		return nil, fmt.Errorf("tenant repo is unavailable")
	}
	limit := uint32(500)
	resp, err := tenantRepo.List(ensureDefaultViewerContext(ctx), &paginationv1.PagingRequest{
		Limit: &limit,
	})
	if err != nil {
		return nil, err
	}
	return filterTenantOptions(resp), nil
}

func filterTenantOptions(resp *identityv1.ListTenantResponse) *identityv1.ListTenantResponse {
	if resp == nil {
		return &identityv1.ListTenantResponse{}
	}
	items := make([]*identityv1.Tenant, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		if item == nil || item.GetId() == 0 {
			continue
		}
		switch item.GetStatus() {
		case identityv1.Tenant_OFF, identityv1.Tenant_FREEZE:
			continue
		default:
			items = append(items, item)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].GetId() < items[j].GetId()
	})
	return &identityv1.ListTenantResponse{
		Items: items,
		Total: uint64(len(items)),
	}
}

func WarmTenantOptionsCache(appCtx *app.AppCtx, tenantRepo repo.TenantRepo) error {
	store, err := newTenantOptionsStore(loadDataConfig(appCtx))
	if err != nil {
		if appCtx != nil {
			appCtx.NewLoggerHelper("user_profile/server").Errorf("init tenant options cache store failed: %s", err.Error())
		}
		return err
	}
	baseCtx := context.Background()
	if appCtx != nil {
		baseCtx = appCtx.AppContext()
	}
	resp, err := store.Refresh(baseCtx, tenantRepo)
	if err != nil {
		if appCtx != nil {
			appCtx.NewLoggerHelper("user_profile/server").Errorf("warm tenant options cache failed: %s", err.Error())
		}
		return err
	}
	if appCtx != nil {
		appCtx.NewLoggerHelper("user_profile/server").Infof("tenant options cache warmed: total=%d", resp.GetTotal())
	}
	return nil
}
