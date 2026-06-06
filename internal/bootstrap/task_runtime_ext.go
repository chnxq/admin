package bootstrap

import (
	"context"

	"admin/internal/service"
	taskloader "admin/internal/task"
)

func configureTaskRuntime(
	services *GeneratedServices,
	data *GeneratedData,
) error {
	if services == nil || data == nil {
		return nil
	}
	bundle, err := taskloader.NewRuntimeBundleFromRepos(
		data.TaskRepo,
		data.TaskLogRepo,
		data.ApiAuditLogRepo,
		data.LoginAuditLogRepo,
		data.PermissionAuditLogRepo,
	)
	if err != nil {
		return err
	}
	return service.BindTaskServices(
		services.Task,
		services.TaskGroup,
		data.TaskRepo,
		data.TaskGroupRepo,
		bundle.Runner,
		bundle.Scheduler,
	)
}

func registerTaskRuntime(
	ctx context.Context,
	components *GeneratedComponents,
) (func(), error) {
	if components == nil || components.Services == nil {
		return func() {}, nil
	}
	return service.RegisterTaskScheduler(
		ctx,
		components.Services.Task,
		components.Services.TaskGroup,
	)
}
