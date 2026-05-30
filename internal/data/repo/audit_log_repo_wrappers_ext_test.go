package repo

import (
	"context"
	"testing"
	"time"

	auditv1 "admin/api/gen/audit/v1"
	v1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
)

type mockApiAuditLogRepo struct {
	listCalled      bool
	writeCalled     bool
	analyticsCalled bool
}

func (m *mockApiAuditLogRepo) List(context.Context, *v1.PagingRequest) (*auditv1.ListApiAuditLogResponse, error) {
	m.listCalled = true
	return &auditv1.ListApiAuditLogResponse{}, nil
}

func (m *mockApiAuditLogRepo) Get(context.Context, *auditv1.GetApiAuditLogRequest) (*auditv1.ApiAuditLog, error) {
	return &auditv1.ApiAuditLog{}, nil
}

func (m *mockApiAuditLogRepo) WriteApiAuditLog(context.Context, *auditv1.ApiAuditLog) error {
	m.writeCalled = true
	return nil
}

func (m *mockApiAuditLogRepo) AnalyticsSummary(context.Context, time.Time) (*ApiAuditAnalyticsSummary, error) {
	m.analyticsCalled = true
	return &ApiAuditAnalyticsSummary{TotalAccesses: 42}, nil
}

type mockLoginAuditLogRepo struct {
	writeCalled bool
}

func (m *mockLoginAuditLogRepo) List(context.Context, *v1.PagingRequest) (*auditv1.ListLoginAuditLogResponse, error) {
	return &auditv1.ListLoginAuditLogResponse{}, nil
}

func (m *mockLoginAuditLogRepo) Get(context.Context, *auditv1.GetLoginAuditLogRequest) (*auditv1.LoginAuditLog, error) {
	return &auditv1.LoginAuditLog{}, nil
}

func (m *mockLoginAuditLogRepo) WriteLoginAuditLog(context.Context, *auditv1.LoginAuditLog) error {
	m.writeCalled = true
	return nil
}

type mockPermissionAuditLogRepo struct {
	writeCalled bool
}

func (m *mockPermissionAuditLogRepo) List(context.Context, *v1.PagingRequest) (*auditv1.ListPermissionAuditLogResponse, error) {
	return &auditv1.ListPermissionAuditLogResponse{}, nil
}

func (m *mockPermissionAuditLogRepo) Get(context.Context, *auditv1.GetPermissionAuditLogRequest) (*auditv1.PermissionAuditLog, error) {
	return &auditv1.PermissionAuditLog{}, nil
}

func (m *mockPermissionAuditLogRepo) WritePermissionAuditLog(context.Context, *auditv1.PermissionAuditLog) error {
	m.writeCalled = true
	return nil
}

func TestWrapApiAuditLogRepo_DelegatesWriterAndAnalytics(t *testing.T) {
	base := &mockApiAuditLogRepo{}
	wrapped := WrapApiAuditLogRepo(base)

	writer, ok := wrapped.(ApiAuditLogWriter)
	if !ok {
		t.Fatalf("wrapped api audit repo must keep ApiAuditLogWriter capability")
	}
	if err := writer.WriteApiAuditLog(context.Background(), &auditv1.ApiAuditLog{}); err != nil {
		t.Fatalf("WriteApiAuditLog failed: %v", err)
	}
	if !base.writeCalled {
		t.Fatalf("expected base WriteApiAuditLog to be called")
	}

	reader, ok := wrapped.(apiAuditLogAnalyticsReader)
	if !ok {
		t.Fatalf("wrapped api audit repo must keep analytics capability")
	}
	summary, err := reader.AnalyticsSummary(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("AnalyticsSummary failed: %v", err)
	}
	if !base.analyticsCalled {
		t.Fatalf("expected base AnalyticsSummary to be called")
	}
	if summary == nil || summary.TotalAccesses != 42 {
		t.Fatalf("unexpected analytics summary: %+v", summary)
	}
}

func TestWrapLoginAuditLogRepo_DelegatesWriter(t *testing.T) {
	base := &mockLoginAuditLogRepo{}
	wrapped := WrapLoginAuditLogRepo(base)

	writer, ok := wrapped.(LoginAuditLogWriter)
	if !ok {
		t.Fatalf("wrapped login audit repo must keep LoginAuditLogWriter capability")
	}
	if err := writer.WriteLoginAuditLog(context.Background(), &auditv1.LoginAuditLog{}); err != nil {
		t.Fatalf("WriteLoginAuditLog failed: %v", err)
	}
	if !base.writeCalled {
		t.Fatalf("expected base WriteLoginAuditLog to be called")
	}
}

func TestWrapPermissionAuditLogRepo_DelegatesWriter(t *testing.T) {
	base := &mockPermissionAuditLogRepo{}
	wrapped := WrapPermissionAuditLogRepo(base)

	writer, ok := wrapped.(PermissionAuditLogWriter)
	if !ok {
		t.Fatalf("wrapped permission audit repo must keep PermissionAuditLogWriter capability")
	}
	if err := writer.WritePermissionAuditLog(context.Background(), &auditv1.PermissionAuditLog{}); err != nil {
		t.Fatalf("WritePermissionAuditLog failed: %v", err)
	}
	if !base.writeCalled {
		t.Fatalf("expected base WritePermissionAuditLog to be called")
	}
}
