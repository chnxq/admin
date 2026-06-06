# Task Runtime Architecture

## 目标

本文用于固定当前 `admin` 中任务运行时的分层与装配链，作为后续继续解耦、新增任务执行器、补充调度能力时的约束文档。

本文不讨论：

- 前端任务管理界面
- `task_group` / `task` / `task_log` 的 CRUD 细节
- 具体任务业务逻辑实现

本文只关注：

- `task` 运行时如何分层
- 应用启动时如何完成装配
- 哪些接口应视为稳定接口
- 哪些旧链路不应恢复

## 当前分层

### 1. `internal/task/runtime`

职责：

- 定义通用运行时接口
- 提供 `Registry`
- 提供 `Runner`
- 提供 `Scheduler`
- 提供 `TaskRuntimeStore` 接口

这一层不应依赖任何具体业务 repo，也不应知道某个任务如何实现。

### 2. `internal/task`

职责：

- 组织任务域公共装配代码
- 提供运行时装配入口
- 提供 `TaskRuntimeStore` 的 repo 适配实现

当前主要文件：

- `loader.go`
- `runtime_store.go`
- `doc.go`

这一层是任务域内部的“公共 glue 层”，可以依赖 repo，也可以依赖 `tasks/<name>`。

### 3. `internal/task/tasks/<name>`

职责：

- 一个具体任务的 contract
- 一个具体任务的 factory
- 一个具体任务的 executor
- 一个具体任务的测试

当前样例：

- `tasks/auditlogcleanup`
- `tasks/taskruntimesummary`
- `tasks/echo`

约束：

- `tasks/<name>` 目录只放具体任务
- 不再把任务域公共 loader、store、registry 装配逻辑放在这里

### 4. `internal/service`

职责：

- 承载 `TaskService` / `TaskGroupService` 的业务逻辑
- 使用已经装配好的 `Runner` / `Scheduler`
- 启动时负责 `Scheduler` 的 start / restore / stop cleanup

约束：

- 不在 `service` 中创建具体执行器
- 不在 `service` 中拼装 registry
- 不在 `service` 中直接依赖 `ApiAuditLogRepo` / `LoginAuditLogRepo` / `PermissionAuditLogRepo` 之类具体任务依赖

### 5. `internal/bootstrap`

职责：

- 应用启动时组织任务运行时装配
- 将 task 运行时接入 generated services
- 启动 scheduler

当前主要文件：

- `generated_hooks_ext.go`
- `task_runtime_ext.go`

约束：

- `generated_hooks_ext.go` 保持为薄入口
- task runtime 的实际 bootstrap 逻辑放在 `task_runtime_ext.go`

## 当前装配链

### 初始化阶段

入口：

- `bootstrap/generated_hooks_ext.go`

调用链：

1. `GeneratedServices.afterInit(data)`
2. `configureTaskRuntime(services, data)`
3. `task.NewRuntimeBundleFromRepos(...)`
4. `service.BindTaskServices(...)`

装配结果：

- `TaskService` 拿到：
  - `taskGroupRepo`
  - `runtimeRunner`
  - `scheduler`
- `TaskGroupService` 拿到：
  - `taskRepo`
  - `runtimeRunner`
  - `scheduler`

### 启动阶段

入口：

- `bootstrap/app.go`

调用链：

1. `newTransportServers(...)`
2. `registerTaskRuntime(ctx, components)`
3. `service.RegisterTaskScheduler(...)`
4. `scheduler.Start()`
5. `scheduler.RestoreTasks(ctx)`

## 当前稳定接口

以下接口可视为当前阶段的稳定接口：

### `internal/task`

- `NewRuntimeBundleFromRepos(...)`
- `NewRunnerFromRepos(...)`
- `NewRegistryFromRepos(...)`
- `NewTaskRuntimeStore(...)`

### `internal/task/runtime`

- `Registry`
- `Runner`
- `Scheduler`
- `TaskRuntimeStore`

### `internal/service`

- `BindTaskServices(...)`
- `RegisterTaskScheduler(...)`

## 不应恢复的旧链路

以下旧实现方向已被明确淘汰，不应在后续修改中恢复：

### 1. `RegisterRuntimeDeps(...)`

不再允许在 `TaskService` / `TaskGroupService` 上保留这类 repo 级依赖注册链。

原因：

- 把具体执行器依赖泄漏到 service 层
- 增加 task 平台与具体任务的耦合

### 2. `service` 内直接创建执行器 / registry / scheduler

不再允许在 `service` 内部完成：

- executor 构造
- registry 构造
- runner 构造
- scheduler 构造

这些责任现在属于：

- `internal/task`
- `internal/task/runtime`

### 3. `tasks` 目录同时承担“具体任务目录”和“公共装配目录”

不再将以下公共代码放回 `internal/task/tasks`：

- loader
- runtime store adapter
- registry assembly helper

## 后续新增任务的约束

新增一个任务时，按以下方向扩展：

1. 在 `internal/task/tasks/<task-name>` 下放置该任务的实现代码
2. 在 `internal/task/loader.go` 中将该任务装配进默认 registry
3. 若该任务需要 repo 适配或公共 adapter，再评估放在：
   - `internal/task`
   - 或 `internal/task/tasks/<task-name>`

判断原则：

- 如果代码只服务于某一个具体任务，放到 `tasks/<name>`
- 如果代码服务于整个任务运行时装配，放到 `internal/task`

## 当前文件组织结论

当前推荐结构如下：

```text
internal/task/
  doc.go
  loader.go
  runtime_store.go
  runtime/
    types.go
    runner.go
    scheduler.go
  tasks/
    auditlogcleanup/
    taskruntimesummary/
    echo/
```

```text
internal/bootstrap/
  generated_hooks_ext.go
  task_runtime_ext.go
```

## 后续建议

后续继续演进时，优先按下面顺序推进：

1. 在本结构下继续增加新的任务样例
2. 继续削弱 task 平台对具体业务 repo 的感知
3. 再评估是否需要把默认任务装配进一步从 `loader.go` 抽成更细粒度模块

在没有明确收益前，不建议再次调整 `runtime / task / tasks / bootstrap / service` 的大层级结构。
