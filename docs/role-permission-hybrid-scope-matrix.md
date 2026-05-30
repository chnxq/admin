# Role / Permission Hybrid Scoped Matrix

## 1. 目标

明确 `role` 与 `permission` 在单库多租户下的访问、写入、绑定边界，避免后续在 repo / service / UI 中各自实现出不一致语义。

本文件只定义第一版可落地规则，优先保证：

- 平台管理员能力完整
- 租户管理员能力清晰
- 租户不能越权修改平台全局授权定义
- 后续可以平滑扩展到模板同步、租户复制、局部覆写

---

## 2. 资源分类

### 2.1 role

`HybridScoped`

当前按两类处理：

- 平台全局角色：`tenant_id IS NULL` 或 `tenant_id = 0`
- 租户角色：`tenant_id = 当前租户 ID`

### 2.2 permission

第一版按 `GlobalScoped Definition` 处理：

- permission 主体由平台统一维护
- menu / api 绑定由平台统一维护
- 租户只消费 permission 结果，不直接维护 permission 定义

备注：

- 这意味着 `permission` 最终分类仍可视为 `HybridScoped`
- 但第一版实现先只开放平台写，租户读

---

## 3. 访问矩阵

### 3.1 role

平台上下文：

- 可查询全部角色
- 可读取全部角色详情
- 可创建平台全局角色
- 可创建任意租户角色
- 可修改全部角色
- 可删除全部角色

租户上下文：

- 可查询：
  - 平台全局角色
  - 本租户角色
- 可读取：
  - 平台全局角色
  - 本租户角色
- 不可读取其它租户角色
- 可创建：
  - 本租户角色
- 不可创建：
  - 平台全局角色
  - 其它租户角色
- 可修改：
  - 本租户角色
- 不可修改：
  - 平台全局角色
  - 其它租户角色
- 可删除：
  - 本租户角色
- 不可删除：
  - 平台全局角色
  - 其它租户角色

### 3.2 permission

平台上下文：

- 可查询全部 permission
- 可读取全部 permission
- 可创建 permission
- 可修改 permission
- 可删除 permission
- 可维护 permission 与 menu / api 绑定

租户上下文：

- 可查询全部 permission
- 可读取全部 permission
- 不可创建 permission
- 不可修改 permission
- 不可删除 permission
- 不可维护 permission 与 menu / api 绑定

---

## 4. 绑定规则

### 4.1 role -> permission

第一版允许：

- 平台全局角色绑定全局 permission
- 租户角色绑定全局 permission

第一版不允许：

- 绑定不存在的 permission
- 绑定未来可能出现的“其它租户私有 permission”

结论：

- 当前 `role_permission` 可保留 `tenant_id`
- 但其 tenant 语义来自 role，而不是来自 permission 主体

### 4.2 user -> role

第一版建议：

- 平台用户可分配全部角色
- 租户用户只能分配：
  - 平台全局角色
  - 本租户角色

明确禁止：

- 给租户 A 用户分配租户 B 角色

### 4.3 permission -> menu / api

第一版建议：

- menu / api 视为平台全局资源
- permission 绑定 menu / api 仅平台可写
- 租户只读取绑定结果

---

## 5. repo 层落地顺序

### 5.1 roleRepo

第一批必须落地：

- `List`：
  - 平台上下文不过滤
  - 租户上下文仅返回 `tenant_id IS NULL OR tenant_id = viewerTenantID`
- `Get`：
  - 租户上下文禁止读取其它租户角色
- `Create`：
  - 平台上下文可按传入 tenant_id 创建
  - 租户上下文默认写入 viewer tenant
  - 租户上下文禁止写入空 tenant 或其它 tenant
- `Update/Delete`：
  - 平台上下文可修改全部
  - 租户上下文仅可修改本租户角色
- `replaceRolePermissions`：
  - 租户上下文仍允许绑定全局 permission

### 5.2 permissionRepo

第一批必须落地：

- `List/Get`：
  - 平台与租户都可读
- `Create/Update/Delete`：
  - 仅平台上下文可写
- `replacePermissionMenus / replacePermissionApis`：
  - 仅平台上下文可写

---

## 6. 前端约束

第一版 UI 应同步反映：

- 租户上下文中：
  - role 列表可显示“全局角色 / 本租户角色”标识
  - 平台全局角色的编辑、删除按钮应禁用
  - permission 页面只读
- 平台上下文中：
  - role 可区分 tenant 归属
  - permission 正常可编辑

---

## 7. 后续演进

后续如需增强，可按下面方向演进：

- 平台角色模板 -> 租户复制
- 租户 permission 局部扩展
- role / permission 的模板同步版本控制
- menu 可见性租户覆写

但这些都不属于第一版落地范围。
