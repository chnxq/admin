package repo

import (
	"context"
	"fmt"

	dictv1 "admin/api/gen/dict/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/dicttype"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// Add DictTypeRepo-specific query helpers and hand-written data access code here.
// This file is created once and is never overwritten by xkit.

func (r *dictTypeRepo) attachDictTypeTenantNames(ctx context.Context, entities []*ent.DictType) []*dictv1.DictType {
	items := make([]*dictv1.DictType, 0, len(entities))
	tenantIDs := make([]uint32, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		tenantIDs = append(tenantIDs, collectTenantIDs(entity.TenantID)...)
	}
	tenantNameMap := loadTenantNameMap(ctx, r.entClient.Client(), tenantIDs)
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		dto := r.mapper.ToDTO(entity)
		if dto == nil {
			continue
		}
		dto.TenantName = tenantNameFromMap(tenantNameMap, entity.TenantID)
		items = append(items, dto)
	}
	return items
}

func (r *dictTypeRepo) dictTypeEnrichListDTOs(ctx context.Context, entities []*ent.DictType) ([]*dictv1.DictType, error) {
	return r.attachDictTypeTenantNames(ctx, entities), nil
}

func (r *dictTypeRepo) dictTypeEnrichGetDTO(ctx context.Context, entities []*ent.DictType) ([]*dictv1.DictType, error) {
	return r.attachDictTypeTenantNames(ctx, entities), nil
}

func (r *dictTypeRepo) dictTypeCustomCreate(ctx context.Context, req *dictv1.CreateDictTypeRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("invalid parameter")
	}

	builder := r.entClient.Client().DictType.Create()
	now, viewer := r.generatedAuditContext(ctx)
	builder.SetNillableTypeCode(req.Data.TypeCode)
	builder.SetNillableTypeName(req.Data.TypeName)
	builder.SetIsEnabled(req.Data.GetIsEnabled())
	builder.SetNillableSortOrder(req.Data.SortOrder)
	tenantID, err := resolveCreateTenantID(ctx, req.Data.TenantId)
	if err != nil {
		return nil, err
	}
	builder.SetNillableTenantID(tenantID)
	builder.SetCreatedAt(now)
	builder.SetCreatedBy(uint32(viewer.UserID()))

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("insert dict_type failed: %s", err.Error())
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (r *dictTypeRepo) dictTypeCustomUpdate(ctx context.Context, req *dictv1.UpdateDictTypeRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("invalid parameter")
	}

	current, err := r.entClient.Client().DictType.Query().Where(dicttype.IDEQ(req.GetId())).Only(ctx)
	if err != nil {
		r.log.Errorf("get dict_type before update failed: %s", err.Error())
		return nil, err
	}
	if err := ensureHybridTenantMutable(ctx, current.TenantID); err != nil {
		return nil, err
	}

	builder := r.entClient.Client().DictType.UpdateOneID(req.GetId())
	now, viewer := r.generatedAuditContext(ctx)
	if req.Data.TypeName != nil {
		builder.SetNillableTypeName(req.Data.TypeName)
	} else if req.GetUpdateMask() != nil && dictTypeFieldMaskContains(req.GetUpdateMask().GetPaths(), "type_name", "typeName") {
		builder.ClearTypeName()
	}
	if req.Data.IsEnabled != nil {
		builder.SetIsEnabled(req.Data.GetIsEnabled())
	}
	if req.Data.SortOrder != nil {
		builder.SetNillableSortOrder(req.Data.SortOrder)
	} else if req.GetUpdateMask() != nil && dictTypeFieldMaskContains(req.GetUpdateMask().GetPaths(), "sort_order", "sortOrder") {
		builder.ClearSortOrder()
	}
	builder.SetUpdatedAt(now)
	builder.SetUpdatedBy(uint32(viewer.UserID()))

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update dict_type failed: %s", err.Error())
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (r *dictTypeRepo) dictTypeCustomDelete(ctx context.Context, req *dictv1.DeleteDictTypeRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, fmt.Errorf("invalid parameter")
	}

	switch typedReq := any(req).(type) {
	case interface{ GetId() uint32 }:
		entity, err := r.entClient.Client().DictType.Query().Where(dicttype.IDEQ(typedReq.GetId())).Only(ctx)
		if err != nil {
			r.log.Errorf("load dict_type before delete failed: %s", err.Error())
			return nil, err
		}
		if err := ensureHybridTenantMutable(ctx, entity.TenantID); err != nil {
			return nil, err
		}
		if err := r.entClient.Client().DictType.DeleteOneID(typedReq.GetId()).Exec(ctx); err != nil {
			r.log.Errorf("delete dict_type failed: %s", err.Error())
			return nil, err
		}
	case interface{ GetIds() []uint32 }:
		entities, err := r.entClient.Client().DictType.Query().Where(dicttype.IDIn(typedReq.GetIds()...)).All(ctx)
		if err != nil {
			r.log.Errorf("load dict_types before delete failed: %s", err.Error())
			return nil, err
		}
		for _, entity := range entities {
			if entity == nil {
				continue
			}
			if err := ensureHybridTenantMutable(ctx, entity.TenantID); err != nil {
				return nil, err
			}
		}
		if _, err := r.entClient.Client().DictType.Delete().Where(dicttype.IDIn(typedReq.GetIds()...)).Exec(ctx); err != nil {
			r.log.Errorf("delete dict_type failed: %s", err.Error())
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid delete request: missing id or ids")
	}

	return &emptypb.Empty{}, nil
}
