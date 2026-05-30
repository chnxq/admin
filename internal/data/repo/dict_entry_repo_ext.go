package repo

import (
	"context"
	"fmt"

	dictv1 "admin/api/gen/dict/v1"
	identityv1 "admin/api/gen/identity/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/dictentry"
	"admin/internal/data/ent/dicttype"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// Add DictEntryRepo-specific query helpers and hand-written data access code here.
// This file is created once and is never overwritten by xkit.

func (r *dictEntryRepo) attachDictEntryTenantNames(ctx context.Context, entities []*ent.DictEntry) ([]*dictv1.DictEntry, error) {
	items := make([]*dictv1.DictEntry, 0, len(entities))
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
	return items, nil
}

func (r *dictEntryRepo) dictEntryEnrichListDTOs(ctx context.Context, entities []*ent.DictEntry) ([]*dictv1.DictEntry, error) {
	return r.attachDictEntryTenantNames(ctx, entities)
}

func (r *dictEntryRepo) dictEntryEnrichGetDTO(ctx context.Context, entities []*ent.DictEntry) ([]*dictv1.DictEntry, error) {
	return r.attachDictEntryTenantNames(ctx, entities)
}

func validateDictEntryTypeTenant(ctx context.Context, txClient *ent.Client, typeID uint32, entryTenantID *uint32) (*uint32, error) {
	if typeID == 0 {
		return nil, fmt.Errorf("invalid dict type id")
	}
	typeEntity, err := txClient.DictType.Query().Where(dicttype.IDEQ(typeID)).Only(ctx)
	if err != nil {
		return nil, err
	}
	if err := ensureHybridTenantAccessible(ctx, typeEntity.TenantID); err != nil {
		return nil, err
	}
	if entryTenantID == nil {
		if typeEntity.TenantID == nil || *typeEntity.TenantID == platformTenantID {
			return nil, nil
		}
		return nil, identityv1.ErrorForbidden("cross-tenant access is forbidden")
	}
	if typeEntity.TenantID == nil {
		return nil, identityv1.ErrorForbidden("cross-tenant access is forbidden")
	}
	if *entryTenantID != *typeEntity.TenantID {
		return nil, identityv1.ErrorForbidden("cross-tenant access is forbidden")
	}
	return typeEntity.TenantID, nil
}

func (r *dictEntryRepo) dictEntryCustomCreate(ctx context.Context, req *dictv1.CreateDictEntryRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("invalid parameter")
	}

	builder := r.entClient.Client().DictEntry.Create()
	now, viewer := r.generatedAuditContext(ctx)
	builder.SetEntryValue(req.Data.GetEntryValue())
	builder.SetNillableNumericValue(req.Data.NumericValue)
	builder.SetIsEnabled(req.Data.GetIsEnabled())
	builder.SetNillableSortOrder(req.Data.SortOrder)
	builder.SetNillableDictTypeID(req.Data.TypeId)
	tenantID, err := resolveCreateTenantID(ctx, req.Data.TenantId)
	if err != nil {
		return nil, err
	}
	resolvedTenantID, err := validateDictEntryTypeTenant(ctx, r.entClient.Client(), req.Data.GetTypeId(), tenantID)
	if err != nil {
		return nil, err
	}
	builder.SetNillableTenantID(resolvedTenantID)
	builder.SetCreatedAt(now)
	builder.SetCreatedBy(uint32(viewer.UserID()))

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("insert dict_entry failed: %s", err.Error())
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (r *dictEntryRepo) dictEntryCustomUpdate(ctx context.Context, req *dictv1.UpdateDictEntryRequest) (*emptypb.Empty, error) {
	if req == nil || req.Data == nil {
		return nil, fmt.Errorf("invalid parameter")
	}

	current, err := r.entClient.Client().DictEntry.Query().Where(dictentry.IDEQ(req.GetId())).Only(ctx)
	if err != nil {
		r.log.Errorf("get dict_entry before update failed: %s", err.Error())
		return nil, err
	}
	if err := ensureHybridTenantMutable(ctx, current.TenantID); err != nil {
		return nil, err
	}

	typeID, err := current.QueryDictType().OnlyID(ctx)
	if err != nil {
		return nil, err
	}
	if req.Data.TypeId != nil {
		typeID = req.Data.GetTypeId()
	}
	if _, err := validateDictEntryTypeTenant(ctx, r.entClient.Client(), typeID, current.TenantID); err != nil {
		return nil, err
	}

	builder := r.entClient.Client().DictEntry.UpdateOneID(req.GetId())
	now, viewer := r.generatedAuditContext(ctx)
	if req.Data.TypeId != nil {
		builder.SetDictTypeID(req.Data.GetTypeId())
	}
	builder.SetEntryValue(req.Data.GetEntryValue())
	if req.Data.NumericValue != nil {
		builder.SetNillableNumericValue(req.Data.NumericValue)
	} else if req.GetUpdateMask() != nil && dictEntryFieldMaskContains(req.GetUpdateMask().GetPaths(), "numeric_value", "numericValue") {
		builder.ClearNumericValue()
	}
	if req.Data.IsEnabled != nil {
		builder.SetIsEnabled(req.Data.GetIsEnabled())
	}
	if req.Data.SortOrder != nil {
		builder.SetNillableSortOrder(req.Data.SortOrder)
	} else if req.GetUpdateMask() != nil && dictEntryFieldMaskContains(req.GetUpdateMask().GetPaths(), "sort_order", "sortOrder") {
		builder.ClearSortOrder()
	}
	builder.SetUpdatedAt(now)
	builder.SetUpdatedBy(uint32(viewer.UserID()))

	if _, err := builder.Save(ctx); err != nil {
		r.log.Errorf("update dict_entry failed: %s", err.Error())
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (r *dictEntryRepo) dictEntryCustomDelete(ctx context.Context, req *dictv1.DeleteDictEntryRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, fmt.Errorf("invalid parameter")
	}

	switch typedReq := any(req).(type) {
	case interface{ GetId() uint32 }:
		entity, err := r.entClient.Client().DictEntry.Query().Where(dictentry.IDEQ(typedReq.GetId())).Only(ctx)
		if err != nil {
			r.log.Errorf("load dict_entry before delete failed: %s", err.Error())
			return nil, err
		}
		if err := ensureHybridTenantMutable(ctx, entity.TenantID); err != nil {
			return nil, err
		}
		if err := r.entClient.Client().DictEntry.DeleteOneID(typedReq.GetId()).Exec(ctx); err != nil {
			r.log.Errorf("delete dict_entry failed: %s", err.Error())
			return nil, err
		}
	case interface{ GetIds() []uint32 }:
		entities, err := r.entClient.Client().DictEntry.Query().Where(dictentry.IDIn(typedReq.GetIds()...)).All(ctx)
		if err != nil {
			r.log.Errorf("load dict_entries before delete failed: %s", err.Error())
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
		if _, err := r.entClient.Client().DictEntry.Delete().Where(dictentry.IDIn(typedReq.GetIds()...)).Exec(ctx); err != nil {
			r.log.Errorf("delete dict_entry failed: %s", err.Error())
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid delete request: missing id or ids")
	}

	return &emptypb.Empty{}, nil
}
