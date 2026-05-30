package service

import (
	"context"
	"fmt"

	identityv1 "admin/api/gen/identity/v1"

	crudviewer "github.com/chnxq/x-crud/viewer"
)

func requireViewerContext(ctx context.Context, action string) (crudviewer.Context, error) {
	viewer, ok := crudviewer.FromContext(ctx)
	if !ok || viewer == nil {
		if action == "" {
			action = "operation"
		}
		return nil, fmt.Errorf("%s requires viewer context", action)
	}
	return viewer, nil
}

func requirePlatformContext(ctx context.Context, action string) (crudviewer.Context, error) {
	viewer, err := requireViewerContext(ctx, action)
	if err != nil {
		return nil, err
	}
	if viewer.IsTenantContext() {
		return nil, identityv1.ErrorForbidden("%s requires platform context", action)
	}
	return viewer, nil
}
