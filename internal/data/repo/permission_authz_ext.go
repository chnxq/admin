package repo

import (
	"context"
	"strings"

	"admin/internal/data/ent/api"
	"admin/internal/data/ent/permission"
	"admin/internal/data/ent/permissionapi"
)

type PermissionAuthorizationReader interface {
	HasAPIAccess(ctx context.Context, permissionCodes []string, path, method string) (bool, error)
}

func (r *permissionRepo) HasAPIAccess(ctx context.Context, permissionCodes []string, path, method string) (bool, error) {
	if r == nil || r.entClient == nil || len(permissionCodes) == 0 {
		return false, nil
	}

	path = strings.TrimSpace(path)
	method = strings.ToUpper(strings.TrimSpace(method))
	if path == "" || method == "" {
		return false, nil
	}

	permissionIDs, err := r.entClient.Client().Permission.Query().
		Where(
			permission.CodeIn(permissionCodes...),
			permission.StatusEQ(permission.StatusOn),
		).
		IDs(ctx)
	if err != nil || len(permissionIDs) == 0 {
		return false, err
	}

	apiIDs, err := r.entClient.Client().Api.Query().
		Where(
			api.PathEQ(path),
			api.MethodEQ(method),
			api.StatusEQ(api.StatusOn),
		).
		IDs(ctx)
	if err != nil || len(apiIDs) == 0 {
		return false, err
	}

	count, err := r.entClient.Client().PermissionApi.Query().
		Where(
			permissionapi.PermissionIDIn(permissionIDs...),
			permissionapi.APIIDIn(apiIDs...),
		).
		Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
