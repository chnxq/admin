# 租户分库存储方案分析（基于当前仓库）

## 1. 目的

本文不是再讨论一版抽象的“未来多租户 SaaS 应该怎么做”，而是基于当前实际仓库状态重新评估：

- `admin`
- `admin-ui`
- `xkit`
- 当前已经接入的宿主模块模式（如 `modules/xdev`）

关注的问题是：

1. 如果从当前 `tenant_id` 逻辑隔离，演进到“平台库 + 租户分库”，会有哪些变化。
2. 与之前相比，这件事现在变容易了什么，变困难了什么。
3. 应该如何判断它是否值得启动。

---

## 2. 先给结论

### 2.1 结论一句话

基于当前仓库状态，**租户分库存储仍然可行，但它已经从“纯业务建模问题”变成了“宿主装配、数据源路由、认证上下文、种子初始化、模块生成链一起调整”的系统级改造**。

### 2.2 与旧分析相比，当前判断有两个变化

第一，**更容易了**：

- 后端已经形成了比较明确的租户边界基础能力。
- 前端已经开始形成“平台可读、跨租户只读、同租户可写”的交互模型。
- 模块已宿主化，`admin` 不再是所有业务代码直接堆在一起的单工程。
- `xkit` 已经具备生成 tenant-scoped CRUD、模块 bootstrap、模块资源/种子同步的能力。

第二，**也更难了**：

- 当前生成链和运行时装配，默认仍是“一个 `AppCtx`，一个数据库配置，一个 `EntClient`，一套 `GeneratedData`”。
- 当前模块虽然宿主化了，但模块 `RegisterData -> GeneratedData -> Repo -> Service` 仍默认绑定单库。
- 前端和后端现在已经有一批租户边界规则，这些规则在“单库 + tenant_id”下成立；一旦改成分库，需要重新定义哪些规则继续保留、哪些规则转化为数据源路由。

### 2.3 最终判断

如果目标是：

- 近期把租户边界补齐
- 降低跨租户脏写/误绑定风险
- 继续扩展模块体系

那么**不建议立刻全面切到租户分库**。  
更稳妥的路线是：

1. 继续把当前单库多租户的边界做扎实。
2. 先把“平台资源 / 租户资源 / 混合资源”的分类模型固定下来。
3. 再把数据源层从单库抽象成“平台库 + 租户库路由”。

如果目标是：

- 支持大租户独立部署
- 做更强物理隔离
- 为高价值租户提供独立备份、迁移、恢复

那么租户分库值得作为**第二阶段架构目标**，但应按分阶段落地，而不是一次性改造。

---

## 3. 以当前仓库为基准，哪些基础已经具备

这一节很重要，因为它直接决定“现在比以前更容易”的部分。

## 3.1 后端租户上下文已经基本打通

当前 `admin/internal/server/viewer_auth.go` 已经不是早期那种固定 `TenantID() == 0` 的状态。

现在它会：

- 通过登录 token 解析当前用户
- 从用户详情中恢复真实 `tenantId`
- 把 `crudviewer.Context` 注入请求上下文

这意味着后端已经有了一个比较稳定的“当前请求属于哪个租户”的入口。

这对未来分库很关键，因为：

- 单库模式下，这个值用于补 `WHERE tenant_id = ...`
- 分库模式下，这个值可以进一步用于“选择哪个租户库”

也就是说，**viewer tenant context 已经具备向数据源路由继续演进的基础**。

## 3.2 `modulex` 已经抽出了可复用的租户边界能力

当前 `admin/shared/modulex/module_shared_ext.go` 已经提供了明确的租户相关基础方法，例如：

- `ViewerTenantID`
- `EnsureTenantAccessible`
- `EnsureHybridTenantAccessible`
- `EnsureHybridTenantMutable`
- `ResolveCreateTenantID`
- `EnsurePlatformOnlyMutable`

这说明当前系统已经不再是“每个 repo 各自猜 tenant 规则”，而是开始把租户能力沉淀到共享层。

这对分库的意义是：

- 现有“谁可读、谁可写、创建时 tenant_id 如何决议”的规则已经部分集中
- 未来如果从“字段过滤”转为“库路由 + 少量平台/混合资源校验”，这些函数仍可作为规则承载点

换句话说，**规则层已经开始成型**，这比旧阶段成熟很多。

## 3.3 repo 层已形成一部分 tenant hook 模式

当前 `admin/internal/data/repo/tenant_scope_ext.go` 和 `xkit` 生成模板中，已经存在明显的租户 repo 策略：

- create 时通过 `ResolveCreateTenantID` 收敛 tenant
- update/delete 时通过 `EnsureTenantAccessible` 或 hybrid 变体校验
- list/count 时自动按 viewer tenant 过滤

这件事的意义是：

- 现在系统已经默认接受“tenant policy 不只在 service 层，而要下沉到 repo 层”
- 未来分库时，repo 层本来就是数据源切换的重要位置

所以从演进角度看，**repo 层已经是正确的切入层之一**。

## 3.4 前端已经形成“跨租户只读详情态”

当前 `admin-ui` 与 `xdev-ui` 已经有一条清晰规则：

- 平台管理员可以看跨租户资源
- 跨租户资源在前端显示为 `详情`，而不是 `修改`
- 确认按钮禁用
- 同时编辑弹框中的 relation 下拉，按当前记录所属租户过滤

这说明前端已经不再是“平台用户看见什么都能改”，而是开始配合后端租户边界做行为收敛。

对分库而言，这很重要，因为未来即使平台用户跨租户查看，往往也意味着：

- 平台用户读取的是某租户库中的数据
- 但仍不一定允许直接修改

当前交互模型与这个方向是兼容的。

## 3.5 宿主模块化已经落地

当前 `admin` 已经明显不是“所有业务都塞进内部目录”的结构，至少在 xdev 路线下，已经形成：

- `admin` 作为宿主
- `shared/modulehost` 作为模块注册协议
- `modules/xdev` 作为业务模块
- 模块自带：
  - data
  - repo
  - service
  - server register
  - bootstrap
  - OpenAPI
  - resource sync
  - default seed

这对租户分库最大的帮助是：

- 可以按模块分阶段迁移，而不是一次性迁移整个 admin
- 可以先选租户属性最清晰的模块试点
- 可以把“数据从平台库走还是租户库走”作为模块级能力扩展

这比旧分析时期“全部逻辑缠在 admin 内部”要好很多。

---

## 4. 但为什么它仍然是一项高成本改造

上面说的是“容易了什么”，这一节说“为什么仍然很难”。

## 4.1 当前生成与装配主链仍然是单库假设

当前主链非常清晰：

1. `bootstrap.Initialize`
2. `newAppContext`
3. `initDataResources`
4. `NewEntClient`
5. `NewGeneratedData`
6. repo/service/server 全部围绕这一个 `EntClient`

无论是 `admin/internal/data/bootstrap/ent_client.gen.go`，还是模块自己的 `modules/xdev/data/bootstrap/ent_client.gen.go`，本质上都是：

- 从 `appCtx.GetConfig().Data.Database`
- 创建一个数据库连接
- 创建一个 `*ent.Client`
- 再生成 `GeneratedData`

也就是说，当前不是“默认支持多个数据源，只是还没用”，而是**运行时骨架本身就是单库骨架**。

如果做分库，至少要回答：

- 平台库 `EntClient` 放哪里
- 租户库 `EntClient` 如何获取
- 一个请求里 repo 到底取哪个 `EntClient`
- 这个选择是请求级、repo 级还是 service 级完成

这部分不解决，下面所有分库讨论都只是概念。

## 4.2 `GeneratedData -> Repo -> Service` 依赖结构需要扩展

当前生成代码中，大量 repo 构造都是这种形式：

- `NewXxxRepo(ctx *app.AppCtx, entClient *entCrud.EntClient[*ent.Client])`

这意味着 repo 默认拿到的就是一个静态 client。

一旦分库，repo 构造可能要变成三类之一：

### 方案 A：repo 直接持有“数据源路由器”

例如：

- `TenantDBRouter`
- `PlatformDBProvider`
- `ResolveTenantClient(ctx)`

优点：

- repo 仍是真正的访问入口

缺点：

- 所有生成 repo 模板都要调整
- 单测模型也要变

### 方案 B：service 先解析 tenant，再注入对应 repo/tx

优点：

- repo 实现可以少动一点

缺点：

- service 复杂度急剧上升
- 生成 service 与手写 service 的边界会变乱

### 方案 C：请求级上下文中挂“当前 tenant 库 client”

优点：

- 对 repo 调用点改动少

缺点：

- 中间件/上下文/事务边界都要重做
- 平台库与租户库混用时更难看清

无论哪条路，都不是“小修小补”。

## 4.3 当前平台资源、租户资源、混合资源还没有完全稳定

虽然这件事已经做了一部分，但从当前仓库状态看，它仍没有完全收口。

例如当前已经能看到三类典型资源：

### 平台资源

- tenant
- 菜单定义的宿主级同步
- API 文档聚合
- 平台管理员登录与认证基础能力

### 租户资源

- org unit
- position
- role
- user 的租户内业务属性
- xdev 的 device / model / group 等

### 混合资源

- 平台用户可读，但租户内可写
- 某些平台菜单、API、权限点对全部租户可见
- 某些资源允许平台态汇总浏览

分库最怕的不是“没有 tenant_id”，而是**资源分类没定清**。  
因为分库之后首先要决定的是：

- 这张表放平台库还是租户库
- 平台是否保留投影表/汇总表
- 平台视角的列表是实时跨库查询，还是读汇总索引

如果当前分类还在变化，分库就会反复返工。

## 4.4 认证与用户模型仍然偏单库

当前 `viewer_auth.go` 的流程仍是：

1. 解析 token
2. 用 `UserRepo` 读用户
3. 用 `RoleRepo` / `PermissionRepo` 算权限

也就是说，当前“认证后的 viewer 恢复”默认依赖一个统一的用户读模型。

如果分库，要先明确：

### 是“平台身份 + 租户成员”模型，还是“每租户独立用户副本”模型？

基于当前代码，更推荐前者：

- 平台库保存 identity/account/credential/membership
- 租户库保存 tenant-local profile / 关系 / 扩展属性

原因很简单：

- 当前登录链已经天然更接近“统一身份入口”
- 如果把完整用户资料拆成多个租户库主数据，再回写平台，会让 `viewer_auth`、token、权限恢复变得非常脆弱

所以从当前仓库演进看，**分库前必须先明确用户模型，否则认证链会成为最大风险点之一**。

## 4.5 种子初始化与模块资源同步会变复杂

当前 `admin` 宿主已经形成：

- 启动时 schema create
- `afterEntSchemaCreate`
- default data seed
- host module resource sync
- host module default seed

这在单库模式下很好理解，因为一切都写到一个库里。

但分库后，至少要拆成：

- 平台库初始化
- 新租户创建时的租户库初始化
- 模块资源是写平台库，还是写租户库
- 模块 default seed 哪些属于平台，哪些属于每租户

尤其是当前 `modulehost.ResourceSyncer` 更偏向“平台菜单资源同步”，这和未来的租户库初始化将不完全同构。

所以模块化虽然让迁移更容易试点，但**种子与资源同步的职责需要重新分层**。

---

## 5. 现在做租户分库，哪些事情比以前更容易

这一节换成“容易度”视角。

## 5.1 容易度提升一：后端租户规则已有公共基础

以前要先解决“tenant 到底是谁说了算”。  
现在至少这些规则已经部分统一：

- viewer tenant 来自认证上下文
- create tenant id 由共享 helper 决议
- mutable/accessibility 已有共享 helper
- xkit 生成代码已开始使用这些 helper

所以未来做分库时，不需要再从零定义租户基础语义。

## 5.2 容易度提升二：模块试点成为可能

现在最现实的路径不是改整个 admin，而是：

1. 先把 `admin` 核心平台资源留在平台库
2. 选一类模块试点租户库
3. 由宿主在模块装配时决定该模块的数据源策略

比如 `xdev` 这种租户属性很强、平台聚合需求相对弱的模块，就是比 `identity/permission` 更适合先试点的类型。

这在旧时期几乎做不到，因为那时没有清晰模块边界。

## 5.3 容易度提升三：前端只读详情态已经减少跨租户写入压力

当前前端已经接受并实现了：

- 平台态跨租户查看
- 非同租户改写禁用
- relation 下拉按当前记录租户过滤

这意味着未来就算后端先做“平台可跨库查看”，前端也不会天然要求“平台必须可跨租户直接改写”。

这会降低一部分产品/交互阻力。

## 5.4 容易度提升四：xkit 已开始承载租户约束生成

当前 `xkit` 已经能生成：

- tenant-scoped repo hook
- 模块宿主骨架
- frontend readonly detail mode
- relation option tenant-aware 行为

这意味着一旦分库方案收敛，后续不是全靠手工修改，而是有机会把一部分模式吸收进生成器。

这对中长期成本控制非常重要。

---

## 6. 现在做租户分库，哪些事情比以前更困难

## 6.1 困难度提升一：因为现在系统更完整了，改动面更广

早期阶段很多东西没建好，反而“改起来没什么可以破坏”。  
现在不是。

当前已经有：

- 宿主模块
- OpenAPI 聚合
- 模块资源同步
- 模块默认种子
- 前端 tenant readonly 模式
- 一批 xkit 生成约束

所以分库改造不再只是“调 repo”：

- 它会影响 bootstrap
- 影响 data provider
- 影响生成器
- 影响前端平台态行为
- 影响模块约定

完整性更强，意味着联动面更大。

## 6.2 困难度提升二：平台视角的“可读汇总”需求已经存在

当前仓库已经明确支持一类平台视角：

- 平台用户可看多租户数据
- 但对跨租户记录只读

在单库里，这只是查询条件和按钮状态问题。  
到了分库，就会变成：

- 平台列表是否要跨所有租户库实时 fan-out 查询
- 是否需要平台侧维护一份只读汇总索引
- 排序、分页、筛选如何做

这比“纯租户内 CRUD”复杂很多。

## 6.3 困难度提升三：生成器与运行时需要重新划清职责

当前 `xkit` 非常擅长：

- 生成单资源 CRUD
- 生成租户边界 repo hook
- 生成模块 bootstrap

但如果做到分库，必须决定：

- 数据源路由逻辑是在宿主手写层
- 还是在模块 bootstrap
- 还是在 repo 生成模板

这个分界如果不先定清，`xkit` 很容易被拉成“大杂烩生成器”。

当前仓库已经经历过一轮“尽量共用，但不要把 host/module 搅乱”的重构，所以这条界线要比以前更谨慎。

---

## 7. 如果决定做，建议的目标架构

这里不写最终实现细节，只写当前仓库最适合演进的方向。

## 7.1 建议采用“平台库 + 租户库路由”的双层模型

### 平台库保存

- tenant 主数据
- account / credential / membership
- 平台菜单、API、权限模板、平台角色等平台资源
- 宿主模块注册信息、模块资源元数据
- 跨租户汇总索引（如果需要）

### 租户库保存

- org unit / position / role 等租户组织权限业务数据
- user 的租户业务属性
- xdev 这类强租户业务模块数据
- 租户内日志、任务、字典等适合本地化的数据

## 7.2 更推荐“平台身份 + 租户成员”而不是“双写完整用户表”

基于当前仓库现状，推荐明确为：

- 平台库：身份主数据
- 租户库：成员业务数据

不建议：

- 平台库与租户库各自保存一份完整 user，再靠 CRUD 同步维持一致

原因：

- 当前认证链已更接近统一身份入口
- 双写完整用户表会把同步、删除、密码、唯一键、授权恢复全部复杂化

## 7.3 数据源路由应先落在宿主数据层，不要先灌进业务模块

建议先做一层宿主级数据源能力，例如概念上提供：

- `PlatformEntClient()`
- `TenantEntClient(ctx, tenantID)`
- `ResolveDataScope(ctx, resourceKind)`

先在宿主 `admin/internal/data/bootstrap` 与 `GeneratedData` 这一层扩展，而不是一开始就让每个模块自己持有数据源路由逻辑。

原因：

- 当前模块宿主化是优势，不能让模块再次各自发明一套 DB routing
- 平台资源与租户资源的判定应由宿主掌握

## 7.4 模块分库存储建议按“模块能力成熟度”分阶段推进

可以按这种顺序考虑：

### 第一批更适合先试点的

- xdev 这种租户强归属模块
- 纯租户内业务模块

### 第二批再考虑的

- org unit / position / role / user profile

### 最后再碰的

- authentication
- token/session
- permission 模板与平台授权
- 平台 API/菜单/资源中心

原因不是这些不重要，而是它们和平台身份/平台资源耦合最深。

---

## 8. 推荐的落地顺序

## 8.1 第一阶段：继续把当前单库 tenant model 固化

目标：

- 把资源分类定清楚
- 把平台可读、同租户可写规则收口
- 把 repo 层 tenant hook 继续补齐
- 把 `xkit` 对 tenant-scoped / hybrid / platform-only 的生成契约稳定下来

这是所有分库的前置条件。

## 8.2 第二阶段：先做宿主级双数据源骨架，但先不迁业务

目标：

- `AppCtx` / config 支持平台库配置与 tenant 库配置
- 提供租户库路由器
- bootstrap 可创建平台 client，并按需创建 tenant client
- 先不大规模改业务表归属

这一阶段的价值是“把基础设施搭起来”，而不是立即见业务收益。

## 8.3 第三阶段：挑一个模块试点迁出

推荐优先考虑：

- `xdev`

因为：

- 租户边界清晰
- 平台汇总依赖较弱
- 已经模块化
- 前后端都已有较完整 tenant 行为验证

## 8.4 第四阶段：再决定是否迁组织权限主业务

这是最需要谨慎的一步。  
因为一旦 org unit / position / role / user membership 迁出，登录恢复、授权装配、平台视角读模型都会受到更明显影响。

---

## 9. 对当前仓库的具体影响清单

如果未来真的启动这件事，当前仓库里最先会动到的区域大致如下：

## 9.1 `admin`

### 一定会受影响

- `internal/data/bootstrap/ent_client.gen.go`
- `internal/bootstrap/generated_servers.gen.go`
- `internal/bootstrap/generated_data_providers.gen.go`
- `internal/server/viewer_auth.go`
- `internal/data/repo/*`
- `shared/modulex/module_shared_ext.go`
- `shared/modulehost/*`
- `internal/data/bootstrap/default_data_ext.go`

### 高风险点

- `GeneratedData` 的结构设计
- 平台库与租户库混合事务
- 资源同步与默认种子职责拆分

## 9.2 `admin-ui`

### 一定会受影响

- 平台态的列表页与详情页行为
- 平台汇总页的分页/排序/搜索策略
- 跨租户只读详情态是否继续保留

### 但相对容易

- 现在前端已经接受“平台可读但不一定可写”的规则
- 前端难点更多在平台汇总查询体验，而不是交互方向本身

## 9.3 `xkit`

### 一定会受影响

- repo 模板
- bootstrap/data provider 模板
- module runtime 模板
- frontend provider/page 模板中的 tenant 语义

### 关键原则

- 不要把“平台库/租户库路由策略”散落到每个生成模块里
- 应先让宿主定义 contract，再由 `xkit` 落具体模板

---

## 10. 最终建议

基于当前仓库状态，我的建议是：

### 短期建议

不要直接启动“全量租户分库改造”。

先继续完成三件事：

1. 固化资源分类：平台资源 / 租户资源 / 混合资源
2. 固化用户模型：平台身份 + 租户成员
3. 固化宿主 contract：数据源路由应由宿主掌控，不由各模块自行决定

### 中期建议

把“平台库 + 租户库”先做成宿主层骨架，但先不迁核心业务。

### 试点建议

如果一定要落一个试点，优先用 `xdev`，不要先用 `identity / permission / org structure`。

### 核心判断

与旧阶段相比，当前仓库：

- **更适合规划租户分库**
- **更不适合仓促切换租户分库**

因为现在基础已经比以前成熟得多，但也正因为更成熟，联动面更广，必须按阶段推进。

---

## 11. 一句话总结

**现在的 `admin/admin-ui/xkit` 已经具备“为租户分库做准备”的结构条件，但还没有具备“低成本直接切到租户分库”的运行时骨架；最正确的路线不是马上切库，而是先把当前单库多租户 contract 固化，再以宿主层双数据源能力为分界，按模块试点迁移。**
