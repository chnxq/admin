# Task 执行器约定

本文约束 `admin/internal/task` 下的任务执行器扩展方式，目标是让任务 CRUD、调度恢复、立即执行、执行日志写入走同一条稳定链路。

## 目录职责

- `internal/service`
  负责任务和任务分组的 CRUD、调度启停、启动恢复、执行日志编排。
- `internal/task`
  负责 `invoke_target` 对应的参数校验、执行逻辑、结果输出。
- `internal/data/repo`
  负责执行器需要的内部能力，例如审计日志清理、任务汇总只读查询等。

执行器不要把业务逻辑塞回 `service` 层的 `switch` / `if` 分支里。新增任务能力时，优先实现新的 `Executor` 并注册到默认 registry。

## 基本接口

每个执行器都实现：

- `InvokeTarget() string`
- `Validate(context.Context, ValidationRequest) error`
- `Execute(context.Context, ExecuteRequest) (string, error)`

约束：

- `InvokeTarget` 必须稳定、唯一、可长期兼容。
- `Validate` 用于创建、编辑、启动前的输入校验。
- `Execute` 返回值应是紧凑、可落盘的结果字符串，优先使用 JSON。

## Invoke Target 命名

推荐格式：

- `system:cleanup:audit-logs`
- `system:task:runtime-summary`

规则：

- 前缀表达领域边界，例如 `system`、`tenant`。
- 中间段表达能力类别，例如 `cleanup`、`task`。
- 最后一段表达动作或资源。
- 一旦进入生产数据，不要随意重命名。

## 输入输出规范

输入 `args` 统一使用 JSON 字符串，不再使用 `k=v` 拼接串。

示例：

```json
{"expireHours":720,"targets":["api","login","permission"]}
```

输出也建议使用 JSON，便于任务日志、调试和后续扩展。

示例：

```json
{"expireHours":720,"deletedApi":18,"deletedLogin":6,"deletedPermission":3,"totalDeleted":27}
```

## 依赖内部能力

执行器如果需要调用 repo / 内部查询能力，不要直接 new service 或耦合 transport。

正确方式：

1. 在 `internal/task/executor.go` 中为该能力定义最小接口。
2. 在 `RuntimeDeps` 中增加可选依赖。
3. 在 `internal/service/task_executor_registry_ext.go` 中从现有 repo 装配该依赖。
4. 在执行器中只依赖这个最小接口。

当前示例：

- `CleanupAuditLogsExecutor`
  调用三类审计日志清理能力。
- `TaskRuntimeSummaryExecutor`
  调用任务只读汇总能力，作为“执行器依赖内部能力”的正式样例。

## 注册策略

默认可用执行器在 `internal/task/registry_factory.go` 的 `NewDefaultRegistry(...)` 中注册。

规则：

- 已经准备投入生产的执行器，注册进默认 registry。
- 实验性或教学型执行器，可以保留在 `internal/task`，但不注册。

## 错误处理

- 参数错误：直接返回明确错误，阻止任务执行。
- 运行错误：返回错误，由 service 层统一写入 `task_log.error`。
- 单任务 cron 非法：启动恢复时只记录并跳过，不能阻塞整个服务启动。

## 多租户要求

- 执行器必须尊重 `task.tenant_id`。
- 后台恢复和调度执行不依赖请求态 `ViewerContext`。
- 若执行器需要跨租户行为，必须显式通过输入参数或内置规则声明，不能隐式放大权限。

## 测试要求

新增执行器至少补两类测试：

- 参数校验测试
- 执行结果测试

若执行器依赖内部能力，使用 fake / stub 接口即可，不要在单测里拉起完整服务。

## SQL 与默认数据约定

- 默认内置任务由 `internal/data/bootstrap/default_data_ext.go` 维护。
- `admin/sql/*demo-data.sql` 只追加高位 demo 数据，不覆盖初始化数据。
- demo task 的 `cron_expression` 必须与后端统一为 6 段格式：

```text
秒 分 时 日 月 周
```
