package bootstrap

import (
	"context"
	"testing"

	authenticationv1 "admin/api/gen/authentication/v1"
	identityv1 "admin/api/gen/identity/v1"
	"admin/internal/data/repo"
	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type stubBootstrapUserRepo struct {
	users []*identityv1.User
}

func (r stubBootstrapUserRepo) List(_ context.Context, req *paginationv1.PagingRequest) (*identityv1.ListUserResponse, error) {
	if req == nil || req.GetFilterExpr() == nil || len(req.GetFilterExpr().GetConditions()) == 0 {
		return &identityv1.ListUserResponse{Items: r.users}, nil
	}
	condition := req.GetFilterExpr().GetConditions()[0]
	value := condition.GetValue()
	items := make([]*identityv1.User, 0, 1)
	for _, user := range r.users {
		if user == nil {
			continue
		}
		switch condition.GetField() {
		case "email":
			if user.GetEmail() == value {
				items = append(items, user)
			}
		case "mobile":
			if user.GetMobile() == value {
				items = append(items, user)
			}
		}
	}
	return &identityv1.ListUserResponse{Items: items}, nil
}

func (r stubBootstrapUserRepo) Get(_ context.Context, req *identityv1.GetUserRequest) (*identityv1.User, error) {
	if req == nil {
		return nil, nil
	}
	switch q := req.GetQueryBy().(type) {
	case *identityv1.GetUserRequest_Id:
		for _, user := range r.users {
			if user != nil && user.GetId() == q.Id {
				return user, nil
			}
		}
	case *identityv1.GetUserRequest_Username:
		for _, user := range r.users {
			if user != nil && user.GetUsername() == q.Username {
				return user, nil
			}
		}
	}
	return nil, nil
}

func (stubBootstrapUserRepo) Create(context.Context, *identityv1.CreateUserRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (stubBootstrapUserRepo) Update(context.Context, *identityv1.UpdateUserRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (stubBootstrapUserRepo) Delete(context.Context, *identityv1.DeleteUserRequest) (*emptypb.Empty, error) {
	return nil, nil
}

func (stubBootstrapUserRepo) UserExists(context.Context, *identityv1.UserExistsRequest) (*identityv1.UserExistsResponse, error) {
	return nil, nil
}

var _ repo.UserRepo = (*stubBootstrapUserRepo)(nil)

func TestResolveLoginRequestUserByEmail(t *testing.T) {
	user := &identityv1.User{
		Id:       auditPtr(uint32(101)),
		TenantId: auditPtr(uint32(9)),
		Username: auditPtr("alice"),
		Email:    auditPtr("alice@example.com"),
	}
	info := resolveLoginRequestUser(context.Background(), &authenticationv1.LoginRequest{
		Identifier: &authenticationv1.LoginRequest_Email{
			Email: "alice@example.com",
		},
	}, stubBootstrapUserRepo{users: []*identityv1.User{user}})
	if info == nil {
		t.Fatalf("expected user info for email login")
	}
	if info.UserID != 101 {
		t.Fatalf("expected user id 101, got %d", info.UserID)
	}
	if info.TenantID != 9 {
		t.Fatalf("expected tenant id 9, got %d", info.TenantID)
	}
	if info.Username != "alice" {
		t.Fatalf("expected username alice, got %q", info.Username)
	}
}

func TestResolveLoginRequestUserByMobile(t *testing.T) {
	user := &identityv1.User{
		Id:       auditPtr(uint32(202)),
		TenantId: auditPtr(uint32(7)),
		Username: auditPtr("bob"),
		Mobile:   auditPtr("13800138000"),
	}
	info := resolveLoginRequestUser(context.Background(), &authenticationv1.LoginRequest{
		Identifier: &authenticationv1.LoginRequest_Mobile{
			Mobile: "13800138000",
		},
	}, stubBootstrapUserRepo{users: []*identityv1.User{user}})
	if info == nil {
		t.Fatalf("expected user info for mobile login")
	}
	if info.UserID != 202 {
		t.Fatalf("expected user id 202, got %d", info.UserID)
	}
	if info.TenantID != 7 {
		t.Fatalf("expected tenant id 7, got %d", info.TenantID)
	}
	if info.Username != "bob" {
		t.Fatalf("expected username bob, got %q", info.Username)
	}
}
