package repo

import (
	"context"

	internalmessagev1 "admin/api/gen/internal_message/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/internalmessage"
	entCrud "github.com/chnxq/x-crud/entgo"
	"github.com/chnxq/x-utils/copierutil"
	"github.com/chnxq/x-utils/mapper"
	"github.com/chnxq/xkitmod/log"
	"github.com/chnxq/xkitpkg/app"
)

type InternalMessageRepo interface {
	ListByPaging(ctx context.Context, req PagingRequest) (*internalmessagev1.ListInternalMessageResponse, error)
	GetByID(ctx context.Context, id uint32) (*internalmessagev1.InternalMessage, error)
	UpdateByID(ctx context.Context, id uint32, data *internalmessagev1.InternalMessage) error
	DeleteByID(ctx context.Context, id uint32) error
	CreateDraft(ctx context.Context, data *internalmessagev1.InternalMessage) (*internalmessagev1.InternalMessage, error)
	CreateAndSend(ctx context.Context, req SendMessageArgs) (uint32, error)
	ListAllUserIDs(ctx context.Context) ([]uint32, error)
	Revoke(ctx context.Context, req RevokeMessageArgs) error
}

type internalMessageRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper *mapper.CopierMapper[internalmessagev1.InternalMessage, ent.InternalMessage]
}

func NewInternalMessageRepo(ctx *app.AppCtx, entClient *entCrud.EntClient[*ent.Client]) *internalMessageRepo {
	repo := &internalMessageRepo{
		entClient: entClient,
		mapper:    mapper.NewCopierMapper[internalmessagev1.InternalMessage, ent.InternalMessage](),
	}
	if ctx != nil {
		repo.log = ctx.NewLoggerHelper("internal_message/data")
	}
	repo.initMapper()
	return repo
}

func (r *internalMessageRepo) initMapper() {
	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(
		mapper.NewEnumTypeConverter[internalmessagev1.InternalMessage_Status, internalmessage.Status](
			internalmessagev1.InternalMessage_Status_name,
			internalmessagev1.InternalMessage_Status_value,
		).NewConverterPair(),
	)
	r.mapper.AppendConverters(
		mapper.NewEnumTypeConverter[internalmessagev1.InternalMessage_Type, internalmessage.Type](
			internalmessagev1.InternalMessage_Type_name,
			internalmessagev1.InternalMessage_Type_value,
		).NewConverterPair(),
	)
}
