package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"

	auditv1 "admin/api/gen/audit/v1"
	permissionv1 "admin/api/gen/permission/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/api"
	"admin/internal/data/ent/menu"
	"admin/internal/data/ent/permission"
	"admin/internal/data/ent/permissionapi"
	"admin/internal/data/ent/permissionmenu"
	"admin/internal/data/ent/rolepermission"
	crudviewer "github.com/chnxq/x-crud/viewer"
	xlog "github.com/chnxq/xkitmod/log"
	"github.com/chnxq/xkitpkg/transport"
	httptransport "github.com/chnxq/xkitpkg/transport/http"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

const (
	permissionAuditTargetPermission      = "permission"
	permissionAuditTargetPermissionGroup = "permission_group"
	permissionAuditTargetRole            = "role"
)

var (
	auditActionAssign = auditv1.PermissionAuditLog_ASSIGN
	auditActionCreate = auditv1.PermissionAuditLog_CREATE
	auditActionDelete = auditv1.PermissionAuditLog_DELETE
	auditActionUpdate = auditv1.PermissionAuditLog_UPDATE
)

type permissionAuditContextKey string

const permissionAuditDisabledContextKey permissionAuditContextKey = "permission_audit_disabled"

type PermissionCodeReader interface {
	GetPermissionCodesByIDs(context.Context, []uint32) ([]string, error)
}

func (r *permissionRepo) GetPermissionCodesByIDs(ctx context.Context, ids []uint32) ([]string, error) {
	if r == nil || r.entClient == nil || len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.entClient.Client().Permission.Query().
		Where(permission.IDIn(ids...)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	codes := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row == nil || row.Code == nil || *row.Code == "" {
			continue
		}
		code := *row.Code
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}

	return codes, nil
}

func (r *permissionRepo) replacePermissionMenus(ctx context.Context, txClient *ent.Client, now time.Time, viewer crudviewer.Context, permissionID uint32, menuIDs []uint32) error {
	if err := validateExistingMenuIDs(ctx, txClient, menuIDs); err != nil {
		return err
	}
	if _, err := txClient.PermissionMenu.Delete().Where(permissionmenu.PermissionIDEQ(permissionID)).Exec(ctx); err != nil {
		return err
	}
	for _, menuID := range menuIDs {
		builder := txClient.PermissionMenu.Create().
			SetPermissionID(permissionID).
			SetMenuID(menuID).
			SetCreatedAt(now).
			SetCreatedBy(uint32(viewer.UserID()))
		if _, err := builder.Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (r *permissionRepo) replacePermissionApis(ctx context.Context, txClient *ent.Client, now time.Time, viewer crudviewer.Context, permissionID uint32, apiIDs []uint32) error {
	if err := validateExistingAPIIDs(ctx, txClient, apiIDs); err != nil {
		return err
	}
	if _, err := txClient.PermissionApi.Delete().Where(permissionapi.PermissionIDEQ(permissionID)).Exec(ctx); err != nil {
		return err
	}
	for _, apiID := range apiIDs {
		builder := txClient.PermissionApi.Create().
			SetPermissionID(permissionID).
			SetAPIID(apiID).
			SetCreatedAt(now).
			SetCreatedBy(uint32(viewer.UserID()))
		if _, err := builder.Save(ctx); err != nil {
			return err
		}
	}
	return nil
}

func validateExistingMenuIDs(ctx context.Context, txClient *ent.Client, menuIDs []uint32) error {
	uniqueIDs := uniqueUint32IDs(menuIDs)
	if len(uniqueIDs) == 0 {
		return nil
	}
	rows, err := txClient.Menu.Query().
		Where(menu.IDIn(uniqueIDs...)).
		All(ctx)
	if err != nil {
		return err
	}
	if len(rows) != len(uniqueIDs) {
		return fmt.Errorf("invalid menu ids")
	}
	return nil
}

func validateExistingAPIIDs(ctx context.Context, txClient *ent.Client, apiIDs []uint32) error {
	uniqueIDs := uniqueUint32IDs(apiIDs)
	if len(uniqueIDs) == 0 {
		return nil
	}
	rows, err := txClient.Api.Query().
		Where(api.IDIn(uniqueIDs...)).
		All(ctx)
	if err != nil {
		return err
	}
	if len(rows) != len(uniqueIDs) {
		return fmt.Errorf("invalid api ids")
	}
	return nil
}

func uniqueUint32IDs(values []uint32) []uint32 {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uint32]struct{}, len(values))
	result := make([]uint32, 0, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (r *permissionRepo) loadPermissionRelations(ctx context.Context, permissionIDs []uint32) (map[uint32][]uint32, map[uint32][]uint32, error) {
	menuMap := make(map[uint32][]uint32, len(permissionIDs))
	apiMap := make(map[uint32][]uint32, len(permissionIDs))
	if len(permissionIDs) == 0 {
		return menuMap, apiMap, nil
	}

	menuRows, err := r.entClient.Client().PermissionMenu.Query().
		Where(permissionmenu.PermissionIDIn(permissionIDs...)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, item := range menuRows {
		if item == nil || item.PermissionID == nil || item.MenuID == nil {
			continue
		}
		current := menuMap[*item.PermissionID]
		exists := false
		for _, id := range current {
			if id == *item.MenuID {
				exists = true
				break
			}
		}
		if !exists {
			menuMap[*item.PermissionID] = append(current, *item.MenuID)
		}
	}

	apiRows, err := r.entClient.Client().PermissionApi.Query().
		Where(permissionapi.PermissionIDIn(permissionIDs...)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, item := range apiRows {
		if item == nil || item.PermissionID == nil || item.APIID == nil {
			continue
		}
		current := apiMap[*item.PermissionID]
		exists := false
		for _, id := range current {
			if id == *item.APIID {
				exists = true
				break
			}
		}
		if !exists {
			apiMap[*item.PermissionID] = append(current, *item.APIID)
		}
	}

	return menuMap, apiMap, nil
}

func (r *permissionRepo) attachPermissionRelations(ctx context.Context, entities []*ent.Permission) ([]*permissionv1.Permission, error) {
	items := make([]*permissionv1.Permission, 0, len(entities))
	permissionIDs := make([]uint32, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		permissionIDs = append(permissionIDs, entity.ID)
	}

	menuMap, apiMap, err := r.loadPermissionRelations(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}

	for _, entity := range entities {
		if entity == nil {
			continue
		}
		dto := r.mapper.ToDTO(entity)
		if dto == nil {
			continue
		}
		dto.MenuIds = menuMap[entity.ID]
		dto.ApiIds = apiMap[entity.ID]
		items = append(items, dto)
	}

	return items, nil
}

func (r *permissionRepo) permissionEnrichListDTOs(ctx context.Context, entities []*ent.Permission) ([]*permissionv1.Permission, error) {
	return r.attachPermissionRelations(ctx, entities)
}

func (r *permissionRepo) permissionEnrichGetDTO(ctx context.Context, entities []*ent.Permission) ([]*permissionv1.Permission, error) {
	return r.attachPermissionRelations(ctx, entities)
}

func (r *permissionRepo) permissionCustomCreate(ctx context.Context, req *permissionv1.CreatePermissionRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("invalid parameter")
	}
	if err := ensurePlatformOnlyMutable(ctx); err != nil {
		return nil, err
	}

	now, viewer := r.generatedAuditContext(ctx)
	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	builder := tx.Client().Permission.Create()
	builder.SetName(req.Data.GetName())
	builder.SetCode(req.Data.GetCode())
	builder.SetNillableGroupID(req.Data.GroupId)
	builder.SetStatus(permission.Status(req.Data.GetStatus().String()))
	builder.SetCreatedAt(now)
	builder.SetCreatedBy(uint32(viewer.UserID()))

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert permission failed: %s", err.Error())
		return nil, err
	}

	if err := r.replacePermissionMenus(ctx, tx.Client(), now, viewer, entity.ID, req.Data.GetMenuIds()); err != nil {
		return nil, err
	}
	if err := r.replacePermissionApis(ctx, tx.Client(), now, viewer, entity.ID, req.Data.GetApiIds()); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	if permissionAuditDisabled(ctx) {
		return &emptypb.Empty{}, nil
	}
	r.writePermissionAuditLog(ctx, attachPermissionAuditRequestMeta(
		newPermissionAuditLogInput(ctx, auditActionCreate, permissionAuditTargetPermission, entity.ID),
		auditRequestID(ctx),
		auditClientIP(ctx),
		"",
	), "", marshalPermissionAuditValue(buildPermissionPointSnapshot(req.Data)))
	return &emptypb.Empty{}, nil
}

func (r *permissionRepo) permissionCustomUpdate(ctx context.Context, req *permissionv1.UpdatePermissionRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("invalid parameter")
	}
	if err := ensurePlatformOnlyMutable(ctx); err != nil {
		return nil, err
	}

	now, viewer := r.generatedAuditContext(ctx)
	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	beforeSnapshot, err := r.permissionPointSnapshotByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	builder := tx.Client().Permission.UpdateOneID(req.GetId())
	builder.SetName(req.Data.GetName())
	builder.SetCode(req.Data.GetCode())
	if req.Data.GroupId != nil {
		builder.SetNillableGroupID(req.Data.GroupId)
	} else if req.GetUpdateMask() != nil && permissionFieldMaskContains(req.GetUpdateMask().GetPaths(), "group_id", "groupId") {
		builder.ClearGroupID()
	}
	builder.SetStatus(permission.Status(req.Data.GetStatus().String()))
	builder.SetUpdatedAt(now)
	builder.SetUpdatedBy(uint32(viewer.UserID()))

	_, err = builder.Save(ctx)
	if err != nil {
		r.log.Errorf("update permission failed: %s", err.Error())
		return nil, err
	}

	if mask := req.GetUpdateMask(); mask == nil || permissionFieldMaskContains(mask.GetPaths(), "menuIds", "menu_ids") {
		if err := r.replacePermissionMenus(ctx, tx.Client(), now, viewer, req.GetId(), req.Data.GetMenuIds()); err != nil {
			return nil, err
		}
	}
	if mask := req.GetUpdateMask(); mask == nil || permissionFieldMaskContains(mask.GetPaths(), "apiIds", "api_ids") {
		if err := r.replacePermissionApis(ctx, tx.Client(), now, viewer, req.GetId(), req.Data.GetApiIds()); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	afterSnapshot, err := r.permissionPointSnapshotByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if permissionAuditDisabled(ctx) || permissionAuditSnapshotsEqual(beforeSnapshot, afterSnapshot) {
		return &emptypb.Empty{}, nil
	}
	r.writePermissionAuditLog(ctx, attachPermissionAuditRequestMeta(
		newPermissionAuditLogInput(ctx, auditActionUpdate, permissionAuditTargetPermission, req.GetId()),
		auditRequestID(ctx),
		auditClientIP(ctx),
		"",
	), beforeSnapshot, afterSnapshot)
	return &emptypb.Empty{}, nil
}

func (r *permissionRepo) permissionCustomDelete(ctx context.Context, req *permissionv1.DeletePermissionRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, fmt.Errorf("invalid parameter")
	}
	if err := ensurePlatformOnlyMutable(ctx); err != nil {
		return nil, err
	}

	permissionIDs, err := r.permissionDeleteIDs(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(permissionIDs) == 0 {
		return &emptypb.Empty{}, nil
	}

	beforeSnapshots, err := r.permissionPointSnapshotsByIDs(ctx, permissionIDs)
	if err != nil {
		return nil, err
	}
	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.Client().PermissionMenu.Delete().Where(permissionmenu.PermissionIDIn(permissionIDs...)).Exec(ctx); err != nil {
		r.log.Errorf("delete permission menus failed: %s", err.Error())
		return nil, err
	}
	if _, err := tx.Client().PermissionApi.Delete().Where(permissionapi.PermissionIDIn(permissionIDs...)).Exec(ctx); err != nil {
		r.log.Errorf("delete permission apis failed: %s", err.Error())
		return nil, err
	}
	if _, err := tx.Client().RolePermission.Delete().Where(rolepermission.PermissionIDIn(permissionIDs...)).Exec(ctx); err != nil {
		r.log.Errorf("delete role permissions failed: %s", err.Error())
		return nil, err
	}
	if _, err := tx.Client().Permission.Delete().Where(permission.IDIn(permissionIDs...)).Exec(ctx); err != nil {
		r.log.Errorf("delete permission failed: %s", err.Error())
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	if permissionAuditDisabled(ctx) {
		return &emptypb.Empty{}, nil
	}
	for _, id := range permissionIDs {
		r.writePermissionAuditLog(ctx, attachPermissionAuditRequestMeta(
			newPermissionAuditLogInput(ctx, auditActionDelete, permissionAuditTargetPermission, id),
			auditRequestID(ctx),
			auditClientIP(ctx),
			"",
		), beforeSnapshots[id], "")
	}

	return &emptypb.Empty{}, nil
}

func (r *permissionRepo) permissionDeleteIDs(ctx context.Context, req *permissionv1.DeletePermissionRequest) ([]uint32, error) {
	if req == nil {
		return nil, fmt.Errorf("invalid parameter")
	}
	switch query := req.GetQueryBy().(type) {
	case *permissionv1.DeletePermissionRequest_Id:
		if query.Id == 0 {
			return nil, fmt.Errorf("invalid delete request: missing id")
		}
		return []uint32{query.Id}, nil
	case *permissionv1.DeletePermissionRequest_Code:
		entity, err := r.entClient.Client().Permission.Query().Where(permission.CodeEQ(query.Code)).Only(ctx)
		if err != nil {
			r.log.Errorf("load permission failed: %s", err.Error())
			return nil, err
		}
		return []uint32{entity.ID}, nil
	case *permissionv1.DeletePermissionRequest_GroupId:
		entities, err := r.entClient.Client().Permission.Query().Where(permission.GroupIDEQ(query.GroupId)).All(ctx)
		if err != nil {
			r.log.Errorf("load permissions failed: %s", err.Error())
			return nil, err
		}
		ids := make([]uint32, 0, len(entities))
		for _, entity := range entities {
			if entity != nil {
				ids = append(ids, entity.ID)
			}
		}
		return ids, nil
	default:
		return nil, fmt.Errorf("invalid delete request: missing id, code or group_id")
	}
}

func (r *permissionRepo) permissionPointSnapshotByID(ctx context.Context, id uint32) (string, error) {
	snapshots, err := r.permissionPointSnapshotsByIDs(ctx, []uint32{id})
	if err != nil {
		return "", err
	}
	return snapshots[id], nil
}

func (r *permissionRepo) permissionPointSnapshotsByIDs(ctx context.Context, ids []uint32) (map[uint32]string, error) {
	entities, err := r.entClient.Client().Permission.Query().Where(permission.IDIn(ids...)).All(ctx)
	if err != nil {
		r.log.Errorf("load permission failed: %s", err.Error())
		return nil, err
	}
	items, err := r.attachPermissionRelations(ctx, entities)
	if err != nil {
		return nil, err
	}
	snapshots := make(map[uint32]string, len(items))
	for _, item := range items {
		if item == nil || item.Id == nil {
			continue
		}
		snapshots[item.GetId()] = marshalPermissionAuditValue(buildPermissionPointSnapshot(item))
	}
	return snapshots, nil
}

func (r *permissionRepo) writePermissionAuditLog(ctx context.Context, log *auditv1.PermissionAuditLog, oldValue, newValue string) {
	if r == nil || r.entClient == nil || log == nil {
		return
	}
	if log.GetOldValue() == "" && oldValue != "" {
		log.OldValue = &oldValue
	}
	if log.GetNewValue() == "" && newValue != "" {
		log.NewValue = &newValue
	}
	_ = writePermissionAuditLogEntity(ctx, r.entClient, r.log, log)
}

func auditRequestID(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	htr, ok := tr.(*httptransport.Transport)
	if !ok || htr.Request() == nil {
		return ""
	}
	for _, key := range []string{"X-Request-ID", "X-Correlation-ID", "x-fc-request-id"} {
		if value := strings.TrimSpace(htr.Request().Header.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func auditClientIP(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	htr, ok := tr.(*httptransport.Transport)
	if !ok || htr.Request() == nil {
		return ""
	}
	req := htr.Request()
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		for _, raw := range strings.Split(xff, ",") {
			ip := strings.TrimSpace(raw)
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	if ip := strings.TrimSpace(req.Header.Get("X-Real-IP")); net.ParseIP(ip) != nil {
		return ip
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil && net.ParseIP(host) != nil {
		return host
	}
	if net.ParseIP(req.RemoteAddr) != nil {
		return req.RemoteAddr
	}
	return ""
}

func loggerOrNil(helper *xlog.Helper) *xlog.Helper {
	return helper
}

func WithPermissionAuditDisabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, permissionAuditDisabledContextKey, true)
}

func permissionAuditDisabled(ctx context.Context) bool {
	disabled, _ := ctx.Value(permissionAuditDisabledContextKey).(bool)
	return disabled
}

func permissionAuditSnapshotsEqual(beforeRaw, afterRaw string) bool {
	before := strings.TrimSpace(beforeRaw)
	after := strings.TrimSpace(afterRaw)
	if before == after {
		return true
	}
	if before == "" || after == "" {
		return false
	}

	var beforeMap map[string]any
	var afterMap map[string]any
	if err := json.Unmarshal([]byte(before), &beforeMap); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(after), &afterMap); err != nil {
		return false
	}
	return reflect.DeepEqual(beforeMap, afterMap)
}
