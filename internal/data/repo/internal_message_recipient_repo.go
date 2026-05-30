package repo

import (
	"context"

	internalmessagev1 "admin/api/gen/internal_message/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/internalmessagerecipient"
	entCrud "github.com/chnxq/x-crud/entgo"
	"github.com/chnxq/x-utils/copierutil"
	"github.com/chnxq/x-utils/mapper"
	"github.com/chnxq/xkitmod/log"
	"github.com/chnxq/xkitpkg/app"
)

type InternalMessageRecipientRepo interface {
	ListUserInbox(ctx context.Context, userID uint32, req InboxPagingRequest) (*internalmessagev1.ListUserInboxResponse, error)
	DeleteFromInbox(ctx context.Context, userID uint32, recipientIDs []uint32) error
	MarkAsRead(ctx context.Context, userID uint32, recipientIDs []uint32) error
}

type internalMessageRecipientRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[internalmessagev1.InternalMessageRecipient, ent.InternalMessageRecipient]
}

func NewInternalMessageRecipientRepo(ctx *app.AppCtx, entClient *entCrud.EntClient[*ent.Client]) *internalMessageRecipientRepo {
	repo := &internalMessageRecipientRepo{
		entClient: entClient,
		mapper: mapper.NewCopierMapper[
			internalmessagev1.InternalMessageRecipient,
			ent.InternalMessageRecipient,
		](),
	}
	if ctx != nil {
		repo.log = ctx.NewLoggerHelper("internal_message_recipient/data")
	}
	repo.initMapper()
	return repo
}

func (r *internalMessageRecipientRepo) initMapper() {
	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(
		mapper.NewEnumTypeConverter[
			internalmessagev1.InternalMessageRecipient_Status,
			internalmessagerecipient.Status,
		](
			internalmessagev1.InternalMessageRecipient_Status_name,
			internalmessagev1.InternalMessageRecipient_Status_value,
		).NewConverterPair(),
	)
}
