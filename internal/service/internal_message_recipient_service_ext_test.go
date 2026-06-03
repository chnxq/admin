package service

import (
	"context"
	"testing"

	v11 "admin/api/gen/internal_message/v1"
	"admin/internal/data/repo"
	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
	crudviewer "github.com/chnxq/x-crud/viewer"
)

type internalMessageRecipientServiceTestViewer struct {
	userID   uint64
	platform bool
	tenant   bool
	tenantID uint64
}

func (v internalMessageRecipientServiceTestViewer) UserID() uint64                    { return v.userID }
func (v internalMessageRecipientServiceTestViewer) TenantID() uint64                  { return v.tenantID }
func (v internalMessageRecipientServiceTestViewer) OrgUnitID() uint64                 { return 0 }
func (v internalMessageRecipientServiceTestViewer) Permissions() []string             { return []string{"*"} }
func (v internalMessageRecipientServiceTestViewer) Roles() []string                   { return []string{"system"} }
func (v internalMessageRecipientServiceTestViewer) DataScope() []crudviewer.DataScope { return nil }
func (v internalMessageRecipientServiceTestViewer) TraceID() string                   { return "" }
func (v internalMessageRecipientServiceTestViewer) HasPermission(string, string) bool { return true }
func (v internalMessageRecipientServiceTestViewer) IsPlatformContext() bool           { return v.platform }
func (v internalMessageRecipientServiceTestViewer) IsTenantContext() bool             { return v.tenant }
func (v internalMessageRecipientServiceTestViewer) IsSystemContext() bool             { return v.platform && !v.tenant }
func (v internalMessageRecipientServiceTestViewer) ShouldAudit() bool                 { return false }

type stubInternalMessageRecipientRepo struct {
	lastDeleteRecipientIDs []uint32
	lastDeleteUserID       uint32
	lastListReq            repo.InboxPagingRequest
	lastListUserID         uint32
	lastReadRecipientIDs   []uint32
	lastReadUserID         uint32
	listResp               *v11.ListUserInboxResponse
}

func (r *stubInternalMessageRecipientRepo) ListUserInbox(_ context.Context, userID uint32, req repo.InboxPagingRequest) (*v11.ListUserInboxResponse, error) {
	r.lastListUserID = userID
	r.lastListReq = req
	if r.listResp != nil {
		return r.listResp, nil
	}
	return &v11.ListUserInboxResponse{}, nil
}

func (r *stubInternalMessageRecipientRepo) DeleteFromInbox(_ context.Context, userID uint32, recipientIDs []uint32) error {
	r.lastDeleteUserID = userID
	r.lastDeleteRecipientIDs = append([]uint32(nil), recipientIDs...)
	return nil
}

func (r *stubInternalMessageRecipientRepo) MarkAsRead(_ context.Context, userID uint32, recipientIDs []uint32) error {
	r.lastReadUserID = userID
	r.lastReadRecipientIDs = append([]uint32(nil), recipientIDs...)
	return nil
}

func TestInternalMessageRecipientServiceListUserInboxUsesViewerUserID(t *testing.T) {
	repoStub := &stubInternalMessageRecipientRepo{
		listResp: &v11.ListUserInboxResponse{
			Total: 1,
			Items: []*v11.InternalMessageRecipient{{Id: uint32Ptr(99)}},
		},
	}
	svc := &InternalMessageRecipientService{internalMessageRecipientRepo: repoStub}
	ctx := crudviewer.WithContext(context.Background(), internalMessageRecipientServiceTestViewer{
		userID:   88,
		platform: true,
	})

	resp, err := svc.listUserInboxImpl(ctx, &paginationv1.PagingRequest{
		Page:     uint32Ptr(2),
		PageSize: uint32Ptr(15),
	})
	if err != nil {
		t.Fatalf("listUserInboxImpl failed: %v", err)
	}
	if resp.GetTotal() != 1 || len(resp.GetItems()) != 1 || resp.GetItems()[0].GetId() != 99 {
		t.Fatalf("unexpected inbox response: %+v", resp)
	}
	if repoStub.lastListUserID != 88 {
		t.Fatalf("expected viewer user id 88, got %d", repoStub.lastListUserID)
	}
	if repoStub.lastListReq.Limit != 15 || repoStub.lastListReq.Offset != 15 {
		t.Fatalf("unexpected paging request: %+v", repoStub.lastListReq)
	}
}

func TestInternalMessageRecipientServiceDeleteAndReadUseViewerUserIDWhenRequestIsZero(t *testing.T) {
	repoStub := &stubInternalMessageRecipientRepo{}
	svc := &InternalMessageRecipientService{internalMessageRecipientRepo: repoStub}
	ctx := crudviewer.WithContext(context.Background(), internalMessageRecipientServiceTestViewer{
		userID:   66,
		platform: true,
	})

	if _, err := svc.deleteNotificationFromInboxImpl(ctx, &v11.DeleteNotificationFromInboxRequest{
		UserId:       0,
		RecipientIds: []uint32{1, 2},
	}); err != nil {
		t.Fatalf("deleteNotificationFromInboxImpl failed: %v", err)
	}
	if repoStub.lastDeleteUserID != 66 {
		t.Fatalf("expected delete user id 66, got %d", repoStub.lastDeleteUserID)
	}
	if len(repoStub.lastDeleteRecipientIDs) != 2 || repoStub.lastDeleteRecipientIDs[0] != 1 || repoStub.lastDeleteRecipientIDs[1] != 2 {
		t.Fatalf("unexpected delete recipient ids: %+v", repoStub.lastDeleteRecipientIDs)
	}

	if _, err := svc.markNotificationAsReadImpl(ctx, &v11.MarkNotificationAsReadRequest{
		UserId:       0,
		RecipientIds: []uint32{3, 4},
	}); err != nil {
		t.Fatalf("markNotificationAsReadImpl failed: %v", err)
	}
	if repoStub.lastReadUserID != 66 {
		t.Fatalf("expected read user id 66, got %d", repoStub.lastReadUserID)
	}
	if len(repoStub.lastReadRecipientIDs) != 2 || repoStub.lastReadRecipientIDs[0] != 3 || repoStub.lastReadRecipientIDs[1] != 4 {
		t.Fatalf("unexpected read recipient ids: %+v", repoStub.lastReadRecipientIDs)
	}
}

func TestCurrentInboxUserIDRejectsMissingViewerUser(t *testing.T) {
	_, err := currentInboxUserID(context.Background(), 0)
	if err == nil {
		t.Fatalf("expected error for missing viewer user id")
	}
}
