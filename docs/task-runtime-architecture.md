# Task Runtime Architecture

## 1. Goal

本文用于固定当前 `admin` 中 task runtime 的分层、装配链和边界约束，作为后续继续新增任务执行器、补测试、做进一步解耦时的基线文档。

本文不覆盖：

- 前端任务管理界面
- `task_group` / `task` / `task_log` 的 CRUD 细节
- 具体任务业务本身

## 2. Current Layering

### `internal/task/runtime`

职责：

- 定义稳定运行时契约
- 提供 `Registry`
- 提供 `Runner`
- 提供 `Scheduler`
- 提供 `TaskRuntimeStore` 接口

约束：

- 不直接依赖具体业务 repo
- 不知道某个具体任务如何实现

### `internal/task`

职责：

- 负责任务域公共装配
- 负责 loader
- 负责 runtime store adapter
- 负责 runtime bundle 组装入口

当前主要文件：

- `loader.go`
- `runtime_store.go`
- `doc.go`

### `internal/task/tasks/<name>`

职责：

- 一个目录只对应一个具体任务
- 放该任务的 contract / factory / executor / tests

当前样例：

- `tasks/auditlogcleanup`
- `tasks/taskruntimesummary`
- `tasks/echo`

### `internal/service`

职责：

- 承载 `TaskService` / `TaskGroupService` 的业务接口
- 使用已装配好的 `Runner` / `Scheduler`
- 提供 runtime 启停入口对接

约束：

- 不在 service 内构造具体执行器
- 不在 service 内拼装 registry
- 不直接持有具体任务业务 repo 依赖

### `internal/bootstrap`

职责：

- 启动期装配 task runtime
- 接入 generated services
- 启动 scheduler

当前关键文件：

- `generated_hooks_ext.go`
- `task_runtime_ext.go`

## 3. Current Wiring Chain

### 初始化阶段

入口：

- `bootstrap/generated_hooks_ext.go`

调用链：

1. `GeneratedServices.afterInit(data)`
2. `configureTaskRuntime(services, data)`
3. `task.NewRuntimeBundleFromRepos(...)`
4. `service.BindTaskServices(...)`

结果：

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

## 4. Stable Interfaces

当前可视为稳定接口：

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

## 5. Runtime Fixes Already Landed

### 5.1 Runtime task load must not require request ViewerContext

调度恢复和后台执行路径已经改成使用 runtime-safe repo 入口，避免：

- `security: missing ViewerContext in context`

### 5.2 Pre-run failure must still write `task_log`

即使失败发生在 `Runner.RunTask(...)` 之前，也必须落失败日志。

当前已由 `Scheduler` + `Runner.RecordTaskFailure(...)` 兜底。

### 5.3 Runtime repo writes/cleanup need runtime context

当前以下 runtime-only repo 入口已补 runtime viewer context：

- 审计日志 cleanup
- `task_log` 写入

### 5.4 Invalid cron must not block service startup

单任务 cron 非法时：

- 记录错误
- 跳过该任务
- 不阻塞整个进程启动

## 6. Default Seed Relationship

当前默认任务种子与 runtime 的关系：

- 默认任务定义由 `internal/data/bootstrap/default_data_ext.go` 维护
- 资源同步每次启动执行
- 默认业务种子只在 seed-domain 为空时执行

这意味着：

- runtime 启动后会使用当前数据库里的任务定义
- 但系统不会因为重启反复刷新默认任务记录

## 7. Deprecated Directions

以下旧方向不应轻易恢复：

### `RegisterRuntimeDeps(...)`

原因：

- 把具体执行器依赖泄漏到 service 层
- 增加 task 平台与具体任务的耦合

### 在 `service` 内直接构造执行器 / registry / scheduler

这些责任当前属于：

- `internal/task`
- `internal/task/runtime`

### 把公共装配代码塞回 `internal/task/tasks`

不应把以下公共代码重新放回 `internal/task/tasks`：

- loader
- runtime store adapter
- registry assembly helper

## 8. Recommended File Structure

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

## 9. Suggested Next Steps

优先顺序建议：

1. 在当前结构下继续增加真实任务样例
2. 补更多 runtime 集成验证
3. 继续削弱 task 平台对具体业务 repo 的感知

在没有明确收益前，不建议再次调整 `runtime / task / tasks / bootstrap / service` 的大层级结构。
