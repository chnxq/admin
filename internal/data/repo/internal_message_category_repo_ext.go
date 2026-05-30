package repo

import (
	"context"

	internalmessagev1 "admin/api/gen/internal_message/v1"
	"admin/internal/data/ent/internalmessagecategory"
)

// Add InternalMessageCategoryRepo-specific query helpers and hand-written data access code here.
// This file is created once and is never overwritten by xkit.

type InternalMessageCategoryLookup interface {
	ListByIDs(ctx context.Context, ids []uint32) ([]*internalmessagev1.InternalMessageCategory, error)
}

func (r *internalMessageCategoryRepo) ListByIDs(ctx context.Context, ids []uint32) ([]*internalmessagev1.InternalMessageCategory, error) {
	if r == nil || r.entClient == nil || len(ids) == 0 {
		return nil, nil
	}

	entities, err := r.entClient.Client().InternalMessageCategory.Query().
		Where(internalmessagecategory.IDIn(ids...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("list internal_message_category by ids failed: %s", err.Error())
		return nil, err
	}

	items := make([]*internalmessagev1.InternalMessageCategory, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		items = append(items, r.mapper.ToDTO(entity))
	}
	return items, nil
}
