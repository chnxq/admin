package server

import (
	"context"
	"fmt"
	"strings"

	identityv1 "admin/api/gen/identity/v1"
	"admin/internal/data/repo"
	paginationv1 "github.com/chnxq/x-crud/api/gen/pagination/v1"
	"github.com/chnxq/xkitpkg/app"
	conf "github.com/chnxq/xkitpkg/conf/v1"
)

const (
	defaultRegistrationTenantID = uint32(1)
)

type generatedDataWithOrgUnitRepo interface {
	OrgUnitRepoProvider() repo.OrgUnitRepo
}

type generatedDataWithPositionRepo interface {
	PositionRepoProvider() repo.PositionRepo
}

type registrationDefaults struct {
	DefaultTenantID     uint32
	DefaultTenantCode   string
	DefaultRoleCode     string
	DefaultOrgUnitCode  string
	DefaultPositionCode string
}

type registrationAssignment struct {
	TenantID    uint32
	RoleIDs     []uint32
	OrgUnitIDs  []uint32
	PositionIDs []uint32
}

type registrationDefaultsResolver struct {
	tenantRepo   repo.TenantRepo
	roleRepo     repo.RoleRepo
	orgUnitRepo  repo.OrgUnitRepo
	positionRepo repo.PositionRepo
	defaults     registrationDefaults
}

func newRegistrationDefaultsResolver(appCtx *app.AppCtx, data GeneratedData) *registrationDefaultsResolver {
	resolver := &registrationDefaultsResolver{
		defaults: loadRegistrationDefaults(appCtx),
	}
	if provider, ok := data.(generatedDataWithTenantRepo); ok {
		resolver.tenantRepo = provider.TenantRepoProvider()
	}
	if provider, ok := data.(generatedDataWithRoleRepo); ok {
		resolver.roleRepo = provider.RoleRepoProvider()
	}
	if provider, ok := data.(generatedDataWithOrgUnitRepo); ok {
		resolver.orgUnitRepo = provider.OrgUnitRepoProvider()
	}
	if provider, ok := data.(generatedDataWithPositionRepo); ok {
		resolver.positionRepo = provider.PositionRepoProvider()
	}
	return resolver
}

func loadRegistrationDefaults(appCtx *app.AppCtx) registrationDefaults {
	defaults := registrationDefaults{
		DefaultTenantID:     defaultRegistrationTenantID,
		DefaultTenantCode:   "default",
		DefaultRoleCode:     repo.DefaultRegistrationRoleCode,
		DefaultOrgUnitCode:  repo.DefaultRegistrationOrgUnitCode,
		DefaultPositionCode: repo.DefaultRegistrationPositionCode,
	}
	if appCtx == nil {
		return defaults
	}

	cfg := appCtx.GetConfig()
	if cfg == nil {
		return defaults
	}
	authn := cfg.GetAuthn()
	if authn == nil {
		return defaults
	}
	registration := authn.GetRegistration()
	if registration == nil {
		return defaults
	}
	return applyRegistrationDefaultsFromConf(defaults, registration)
}

func applyRegistrationDefaultsFromConf(defaults registrationDefaults, registration *conf.Authentication_Registration) registrationDefaults {
	if registration == nil {
		return defaults
	}
	if registration.GetDefaultTenantId() > 0 {
		defaults.DefaultTenantID = registration.GetDefaultTenantId()
	}
	if value := strings.TrimSpace(registration.GetDefaultTenantCode()); value != "" {
		defaults.DefaultTenantCode = value
	}
	if value := strings.TrimSpace(registration.GetDefaultRoleCode()); value != "" {
		defaults.DefaultRoleCode = value
	}
	if value := strings.TrimSpace(registration.GetDefaultOrgUnitCode()); value != "" {
		defaults.DefaultOrgUnitCode = value
	}
	if value := strings.TrimSpace(registration.GetDefaultPositionCode()); value != "" {
		defaults.DefaultPositionCode = value
	}
	return defaults
}

func (r *registrationDefaultsResolver) resolveRegistrationAssignment(ctx context.Context, requestedTenantCode string) (*registrationAssignment, error) {
	if r == nil {
		return nil, nil
	}
	tenantID, err := r.resolveTenantID(ctx, requestedTenantCode)
	if err != nil {
		return nil, err
	}
	return r.resolveRegistrationAssignmentForTenantID(ctx, tenantID)
}

func (r *registrationDefaultsResolver) resolveRegistrationAssignmentForTenantID(ctx context.Context, tenantID uint32) (*registrationAssignment, error) {
	if r == nil || tenantID == 0 {
		return &registrationAssignment{}, nil
	}

	roleID, err := r.lookupRoleID(ctx, tenantID, r.defaults.DefaultRoleCode)
	if err != nil {
		return nil, err
	}
	orgUnitID, err := r.lookupOrgUnitID(ctx, tenantID, r.defaults.DefaultOrgUnitCode)
	if err != nil {
		return nil, err
	}
	positionID, err := r.lookupPositionID(ctx, tenantID, r.defaults.DefaultPositionCode)
	if err != nil {
		return nil, err
	}

	assignment := &registrationAssignment{TenantID: tenantID}
	if roleID > 0 {
		assignment.RoleIDs = []uint32{roleID}
	}
	if orgUnitID > 0 {
		assignment.OrgUnitIDs = []uint32{orgUnitID}
	}
	if positionID > 0 {
		assignment.PositionIDs = []uint32{positionID}
	}
	return assignment, nil
}

func (r *registrationDefaultsResolver) resolveTenantID(ctx context.Context, requestedTenantCode string) (uint32, error) {
	lookupCtx := ensureDefaultViewerContext(ctx)
	if code := strings.TrimSpace(requestedTenantCode); code != "" {
		if tenantID, err := r.lookupTenantIDByCode(lookupCtx, code); err != nil {
			return 0, err
		} else if tenantID > 0 {
			return tenantID, nil
		}
	}
	if r.defaults.DefaultTenantID > 0 {
		if tenantID, err := r.lookupTenantIDByID(lookupCtx, r.defaults.DefaultTenantID); err != nil {
			return 0, err
		} else if tenantID > 0 {
			return tenantID, nil
		}
	}
	if code := strings.TrimSpace(r.defaults.DefaultTenantCode); code != "" {
		return r.lookupTenantIDByCode(lookupCtx, code)
	}
	return 0, nil
}

func (r *registrationDefaultsResolver) lookupTenantIDByID(ctx context.Context, tenantID uint32) (uint32, error) {
	if r.tenantRepo == nil || tenantID == 0 {
		return 0, nil
	}
	tenantDTO, err := r.tenantRepo.Get(ctx, &identityv1.GetTenantRequest{
		QueryBy: &identityv1.GetTenantRequest_Id{Id: tenantID},
	})
	if err != nil || tenantDTO == nil {
		return 0, err
	}
	return tenantDTO.GetId(), nil
}

func (r *registrationDefaultsResolver) lookupTenantIDByCode(ctx context.Context, code string) (uint32, error) {
	if r.tenantRepo == nil || strings.TrimSpace(code) == "" {
		return 0, nil
	}
	limit := uint32(1)
	resp, err := r.tenantRepo.List(ctx, &paginationv1.PagingRequest{
		Limit: &limit,
		FilteringType: &paginationv1.PagingRequest_FilterExpr{
			FilterExpr: &paginationv1.FilterExpr{
				Type: paginationv1.ExprType_AND,
				Conditions: []*paginationv1.FilterCondition{
					{
						Field: "code",
						Op:    paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{
							Value: strings.TrimSpace(code),
						},
					},
				},
			},
		},
	})
	if err != nil || resp == nil || len(resp.GetItems()) == 0 || resp.GetItems()[0] == nil {
		return 0, err
	}
	return resp.GetItems()[0].GetId(), nil
}

func (r *registrationDefaultsResolver) lookupRoleID(ctx context.Context, tenantID uint32, code string) (uint32, error) {
	if r.roleRepo == nil || tenantID == 0 || strings.TrimSpace(code) == "" {
		return 0, nil
	}
	limit := uint32(1)
	resp, err := r.roleRepo.List(ensureDefaultViewerContext(ctx), &paginationv1.PagingRequest{
		Limit: &limit,
		FilteringType: &paginationv1.PagingRequest_FilterExpr{
			FilterExpr: &paginationv1.FilterExpr{
				Type: paginationv1.ExprType_AND,
				Conditions: []*paginationv1.FilterCondition{
					{
						Field: "code",
						Op:    paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{
							Value: strings.TrimSpace(code),
						},
					},
					{
						Field: "tenant_id",
						Op:    paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{
							Value: fmt.Sprintf("%d", tenantID),
						},
					},
				},
			},
		},
	})
	if err != nil || resp == nil || len(resp.GetItems()) == 0 || resp.GetItems()[0] == nil {
		return 0, err
	}
	return resp.GetItems()[0].GetId(), nil
}

func (r *registrationDefaultsResolver) lookupOrgUnitID(ctx context.Context, tenantID uint32, code string) (uint32, error) {
	if r.orgUnitRepo == nil || tenantID == 0 || strings.TrimSpace(code) == "" {
		return 0, nil
	}
	limit := uint32(1)
	resp, err := r.orgUnitRepo.List(ensureDefaultViewerContext(ctx), &paginationv1.PagingRequest{
		Limit: &limit,
		FilteringType: &paginationv1.PagingRequest_FilterExpr{
			FilterExpr: &paginationv1.FilterExpr{
				Type: paginationv1.ExprType_AND,
				Conditions: []*paginationv1.FilterCondition{
					{
						Field: "code",
						Op:    paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{
							Value: strings.TrimSpace(code),
						},
					},
					{
						Field: "tenant_id",
						Op:    paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{
							Value: fmt.Sprintf("%d", tenantID),
						},
					},
				},
			},
		},
	})
	if err != nil || resp == nil || len(resp.GetItems()) == 0 || resp.GetItems()[0] == nil {
		return 0, err
	}
	return resp.GetItems()[0].GetId(), nil
}

func (r *registrationDefaultsResolver) lookupPositionID(ctx context.Context, tenantID uint32, code string) (uint32, error) {
	if r.positionRepo == nil || tenantID == 0 || strings.TrimSpace(code) == "" {
		return 0, nil
	}
	limit := uint32(1)
	resp, err := r.positionRepo.List(ensureDefaultViewerContext(ctx), &paginationv1.PagingRequest{
		Limit: &limit,
		FilteringType: &paginationv1.PagingRequest_FilterExpr{
			FilterExpr: &paginationv1.FilterExpr{
				Type: paginationv1.ExprType_AND,
				Conditions: []*paginationv1.FilterCondition{
					{
						Field: "code",
						Op:    paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{
							Value: strings.TrimSpace(code),
						},
					},
					{
						Field: "tenant_id",
						Op:    paginationv1.Operator_EQ,
						ValueOneof: &paginationv1.FilterCondition_Value{
							Value: fmt.Sprintf("%d", tenantID),
						},
					},
				},
			},
		},
	})
	if err != nil || resp == nil || len(resp.GetItems()) == 0 || resp.GetItems()[0] == nil {
		return 0, err
	}
	return resp.GetItems()[0].GetId(), nil
}
