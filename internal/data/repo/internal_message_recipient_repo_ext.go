package repo

import (
	"context"
	"fmt"
	"sort"
	"time"

	internalmessagev1 "admin/api/gen/internal_message/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/internalmessage"
	"admin/internal/data/ent/internalmessagerecipient"
	"admin/internal/data/ent/predicate"
	entsql "entgo.io/ent/dialect/sql"
)

func (r *internalMessageRecipientRepo) ListUserInbox(ctx context.Context, userID uint32, req InboxPagingRequest) (*internalmessagev1.ListUserInboxResponse, error) {
	if r == nil || r.entClient == nil {
		return nil, fmt.Errorf("internal message recipient repo is not initialized")
	}
	if userID == 0 {
		return nil, internalmessagev1.ErrorBadRequest("user id is required")
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

	visibleMessagePredicate := recipientVisibleMessagePredicate()
	basePredicates := []predicate.InternalMessageRecipient{
		internalmessagerecipient.RecipientUserIDEQ(userID),
		internalmessagerecipient.StatusNEQ(internalmessagerecipient.StatusRevoked),
		internalmessagerecipient.StatusNEQ(internalmessagerecipient.StatusDeleted),
		visibleMessagePredicate,
	}

	baseQuery := r.entClient.Client().InternalMessageRecipient.Query().
		Where(basePredicates...).
		Order(internalmessagerecipient.ByCreatedAt()).
		Limit(limit).
		Offset(offset)

	entities, err := baseQuery.All(ctx)
	if err != nil {
		r.log.Errorf("list inbox recipients failed: %s", err.Error())
		return nil, err
	}
	total, err := r.entClient.Client().InternalMessageRecipient.Query().
		Where(basePredicates...).
		Count(ctx)
	if err != nil {
		r.log.Errorf("count inbox recipients failed: %s", err.Error())
		return nil, err
	}

	messageIDs := make([]uint32, 0, len(entities))
	for _, entity := range entities {
		if entity == nil || entity.MessageID == nil || *entity.MessageID == 0 {
			continue
		}
		messageIDs = append(messageIDs, *entity.MessageID)
	}
	messageMap, err := r.loadMessagesByIDs(ctx, messageIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*internalmessagev1.InternalMessageRecipient, 0, len(entities))
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		dto := r.mapper.ToDTO(entity)
		if dto == nil {
			continue
		}
		msg := messageMap[dto.GetMessageId()]
		if msg == nil {
			continue
		}
		if msg.Status != nil &&
			(*msg.Status == internalmessage.StatusRevoked || *msg.Status == internalmessage.StatusDeleted) {
			continue
		}
		dto.Title = msg.Title
		dto.Content = msg.Content
		items = append(items, dto)
	}

	return &internalmessagev1.ListUserInboxResponse{
		Items: items,
		Total: uint64(total),
	}, nil
}

func (r *internalMessageRecipientRepo) DeleteFromInbox(ctx context.Context, userID uint32, recipientIDs []uint32) error {
	if r == nil || r.entClient == nil {
		return fmt.Errorf("internal message recipient repo is not initialized")
	}
	if userID == 0 {
		return internalmessagev1.ErrorBadRequest("user id is required")
	}
	ids := dedupeUint32(recipientIDs)
	if len(ids) == 0 {
		return nil
	}

	_, err := r.entClient.Client().InternalMessageRecipient.Delete().
		Where(
			internalmessagerecipient.IDIn(ids...),
			internalmessagerecipient.RecipientUserIDEQ(userID),
		).
		Exec(ctx)
	if err != nil {
		r.log.Errorf("delete inbox notifications failed: %s", err.Error())
		return err
	}
	return nil
}

func (r *internalMessageRecipientRepo) MarkAsRead(ctx context.Context, userID uint32, recipientIDs []uint32) error {
	if r == nil || r.entClient == nil {
		return fmt.Errorf("internal message recipient repo is not initialized")
	}
	if userID == 0 {
		return internalmessagev1.ErrorBadRequest("user id is required")
	}
	ids := dedupeUint32(recipientIDs)
	if len(ids) == 0 {
		return nil
	}
	sort.SliceStable(ids, func(i, j int) bool { return ids[i] < ids[j] })

	now := time.Now()
	_, err := r.entClient.Client().InternalMessageRecipient.Update().
		Where(
			internalmessagerecipient.IDIn(ids...),
			internalmessagerecipient.RecipientUserIDEQ(userID),
		).
		SetStatus(internalmessagerecipient.StatusRead).
		SetReadAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		r.log.Errorf("mark inbox notifications as read failed: %s", err.Error())
		return err
	}
	return nil
}

func (r *internalMessageRecipientRepo) loadMessagesByIDs(ctx context.Context, ids []uint32) (map[uint32]*ent.InternalMessage, error) {
	result := make(map[uint32]*ent.InternalMessage, len(ids))
	ids = dedupeUint32(ids)
	if len(ids) == 0 {
		return result, nil
	}

	messageRows, err := r.entClient.Client().InternalMessage.Query().
		Where(internalmessage.IDIn(ids...)).
		All(ctx)
	if err != nil {
		r.log.Errorf("load inbox related messages failed: %s", err.Error())
		return nil, err
	}
	for _, messageRow := range messageRows {
		if messageRow == nil {
			continue
		}
		result[messageRow.ID] = messageRow
	}
	return result, nil
}

func recipientVisibleMessagePredicate() predicate.InternalMessageRecipient {
	return func(selector *entsql.Selector) {
		messageTable := entsql.Table(internalmessage.Table)
		selector.Where(
			entsql.Exists(
				entsql.Select().
					From(messageTable).
					Where(
						entsql.And(
							entsql.ColumnsEQ(selector.C(internalmessagerecipient.FieldMessageID), messageTable.C(internalmessage.FieldID)),
							entsql.NEQ(messageTable.C(internalmessage.FieldStatus), internalmessage.StatusRevoked),
							entsql.NEQ(messageTable.C(internalmessage.FieldStatus), internalmessage.StatusDeleted),
						),
					),
			),
		)
	}
}
