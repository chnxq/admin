# Task 模块解耦设计草案

## 目标

当前 `task` 模块已经具备任务 CRUD、调度恢复、立即执行、执行日志记录等能力，但实现上仍然存在一类不合理耦合：

- `TaskService`
- `TaskGroupService`
- `taskScheduler`
- `RegisterTaskScheduler`

都直接认识具体业务依赖，例如：

- `repo.ApiAuditLogRepo`
- `repo.LoginAuditLogRepo`
- `repo.PermissionAuditLogRepo`

这种结构虽然能工作，但会导致 `task` 模块与“将来要被执行的任务”形成反向耦合。任务平台本应只负责“调度”和“分发”，不应知道“删除过期日志”具体要清理哪些表、依赖哪些 repo。

本文先以“删除过期日志”任务为例，给出通用解耦方案。

## 当前耦合点

### 1. service 层直接持有业务 repo

当前 `admin/internal/service/task_runtime_ext.go` 中：

- `TaskService.RegisterRuntimeDeps(...)`
- `TaskGroupService.RegisterRuntimeDeps(...)`

直接接收：

- `TaskLogRepo`
- `ApiAuditLogRepo`
- `LoginAuditLogRepo`
- `PermissionAuditLogRepo`

问题：

- `TaskService`/`TaskGroupService` 只应关心任务运行时控制，不应关心某个具体执行器依赖哪些日志仓库。
- 每增加一种新任务，都可能继续向 `RegisterRuntimeDeps(...)` 追加新的 repo/service 参数，导致参数表无限膨胀。

### 2. scheduler 直接持有具体业务 repo

当前 `admin/internal/service/task_scheduler_ext.go` 中的 `taskScheduler` 直接包含：

- `taskLogRepo`
- `apiAuditLogRepo`
- `loginAuditLogRepo`
- `permissionAuditLogRepo`

问题：

- 调度器本质上只是“按 cron 触发某个任务”。
- 调度器不应该知道任务执行体内部的依赖。
- 一旦新增其它执行器，比如“同步缓存”“重建索引”“发送汇总通知”，调度器也会被迫继续扩充字段。

### 3. executeTaskOnce 的签名泄漏具体业务依赖

当前 `executeTaskOnce(...)` 形如：

```go
func executeTaskOnce(
    ctx context.Context,
    taskItem *taskv1.Task,
    overrideInput string,
    taskLogRepo repo.TaskLogRepo,
    apiAuditLogRepo repo.ApiAuditLogRepo,
    loginAuditLogRepo repo.LoginAuditLogRepo,
    permissionAuditLogRepo repo.PermissionAuditLogRepo,
    executorRegistry *taskruntime.Registry,
) error
```

问题：

- 该函数名义上是“执行一个任务”，但签名却显式暴露了“删除过期日志”任务的实现细节。
- 这意味着任务运行时基础设施已经被单个执行器污染。

### 4. registry 工厂仍依赖具体 repo 类型

当前 `admin/internal/service/task_executor_registry_ext.go` 中：

- `newTaskExecutorRegistry(...)`

直接接收多个具体 repo，再映射成 `task.RuntimeDeps`。

这比把 repo 直接塞进执行器更好一些，但边界仍然在 `service` 层，不够内聚。

## 正确的职责划分

推荐拆成四层。

### 1. Task 平台层

目录建议：

- `admin/internal/task/runtime`

职责：

- 维护执行器注册表
- 根据 `invoke_target` 查找执行器
- 调用执行器的 `Validate / Execute`
- 编排任务日志写入
- 向调度器暴露统一运行入口

这一层不应该引用任何 `ApiAuditLogRepo`、`LoginAuditLogRepo` 之类的具体业务仓库。

### 2. 调度层

可继续放在：

- `admin/internal/service/task_scheduler_ext.go`

或后续迁移到：

- `admin/internal/task/runtime/scheduler.go`

职责：

- 管理 cron parser
- 注册、移除、恢复任务
- 在触发时调用统一的 runtime 执行入口

这一层只应依赖：

- `TaskRuntimeRepo`
- 统一的 `Runner` 或 `ExecutorRegistry`
- `TaskLogWriter`

而不应依赖具体业务执行器的 repo。

### 3. 执行器层

目录建议：

- `admin/internal/task/executors`

每个执行器一个文件或一个子目录，例如：

- `cleanup_audit_logs.go`
- `rebuild_cache.go`
- `sync_routes.go`

职责：

- 只实现单一业务任务
- 自己声明所需依赖
- 自己解析 `args`
- 自己负责校验输入和返回输出摘要

### 4. 执行器依赖装配层

目录建议：

- `admin/internal/task/factory`

职责：

- 从现有 repo/service 组装出各执行器实例
- 统一创建默认 registry

也就是说，`task` 平台只拿到最终的 `Registry`，不再关心其中有哪些执行器、这些执行器又依赖哪些 repo。

## 推荐的通用模型

### A. 保留统一执行器接口

当前 `admin/internal/task/executor.go` 里的接口方向是对的，可以保留：

```go
type Executor interface {
    InvokeTarget() string
    Validate(context.Context, ValidationRequest) error
    Execute(context.Context, ExecuteRequest) (string, error)
}
```

这是稳定边界。

### B. 将依赖收敛到执行器自身

以“删除过期日志”为例，不应让 `TaskService` 或 `taskScheduler` 认识：

- `ApiAuditLogRepo`
- `LoginAuditLogRepo`
- `PermissionAuditLogRepo`

而应让执行器自己依赖一个最小接口。

例如：

```go
type AuditLogCleanupStore interface {
    CleanupAPIBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error)
    CleanupLoginBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error)
    CleanupPermissionBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error)
}
```

然后由执行器持有它：

```go
type CleanupAuditLogsExecutor struct {
    store AuditLogCleanupStore
}
```

这样变化就被约束在执行器内部：

- 新增一个清理目标，只改执行器和其 store
- 不需要改 `TaskService`
- 不需要改 `taskScheduler`
- 不需要改运行时统一入口

### C. Task 平台只依赖 Registry

推荐将运行时收敛成一类统一对象，例如：

```go
type Runner struct {
    registry    *Registry
    taskLogRepo repo.TaskLogWriter
}
```

对外只暴露：

- `RunTask(ctx, taskItem, overrideInput)`
- `ValidateTask(ctx, taskItem, raw)`

这样：

- `TaskService` 只依赖 `Runner`
- `TaskGroupService` 只依赖 `Runner`
- `taskScheduler` 只依赖 `Runner`

而不是每一层都自己拿一组 repo 再拼装调用。

## 删除过期日志任务的推荐解法

### 方案一：执行器依赖单一聚合接口

定义：

```go
type AuditLogCleanupStore interface {
    CleanupAPIBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error)
    CleanupLoginBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error)
    CleanupPermissionBefore(ctx context.Context, tenantID *uint32, before time.Time) (int, error)
}
```

优点：

- 执行器只有一个依赖
- 未来容易 mock
- service 层完全不知道三个日志 repo 的存在

缺点：

- 需要新增一个装配对象，把三个 repo 适配成一个 store

这是推荐方案。

### 方案二：执行器直接依赖三个最小清理接口

定义：

- `ApiAuditLogCleaner`
- `LoginAuditLogCleaner`
- `PermissionAuditLogCleaner`

优点：

- 与当前 `task.RuntimeDeps` 思路接近，迁移成本小

缺点：

- 执行器构造函数参数会变多
- 后续类似任务增多时，factory 仍会膨胀

如果想最小改造，可以先走这个方案，再收敛到方案一。

## 建议的目标结构

建议最终形成下面的关系：

```text
TaskService/TaskGroupService
    -> taskruntime.Runner
        -> task.Registry
            -> CleanupAuditLogsExecutor
                -> AuditLogCleanupStore
            -> OtherExecutor...

taskScheduler
    -> taskruntime.Runner

factory.NewDefaultRegistry(...)
    -> 组装全部执行器
```

关键点：

- `TaskService` 不直接依赖任何具体业务 repo
- `taskScheduler` 不直接依赖任何具体业务 repo
- “删除过期日志”只是众多执行器之一
- 执行器依赖的业务 repo 只出现在 factory/adapter 层

## 迁移步骤建议

### 第一步：引入统一 Runner

先抽出：

- `ValidateTask(...)`
- `RunTask(...)`
- `writeTaskExecutionLog(...)`

让 `TaskService` 和 `taskScheduler` 都只调用 `Runner`。

这一步不改业务行为，只收敛调用链。

### 第二步：把执行器装配迁移到 `internal/task`

将：

- `newTaskExecutorRegistry(...)`

从 `internal/service` 移到 `internal/task/factory` 或相近位置。

让 registry 的创建责任回到 task 领域内部。

### 第三步：把日志清理依赖收敛为单一 store

新增一个 adapter，例如：

- `audit_log_cleanup_store.go`

内部组合：

- `ApiAuditLogRepo`
- `LoginAuditLogRepo`
- `PermissionAuditLogRepo`

但对执行器只暴露一个 `AuditLogCleanupStore`。

### 第四步：删除 service 层上的具体 repo 字段

最终从下列对象中去掉：

- `TaskService`
- `TaskGroupService`
- `taskScheduler`

上的：

- `apiAuditLogRepo`
- `loginAuditLogRepo`
- `permissionAuditLogRepo`

以及与之对应的注入路径。

### 第五步：保留 TaskLogRepo，但只留在 Runner

`TaskLogRepo` 与任务平台本身有关，因为它属于运行日志基础设施，不属于某个具体业务任务。

因此：

- `TaskLogRepo` 可以留在统一 `Runner`
- 但不应分散在 `TaskService`、`TaskGroupService`、`taskScheduler` 三处重复持有

## 对生成边界的启发

这件事也反过来说明：

- `TaskService` 的运行时依赖不应继续扩张
- 一旦 service 结构体需要不断增加具体业务 repo 字段，说明边界错了

因此后续在 `xkit` 层面，应该尽量避免把“具体执行器依赖”沉淀为 service 生成能力，而应把生成边界收缩为：

- 任务 CRUD
- 基础调度入口
- 统一 runtime 持有

而把具体执行器依赖留在手写的 task 领域模块中。

## 本轮建议结论

建议采用下面的方向：

1. `task` 平台层只依赖 `Runner + Registry + TaskLogWriter`
2. 具体任务依赖下沉到执行器
3. 执行器依赖通过 `internal/task` 内部 factory/adapter 装配
4. `ApiAuditLogRepo` / `LoginAuditLogRepo` / `PermissionAuditLogRepo` 不再直接出现在 `TaskService`、`TaskGroupService`、`taskScheduler` 中

如果按这个方向推进，后续新增任何任务执行器，都不会再反向污染任务平台本身。
