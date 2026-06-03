# 任务管理开发计划

## 1. 目标与范围

本次开发目标是在 `admin` / `admin-ui` 现有架构内，补齐一套完整的任务调度管理能力，覆盖以下范围：

- 任务分组管理
- 任务管理
- 任务执行日志观察
- 任务启动、停止、立即执行等操作
- 应用启动时根据 `task` 表加载并启动调度
- 初始化默认任务分组与默认任务
- 落地一个内置任务：删除过期日志

本次以现有 `task` / `task_group` / `task_log` schema、repo、proto 为基础，不重新设计存储结构。

## 2. 设计原则

- 遵循 `admin` 现有分层：`proto -> repo -> service -> bootstrap/server`
- 遵循 `admin-ui` 现有页面组织与交互风格，优先复用 Vben 与现有表格/表单能力
- 菜单、权限、初始化数据统一走当前 V2 初始化链路，不做临时旁路
- 调度能力参考 `D:\GoProjects\chnxq\xadmin`，但仅借鉴启动与调度编排思路，不直接照搬旧 task 模型
- 内置任务执行器以“可扩展注册”的方式组织，避免后续继续堆 if/else
- 后端保持 `task_group`、`task`、`task_log` 三套资源边界独立，前端只做界面聚合

## 3. 现状判断

当前 `admin` 已具备基础任务资源链路：

- proto 已有：
  - `task/v1/task.proto`
  - `task/v1/task_group.proto`
  - `task/v1/task_log.proto`
  - `admin/v1/i_task.proto`
  - `admin/v1/i_task_group.proto`
  - `admin/v1/i_task_log.proto`
- repo 已有：
  - `task_repo.gen.go`
  - `task_group_repo.gen.go`
  - `task_log_repo.gen.go`
- service 已有基础 CRUD：
  - `TaskService`
  - `TaskGroupService`
  - `TaskLogService`
- 启动入口已有：
  - `admin/internal/bootstrap/hooks.go`
  - 已接入 `server.NewAsynqServer(appCtx)`

当前缺口主要在：

- task service 只有基础接口，缺少完整调度编排实现
- task group 的批量控制逻辑未落地
- task log 仅有资源接口，缺少真实执行写入链路
- 默认菜单、权限点、角色授权、初始任务数据均未补齐
- 前端尚无任务管理聚合页与任务日志页
- “删除过期日志”任务执行器尚未实现

## 4. 菜单与权限规划

### 4.1 菜单组织

新增一个与“系统管理”“审计日志”平级的一级菜单，建议命名：

- `任务调度`

其下挂两个二级菜单：

- `任务管理`
- `任务执行日志`

建议路径与组件规划：

- `menu.task.moduleName` -> 任务调度
- `menu.task.task` -> 任务管理
- `menu.task.log` -> 任务执行日志

前端视图目录建议：

- `admin-ui/apps/web-antd/src/views/task/index.vue`
- `admin-ui/apps/web-antd/src/views/task/log/index.vue`

其中：

- `任务管理` 页面合并展示“任务分组 + 任务管理”
- `任务执行日志` 页面独立

### 4.2 权限点规划

继续使用当前“功能分组 + 服务分组导出”的权限模型。

功能分组需生成的菜单权限点至少包括：

- `tasks:view`
- `tasks:create`
- `tasks:edit`
- `tasks:delete`
- `tasks:start`
- `tasks:stop`
- `tasks:run:once`
- `task:groups:create`
- `task:groups:edit`
- `task:groups:delete`
- `task:groups:start`
- `task:groups:stop`
- `task:groups:run:once`
- `task:logs:view`
- `task:logs:delete`
- `task:logs:export`

说明：

- 虽然前端合并页面，但后端仍保留任务分组与任务管理独立权限点
- 页面内按钮显隐仍按分组操作权限、任务操作权限、日志操作权限分别判断

如果菜单按钮上显式配置 authority，则优先沿用菜单 authority；否则由 API 聚合规则自动归并。

## 5. 后端开发计划

### 第一阶段：调度基础设施接入

目标：让 `admin` 启动时能够从任务表恢复调度，并为后续任务操作提供统一入口。

实施项：

1. 在 `admin/internal/server` 补充 task scheduler 适配层
2. 为 `TaskService` 注入 scheduler 能力，而不是仅停留在 repo CRUD
3. 在 `server.NewAsynqServer(appCtx)` 中完成：
   - 创建调度器
   - 注册内置任务执行器
   - 加载已启用任务
   - 注册周期任务
4. 约束调度范围：
   - 先支持基于 `cron_expression` 的周期任务
   - `RunOnce` 走一次性投递
   - `Start/Stop` 通过调度器增删注册项实现

建议新增的内部抽象：

- `TaskScheduler`
  - 注册周期任务
  - 移除周期任务
  - 投递立即执行任务
  - 刷新单任务调度
  - 批量加载全部启用任务
- `TaskExecutor`
  - 根据 `invoke_target` 执行业务逻辑
  - 输出 `task_log`

### 第二阶段：TaskService / TaskGroupService 完整业务化

目标：把当前资源 CRUD 补成真正可用的调度服务。

`TaskService` 需要补齐：

- `Create`
  - 参数校验
  - cron 合法性校验
  - `invoke_target` 合法性校验
  - 持久化后根据状态决定是否注册调度
- `Update`
  - 更新后刷新调度
  - 如 cron / args / status 变化，移除旧注册再注册新任务
- `Delete`
  - 删除前停止调度
- `Start`
  - 将任务状态切到运行态并注册调度
- `Stop`
  - 将任务状态切到停止态并移除调度
- `RunOnce`
  - 支持临时输入参数覆盖
  - 触发一次执行并写入 task_log

`TaskGroupService` 需要补齐：

- 分组 CRUD 约束
  - 删除前检查是否仍有关联任务
- `Start`
  - 批量启动组下全部任务
- `Stop`
  - 批量停止组下全部任务
- `RunOnce`
  - 批量执行组下全部任务

建议策略：

- 任务分组本身不单独维护一个复杂运行状态
- 分组控制仅作为“批量操作入口”
- 真正调度状态仍以 task.status 与 scheduler 注册结果为准

### 第三阶段：任务执行日志链路

目标：任务一旦执行，无论成功失败，都能形成可查询日志。

实施项：

1. 增加统一执行包装器
2. 记录字段：
   - `task_id`
   - `input`
   - `output`
   - `error`
   - `status`
   - `process_time`
   - `execute_time`
   - `tenant_id`
3. 失败不影响日志入库
4. `RunOnce` 与定时调度走同一条执行包装器

建议补充能力：

- task log 列表支持按 `task_id`、状态、执行时间范围过滤
- task log 详情展示长文本输出与错误信息

### 第四阶段：内置任务执行器

目标：先落地一个真正可用的系统维护任务，并形成后续扩展模式。

本期先实现一个内置 `invoke_target`：

- `system:cleanup:audit-logs`

参数建议统一使用 JSON 字符串存入 `task.args`，例如：

```json
{
  "expireHours": 720,
  "targets": ["api", "login", "permission"]
}
```

执行逻辑：

1. 解析 `expireHours`
2. 计算阈值时间：`now - expireHours`
3. 按 `targets` 删除以下日志表中阈值时间之前的数据
   - API 审计日志
   - 登录审计日志
   - 权限审计日志
4. 输出删除结果摘要到 `task_log.output`

建议输出示例：

```text
cleanup completed: api=120, login=48, permission=7
```

实现建议：

- 不直接写 SQL
- 复用现有 repo / ent 能力
- 单个目标删除失败时记录失败并终止本次执行
- 后续如需扩展“操作审计日志”“策略评估日志”，只需要扩展 `targets`

### 第五阶段：初始化数据

目标：新系统启动或同步初始化后，任务菜单、权限与默认任务即可使用。

需补齐的初始化数据：

1. 默认菜单
   - 任务调度
   - 任务管理
   - 任务执行日志
2. 默认功能权限点
   - 菜单查看
   - CRUD
   - 启动/停止/立即执行
   - 日志查看/删除/导出
3. 默认角色授权
   - 平台管理员可获得任务调度全量功能权限
4. 默认任务分组
   - `系统维护`
5. 默认任务
   - `删除过期日志`

默认任务建议初值：

- 分组：`系统维护`
- 任务名称：`删除过期日志`
- 任务类型：`FUNCTION`
- 调用目标：`system:cleanup:audit-logs`
- cron：`0 3 * * *`
- 参数：

```json
{
  "expireHours": 720,
  "targets": ["api", "login", "permission"]
}
```

- 状态建议：`STOPPED`

说明：

- 默认停用更安全，避免初始化后立即清理用户已有历史数据
- 如果后续明确要求开箱即用，可改为 `RUNNING`

## 6. 前端开发计划

### 第一阶段：API 与类型整理

目标：确保 `admin-ui` 使用的是当前新版 task API，而不是历史遗留接口。

实施项：

1. 清理 task 旧生成物残留
2. 重新生成 `admin-ui` 的 task / task-group / task-log API 客户端
3. 确认不再残留旧接口概念，例如：
   - `ListTaskTypeNameResponse`
   - `RestartAllTaskResponse`
   - `ControlTaskRequest`

### 第二阶段：任务管理聚合页面

页面目标：

- 在一个页面内同时组织任务分组与任务管理
- 页面结构借鉴“字典管理”或“权限点管理”的组织方式
- 左侧或上方展示任务分组
- 右侧或下方展示当前分组下的任务列表
- 在同一页面内完成分组操作与任务操作

实现建议：

- 页面风格参考现有系统管理复合页
- 使用 `Page + useVbenVxeGrid + VbenForm + Modal/Drawer`

分组区域字段建议：

- 分组ID
- 分组名称
- 备注
- 租户ID
- 创建时间
- 操作列

任务区域字段建议：

- 任务ID
- 任务名称
- 所属分组
- 任务类型
- cron 表达式
- 调用目标
- 重试次数
- 是否并发
- 状态
- 最近更新时间
- 操作列

分组区域操作建议：

- 新增分组
- 编辑分组
- 删除分组
- 启动整组
- 停止整组
- 立即执行整组

任务区域操作建议：

- 新增任务
- 编辑任务
- 删除任务
- 启动任务
- 停止任务
- 立即执行任务
- 查看最近执行状态

表单重点：

- `group_id` 下拉选择
- `task_type` 枚举选择
- `cron_expression` 文本输入
- `invoke_target` 下拉或可选输入
- `args` JSON 文本输入
- `retry` 数值限制 0-5
- `concurrent` 开关
- `status` 选择
- `remark` 文本域

可以增加两个增强能力，但放在本期次优先：

- cron 填写说明
- args JSON 示例提示

### 第三阶段：任务执行日志页面

页面目标：

- 查询任务执行日志
- 查看日志详情
- 删除日志
- 导出日志

页面风格建议直接参考：

- `admin-ui/apps/web-antd/src/views/app/log/api-audit-log/index.vue`

列表字段建议：

- 日志ID
- 任务名称
- 状态
- 执行时间
- 耗时
- 输入摘要
- 输出摘要
- 错误摘要

详情弹窗建议展示：

- 输入
- 输出
- 错误信息
- 原始执行时间

## 7. 调度实现方案

### 7.1 启动链路

目标链路：

1. 应用启动
2. `bootstrap.NewManualServers`
3. `server.NewAsynqServer(appCtx)`
4. 初始化 task scheduler
5. 注册任务执行器
6. 从数据库加载可运行任务
7. 注册周期任务

### 7.2 执行链路

目标链路：

1. 定时触发或点击立即执行
2. 进入统一 task handler
3. 按 `invoke_target` 路由到具体执行器
4. 执行业务
5. 写入 `task_log`
6. 返回执行结果

### 7.3 刷新策略

任务发生以下变更时，应立即同步调度器：

- 创建任务
- 更新任务
- 删除任务
- 启动任务
- 停止任务

推荐实现：

- 单任务刷新优先
- 必要时提供全量重载方法，用于初始化或故障恢复

## 8. 删除过期日志任务的专项方案

### 8.1 功能目标

定期清理以下日志资源中的历史数据：

- API 审计日志
- 登录审计日志
- 权限审计日志

### 8.2 参数约束

- `expireHours` 必填，单位小时，必须大于 0
- `targets` 必填，至少包含一个目标
- 允许值：
  - `api`
  - `login`
  - `permission`

### 8.3 执行结果

成功时：

- 返回各类日志删除条数
- 写入 task log

失败时：

- 记录失败原因
- task log 标记失败

### 8.4 安全约束

- 不允许 `expireHours <= 0`
- 不允许空 `targets`
- 删除操作必须带时间条件
- 默认任务初始状态建议关闭

## 9. 分阶段实施顺序

建议按以下顺序落地：

1. 后端调度基础设施
2. 内置任务执行器与 task log 写入
3. task / task_group service 业务补齐
4. 默认菜单、权限、初始化任务数据
5. 任务管理聚合页
6. 任务执行日志页
7. 联调与验证

这样安排的原因：

- 后端链路先稳定，前端才有明确对接目标
- 初始化数据应在菜单、权限、页面路径稳定后再一次性补齐
- “删除过期日志”是第一条真实任务，适合作为联调样例贯穿全流程

## 10. 验收标准

后端验收：

- 应用重启后可从 `task` 表恢复已启用任务
- 任务的启动、停止、立即执行都能生效
- 执行结果能落到 `task_log`
- “删除过期日志”能按参数删除三类日志
- 初始化后可看到默认任务分组与默认任务

前端验收：

- 菜单中出现“任务调度”一级菜单
- 能在一个聚合页面内完成任务分组 CRUD 与批量操作
- 能在同一页面内完成任务 CRUD 与启动/停止/立即执行
- 能查看任务执行日志详情
- 权限不足时按钮按现有访问控制逻辑收敛显示

数据验收：

- 菜单、权限点、角色授权与当前权限初始化体系一致
- 默认任务分组为“系统维护”
- 默认任务为“删除过期日志”

## 11. 风险与注意事项

### 11.1 API 边界风险

当前 proto 里虽然已有 `Start/Stop/RunOnce`，但前端生成代码可能仍残留旧接口产物。正式开发前需先做一次 task 相关前端 API 整理。

### 11.2 调度与状态一致性

数据库中的 `task.status` 与内存调度器注册状态必须保持一致。需要明确：

- 启动成功后再写运行态，还是先写状态再注册
- 更新失败时如何回滚

建议优先保证“数据库状态表达用户意图，调度器按结果补偿刷新”。

### 11.3 多租户边界

task / task_group / task_log 均带 `tenant_id`。本期默认初始化任务建议使用全局租户 `0`。后续若扩展租户级任务，需要再明确：

- 租户可见性
- 租户级调度隔离
- 执行上下文中的租户注入

### 11.4 参考项目差异

`D:\GoProjects\chnxq\xadmin` 的任务调度实现可借鉴启动与注册思路，但其 task 模型与当前 `admin` 新版结构并不一致，不能直接复制。

## 12. 本计划对应的首轮开发拆分

建议第一轮直接完成以下最小闭环：

1. 后端调度器启动与 task 表加载
2. `删除过期日志` 执行器
3. task log 写入
4. 初始化菜单、权限、任务分组、默认任务
5. 任务管理聚合页
6. 任务执行日志页

说明：

- 后端仍保持 `task_group`、`task`、`task_log` 三套资源独立
- 前端只合并“任务分组 + 任务管理”界面，不合并数据模型与服务边界
