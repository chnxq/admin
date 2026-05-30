package repo

import (
	"context"
	"testing"
	"time"

	"admin/internal/data/ent"
	"admin/internal/data/ent/internalmessage"
	"admin/internal/data/ent/internalmessagerecipient"
	_ "admin/internal/data/ent/runtime"
	entsql "entgo.io/ent/dialect/sql"
	entCrud "github.com/chnxq/x-crud/entgo"
	_ "github.com/mattn/go-sqlite3"
)

func newInternalMessageRecipientRepoForTest(t *testing.T, dbName string) (*internalMessageRecipientRepo, *ent.Client) {
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
	repo := NewInternalMessageRecipientRepo(nil, entClient)
	return repo, client
}

func TestInternalMessageRecipientRepoListUserInboxFiltersRevokedItems(t *testing.T) {
	repo, client := newInternalMessageRecipientRepoForTest(t, "internal-message-inbox-filter-revoked")
	ctx := withInternalMessageTestViewer(context.Background())
	now := time.Now()

	activeMessage, err := client.InternalMessage.Create().
		SetTitle("active").
		SetContent("active-content").
		SetSenderID(1).
		SetStatus(internalmessage.StatusPublished).
		SetType(internalmessage.TypeNotification).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		t.Fatalf("create active message failed: %v", err)
	}
	revokedMessage, err := client.InternalMessage.Create().
		SetTitle("revoked").
		SetContent("revoked-content").
		SetSenderID(1).
		SetStatus(internalmessage.StatusRevoked).
		SetType(internalmessage.TypeNotification).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		t.Fatalf("create revoked message failed: %v", err)
	}

	if _, err := client.InternalMessageRecipient.Create().
		SetMessageID(activeMessage.ID).
		SetRecipientUserID(500).
		SetStatus(internalmessagerecipient.StatusSent).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx); err != nil {
		t.Fatalf("create active recipient failed: %v", err)
	}

	if _, err := client.InternalMessageRecipient.Create().
		SetMessageID(revokedMessage.ID).
		SetRecipientUserID(500).
		SetStatus(internalmessagerecipient.StatusSent).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx); err != nil {
		t.Fatalf("create revoked recipient failed: %v", err)
	}

	if _, err := client.InternalMessageRecipient.Create().
		SetMessageID(activeMessage.ID).
		SetRecipientUserID(500).
		SetStatus(internalmessagerecipient.StatusRevoked).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx); err != nil {
		t.Fatalf("create revoked-status recipient failed: %v", err)
	}

	result, err := repo.ListUserInbox(ctx, 500, InboxPagingRequest{
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListUserInbox failed: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected total 1 visible inbox item, got %d", result.Total)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 visible inbox item, got %d", len(result.Items))
	}
	if result.Items[0].GetMessageId() != activeMessage.ID {
		t.Fatalf("expected active message id %d, got %d", activeMessage.ID, result.Items[0].GetMessageId())
	}
}
