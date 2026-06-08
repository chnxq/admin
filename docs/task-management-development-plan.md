# Task Management Development Plan

## 1. Goal And Scope

本计划用于说明 `admin` / `admin-ui` 中任务调度能力的设计目标、已落地内容和后续收尾方向。

目标范围：

- 任务分组管理
- 任务管理
- 任务执行日志
- 启动、停止、立即执行等操作
- 应用启动时从 `task` 表恢复调度
- 初始内置任务与默认菜单/权限联动
- 至少一条真实内置任务执行链路

当前基础数据模型：

- `task_group`
- `task`
- `task_log`

## 2. 已落地的核心结果

### 后端

已完成：

- task 三表结构与 proto 对齐
- task runtime 分层重构
- 启动时恢复调度
- `RunOnce` / 调度执行统一走 runtime runner
- 执行日志写入 `task_log`
- 内置任务样例：
  - `system:cleanup:audit-logs`
  - `system:task:runtime-summary`
- 多轮 runtime bug 修复：
  - runtime repo 读取缺少 `ViewerContext`
  - 调度前置失败未写 `task_log`
  - cleanup / task_log repo 运行时调用缺少 `ViewerContext`
  - 非法 cron 只跳过单任务，不阻塞服务启动

### 前端

已完成：

- “任务调度”一级菜单
- 合并式任务管理界面
  - 左侧任务分组
  - 右侧任务列表
- 任务执行日志界面
- 多轮与系统现有列表风格对齐
- 任务日志按 `taskId` 解析并显示任务名称
- cron 编辑器与自然语言展示

## 3. 当前资源边界

后端保持三套资源独立：

- `task_group`
- `task`
- `task_log`

前端只做界面聚合，不合并资源模型。

## 4. 菜单与权限

当前目标菜单结构：

- `任务调度`
  - `任务管理`
  - `任务执行日志`

推荐权限点保持独立：

- 任务分组相关
  - `task:groups:create`
  - `task:groups:edit`
  - `task:groups:delete`
  - `task:groups:start`
  - `task:groups:stop`
  - `task:groups:run:once`
- 任务相关
  - `tasks:view`
  - `tasks:create`
  - `tasks:edit`
  - `tasks:delete`
  - `tasks:start`
  - `tasks:stop`
  - `tasks:run:once`
- 任务日志相关
  - `task:logs:view`
  - `task:logs:delete`
  - `task:logs:export`

## 5. 当前后端实现状态

### 调度基础设施

当前已不是计划状态，而是已实现状态：

- `Scheduler` 已启动并支持恢复任务
- `Runner` 已统一编排执行与日志写入
- `Registry` 已经只负责 invoke target 分发
- runtime 装配责任已收敛到 `internal/task` 与 `bootstrap/task_runtime_ext.go`

### 任务执行日志

已具备：

- 成功/失败状态
- 输入/输出/错误信息
- 执行时间
- 处理耗时
- 租户字段

### 内置任务

当前默认样例：

- 删除过期日志
- 任务运行时概览

## 6. 当前前端实现状态

已落地：

- 合并式任务管理页面
- 任务日志列表与详情
- cron 规则编辑与自然语言展示
- 多轮按钮、工具栏、表格风格对齐

当前前端工作不应再按“空壳页面”理解，而应在现有实现上继续精修。

## 7. 默认数据与初始化

默认任务相关初始化现已纳入：

- `internal/data/bootstrap/default_data_ext.go`

当前初始化策略：

- 资源同步每次启动执行
- 业务默认种子仅在 seed-domain 为空时执行

这意味着：

- 不再因为重启反复刷新默认任务、默认角色、默认用户
- 但菜单/API/权限同步仍然每次执行

## 8. 第一条真实内置任务

### 任务

- `system:cleanup:audit-logs`

### 参数

```json
{"expireHours":720,"targets":["api","login","permission"]}
```

### 语义

根据阈值时间清理：

- API 审计日志
- 登录审计日志
- 权限审计日志

执行结果写入 `task_log.output`。

## 9. 当前未完成或适合继续推进的部分

- 增加真实集成验证：
  - 启动恢复 runnable task
  - 定时触发
  - 失败日志写入
  - 非法 cron 跳过
- 继续梳理任务权限初始化与默认角色授予
- 继续补充更多真实内置任务样例
- 视需要决定默认种子后续是否演进为版本化 seed migration

## 10. 注意事项

- cron 当前统一为 6 段格式：

```text
秒 分 时 日 月 周
```

- 前端 cron 编辑器与后端校验必须保持一致。
- 不要恢复旧的 service 级 runtime 依赖注入方式。
- 不要把具体任务依赖重新耦合回 `TaskService` / `TaskGroupService` / scheduler。

## 11. 相关文档

- `admin/docs/task-runtime-architecture.md`
- `admin/docs/task-decoupling-design.md`
- `admin/docs/task-executor-convention.md`
