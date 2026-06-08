package task

import (
	"admin/internal/data/repo"
	taskruntime "admin/internal/task/runtime"
	"admin/internal/task/tasks/auditlogcleanup"
	"admin/internal/task/tasks/taskruntimesummary"
)

type Loader struct{}

type RuntimeBundle struct {
	Runner    *taskruntime.Runner
	Scheduler *taskruntime.Scheduler
}

func NewLoader() *Loader {
	return &Loader{}
}

func (l *Loader) LoadFromRepos(
	taskRepo repo.TaskRepo,
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
) ([]taskruntime.Executor, error) {
	return []taskruntime.Executor{
		auditlogcleanup.NewAuditLogCleanupExecutor(apiAuditLogRepo, loginAuditLogRepo, permissionAuditLogRepo),
		taskruntimesummary.NewTaskRuntimeSummaryExecutor(taskRepo),
	}, nil
}

func NewRegistryFromRepos(
	taskRepo repo.TaskRepo,
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
) (*taskruntime.Registry, error) {
	loader := NewLoader()
	executors, err := loader.LoadFromRepos(taskRepo, apiAuditLogRepo, loginAuditLogRepo, permissionAuditLogRepo)
	if err != nil {
		return nil, err
	}
	return taskruntime.NewRegistry(executors...)
}

func NewRunnerFromRepos(
	taskRepo repo.TaskRepo,
	taskLogRepo repo.TaskLogRepo,
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
) (*taskruntime.Runner, error) {
	registry, err := NewRegistryFromRepos(taskRepo, apiAuditLogRepo, loginAuditLogRepo, permissionAuditLogRepo)
	if err != nil {
		return nil, err
	}
	var logWriter taskruntime.TaskLogWriter
	if value, ok := taskLogRepo.(repo.TaskLogWriter); ok {
		logWriter = value
	}
	return taskruntime.NewRunner(registry, logWriter), nil
}

func NewRuntimeBundleFromRepos(
	taskRepo repo.TaskRepo,
	taskLogRepo repo.TaskLogRepo,
	apiAuditLogRepo repo.ApiAuditLogRepo,
	loginAuditLogRepo repo.LoginAuditLogRepo,
	permissionAuditLogRepo repo.PermissionAuditLogRepo,
) (*RuntimeBundle, error) {
	runner, err := NewRunnerFromRepos(
		taskRepo,
		taskLogRepo,
		apiAuditLogRepo,
		loginAuditLogRepo,
		permissionAuditLogRepo,
	)
	if err != nil {
		return nil, err
	}
	store, err := NewTaskRuntimeStore(taskRepo)
	if err != nil {
		return nil, err
	}
	return &RuntimeBundle{
		Runner:    runner,
		Scheduler: taskruntime.NewScheduler(store, runner),
	}, nil
}
