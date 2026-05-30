package repo

import (
	internalmessagev1 "admin/api/gen/internal_message/v1"
	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
)

type PagingRequest struct {
	Limit   int
	Offset  int
	Sorting []*paginationv1.Sorting
}

type InboxPagingRequest struct {
	Limit  int
	Offset int
}

type SendMessageArgs struct {
	Title        string
	Content      string
	Type         internalmessagev1.InternalMessage_Type
	CategoryID   *uint32
	SenderID     uint32
	TenantID     uint32
	RecipientIDs []uint32
}

type RevokeMessageArgs struct {
	MessageID uint32
	SenderID  uint32
}
