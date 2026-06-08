# Task Executor Convention

本文约束 `admin/internal/task` 下任务执行器的组织方式，目标是让任务 CRUD、调度恢复、立即执行、执行日志写入走同一条稳定链路，并避免 task 平台再次反向耦合到具体业务 repo。

## 1. 当前分层

- `internal/task/runtime`
  - 定义稳定运行时契约。
  - 提供 `Registry`、`Runner`、`Scheduler`、`TaskRuntimeStore`。
- `internal/task`
  - 负责任务域公共装配。
  - 负责 loader、runtime store adapter、runtime bundle 组装。
- `internal/task/tasks/<name>`
  - 每个目录只放一个具体任务。
  - 负责该任务的输入校验、执行逻辑、工厂、测试。
- `internal/service`
  - 只消费已装配好的 `Runner` / `Scheduler`。
  - 不再直接拼装具体执行器依赖。

## 2. 执行器接口

每个任务执行器都实现：

- `InvokeTarget() string`
- `Validate(context.Context, ValidationRequest) error`
- `Execute(context.Context, ExecuteRequest) (string, error)`

约束：

- `InvokeTarget` 必须稳定、唯一、可长期兼容。
- `Validate` 负责创建、编辑、启动前参数校验。
- `Execute` 返回值应为紧凑且可落盘的结果字符串，优先使用 JSON。

## 3. Invoke Target 命名

推荐格式：

- `system:cleanup:audit-logs`
- `system:task:runtime-summary`

规则：

- 第一段表达领域边界，例如 `system`、`tenant`。
- 中间段表达能力类别，例如 `cleanup`、`task`。
- 最后一段表达资源或动作。
- 一旦进入生产数据，不应随意重命名。

## 4. 输入与输出规范

`task.args` 统一使用 JSON 字符串，不再使用 `k=v` 形式。

示例输入：

```json
{"expireHours":720,"targets":["api","login","permission"]}
```

建议输出也使用 JSON，便于写入 `task_log.output`、调试和后续扩展。

示例输出：

```json
{"expireHours":720,"deletedApi":18,"deletedLogin":6,"deletedPermission":3,"totalDeleted":27}
```

## 5. 依赖注入原则

执行器如需内部业务能力，不直接 new service，也不直接感知 transport。

正确方式：

1. 在具体任务目录内声明最小依赖接口。
2. 由 `internal/task` 装配层提供该接口的实际适配。
3. 由 `loader.go` 把执行器注册进默认 registry。
4. `service` 和 `bootstrap` 只拿最终的 `Runner` / `Scheduler`。

当前样例：

- `tasks/auditlogcleanup`
  - 依赖审计日志清理能力。
- `tasks/taskruntimesummary`
  - 依赖任务运行时汇总只读能力。
- `tasks/echo`
  - 作为最小可运行示例。

## 6. 注册策略

默认执行器通过 `internal/task/loader.go` 注册。

规则：

- 已准备投入运行的执行器，注册进默认 loader。
- 实验性执行器可以保留在 `tasks/<name>` 下，但不接入 loader。

## 7. 错误处理

- 参数错误：直接返回，阻止任务执行。
- 运行错误：返回错误，由 `Runner` 统一写入 `task_log.error`。
- 调度前置失败：也必须落 `task_log`，不能只报错不留痕。
- 单任务 cron 非法：启动恢复时只记录并跳过该任务，不能阻塞整个服务启动。

## 8. 多租户与运行上下文

- 执行器必须尊重 `task.tenant_id`。
- 后台恢复和调度执行不依赖请求态 `ViewerContext`。
- runtime 相关 repo 入口必须自行补齐 runtime viewer context。
- 若执行器需要跨租户行为，必须通过明确规则声明，不能隐式放大权限。

## 9. 测试要求

新增执行器至少补两类测试：

- 参数校验测试
- 执行结果测试

若执行器依赖内部能力，应使用 fake/stub 接口，不在单测中启动完整服务。

另外，涉及 runtime 调度行为时，应优先补到：

- `internal/task/runtime/runner_test.go`
- `internal/task/runtime/scheduler_test.go`
- `internal/task/runtime_store_test.go`

## 10. SQL 与默认数据

- 默认内置任务由 `internal/data/bootstrap/default_data_ext.go` 维护。
- 当前默认种子语义：
  - 资源同步每次启动执行。
  - 业务默认种子仅在空 seed-domain 时初始化。
- `admin/sql/*demo-data.sql` 只追加高位 demo 数据，不覆盖初始化数据。
- demo task 的 `cron_expression` 必须与后端统一为 6 段格式：

```text
秒 分 时 日 月 周
```

## 11. 不应恢复的旧链路

以下方向不建议恢复：

- `RegisterRuntimeDeps(...)`
- service 内部直接构造具体执行器或 registry
- scheduler 直接依赖 `ApiAuditLogRepo` / `LoginAuditLogRepo` / `PermissionAuditLogRepo`
- 把公共 loader/store/registry 装配代码重新塞回 `internal/task/tasks`
