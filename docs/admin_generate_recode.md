# admin 后端生成与 Phase 2 对齐记录

本文记录本仓库从 `xkit/examples/admin` 重新生成后端应用，并按 `xkit-helper` Phase 2 与参考项目对齐手写业务细节的实际过程。

## 1. Phase 1 生成输入

- 工作区根目录：`D:\GoProjects\XAdmin`
- 生成器仓库：`D:\GoProjects\XAdmin\xkit`
- 示例输入目录：`D:\GoProjects\XAdmin\xkit\examples\admin`
- 目标仓库：`D:\GoProjects\XAdmin\admin`
- `ProjectName`：`admin`
- `Module`：`admin`
- `AppName`：`Admin`
- `TypeScriptRoot`：`D:\GoProjects\XAdmin\admin\.generated-ui`

实际执行命令：

```powershell
D:\GoProjects\XAdmin\xkit\examples\generateAll.ps1
```

交互输入：

```text
ProjectName: admin
Module [admin]:
AppName [admin]: Admin
TypeScriptRoot [D:\GoProjects\XAdmin\admin\.generated-ui]:
Continue with this plan? [Y/n]: Y
```

`ProjectName=admin` 时，目标配置使用了避免覆盖规范配置的路径：

```text
D:\GoProjects\XAdmin\xkit\examples\admin\admin-target-config\admin.yaml
```

生成链路已按规范配置重新对齐：

- 复制 `xkit-template` 静态模板。
- 导入 `xkit/examples/admin` 下的 `api`、`schema` 和配置素材。
- 使用规范配置生成 Go API、OpenAPI、TypeScript、Ent 和 xkit 动态代码。
- Ent 生成已启用 `sql/modifier`，因此目标项目不再需要参考项目里的旧兼容文件 `internal/data/ent/query_modify_ext.go`。

Phase 1 验证结果：

```powershell
go test ./...
```

结果：通过。

## 2. Phase 2 参考项目

- `ReferenceSource`：`D:\GoProjects\XAdmin\admin-org`
- 解析方式：本地路径
- 参考项目提交：`cba6fe1`
- 目标项目基线提交：`0c27fd4`

Phase 2 的目标是恢复参考项目中不会由 Phase 1 自动生成的手写运行时行为，包括认证、验证码、审计、菜单、权限、站内消息、默认数据和 SQL 数据。

## 3. Phase 2 同步范围

已从参考项目同步的主要运行时文件：

- `internal/server/auth_password_ext.go`
- `internal/server/auth_support_ext.go`
- `internal/server/captcha_ext.go`
- `internal/server/viewer_auth.go`
- `internal/server/manual_http_data.go`
- `internal/server/manual_http_test.go`
- `internal/server/options.go`
- `internal/server/http_options.go`
- `internal/server/grpc_options.go`
- `internal/server/sse.go`

已同步的 bootstrap 和默认数据：

- `internal/bootstrap/db_logging_ext.go`
- `internal/bootstrap/generated_hooks_ext.go`
- `internal/bootstrap/generated_servers_ext.go`
- `internal/data/bootstrap/default_data_ext.go`
- `internal/data/bootstrap/ent_client_ext.go`

已同步的 repo/service 扩展：

- `internal/data/repo/*_ext.go`
- `internal/service/*_ext.go`
- `internal/data/repo/audit_log_repo_wrappers_ext.go`
- `internal/data/repo/audit_log_time_filter_ext.go`
- 相关 repo 扩展测试文件

已同步的 SQL 数据：

- `sql/mysql-default-data.sql`
- `sql/mysql-demo-data.sql`
- `sql/postgresql-default-data.sql`
- `sql/postgresql-demo-data.sql`

## 4. 特殊处理

站内消息 repo 在参考项目中是手写实现，目标项目 Phase 1 生成的是空接口型 repo。为避免同名类型和构造函数冲突，已删除目标项目中的：

```text
internal/data/repo/internal_message_repo.gen.go
internal/data/repo/internal_message_recipient_repo.gen.go
```

并同步参考项目中的：

```text
internal/data/repo/internal_message_repo.go
internal/data/repo/internal_message_recipient_repo.go
internal/data/repo/internal_message_types.go
```

API 同步、权限同步和站内消息相关 special 方法在参考项目中仍位于 `*.gen.go` 中，但目标项目 Phase 1 生成结果是 TODO 空实现。为恢复参考运行时行为，本次同步了以下参考实现：

```text
internal/service/api_service.gen.go
internal/service/permission_service.gen.go
internal/service/internal_message_service.gen.go
internal/service/internal_message_recipient_service.gen.go
```

这属于当前生成器边界问题，后续应优先回归到 xkit 配置或 special method 生成策略中，而不是长期依赖手工覆盖 `*.gen.go`。

由于目标项目的 Ent 代码已经通过 `sql/modifier` 生成了查询 `Modify(...)` 方法，本次没有复制参考项目的：

```text
internal/data/ent/query_modify_ext.go
```

同步参考扩展后出现过两个 helper 重复定义：

```text
permissionFieldMaskContains
roleFieldMaskContains
```

修复策略是保留目标项目生成代码中的 helper，删除扩展文件里的旧定义。

## 5. Phase 2 实际命令

核心同步命令按文件边界执行，避免覆盖普通 CRUD 生成文件：

```powershell
# 读取技能和 Phase 2 热点说明
Get-Content D:\GoProjects\XAdmin\xkit\xkit-helper\SKILL.md
Get-Content D:\GoProjects\XAdmin\xkit\xkit-helper\references\phase-2-repair-hotspots.md

# 检查参考项目、目标项目和 xkit 状态
git -C D:\GoProjects\XAdmin\admin status --short
git -C D:\GoProjects\XAdmin\admin-org status --short
git -C D:\GoProjects\XAdmin\xkit status --short

# 同步参考项目手写扩展、server、bootstrap、repo/service 扩展和 SQL 数据
# 具体文件范围见上文 “Phase 2 同步范围”

# 删除会与手写消息 repo 冲突的生成文件
Remove-Item D:\GoProjects\XAdmin\admin\internal\data\repo\internal_message_repo.gen.go
Remove-Item D:\GoProjects\XAdmin\admin\internal\data\repo\internal_message_recipient_repo.gen.go

# 补齐新增依赖
go mod tidy

# 验证
go test ./...
```

## 6. 验证结果

最终验证命令：

```powershell
go test ./...
```

最终结果：通过。

本阶段完成后，目标项目已经具备参考项目中的主要手写业务能力，但 Phase 3 仍需要从功能视角重新对比 `admin` 与 `admin-org`，并将剩余差异写入：

```text
D:\GoProjects\XAdmin\admin\docs\xkit-helper-feature-gap-review.md
```

## 7. Phase 3 功能差异评审

Phase 3 已按功能面完成目标项目与参考项目的差异评审，结论文件：

```text
D:\GoProjects\XAdmin\admin\docs\xkit-helper-feature-gap-review.md
```

当前优先级最高的剩余差异是用户、角色、权限相关关系表维护和权限审计触发链，应优先回归到 xkit 生成器或扩展边界中。

后续 Phase 3 修复已完成上述最高优先级差异：

- `xkit` repo 生成模板已增加可选 CRUD hook 和 DTO 关系回填 hook。
- `admin/internal/data/repo/user_repo_ext.go` 已实现用户组织、岗位、角色关系回填与维护，以及用户删除关系清理。
- `admin/internal/data/repo/role_repo_ext.go` 已实现角色权限维护、角色删除关系清理和权限审计写入。
- `admin/internal/data/repo/permission_repo_ext.go` 已实现权限点菜单/API 绑定维护、权限删除关系清理和权限审计写入。
- `admin/internal/data/repo/relation_crud_repo_ext_test.go` 已覆盖上述关系和审计链路。

当前仍需在真实前后端环境执行登录态 E2E：登录、进入 `/analytics`、用户角色变更、权限变更、消息发送/撤销、日志查询。
