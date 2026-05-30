# xkit-helper Phase 3 功能差异评审

## 1. 评审范围

- 评审日期：2026-05-25
- 目标项目：`D:\GoProjects\XAdmin\admin`
- 参考项目：`D:\GoProjects\XAdmin\admin-org`
- `ReferenceSource` 解析方式：本地路径
- 目标项目基线提交：`0c27fd4`
- 参考项目提交：`cba6fe1`
- 目标项目当前状态：包含 Phase 2 对齐后的未提交改动

本评审按 `xkit-helper` Phase 3 要求，从功能差异出发，而不是只列文件 diff。评审重点包括认证、验证码、权限、菜单、审计、分析页、站内消息、默认数据、SQL 资产和生成器边界。

## 2. 验证基线

已执行目标项目测试：

```powershell
cd D:\GoProjects\XAdmin\admin
go test ./...
```

结果：通过。

已执行关系与审计回归测试：

```powershell
cd D:\GoProjects\XAdmin\admin
go test ./internal/data/repo
```

结果：通过。覆盖用户关系回填与维护、用户删除关系清理、角色权限关系维护、权限菜单/API 绑定、角色/权限删除关系清理、权限审计日志写入。

已执行 xkit 生成器测试：

```powershell
cd D:\GoProjects\XAdmin\xkit
go test ./cmd/xkit ./internal/...
```

结果：通过。

已执行参考项目测试：

```powershell
cd D:\GoProjects\XAdmin\admin-org
go test ./...
```

结果：通过。

已执行后端菜单组件到前端页面的路径扫描：

```text
admin: all backend menu components map to frontend views
admin-org: all backend menu components map to frontend views
```

前端运行契约核对结果：

- `admin-ui/apps/web-antd/src/preferences.ts` 使用 `accessMode: 'backend'`。
- `admin-ui/apps/web-antd/src/api/admin/user.ts` 的默认 `homePath` 是 `/analytics`。
- `admin` 与 `admin-org` 的后端菜单组件均能映射到 `admin-ui/apps/web-antd/src/views/**/*.vue`。
- 本轮未执行带登录态的浏览器 E2E，只完成代码级和测试级验证。

## 3. 功能差异结论

### 3.1 认证、会话与验证码

目标状态：已基本对齐参考项目。

参考状态：`admin-org` 已具备密码校验、token 支持、验证码生成和登录辅助逻辑。

证据：

- `internal/server/auth_password_ext.go` 与参考项目一致。
- `internal/server/auth_support_ext.go` 与参考项目一致。
- `internal/server/captcha_ext.go` 与参考项目一致。
- `internal/server/viewer_auth.go` 与参考项目一致。

差异分类：已对齐。

建议：暂无功能修复项。后续如果接入真实部署环境，应补充登录成功、错误密码、错误验证码、token 过期和刷新 token 的接口级测试。

### 3.2 后端动态路由、菜单与 `/analytics`

目标状态：目标项目已具备参考项目能力，并在 `/analytics` 可达性上做了更强兜底。

参考状态：`admin-org` 的菜单配置包含 `/dashboard -> /analytics`，但 `GetNavigation` 直接返回按角色菜单过滤后的结果；当用户角色或菜单为空时，参考项目可能返回空路由。

目标差异：

- `admin/internal/server/manual_http_data.go` 中 `GetNavigation` 在空菜单时返回 `defaultNavigationRoutes()`。
- `admin/internal/server/manual_http_data.go` 增加 `loadUserRoleIDs`，优先通过 `repo.UserRoleIDReader` 从用户角色关系读取角色。
- `admin/internal/data/repo/user_repo_ext.go` 增加 `ListRoleIDsByUserID`，避免只依赖 regenerated `userRepo.Get().GetRoleIds()`。

差异分类：有意增强。

建议：保留目标项目当前实现。Phase 2 的缺陷根因是新生成的 `user_repo.gen.go` 不再回填 `role_ids`，如果只按参考项目的 `GetNavigation` 逻辑复制，会让 backend access 模式下的 `/analytics` 继续不可达。

### 3.3 用户管理、组织、岗位和角色关系

目标状态：已修复并回归到生成器扩展边界。

参考状态：`admin-org/internal/data/repo/user_repo.gen.go` 中包含较完整的用户关系逻辑：

- `attachRelations` 会在用户列表和详情中回填组织、岗位、角色 ID 和名称。
- `replaceUserOrgUnits` 会维护 `sys_user_org_units`。
- `replaceUserPositions` 会维护 `sys_user_positions`。
- `replaceUserRoles` 会维护 `sys_user_roles`。
- `Create` 和 `Update` 会调用上述关系维护逻辑。

修复方式：

- `xkit/internal/codegen/template/repo_file.tmpl` 为 `List`、`Get`、`Create`、`Update`、`Delete` 生成可选 hook。
- `admin/internal/data/repo/user_repo_ext.go` 实现 `userEnrichListDTOs`、`userEnrichGetDTO`、`userCustomCreate`、`userCustomUpdate`、`userCustomDelete`。
- 用户列表和详情会回填组织、岗位、角色 ID、名称和角色编码。
- 用户创建和更新会维护 `sys_user_org_units`、`sys_user_positions`、`sys_user_roles`。
- 用户创建和密码更新会写入规范化密码凭证。
- 用户删除会事务内清理用户组织、岗位、角色和凭证关系。

差异分类：已修复。

验证：

- `internal/data/repo/relation_crud_repo_ext_test.go` 覆盖用户创建关系落库、详情关系回填、密码凭证规范化、用户删除关系清理。

### 3.4 角色、权限点、菜单/API 绑定和权限审计

目标状态：已修复并保留在扩展文件中。

参考状态：`admin-org` 的角色和权限生成文件中包含运行时业务逻辑：

- `role_repo.gen.go` 会在角色创建、更新、删除时维护 `role_permissions`。
- `role_repo.gen.go` 会写入权限审计日志。
- `permission_repo.gen.go` 会维护 `permission_menus` 和 `permission_apis`。
- `permission_repo.gen.go` 会在权限点创建、更新、删除时写入权限审计日志。

修复方式：

- `admin/internal/data/repo/role_repo_ext.go` 实现角色列表/详情权限回填，以及角色创建、更新、删除的自定义 CRUD hook。
- 角色创建和更新会维护 `sys_role_permissions`。
- 角色删除会事务内清理 `sys_role_permissions` 和 `sys_user_roles`，提交后写入权限审计日志。
- `admin/internal/data/repo/permission_repo_ext.go` 实现权限点列表/详情菜单/API 回填，以及权限点创建、更新、删除的自定义 CRUD hook。
- 权限点创建和更新会维护 `sys_permission_menus` 和 `sys_permission_apis`。
- 权限点删除会事务内清理 `sys_permission_menus`、`sys_permission_apis` 和 `sys_role_permissions`，提交后写入权限审计日志。

差异分类：已修复。

验证：

- `internal/data/repo/relation_crud_repo_ext_test.go` 覆盖角色权限关系落库、角色删除关系清理、权限菜单/API 绑定落库、权限删除关系清理、角色和权限点审计日志写入。

### 3.5 审计日志和 DB logging

目标状态：已基本对齐参考项目。

参考状态：`admin-org` 支持登录日志、API 日志、权限审计日志、操作日志和 DB logging 相关扩展。

目标状态：

- `internal/bootstrap/db_logging_ext.go` 已同步。
- `internal/data/repo/audit_log_repo_wrappers_ext.go` 已同步。
- `internal/data/repo/audit_log_time_filter_ext.go` 已同步。
- `internal/data/repo/api_audit_log_repo_ext.go`、`login_audit_log_repo_ext.go`、`permission_audit_log_repo_ext.go` 等扩展已同步。

差异分类：已对齐。权限审计触发链已通过角色和权限点 CRUD hook 补齐。

建议：

- DB logging 和日志查询能力可视为可用。
- 权限日志写入已具备 repo 层测试覆盖，后续仍需要用真实前端操作执行一次端到端验证。

### 3.6 分析页和访问量数据

目标状态：已基本对齐参考项目。

参考状态：`admin-org/internal/server/manual_http_data.go` 的分析页使用用户数量和 API 审计日志聚合数据。

目标状态：

- `admin/internal/server/manual_http_data.go` 已包含 `GetAnalyticsDashboard`。
- `apiAnalyticsReader.AnalyticsSummary` 接口存在，数据来源是 API 审计日志聚合。
- 前端 `homePath` 是 `/analytics`，目标后端默认路由也包含 `/analytics`。

差异分类：已对齐并增强。

建议：

- 后续需要用真实登录态访问分析页，确认 API audit log 写入后访问量不再为 0。
- 如果仍出现访问量为 0，应优先检查 `rest-enable_db_logging`、API 审计日志写入链路和时间范围过滤，而不是菜单路由。

### 3.7 站内消息和 SSE

目标状态：已基本对齐参考项目。

参考状态：`admin-org` 使用手写站内消息 repo，支持消息收件人状态、撤销规则和 SSE server 共享。

目标状态：

- `internal/data/repo/internal_message_repo.go` 已存在。
- `internal/data/repo/internal_message_recipient_repo.go` 已存在。
- `internal/data/repo/internal_message_types.go` 已存在。
- `internal/service/internal_message_service.gen.go` 与参考项目一致。
- `internal/service/internal_message_service_ext.go` 与参考项目一致。
- `internal/server/sse.go` 中共享 SSE server 逻辑已对齐。

差异分类：已对齐。

建议：

- 保留删除 `internal_message_repo.gen.go` 和 `internal_message_recipient_repo.gen.go` 的策略，避免与手写 repo 冲突。
- 后续应把站内消息 repo 的手写实现回归到 xkit 模板或生成器策略，否则下一次全量生成仍会产生空接口型 repo。

### 3.8 默认数据和 SQL 资产

目标状态：已对齐参考项目。

参考状态：包含 MySQL/PostgreSQL 默认数据和演示数据。

目标状态：

- `sql/mysql-default-data.sql` 存在且大小与参考项目一致。
- `sql/mysql-demo-data.sql` 存在且大小与参考项目一致。
- `sql/postgresql-default-data.sql` 存在且大小与参考项目一致。
- `sql/postgresql-demo-data.sql` 存在且大小与参考项目一致。
- `internal/data/bootstrap/default_data_ext.go` 与参考项目一致。

差异分类：已对齐。

建议：暂无功能修复项。后续如默认菜单或权限码继续变化，应优先回归到 `xkit/examples/admin` 或 SQL 生成素材。

### 3.9 文档、生成产物和模板边界

目标状态：存在可接受差异和一个文档补齐项。

参考状态：

- `admin-org` 有 `README.en.md`、`README.zh.md` 和 `docs/generated-code-files.md`。
- `admin-org` 有旧兼容文件 `internal/data/ent/query_modify_ext.go`。

目标差异：

- `admin` 当前没有 `README.en.md`、`README.zh.md` 和 `docs/generated-code-files.md`。
- `admin` 没有 `internal/data/ent/query_modify_ext.go`，因为当前 Ent 生成已通过 `sql/modifier` 生成 `Modify(...)` 能力。
- `admin` 有 `.generated-ui/apps/web-antd/src/api/generated/admin/service/v1/index.ts`，这是 Phase 1 生成 TypeScript API 的目标产物，参考项目没有该文件。

差异分类：

- 缺少 `README.en.md`、`README.zh.md`、`docs/generated-code-files.md`：optional。
- 缺少 `query_modify_ext.go`：intentional。
- 多出 `.generated-ui/.../index.ts`：intentional。

建议：

- 若目标项目需要作为长期样板仓库，应补一份 `docs/generated-code-files.md`，用于说明哪些文件由 xkit 生成、哪些文件是手写扩展。
- `README.en.md` 和 `README.zh.md` 可后置，不阻塞当前 Phase 3。

## 4. 阻塞项、重要项和有意差异

### Blocking

当前未发现会阻止目标项目启动或登录后进入首页的阻塞项。

已知 `/analytics` 问题在目标项目中通过后端默认路由兜底和用户角色关系读取增强缓解。代码级路径扫描确认后端菜单组件均能映射到前端页面。

### Important

- 真实前后端登录态 E2E 尚未执行：登录、进入 `/analytics`、用户角色变更、权限变更、消息发送/撤销、日志查询还需要在运行环境中验证。
- 站内消息手写 repo 仍未回归为通用生成能力，当前通过 `xkit/examples/admin/admin-config/admin.yaml` 关闭 `internal_message` 和 `internal_message_recipient` 的 repo CRUD 生成来避免冲突。

### Optional

- 补齐目标项目文档：`README.en.md`、`README.zh.md`、`docs/generated-code-files.md`。
- 增加带真实数据库的集成测试，覆盖用户、角色、权限、菜单、分析页和站内消息。

### Intentional

- 不复制 `internal/data/ent/query_modify_ext.go`，因为新 Ent 生成已具备 `Modify(...)`。
- 保留 `.generated-ui` TypeScript API 输出，这是新生成流程的产物。
- 目标项目 `GetNavigation` 比参考项目多了空菜单默认路由兜底，这是为 backend access 前端首页可达性做的修复。
- 目标项目通过 `UserRoleIDReader` 读取角色关系，是对 regenerated `userRepo.Get().GetRoleIds()` 为空问题的修复。

## 5. 下一步建议

1. 在真实前后端环境执行一次登录态 E2E：登录、进入 `/analytics`、用户角色变更、权限变更、消息发送/撤销、日志查询。
2. 将站内消息手写 repo 能力继续回归到 xkit 生成器或模板策略，避免后续项目仍需删除生成 repo。
3. 为 relation-aware CRUD 设计更结构化的 xkit 配置，而不是只依赖当前 optional hook 命名约定。
4. 如目标项目需要作为长期样板仓库，补齐 `docs/generated-code-files.md`、`README.en.md` 和 `README.zh.md`。
