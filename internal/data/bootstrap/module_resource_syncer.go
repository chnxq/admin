package bootstrap

import (
	"context"
	"fmt"
	"time"

	resourcev1 "admin/api/gen/resource/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/menu"
	"admin/internal/data/ent/predicate"
	modulehost "admin/shared/modulehost"

	entCrud "github.com/chnxq/x-crud/entgo"
)

type hostResourceSyncer struct {
	entClient *entCrud.EntClient[*ent.Client]
	now       time.Time
}

func newHostResourceSyncer(entClient *entCrud.EntClient[*ent.Client], now time.Time) modulehost.ResourceSyncer {
	if entClient == nil || entClient.Client() == nil {
		return nil
	}
	return &hostResourceSyncer{entClient: entClient, now: now}
}

func (s *hostResourceSyncer) UpsertMenus(ctx context.Context, menus []modulehost.MenuResource) error {
	if s == nil || s.entClient == nil || len(menus) == 0 {
		return nil
	}
	for _, item := range menus {
		if _, err := s.ensureMenu(ctx, item, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *hostResourceSyncer) ensureMenu(ctx context.Context, item modulehost.MenuResource, parentID *uint32) (*ent.Menu, error) {
	existing, err := s.findMenu(ctx, item, parentID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		existing, err = s.createMenu(ctx, item, parentID)
		if err != nil {
			return nil, err
		}
	} else {
		existing, err = s.updateMenu(ctx, existing, item, parentID)
		if err != nil {
			return nil, err
		}
	}
	for _, child := range item.Children {
		if _, err := s.ensureMenu(ctx, child, &existing.ID); err != nil {
			return nil, err
		}
	}
	return existing, nil
}

func (s *hostResourceSyncer) findMenu(ctx context.Context, item modulehost.MenuResource, parentID *uint32) (*ent.Menu, error) {
	if item.Path != "" {
		entity, err := s.findMenuBy(ctx, parentID, menu.PathEQ(item.Path))
		if err != nil || entity != nil {
			return entity, err
		}
	}
	if item.Name != "" {
		return s.findMenuBy(ctx, parentID, menu.NameEQ(item.Name))
	}
	return nil, nil
}

func (s *hostResourceSyncer) findMenuBy(ctx context.Context, parentID *uint32, predicates ...predicate.Menu) (*ent.Menu, error) {
	query := s.entClient.Client().Menu.Query()
	if parentID == nil {
		query = query.Where(menu.ParentIDIsNil())
	} else {
		query = query.Where(menu.ParentIDEQ(*parentID))
	}
	entity, err := query.Where(predicates...).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (s *hostResourceSyncer) createMenu(ctx context.Context, item modulehost.MenuResource, parentID *uint32) (*ent.Menu, error) {
	builder := s.entClient.Client().Menu.Create().
		SetType(resourceMenuType(item.Type)).
		SetPath(item.Path).
		SetName(item.Name).
		SetComponent(item.Component).
		SetStatus(menu.StatusOn).
		SetMeta(resourceMenuMeta(item.Meta)).
		SetCreatedAt(s.now).
		SetCreatedBy(0).
		SetUpdatedAt(s.now).
		SetUpdatedBy(0)
	if parentID != nil {
		builder.SetParentID(*parentID)
	}
	if item.Redirect != "" {
		builder.SetRedirect(item.Redirect)
	}
	entity, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create module menu %s: %w", item.Path, err)
	}
	return entity, nil
}

func (s *hostResourceSyncer) updateMenu(ctx context.Context, existing *ent.Menu, item modulehost.MenuResource, parentID *uint32) (*ent.Menu, error) {
	builder := s.entClient.Client().Menu.UpdateOneID(existing.ID).
		SetType(resourceMenuType(item.Type)).
		SetPath(item.Path).
		SetName(item.Name).
		SetComponent(item.Component).
		SetStatus(menu.StatusOn).
		SetMeta(resourceMenuMeta(item.Meta)).
		SetUpdatedAt(s.now).
		SetUpdatedBy(0)
	if parentID != nil {
		builder.SetParentID(*parentID)
	} else {
		builder.ClearParentID()
	}
	if item.Redirect != "" {
		builder.SetRedirect(item.Redirect)
	} else {
		builder.ClearRedirect()
	}
	entity, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update module menu %s: %w", item.Path, err)
	}
	return entity, nil
}

func resourceMenuType(value modulehost.MenuType) menu.Type {
	switch value {
	case modulehost.MenuTypeCatalog:
		return menu.TypeCatalog
	case modulehost.MenuTypeEmbedded:
		return menu.TypeEmbedded
	case modulehost.MenuTypeLink:
		return menu.TypeLink
	case modulehost.MenuTypeButton:
		return menu.TypeButton
	default:
		return menu.TypeMenu
	}
}

func resourceMenuMeta(meta modulehost.MenuMeta) *resourcev1.MenuMeta {
	return &resourcev1.MenuMeta{
		Authority:       append([]string(nil), meta.Authority...),
		Icon:            meta.Icon,
		Link:            meta.Link,
		OpenInNewWindow: meta.OpenInNewWindow,
		Title:           meta.Title,
	}
}
