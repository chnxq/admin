package repo

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	internalmessagev1 "admin/api/gen/internal_message/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/internalmessage"
	"admin/internal/data/ent/internalmessagerecipient"
	"admin/internal/data/ent/predicate"
	"admin/internal/data/ent/tenant"
	"entgo.io/ent/dialect/sql"
	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
	crudviewer "github.com/chnxq/x-crud/viewer"
	"github.com/chnxq/x-utils/mapper"
)

func (r *internalMessageRepo) ListByPaging(ctx context.Context, req PagingRequest) (*internalmessagev1.ListInternalMessageResponse, error) {
	if r == nil || r.entClient == nil {
		return nil, fmt.Errorf("internal message repo is not initialized")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	viewerUserID := viewerUserIDFromContext(ctx)
	predicates := []predicate.InternalMessage{
		internalmessage.StatusNEQ(internalmessage.StatusDeleted),
		internalmessage.Or(
			internalmessage.StatusNEQ(internalmessage.StatusRevoked),
			internalmessage.SenderIDEQ(viewerUserID),
		),
	}

	query := r.entClient.Client().InternalMessage.Query().
		Where(predicates...).
		Limit(limit).
		Offset(offset)
	applyInternalMessageSorting(query, req.Sorting)

	items, err := query.All(ctx)
	if err != nil {
		r.log.Errorf("list internal messages failed: %s", err.Error())
		return nil, err
	}
	total, err := r.entClient.Client().InternalMessage.Query().
		Where(predicates...).
		Count(ctx)
	if err != nil {
		r.log.Errorf("count internal messages failed: %s", err.Error())
		return nil, err
	}

	return &internalmessagev1.ListInternalMessageResponse{
		Items: toInternalMessageDTOs(ctx, r.entClient.Client(), r.mapper, items),
		Total: uint64(total),
	}, nil
}

func applyInternalMessageSorting(query *ent.InternalMessageQuery, sorting []*paginationv1.Sorting) {
	if query == nil {
		return
	}

	orderTerms := make([]internalmessage.OrderOption, 0, len(sorting))
	for _, item := range sorting {
		if item == nil {
			continue
		}
		field := strings.TrimSpace(item.GetField())
		if field == "" {
			continue
		}

		opts := []sql.OrderTermOption{}
		if item.GetDirection() == paginationv1.Sorting_DESC {
			opts = append(opts, sql.OrderDesc())
		}

		switch field {
		case "content":
			orderTerms = append(orderTerms, internalmessage.ByContent(opts...))
		case "created_at":
			orderTerms = append(orderTerms, internalmessage.ByCreatedAt(opts...))
		case "status":
			orderTerms = append(orderTerms, internalmessage.ByStatus(opts...))
		case "title":
			orderTerms = append(orderTerms, internalmessage.ByTitle(opts...))
		case "type":
			orderTerms = append(orderTerms, internalmessage.ByType(opts...))
		}
	}

	if len(orderTerms) == 0 {
		orderTerms = append(orderTerms, internalmessage.ByCreatedAt(sql.OrderDesc()))
	}
	query.Order(orderTerms...)
}

func (r *internalMessageRepo) GetByID(ctx context.Context, id uint32) (*internalmessagev1.InternalMessage, error) {
	if r == nil || r.entClient == nil {
		return nil, fmt.Errorf("internal message repo is not initialized")
	}
	if id == 0 {
		return nil, internalmessagev1.ErrorBadRequest("message id is required")
	}

	entity, err := r.entClient.Client().InternalMessage.Query().
		Where(
			internalmessage.IDEQ(id),
			internalmessage.StatusNEQ(internalmessage.StatusDeleted),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, internalmessagev1.ErrorNotFound("message not found")
		}
		r.log.Errorf("get internal message failed: %s", err.Error())
		return nil, err
	}
	if entity.Status != nil && *entity.Status == internalmessage.StatusRevoked {
		viewerUserID := viewerUserIDFromContext(ctx)
		if entity.SenderID == nil || *entity.SenderID != viewerUserID {
			return nil, internalmessagev1.ErrorNotFound("message not found")
		}
	}
	items := toInternalMessageDTOs(ctx, r.entClient.Client(), r.mapper, []*ent.InternalMessage{entity})
	if len(items) == 0 {
		return nil, internalmessagev1.ErrorNotFound("message not found")
	}
	return items[0], nil
}

func (r *internalMessageRepo) UpdateByID(ctx context.Context, id uint32, data *internalmessagev1.InternalMessage) error {
	if r == nil || r.entClient == nil {
		return fmt.Errorf("internal message repo is not initialized")
	}
	if id == 0 || data == nil {
		return internalmessagev1.ErrorBadRequest("invalid update payload")
	}

	now := time.Now()
	builder := r.entClient.Client().InternalMessage.UpdateOneID(id).
		SetNillableTitle(data.Title).
		SetNillableContent(data.Content).
		SetNillableCategoryID(data.CategoryId).
		SetNillableSenderID(data.SenderId).
		SetNillableUpdatedBy(data.UpdatedBy).
		SetUpdatedAt(now)

	if data.Status != nil {
		builder.SetStatus(internalmessage.Status(data.GetStatus().String()))
	}
	if data.Type != nil {
		builder.SetType(internalmessage.Type(data.GetType().String()))
	}

	if _, err := builder.Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return internalmessagev1.ErrorNotFound("message not found")
		}
		r.log.Errorf("update internal message failed: %s", err.Error())
		return err
	}
	return nil
}

func (r *internalMessageRepo) DeleteByID(ctx context.Context, id uint32) error {
	if r == nil || r.entClient == nil {
		return fmt.Errorf("internal message repo is not initialized")
	}
	if id == 0 {
		return internalmessagev1.ErrorBadRequest("message id is required")
	}
	if err := r.entClient.Client().InternalMessage.DeleteOneID(id).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return internalmessagev1.ErrorNotFound("message not found")
		}
		r.log.Errorf("delete internal message failed: %s", err.Error())
		return err
	}
	return nil
}

func (r *internalMessageRepo) CreateDraft(ctx context.Context, data *internalmessagev1.InternalMessage) (*internalmessagev1.InternalMessage, error) {
	if r == nil || r.entClient == nil {
		return nil, fmt.Errorf("internal message repo is not initialized")
	}
	if data == nil {
		return nil, internalmessagev1.ErrorBadRequest("message data is required")
	}

	now := time.Now()
	builder := r.entClient.Client().InternalMessage.Create().
		SetNillableTenantID(data.TenantId).
		SetNillableTitle(data.Title).
		SetNillableContent(data.Content).
		SetNillableCategoryID(data.CategoryId).
		SetNillableCreatedBy(data.CreatedBy).
		SetCreatedAt(now)

	if data.SenderId != nil {
		builder.SetSenderID(data.GetSenderId())
	}
	if data.Status != nil {
		builder.SetStatus(internalmessage.Status(data.GetStatus().String()))
	}
	if data.Type != nil {
		builder.SetType(internalmessage.Type(data.GetType().String()))
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("create internal message failed: %s", err.Error())
		return nil, err
	}
	return r.mapper.ToDTO(entity), nil
}

func (r *internalMessageRepo) CreateAndSend(ctx context.Context, req SendMessageArgs) (uint32, error) {
	if r == nil || r.entClient == nil {
		return 0, fmt.Errorf("internal message repo is not initialized")
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Content) == "" {
		return 0, internalmessagev1.ErrorBadRequest("title and content are required")
	}
	if len(req.RecipientIDs) == 0 {
		return 0, internalmessagev1.ErrorBadRequest("recipient ids are required")
	}

	recipientIDs := dedupeUint32(req.RecipientIDs)
	sort.SliceStable(recipientIDs, func(i, j int) bool { return recipientIDs[i] < recipientIDs[j] })
	now := time.Now()

	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		return 0, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	create := tx.Client().InternalMessage.Create().
		SetTitle(req.Title).
		SetContent(req.Content).
		SetSenderID(req.SenderID).
		SetStatus(internalmessage.StatusPublished).
		SetType(internalmessage.Type(req.Type.String())).
		SetCreatedAt(now)
	if req.TenantID > 0 {
		create.SetTenantID(req.TenantID)
	}
	if req.CategoryID != nil {
		create.SetCategoryID(*req.CategoryID)
	}
	if req.SenderID > 0 {
		create.SetCreatedBy(req.SenderID)
	}

	messageEntity, err := create.Save(ctx)
	if err != nil {
		r.log.Errorf("insert internal message failed: %s", err.Error())
		return 0, err
	}

	recipientBuilders := make([]*ent.InternalMessageRecipientCreate, 0, len(recipientIDs))
	for _, uid := range recipientIDs {
		if uid == 0 {
			continue
		}
		builder := tx.Client().InternalMessageRecipient.Create().
			SetMessageID(messageEntity.ID).
			SetRecipientUserID(uid).
			SetStatus(internalmessagerecipient.StatusSent).
			SetCreatedAt(now).
			SetUpdatedAt(now)
		if req.TenantID > 0 {
			builder.SetTenantID(req.TenantID)
		}
		recipientBuilders = append(recipientBuilders, builder)
	}
	if len(recipientBuilders) == 0 {
		return 0, internalmessagev1.ErrorBadRequest("no valid recipients")
	}
	if _, err := tx.Client().InternalMessageRecipient.CreateBulk(recipientBuilders...).Save(ctx); err != nil {
		r.log.Errorf("insert internal message recipients failed: %s", err.Error())
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	committed = true
	return messageEntity.ID, nil
}

func (r *internalMessageRepo) Revoke(ctx context.Context, req RevokeMessageArgs) error {
	if r == nil || r.entClient == nil {
		return fmt.Errorf("internal message repo is not initialized")
	}
	if req.MessageID == 0 {
		return internalmessagev1.ErrorBadRequest("message id is required")
	}
	if req.SenderID == 0 {
		return internalmessagev1.ErrorUnauthorized("sender id is required")
	}

	now := time.Now()

	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	messageEntity, err := tx.Client().InternalMessage.Query().
		Where(
			internalmessage.IDEQ(req.MessageID),
			internalmessage.StatusNEQ(internalmessage.StatusDeleted),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return internalmessagev1.ErrorNotFound("message not found")
		}
		r.log.Errorf("load internal message for revoke failed: %s", err.Error())
		return err
	}

	if messageEntity.SenderID == nil || *messageEntity.SenderID != req.SenderID {
		return internalmessagev1.ErrorForbidden("only sender can revoke message")
	}
	if messageEntity.Status != nil && *messageEntity.Status == internalmessage.StatusRevoked {
		return internalmessagev1.ErrorConflict("message has already been revoked")
	}

	if now.After(messageEntity.CreatedAt.Add(30 * time.Minute)) {
		return internalmessagev1.ErrorPreconditionFailed("message cannot be revoked after 30 minutes")
	}

	readCount, err := tx.Client().InternalMessageRecipient.Query().
		Where(
			internalmessagerecipient.MessageIDEQ(req.MessageID),
			internalmessagerecipient.Or(
				internalmessagerecipient.ReadAtNotNil(),
				internalmessagerecipient.StatusEQ(internalmessagerecipient.StatusRead),
			),
		).
		Count(ctx)
	if err != nil {
		r.log.Errorf("count message read recipients failed: %s", err.Error())
		return err
	}
	if readCount > 0 {
		return internalmessagev1.ErrorPreconditionFailed("message cannot be revoked after any recipient has read it")
	}

	if _, err := tx.Client().InternalMessage.UpdateOneID(req.MessageID).
		SetStatus(internalmessage.StatusRevoked).
		SetUpdatedAt(now).
		SetUpdatedBy(req.SenderID).
		Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return internalmessagev1.ErrorNotFound("message not found")
		}
		r.log.Errorf("revoke internal message failed: %s", err.Error())
		return err
	}

	if _, err := tx.Client().InternalMessageRecipient.Update().
		Where(internalmessagerecipient.MessageIDEQ(req.MessageID)).
		SetStatus(internalmessagerecipient.StatusRevoked).
		SetUpdatedAt(now).
		Save(ctx); err != nil {
		r.log.Errorf("revoke internal message recipients failed: %s", err.Error())
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (r *internalMessageRepo) ListAllUserIDs(ctx context.Context) ([]uint32, error) {
	if r == nil || r.entClient == nil {
		return nil, fmt.Errorf("internal message repo is not initialized")
	}

	userRows, err := r.entClient.Client().User.Query().All(ctx)
	if err != nil {
		r.log.Errorf("list users for internal message failed: %s", err.Error())
		return nil, err
	}

	ids := make([]uint32, 0, len(userRows))
	for _, userRow := range userRows {
		if userRow == nil || userRow.ID == 0 {
			continue
		}
		ids = append(ids, userRow.ID)
	}
	return dedupeUint32(ids), nil
}

func toInternalMessageDTOs(
	ctx context.Context,
	client *ent.Client,
	mapperIns *mapper.CopierMapper[internalmessagev1.InternalMessage, ent.InternalMessage],
	entities []*ent.InternalMessage,
) []*internalmessagev1.InternalMessage {
	if mapperIns == nil || len(entities) == 0 {
		return nil
	}
	tenantNameMap := loadTenantNamesForInternalMessages(ctx, client, entities)
	items := make([]*internalmessagev1.InternalMessage, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		item := mapperIns.ToDTO(entity)
		if item == nil {
			continue
		}
		if entity.TenantID != nil {
			if tenantName := resolvedTenantName(tenantNameMap, *entity.TenantID); tenantName != "" {
				item.TenantName = &tenantName
			}
		}
		items = append(items, item)
	}
	return items
}

func resolvedTenantName(tenantNameMap map[uint32]string, tenantID uint32) string {
	return strings.TrimSpace(resolvedTenantDisplayName(tenantNameMap, tenantID))
}

func loadTenantNamesForInternalMessages(
	ctx context.Context,
	client *ent.Client,
	entities []*ent.InternalMessage,
) map[uint32]string {
	if client == nil || len(entities) == 0 {
		return nil
	}
	tenantIDs := make([]uint32, 0, len(entities))
	seen := make(map[uint32]struct{}, len(entities))
	for _, entity := range entities {
		if entity == nil || entity.TenantID == nil || *entity.TenantID == platformTenantID {
			continue
		}
		tenantID := *entity.TenantID
		if _, ok := seen[tenantID]; ok {
			continue
		}
		seen[tenantID] = struct{}{}
		tenantIDs = append(tenantIDs, tenantID)
	}
	if len(tenantIDs) == 0 {
		return nil
	}
	rows, err := client.Tenant.Query().
		Where(tenant.IDIn(tenantIDs...)).
		All(ctx)
	if err != nil {
		return nil
	}
	result := make(map[uint32]string, len(rows))
	for _, row := range rows {
		if row == nil || row.Name == nil {
			continue
		}
		result[row.ID] = *row.Name
	}
	return result
}

func dedupeUint32(values []uint32) []uint32 {
	if len(values) == 0 {
		return nil
	}
	result := make([]uint32, 0, len(values))
	seen := make(map[uint32]struct{}, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func viewerUserIDFromContext(ctx context.Context) uint32 {
	viewer, ok := crudviewer.FromContext(ctx)
	if !ok || viewer == nil {
		return 0
	}
	if viewer.UserID() == 0 {
		return 0
	}
	return uint32(viewer.UserID())
}
