package repo

import (
	"context"
	"io"
	"testing"
	"time"

	authenticationv1 "admin/api/gen/authentication/v1"
	dictv1 "admin/api/gen/dict/v1"
	storagev1 "admin/api/gen/storage/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/file"
	"admin/internal/data/ent/loginpolicy"
	_ "admin/internal/data/ent/runtime"

	entsql "entgo.io/ent/dialect/sql"
	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
	entCrud "github.com/chnxq/x-crud/entgo"
	crudviewer "github.com/chnxq/x-crud/viewer"
	"github.com/chnxq/x-utils/mapper"
	xlog "github.com/chnxq/xkitmod/log"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func newTenantBoundaryEntClientForTest(t *testing.T, dbName string) (*entCrud.EntClient[*ent.Client], *ent.Client) {
	t.Helper()

	driver, err := entsql.Open("sqlite3", "file:"+dbName+"?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("open sqlite driver failed: %v", err)
	}
	client := ent.NewClient(ent.Driver(driver))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema failed: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = driver.Close()
	})

	return entCrud.NewEntClient[*ent.Client](client, driver), client
}

func newTenantBoundaryLoggerForTest() *xlog.Helper {
	return xlog.NewHelper(xlog.NewStdLogger(io.Discard))
}

func seedDictTypeForTenantBoundaryTest(t *testing.T, entClient *entCrud.EntClient[*ent.Client], id uint32, typeCode, typeName string, tenantID *uint32) {
	t.Helper()

	query := "INSERT INTO sys_dict_types (id, type_code, type_name, is_enabled, sort_order, tenant_id, created_by, updated_by, deleted_by, created_at, updated_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	now := time.Now()
	var tenantValue any
	if tenantID != nil {
		tenantValue = *tenantID
	}
	if err := entClient.Exec(context.Background(), query, []any{id, typeCode, typeName, true, 0, tenantValue, 0, 0, 0, now, now, nil}, nil); err != nil {
		t.Fatalf("seed dict type failed: %v", err)
	}
}

func tenantBoundaryPlatformCtx() context.Context {
	return crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		platform: true,
		tenant:   false,
		tenantID: 0,
	})
}

func tenantBoundaryTenantCtx(tenantID uint64) context.Context {
	return crudviewer.WithContext(context.Background(), tenantScopeTestViewer{
		platform: true,
		tenant:   true,
		tenantID: tenantID,
	})
}

func newDictTypeRepoForTenantBoundaryTest(entClient *entCrud.EntClient[*ent.Client]) *dictTypeRepo {
	repo := &dictTypeRepo{
		log:       newTenantBoundaryLoggerForTest(),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[dictv1.DictType, ent.DictType](),
	}
	repo.init()
	return repo
}

func newDictEntryRepoForTenantBoundaryTest(entClient *entCrud.EntClient[*ent.Client]) *dictEntryRepo {
	repo := &dictEntryRepo{
		log:       newTenantBoundaryLoggerForTest(),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[dictv1.DictEntry, ent.DictEntry](),
	}
	repo.init()
	return repo
}

func newFileRepoForTenantBoundaryTest(entClient *entCrud.EntClient[*ent.Client]) *fileRepo {
	repo := &fileRepo{
		log:       newTenantBoundaryLoggerForTest(),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[storagev1.File, ent.File](),
	}
	repo.init()
	return repo
}

func newLoginPolicyRepoForTenantBoundaryTest(entClient *entCrud.EntClient[*ent.Client]) *loginPolicyRepo {
	repo := &loginPolicyRepo{
		log:       newTenantBoundaryLoggerForTest(),
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[authenticationv1.LoginPolicy, ent.LoginPolicy](),
	}
	repo.init()
	return repo
}

func TestDictTypeRepoTenantViewerSeesGlobalAndOwnOnly(t *testing.T) {
	entClient, _ := newTenantBoundaryEntClientForTest(t, "tenant-boundary-dict-type-list")
	repo := newDictTypeRepoForTenantBoundaryTest(entClient)
	ctx := tenantBoundaryTenantCtx(101)

	globalTenant := uint32(0)
	seedDictTypeForTenantBoundaryTest(t, entClient, 1, "global", "global", &globalTenant)
	ownTenant := uint32(101)
	seedDictTypeForTenantBoundaryTest(t, entClient, 2, "own", "own", &ownTenant)
	otherTenant := uint32(202)
	seedDictTypeForTenantBoundaryTest(t, entClient, 3, "other", "other", &otherTenant)

	resp, err := repo.List(ctx, &paginationv1.PagingRequest{NoPaging: boolPtr(true)})
	if err != nil {
		t.Fatalf("list dict types failed: %v", err)
	}
	if len(resp.GetItems()) != 2 {
		t.Fatalf("expected 2 visible dict types, got %d", len(resp.GetItems()))
	}
	if _, err := repo.Get(ctx, &dictv1.GetDictTypeRequest{QueryBy: &dictv1.GetDictTypeRequest_Id{Id: 3}}); !dictv1.IsForbidden(err) {
		t.Fatalf("expected forbidden when reading other tenant dict type, got %v", err)
	}
}

func TestDictTypeRepoTenantViewerCannotMutateGlobalOrOtherTenant(t *testing.T) {
	entClient, _ := newTenantBoundaryEntClientForTest(t, "tenant-boundary-dict-type-mutate")
	repo := newDictTypeRepoForTenantBoundaryTest(entClient)
	ctx := tenantBoundaryTenantCtx(101)

	globalTenant := uint32(0)
	seedDictTypeForTenantBoundaryTest(t, entClient, 1, "global", "global", &globalTenant)

	typeName := "global-updated"
	if _, err := repo.Update(ctx, &dictv1.UpdateDictTypeRequest{
		Id:   1,
		Data: &dictv1.DictType{TypeName: &typeName},
	}); !dictv1.IsForbidden(err) {
		t.Fatalf("expected forbidden update, got %v", err)
	}
	if _, err := repo.Delete(ctx, &dictv1.DeleteDictTypeRequest{Ids: []uint32{1}}); !dictv1.IsForbidden(err) {
		t.Fatalf("expected forbidden delete, got %v", err)
	}

	otherTenant := uint32(202)
	createdCode := "tenant-created"
	createdName := "tenant-created"
	if _, err := repo.Create(ctx, &dictv1.CreateDictTypeRequest{
		Data: &dictv1.DictType{TypeCode: &createdCode, TypeName: &createdName, TenantId: &otherTenant},
	}); !dictv1.IsForbidden(err) {
		t.Fatalf("expected tenant create with other tenant id forbidden, got %v", err)
	}
}

func TestDictEntryRepoTenantAndTypeOwnershipMustMatch(t *testing.T) {
	entClient, client := newTenantBoundaryEntClientForTest(t, "tenant-boundary-dict-entry")
	repo := newDictEntryRepoForTenantBoundaryTest(entClient)
	seedCtx := tenantBoundaryPlatformCtx()
	ctx := tenantBoundaryTenantCtx(101)

	globalTenant := uint32(0)
	seedDictTypeForTenantBoundaryTest(t, entClient, 1, "global", "global", &globalTenant)
	ownTenant := uint32(101)
	seedDictTypeForTenantBoundaryTest(t, entClient, 2, "own", "own", &ownTenant)
	otherTenant := uint32(202)
	seedDictTypeForTenantBoundaryTest(t, entClient, 3, "other", "other", &otherTenant)

	ownTypeID := uint32(2)
	ownEntry, err := client.DictEntry.Create().SetEntryValue("own-entry").SetDictTypeID(ownTypeID).SetTenantID(ownTenant).Save(seedCtx)
	if err != nil {
		t.Fatalf("seed own dict entry failed: %v", err)
	}

	globalEntryValue := "global-entry"
	globalTypeID := uint32(1)
	if _, err := repo.Create(ctx, &dictv1.CreateDictEntryRequest{
		Data: &dictv1.DictEntry{
			TypeId:     &globalTypeID,
			EntryValue: &globalEntryValue,
		},
	}); !dictv1.IsForbidden(err) {
		t.Fatalf("expected tenant viewer create on global dict type forbidden, got %v", err)
	}

	otherTypeID := uint32(3)
	if _, err := repo.Create(ctx, &dictv1.CreateDictEntryRequest{
		Data: &dictv1.DictEntry{
			TypeId:     &otherTypeID,
			EntryValue: &globalEntryValue,
		},
	}); !dictv1.IsForbidden(err) {
		t.Fatalf("expected tenant viewer create on other tenant dict type forbidden, got %v", err)
	}

	resp, err := repo.List(ctx, &paginationv1.PagingRequest{NoPaging: boolPtr(true)})
	if err != nil {
		t.Fatalf("list dict entries failed: %v", err)
	}
	if len(resp.GetItems()) != 1 {
		t.Fatalf("expected 1 visible dict entry, got %d", len(resp.GetItems()))
	}

	otherEntry, err := client.DictEntry.Create().SetEntryValue("other-entry").SetDictTypeID(otherTypeID).SetTenantID(otherTenant).Save(seedCtx)
	if err != nil {
		t.Fatalf("create other tenant dict entry failed: %v", err)
	}
	if _, err := client.DictEntry.Create().SetEntryValue("global-existing").SetDictTypeID(globalTypeID).SetTenantID(0).Save(seedCtx); err != nil {
		t.Fatalf("create global dict entry failed: %v", err)
	}
	if _, err := repo.Delete(ctx, &dictv1.DeleteDictEntryRequest{Ids: []uint32{otherEntry.ID}}); !dictv1.IsForbidden(err) {
		t.Fatalf("expected forbidden delete for other tenant dict entry, got %v", err)
	}

	updatedValue := "own-entry-updated"
	if _, err := repo.Update(ctx, &dictv1.UpdateDictEntryRequest{
		Id: ownEntry.ID,
		Data: &dictv1.DictEntry{
			TypeId:     &ownTypeID,
			EntryValue: &updatedValue,
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"entry_value"}},
	}); err != nil {
		t.Fatalf("update own dict entry failed: %v", err)
	}
}

func TestFileRepoTenantBoundary(t *testing.T) {
	entClient, client := newTenantBoundaryEntClientForTest(t, "tenant-boundary-file")
	repo := newFileRepoForTenantBoundaryTest(entClient)
	seedCtx := tenantBoundaryPlatformCtx()
	ctx := tenantBoundaryTenantCtx(101)

	ownTenant := uint32(101)
	otherTenant := uint32(202)
	if _, err := client.File.Create().SetFileName("own-file").SetTenantID(ownTenant).Save(seedCtx); err != nil {
		t.Fatalf("create own file failed: %v", err)
	}
	otherFile, err := client.File.Create().SetFileName("other-file").SetTenantID(otherTenant).Save(seedCtx)
	if err != nil {
		t.Fatalf("create other file failed: %v", err)
	}

	fileName := "tenant-created"
	if _, err := repo.Create(ctx, &storagev1.CreateFileRequest{
		Data: &storagev1.File{FileName: &fileName},
	}); err != nil {
		t.Fatalf("tenant create file failed: %v", err)
	}
	created, err := client.File.Query().Where(file.FileNameEQ(fileName)).Only(seedCtx)
	if err != nil {
		t.Fatalf("load created file failed: %v", err)
	}
	if created.TenantID == nil || *created.TenantID != ownTenant {
		t.Fatalf("expected created file tenant_id=101, got %+v", created.TenantID)
	}

	resp, err := repo.List(ctx, &paginationv1.PagingRequest{NoPaging: boolPtr(true)})
	if err != nil {
		t.Fatalf("list files failed: %v", err)
	}
	if len(resp.GetItems()) != 2 {
		t.Fatalf("expected 2 visible files, got %d", len(resp.GetItems()))
	}
	if _, err := repo.Get(ctx, &storagev1.GetFileRequest{QueryBy: &storagev1.GetFileRequest_Id{Id: otherFile.ID}}); !storagev1.IsForbidden(err) {
		t.Fatalf("expected forbidden get for other tenant file, got %v", err)
	}
	if _, err := repo.Delete(ctx, &storagev1.DeleteFileRequest{QueryBy: &storagev1.DeleteFileRequest_Id{Id: otherFile.ID}}); !storagev1.IsForbidden(err) {
		t.Fatalf("expected forbidden delete for other tenant file, got %v", err)
	}
}

func TestLoginPolicyRepoTenantBoundary(t *testing.T) {
	entClient, client := newTenantBoundaryEntClientForTest(t, "tenant-boundary-login-policy")
	repo := newLoginPolicyRepoForTenantBoundaryTest(entClient)
	seedCtx := tenantBoundaryPlatformCtx()
	ctx := tenantBoundaryTenantCtx(101)

	ownTenant := uint32(101)
	otherTenant := uint32(202)
	if _, err := client.LoginPolicy.Create().SetTenantID(ownTenant).SetValue("10.0.0.1").Save(seedCtx); err != nil {
		t.Fatalf("create own login policy failed: %v", err)
	}
	otherPolicy, err := client.LoginPolicy.Create().SetTenantID(otherTenant).SetValue("10.0.0.2").Save(seedCtx)
	if err != nil {
		t.Fatalf("create other tenant login policy failed: %v", err)
	}

	value := "10.0.0.3"
	if _, err := repo.Create(ctx, &authenticationv1.CreateLoginPolicyRequest{
		Data: &authenticationv1.LoginPolicy{Value: &value},
	}); err != nil {
		t.Fatalf("tenant create login policy failed: %v", err)
	}
	created, err := client.LoginPolicy.Query().Where(loginpolicy.ValueEQ(value)).Only(seedCtx)
	if err != nil {
		t.Fatalf("load created login policy failed: %v", err)
	}
	if created.TenantID == nil || *created.TenantID != ownTenant {
		t.Fatalf("expected created login policy tenant_id=101, got %+v", created.TenantID)
	}

	resp, err := repo.List(ctx, &paginationv1.PagingRequest{NoPaging: boolPtr(true)})
	if err != nil {
		t.Fatalf("list login policies failed: %v", err)
	}
	if len(resp.GetItems()) != 2 {
		t.Fatalf("expected 2 visible login policies, got %d", len(resp.GetItems()))
	}
	if _, err := repo.Get(ctx, &authenticationv1.GetLoginPolicyRequest{QueryBy: &authenticationv1.GetLoginPolicyRequest_Id{Id: otherPolicy.ID}}); !authenticationv1.IsForbidden(err) {
		t.Fatalf("expected forbidden get for other tenant login policy, got %v", err)
	}
	if _, err := repo.Delete(ctx, &authenticationv1.DeleteLoginPolicyRequest{QueryBy: &authenticationv1.DeleteLoginPolicyRequest_Id{Id: otherPolicy.ID}}); !authenticationv1.IsForbidden(err) {
		t.Fatalf("expected forbidden delete for other tenant login policy, got %v", err)
	}
}
