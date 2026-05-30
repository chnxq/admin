package repo

import (
	"context"
	"testing"
	"time"

	internalmessagev1 "admin/api/gen/internal_message/v1"
	"admin/internal/data/ent"
	"admin/internal/data/ent/internalmessage"
	"admin/internal/data/ent/internalmessagerecipient"
	_ "admin/internal/data/ent/runtime"
	entsql "entgo.io/ent/dialect/sql"
	entCrud "github.com/chnxq/x-crud/entgo"
	crudviewer "github.com/chnxq/x-crud/viewer"
	_ "github.com/mattn/go-sqlite3"
)

type internalMessageTestViewer struct{}

func (internalMessageTestViewer) UserID() uint64                    { return 1 }
func (internalMessageTestViewer) TenantID() uint64                  { return 1 }
func (internalMessageTestViewer) OrgUnitID() uint64                 { return 0 }
func (internalMessageTestViewer) Permissions() []string             { return []string{"*"} }
func (internalMessageTestViewer) Roles() []string                   { return []string{"system"} }
func (internalMessageTestViewer) DataScope() []crudviewer.DataScope { return nil }
func (internalMessageTestViewer) TraceID() string                   { return "" }
func (internalMessageTestViewer) HasPermission(string, string) bool { return true }
func (internalMessageTestViewer) IsPlatformContext() bool           { return true }
func (internalMessageTestViewer) IsTenantContext() bool             { return true }
func (internalMessageTestViewer) IsSystemContext() bool             { return true }
func (internalMessageTestViewer) ShouldAudit() bool                 { return false }

func withInternalMessageTestViewer(ctx context.Context) context.Context {
	return crudviewer.WithContext(ctx, internalMessageTestViewer{})
}

func newInternalMessageRepoForTest(t *testing.T, dbName string) (*internalMessageRepo, *ent.Client) {
	t.Helper()
	driver, err := entsql.Open("sqlite3", "file:"+dbName+"?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("open sqlite driver failed: %v", err)
	}
	client := ent.NewClient(ent.Driver(driver))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema failed: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = driver.Close()
	})

	entClient := entCrud.NewEntClient[*ent.Client](client, driver)
	repo := NewInternalMessageRepo(nil, entClient)
	return repo, client
}

func mustCreateMessageForRevokeTest(
	t *testing.T,
	client *ent.Client,
	senderID uint32,
	createdAt time.Time,
	status internalmessage.Status,
	recipientStatuses []internalmessagerecipient.Status,
	recipientReadAt []time.Time,
) uint32 {
	t.Helper()
	seedCtx := withInternalMessageTestViewer(context.Background())
	messageEntity, err := client.InternalMessage.Create().
		SetTitle("test title").
		SetContent("test content").
		SetSenderID(senderID).
		SetStatus(status).
		SetType(internalmessage.TypeNotification).
		SetCreatedAt(createdAt).
		SetUpdatedAt(createdAt).
		Save(seedCtx)
	if err != nil {
		t.Fatalf("create internal message failed: %v", err)
	}

	for index, recipientStatus := range recipientStatuses {
		builder := client.InternalMessageRecipient.Create().
			SetMessageID(messageEntity.ID).
			SetRecipientUserID(uint32(index + 100)).
			SetStatus(recipientStatus).
			SetCreatedAt(createdAt).
			SetUpdatedAt(createdAt)
		if index < len(recipientReadAt) && !recipientReadAt[index].IsZero() {
			builder.SetReadAt(recipientReadAt[index])
		}
		if _, err := builder.Save(seedCtx); err != nil {
			t.Fatalf("create internal message recipient failed: %v", err)
		}
	}

	return messageEntity.ID
}

func TestInternalMessageRepoRevokeRejectsAfter30Minutes(t *testing.T) {
	repo, client := newInternalMessageRepoForTest(t, "internal-message-revoke-time-window")

	messageID := mustCreateMessageForRevokeTest(
		t,
		client,
		7,
		time.Now().Add(-31*time.Minute),
		internalmessage.StatusPublished,
		[]internalmessagerecipient.Status{internalmessagerecipient.StatusSent},
		nil,
	)

	err := repo.Revoke(withInternalMessageTestViewer(context.Background()), RevokeMessageArgs{
		MessageID: messageID,
		SenderID:  7,
	})
	if err == nil {
		t.Fatalf("expected revoke to fail when message is older than 30 minutes")
	}
	if !internalmessagev1.IsPreconditionFailed(err) {
		t.Fatalf("expected precondition failed, got: %v", err)
	}
}

func TestInternalMessageRepoRevokeRejectsWhenAnyRecipientRead(t *testing.T) {
	repo, client := newInternalMessageRepoForTest(t, "internal-message-revoke-read-block")

	createdAt := time.Now().Add(-5 * time.Minute)
	messageID := mustCreateMessageForRevokeTest(
		t,
		client,
		8,
		createdAt,
		internalmessage.StatusPublished,
		[]internalmessagerecipient.Status{
			internalmessagerecipient.StatusSent,
			internalmessagerecipient.StatusRead,
		},
		[]time.Time{
			time.Time{},
			time.Now(),
		},
	)

	err := repo.Revoke(withInternalMessageTestViewer(context.Background()), RevokeMessageArgs{
		MessageID: messageID,
		SenderID:  8,
	})
	if err == nil {
		t.Fatalf("expected revoke to fail when any recipient has read the message")
	}
	if !internalmessagev1.IsPreconditionFailed(err) {
		t.Fatalf("expected precondition failed, got: %v", err)
	}
}

func TestInternalMessageRepoRevokeRejectsNonSender(t *testing.T) {
	repo, client := newInternalMessageRepoForTest(t, "internal-message-revoke-sender-check")

	messageID := mustCreateMessageForRevokeTest(
		t,
		client,
		11,
		time.Now().Add(-2*time.Minute),
		internalmessage.StatusPublished,
		[]internalmessagerecipient.Status{internalmessagerecipient.StatusSent},
		nil,
	)

	err := repo.Revoke(withInternalMessageTestViewer(context.Background()), RevokeMessageArgs{
		MessageID: messageID,
		SenderID:  12,
	})
	if err == nil {
		t.Fatalf("expected revoke to fail for non-sender")
	}
	if !internalmessagev1.IsForbidden(err) {
		t.Fatalf("expected forbidden, got: %v", err)
	}
}

func TestInternalMessageRepoRevokeSuccessMarksMessageAndRecipientsRevoked(t *testing.T) {
	repo, client := newInternalMessageRepoForTest(t, "internal-message-revoke-success")

	messageID := mustCreateMessageForRevokeTest(
		t,
		client,
		21,
		time.Now().Add(-10*time.Minute),
		internalmessage.StatusPublished,
		[]internalmessagerecipient.Status{
			internalmessagerecipient.StatusSent,
			internalmessagerecipient.StatusReceived,
		},
		nil,
	)

	ctx := withInternalMessageTestViewer(context.Background())
	err := repo.Revoke(ctx, RevokeMessageArgs{
		MessageID: messageID,
		SenderID:  21,
	})
	if err != nil {
		t.Fatalf("expected revoke success, got: %v", err)
	}

	messageEntity, err := client.InternalMessage.Get(ctx, messageID)
	if err != nil {
		t.Fatalf("load message failed: %v", err)
	}
	if messageEntity.Status == nil || *messageEntity.Status != internalmessage.StatusRevoked {
		t.Fatalf("expected message status revoked, got: %+v", messageEntity.Status)
	}

	recipientRows, err := client.InternalMessageRecipient.Query().
		Where(internalmessagerecipient.MessageIDEQ(messageID)).
		All(ctx)
	if err != nil {
		t.Fatalf("load recipients failed: %v", err)
	}
	if len(recipientRows) == 0 {
		t.Fatalf("expected recipients to exist")
	}
	for _, item := range recipientRows {
		if item.Status == nil || *item.Status != internalmessagerecipient.StatusRevoked {
			t.Fatalf("expected recipient status revoked, got: %+v", item.Status)
		}
	}
}

func TestInternalMessageRepoGetByIDHidesRevokedMessageForNonSender(t *testing.T) {
	repo, client := newInternalMessageRepoForTest(t, "internal-message-get-by-id-revoked-visibility")
	ctx := withInternalMessageTestViewer(context.Background())
	now := time.Now()

	messageEntity, err := client.InternalMessage.Create().
		SetTitle("revoked-visible-only-to-sender").
		SetContent("content").
		SetSenderID(31).
		SetStatus(internalmessage.StatusRevoked).
		SetType(internalmessage.TypeNotification).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		t.Fatalf("create revoked message failed: %v", err)
	}

	_, err = repo.GetByID(ctx, messageEntity.ID)
	if err == nil {
		t.Fatalf("expected revoked message hidden for non-sender")
	}
	if !internalmessagev1.IsNotFound(err) {
		t.Fatalf("expected not found, got: %v", err)
	}
}
