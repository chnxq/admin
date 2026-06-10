# 用户注册与第三方认证改造规划

## 1. 目标与范围

本规划用于指导 `admin` 与 `admin-ui` 后续完成以下能力建设：

1. 用户注册能力补齐，包括用户名、邮箱、手机号注册。
2. 第三方账号注册/登录接入，包括微信、支付宝、抖音、GitHub。
3. 已注册用户绑定/解绑第三方账号。
4. 登录页改造，为未来微信/支付宝小程序登录、扫码登录、授权确认页预留结构。
5. 后端认证模型提前为微信/支付宝小程序认证授权做准备，避免后续再推翻现有 `authentication` 体系。

本阶段输出的是规划，不直接进入编码实现。

## 2. 现有基础盘点

### 2.1 后端已有基础

当前后端并不是从零开始：

1. `admin/internal/data/ent/schema/user_credential.go`
   已具备统一认证凭据模型，包含：
   - `identity_type`
   - `credential_type`
   - `provider`
   - `provider_account_id`
   - `extra_info`
   - 激活/重置 token 字段
2. `sys_user_credentials` 已存在 `(tenant_id, provider, provider_account_id)` 唯一索引。
   这对第三方绑定非常关键，说明“租户内第三方账号唯一绑定”已经有天然承载点。
3. `admin/api/protos/authentication/v1/authentication.proto`
   已有：
   - `RegisterUser`
   - `Login`
   - `RefreshToken`
   - `GrantType`
   - `ClientType`
4. `admin/api/protos/authentication/v1/oauth.proto`
   已经有一套 OAuth 关联账号接口雏形，例如：
   - `StartLinkOAuth`
   - `ConfirmLinkOAuth`
   - `LinkOAuth`
   - `UnlinkOAuth`
   - `ListLinkedAccounts`
5. 当前登录/刷新令牌链路已经完成基于 `xkitpkg/cache` 的 token store 改造：
   - access token / refresh token 的服务端有效性校验不再依赖数据库会话表
   - 单节点默认可走 memory cache
   - 多节点可切到 Redis / Redis Cluster
   这说明后续扫码登录、授权确认、小程序短期流程态继续落在 cache 上是符合当前系统演进方向的。

结论：后端应优先扩展现有 `authentication + user_credential + oauth`，而不是再造一套独立的“社交登录表”和“社交登录服务”。

### 2.2 前端已有基础

`admin-ui` 已有认证页基础骨架：

1. `apps/web-antd/src/views/_core/authentication/login.vue`
   当前只是关闭了：
   - 注册
   - 二维码登录
   - 第三方登录
2. 路由已存在：
   - `login.vue`
   - `code-login.vue`
   - `qrcode-login.vue`
   - `register.vue`
3. Vben 公共组件已存在：
   - `AuthenticationLogin`
   - `AuthenticationQrCodeLogin`
   - `third-party-login.vue`

结论：前端也应基于现有 Vben 认证页体系增量改造，不应重建一套完全平行的登录框架。

## 3. 设计原则

### 3.1 一个用户，多个凭据

统一原则是：

1. `sys_users` 表示用户主体。
2. `sys_user_credentials` 表示登录/认证/绑定方式。
3. 一个用户可拥有多个凭据：
   - 用户名密码
   - 邮箱密码
   - 手机号验证码
   - GitHub OAuth
   - 微信开放平台账号
   - 支付宝用户标识
   - 抖音开放平台账号
   - 微信/支付宝小程序 openid 体系

这样后续增加认证渠道时，只是新增凭据类型和流程，而不是新增一套用户体系。

### 3.2 登录、注册、绑定三条流程共用同一凭据模型

统一收敛到三个语义：

1. `登录`
   已知外部身份，找到本地用户并签发 token。
2. `注册`
   外部身份首次进入，尚未绑定本地用户，需要创建用户并建立首个凭据。
3. `绑定`
   当前已登录用户，将新的外部身份挂到自己名下。

### 3.3 将“小程序登录”和“网页 OAuth 登录”区分建模

这两类流程不能混为一谈：

1. GitHub、微信网站应用、抖音开放平台网页授权，本质是浏览器 OAuth/授权码回调流程。
2. 微信/支付宝小程序登录，本质是小程序端拿 `code/authCode`，后端向平台换取 `openid/session/user_id` 等身份信息。

因此后端接口层面要把“浏览器 OAuth 回调式登录”和“小程序 code 换 session 式登录”拆开。

### 3.4 先支持账号绑定，再支持账号合并

首期先做：

1. 已登录账号绑定第三方。
2. 第三方首次登录时可选择：
   - 自动注册新用户
   - 绑定已有账号

不建议首期就做复杂的“多账号自动合并”。

## 4. 建议的数据模型

## 4.1 继续沿用 `sys_user_credentials`

建议继续使用现有字段承担主绑定关系：

1. `identity_type`
   - 用户名/邮箱/手机号
   - 第三方 OAuth
   - 设备身份
2. `credential_type`
   - `PASSWORD_HASH`
   - `OAUTH_TOKEN`
   - `OPENID_CONNECT_ID_TOKEN`
   - `SMS_OTP`
   - `TEMPORARY_CREDENTIAL`
3. `provider`
   建议规范化为稳定字符串：
   - `github`
   - `wechat_web`
   - `wechat_miniapp`
   - `alipay_miniapp`
   - `douyin_open`
   - 后续可扩展 `wechat_official`, `apple`, `google`
4. `provider_account_id`
   存平台主标识：
   - GitHub `id`
   - 微信 `unionid/openid`
   - 支付宝 `user_id`
   - 抖音平台用户唯一标识
5. `identifier`
   建议作为“登录时用于查找的主索引值”，通常与 `provider_account_id` 一致或派生一致值。

### 4.2 建议补充一张第三方档案表

`user_credential` 足以承载绑定关系，但不足以优雅承载第三方展示资料。建议新增：

`sys_user_oauth_profiles`

建议字段：

1. `id`
2. `tenant_id`
3. `user_id`
4. `credential_id`
5. `provider`
6. `provider_account_id`
7. `union_id`
8. `open_id`
9. `app_id`
10. `nickname`
11. `avatar`
12. `email`
13. `mobile`
14. `raw_profile_json`
15. `access_token_expires_at`
16. `refresh_token_expires_at`
17. `last_sync_at`
18. `created_at`
19. `updated_at`

职责划分：

1. `user_credential`
   负责认证和唯一性。
2. `user_oauth_profiles`
   负责展示资料、平台返回扩展字段、token 元数据。

### 4.3 建议引入基于 cache 的认证流程状态存储

如果要做扫码登录、授权确认、绑定流程，不建议新增数据库表来承载这类短期状态。

更合适的方案是引入一套基于 cache 的认证流程状态存储，建议挂在 `xkitpkg/cache` 抽象之上，由 `admin` 按配置选择具体实现：

1. 单节点部署使用 memory cache。
2. 多节点部署使用 Redis。

这套状态存储承载的是“认证流程状态”，不是当前系统已有的 JWT 登录态本身。

建议承载的数据项包括：

1. `scene`
   - `login_qr`
   - `bind_qr`
   - `miniapp_auth`
2. `session_token`
3. `state`
4. `status`
   - `pending`
   - `scanned`
   - `confirmed`
   - `expired`
   - `canceled`
5. `tenant_id`
6. `user_id`
7. `provider`
8. `provider_account_id`
9. `client_type`
10. `redirect_uri`
11. `extra_info`
12. `expired_at`
13. `confirmed_at`

这套 cache 状态用于统一承载：

1. PC 扫码登录状态。
2. 绑定确认状态。
3. 授权页确认状态。
4. 小程序 code 交换过程中的临时上下文。

设计边界：

1. 它不替代 access token / refresh token。
2. 它只负责短期、可过期、可丢弃的流程态。
3. 是否持久审计，交由审计日志体系处理，而不是依赖 cache 本身。

补充边界：

1. 当前系统已经存在一套 `token store`，用于 access token / refresh token 的服务端撤销与校验。
2. 本规划新增的是 `auth flow store`，用于承载扫码登录、授权确认、首次绑定/注册选择、小程序 code 交换等短期流程态。
3. 两者都可以复用 `xkitpkg/cache.AdapterCache`，但 key 前缀、数据结构、生命周期、调用入口应严格分离，避免把“登录令牌状态”和“认证流程状态”混成一套记录。

## 5. Provider 标准化建议

建议先明确 provider 语义，避免后续混乱。这里区分两层：

1. `存储层 / 领域层`
   使用稳定字符串，如 `github`、`wechat_web`、`wechat_miniapp`。
2. `传输层 / proto 层`
   可以保留现有 `oauth.proto` 中的 `OAuthProvider` 枚举，但需要在 service 层维护“枚举 <-> 稳定字符串 provider key”的映射。

也就是说，数据库与业务主流程不要直接依赖 proto enum 的字面值。

建议先明确 provider 语义：

| provider | 场景 | 主标识建议 |
| --- | --- | --- |
| `github` | GitHub OAuth 网页登录 | GitHub user id |
| `wechat_web` | 微信开放平台网站扫码登录 | unionid 优先，缺失时 openid |
| `wechat_miniapp` | 微信小程序登录 | unionid 优先，缺失时 openid |
| `alipay_miniapp` | 支付宝小程序登录 | user_id |
| `douyin_open` | 抖音开放平台网页登录/小程序相关授权 | 平台用户唯一标识 |

约束建议：

1. 同租户下 `(provider, provider_account_id)` 唯一。
2. 如果平台提供 `unionid` 且业务允许跨 app 统一识别，则优先把 `unionid` 作为主标识。
3. 如果只有 `openid`，则以 `openid + provider + app_id` 维持唯一性，必要时把 `app_id` 存入扩展档案表。

## 6. 后端流程规划

### 6.1 用户名/邮箱/手机号注册

首期建议支持三种注册方式：

1. 用户名 + 密码
2. 邮箱 + 密码 + 验证码
3. 手机号 + 验证码 + 可选密码

标准流程：

1. 前端提交注册请求。
2. 服务端校验：
   - 租户
   - 用户名/邮箱/手机号唯一性
   - 验证码
   - 密码强度
3. 创建 `sys_users`
4. 创建 `sys_user_credentials`
5. 返回基础 token 或返回“待激活/待登录”状态

建议不要把注册逻辑硬塞进 `Login` 接口，应新增明确的注册接口族。

### 6.2 第三方首次登录

第三方首次登录有两种落地模式：

1. `首次登录自动注册`
   适合开放型产品。
2. `首次登录先补资料/确认绑定`
   适合后台型产品和租户隔离更强的场景。

建议 `admin` 默认采用第二种：

1. 第三方完成授权后，后端先识别外部身份。
2. 若本地无绑定：
   - 返回 `UNBOUND` 状态
   - 携带 provider profile 摘要
   - 前端进入“绑定已有账号 / 创建新账号”选择页
3. 用户确认后：
   - 绑定已有账号，或
   - 创建用户并绑定首个第三方凭据

### 6.3 已有账号绑定第三方

标准流程：

1. 用户已登录后台。
2. 在“账号安全/第三方账号”页点击绑定。
3. 前端跳转外部授权，或展示二维码。
4. 回调后后端校验：
   - 当前登录态
   - `state`
   - 第三方身份唯一性
5. 若第三方账号未被占用，则创建新 credential。
6. 若已绑定其他用户，则返回明确错误。

### 6.4 第三方登录后的令牌策略

建议：

1. XAdmin 自己仍然签发自己的 access/refresh token。
2. 外部平台 access token 不直接作为系统登录 token 使用。
3. 外部 token 仅作为：
   - 拉取用户资料
   - 后续平台接口调用
   - 刷新平台授权状态

这样平台切换与业务认证解耦，安全边界更清晰。

### 6.5 微信/支付宝/抖音/GitHub 分别建议

#### GitHub

适合作为第一批接入 provider：

1. 标准 OAuth 2.0 授权码流程清晰。
2. 文档稳定。
3. 前后端联调成本最低。

建议：

1. 先支持网页登录。
2. 通过 GitHub `code -> access_token -> user profile` 建立绑定。
3. 做成后续 provider 接入的模板实现。

#### 微信网站扫码登录

建议建模为 `wechat_web`：

1. 前端点击“微信登录”。
2. 跳转或弹出二维码授权页。
3. 用户扫码确认。
4. 后端根据 `code` 换取身份信息。
5. 按绑定/注册策略落地。

注意：

1. 网站微信登录与小程序登录不是一套账号交换接口。
2. `unionid`、`openid`、不同应用主体之间的关系要单独建模。

#### 微信小程序登录

建议建模为 `wechat_miniapp`：

1. 小程序前端调用平台登录能力拿 `code`。
2. 后端调用平台接口换取 `openid/session_key/unionid`。
3. 后端按 `unionid/openid` 查找或创建绑定。

需要单独设计：

1. `miniapp login`
2. `bind miniapp account`
3. `miniapp session exchange`

#### 支付宝小程序登录

建议建模为 `alipay_miniapp`：

1. 小程序端拿 `authCode`。
2. 后端换取支付宝用户标识与资料。
3. 再进入绑定/注册分支。

#### 抖音

建议先只做接口预留，不在第一阶段承诺完整上线。

原因：

1. 接入形态相对更复杂。
2. 公开资料与生态稳定性弱于 GitHub 和 GitHub 风格 OAuth。
3. 首期收益不如 GitHub/微信高。

但数据模型和 API 要预留 provider 能力，不要把 GitHub/微信写死。

## 7. 建议的后端 API 拆分

当前不建议把所有能力都塞进 `AuthenticationService.Login`。

建议分成四组服务，但要注意“首期可先在现有服务内增量演进，后续再视规模独立拆分 proto/service”。

### 7.1 注册服务

从长期看，注册能力可以独立为 `RegistrationService`；但结合当前已有 `AuthenticationService.RegisterUser`，首期更实际的做法是：

1. 先扩展 `AuthenticationService` 内的注册能力与请求模型。
2. 当邮箱/手机号/邀请码/激活确认等流程明显膨胀后，再拆出独立注册服务。

如果拆分，新增建议：

1. `RegisterByUsername`
2. `RegisterByEmail`
3. `RegisterByMobile`
4. `CompleteRegistration`

也可以统一成：

1. `StartRegistration`
2. `ConfirmRegistration`

但首期为了清晰，建议显式接口更好。

### 7.2 第三方登录服务

新增建议：

1. `StartSocialLogin`
2. `CompleteSocialLogin`
3. `ExchangeMiniAppCode`
4. `ConfirmBindOrRegister`

其中：

1. `StartSocialLogin`
   返回授权地址、二维码地址、state、session id。
2. `CompleteSocialLogin`
   处理 GitHub/微信网站应用回调。
3. `ExchangeMiniAppCode`
   处理微信/支付宝小程序 `code/authCode`。
4. `ConfirmBindOrRegister`
   在 `UNBOUND` 场景下完成“绑定已有账号”或“注册新账号”。

说明：

1. 这组服务建议作为“新增服务族”优先落地，因为它和现有 `AuthenticationService.Login/RefreshToken` 不是同一类请求。
2. `CompleteSocialLogin` 与 `ExchangeMiniAppCode` 完成的是“识别外部身份 + 返回绑定结果 / 未绑定状态”，不是直接等价于“本系统登录成功”。
3. 真正签发本系统 access/refresh token 的动作，应在绑定成功或识别到已绑定用户之后，由统一认证链完成。

### 7.3 第三方绑定管理服务

现有 `oauth.proto` 已有基础，可继续扩展：

1. `ListLinkedAccounts`
2. `StartLinkOAuth`
3. `ConfirmLinkOAuth`
4. `UnlinkOAuth`
5. `RefreshOAuthToken`

还可新增：

1. `SyncLinkedProfile`
2. `SetPrimaryLoginMethod`

说明：

1. `oauth.proto` 更适合承载“已登录用户对外部账号的管理与绑定”。
2. `第三方首次登录`、`扫码登录`、`小程序 code 交换` 这类“未建立本地用户会话前”的流程，不建议继续硬塞进 `oauth.proto`。
3. 因此建议保留 `oauth.proto` 作为“账号绑定管理”边界，新建一组更偏认证入口的 social auth proto/service。

### 7.4 认证流程状态服务

建议新增统一会话接口：

1. `CreateAuthSession`
2. `GetAuthSession`
3. `ConfirmAuthSession`
4. `CancelAuthSession`
5. `PollAuthSession`

这组接口将来既可服务扫码登录，也可服务绑定确认。

其底层状态建议放在 cache 中，而不是数据库表中。

命名建议：

1. 对外 API 可以继续使用 `AuthSession` 这类易理解名称。
2. 但实现与文档内部应统一理解为 `cache-based auth flow store`，避免被误解成数据库会话表，或与现有 token store 混淆。

## 8. 登录页与前端交互规划

### 8.1 登录主页面

基于 `AuthenticationLogin` 改造，而不是重写。

建议主页面包含四个入口：

1. 用户名/密码登录
2. 手机验证码登录
3. 扫码登录
4. 第三方登录

登录页布局建议：

1. 左侧继续保留系统品牌和说明。
2. 右侧保持 Vben 登录卡片结构。
3. 卡片底部增加“第三方登录”区。
4. 将“注册”入口恢复出来。

### 8.2 第三方登录区

建议第一阶段展示：

1. GitHub
2. 微信
3. 支付宝

其中：

1. GitHub
   直接跳转授权。
2. 微信
   网页端优先用扫码式。
3. 支付宝
   PC 端可先隐藏，仅在后续支持网页授权时打开。

### 8.3 扫码登录页

当前已有 `qrcode-login.vue` 路由，可直接承接。

建议交互：

1. 显示二维码
2. 轮询扫码状态
3. 展示状态变迁：
   - 待扫码
   - 已扫码待确认
   - 已确认登录
   - 已失效
4. 支持“刷新二维码”

### 8.4 第三方首次登录落地页

这是本次改造里最关键但当前缺失的页面。

建议新增一个“绑定或创建账号”页，场景是：

1. 第三方身份已认证成功。
2. 本地尚无绑定用户。

页面提供两个动作：

1. 绑定已有账号
2. 创建新账号

页面需展示：

1. 第三方头像
2. 第三方昵称
3. provider 名称
4. 风险提示

### 8.5 个人中心第三方绑定页

建议后续在“个人中心/安全设置”中新增：

1. 已绑定账号列表
2. 绑定按钮
3. 解绑按钮
4. 主登录方式标记
5. 最后同步时间

## 9. 小程序登录的专项预留

本次规划必须为未来小程序预留，而不是只做网页登录。

### 9.1 微信小程序

建议预留：

1. `client_type = miniapp`
2. provider = `wechat_miniapp`
3. 专用接口 `ExchangeMiniAppCode`
4. 小程序设备/会话信息扩展字段：
   - app id
   - device id
   - unionid/openid
   - scene

### 9.2 支付宝小程序

建议预留：

1. provider = `alipay_miniapp`
2. `authCode` 交换用户身份
3. 与网页 OAuth 登录完全分开的 handler

### 9.3 小程序授权与后端边界

建议严格遵循：

1. 小程序只负责拿平台 code。
2. 所有 token/session/openid 交换都放到后端。
3. 前端不直接持久化第三方 access token。

## 10. 安全与风控要求

### 10.1 OAuth 基础安全

必须具备：

1. `state` 校验
2. 回调地址白名单
3. PKCE
   对支持的 provider 优先启用
4. access token 不明文回传前端
   除非明确是外部平台前端直接使用场景

### 10.2 多租户语义

需要明确：

1. 同一个 GitHub/微信账号是否允许绑定多个租户下的不同用户。
2. 平台租户和业务租户的绑定隔离如何定义。

建议默认策略：

1. 第三方绑定唯一约束先控制在租户内。
2. 若未来需要“跨租户唯一账号”，再升级为全局身份映射表。

### 10.3 绑定冲突处理

必须明确返回以下错误语义：

1. 第三方账号已被其他用户绑定
2. 当前用户已绑定同类账号
3. 平台返回身份缺失
4. 授权已过期
5. 二维码会话已失效

### 10.4 审计

建议新增或扩展审计事件：

1. 注册成功
2. 第三方首次登录
3. 绑定第三方账号
4. 解绑第三方账号
5. 扫码确认登录
6. 小程序 code 交换失败

## 11. 数据库变更建议

建议数据库变更分两层。

### 11.1 第一层，最小变更可先做

1. 继续使用 `sys_user_credentials`
2. 规范 `provider` 值
3. 规范 `extra_info` 结构
4. 增补必要索引与注释

### 11.2 第二层，增强能力

新增：

1. `sys_user_oauth_profiles`

说明：

1. 认证流程短期状态不建议建表。
2. 这部分优先通过 cache 承载。
3. 只有当后续明确出现“需要长期追踪某类绑定操作实体”的业务需求时，才考虑补专门持久化表，而不在首期预设 `sys_user_bind_operations`。

## 12. 推荐的实施顺序

### 阶段一：收口现有模型

1. 明确 provider 枚举与命名规范
2. 梳理 `user_credential` 在登录/绑定/注册中的职责
3. 明确是否新增 `user_oauth_profiles`，以及认证流程状态在 cache 中的承载结构
4. 明确 `token store` 与 `auth flow store` 的边界、key 命名规范、TTL 策略
5. 明确 `oauth.proto` 与后续 social auth proto 的职责边界

### 阶段二：打通 GitHub 登录与绑定

1. 新增 GitHub provider 配置
2. 落地 `auth flow store`
3. 完成 GitHub 网页登录入口与回调
4. 完成首次登录落地页
5. 完成已登录用户绑定 GitHub

原因：

1. 文档清晰
2. 实现标准
3. 可作为后续 provider 模板

### 阶段三：恢复并改造登录页

1. 打开注册入口
2. 打开第三方登录区域
3. 打开二维码登录入口
4. 接入“未绑定时的补充流程页”
5. 将登录页交互与第二阶段的 GitHub/social auth 实际链路接通

### 阶段四：接入微信网页登录

1. 二维码授权
2. 回调处理
3. 绑定/注册分流

### 阶段五：接入微信/支付宝小程序

1. 新增专用 code 交换接口
2. 引入 miniapp client type
3. 小程序端与后台端联调

### 阶段六：接入抖音

1. 先做 provider 框架适配
2. 再根据最终业务场景决定做网页授权还是小程序授权

## 13. 可借鉴的后端代码/库

以下内容更适合“借鉴结构与模式”，不建议直接照搬。

### 13.1 Go OAuth 基础库

1. `golang.org/x/oauth2`
   - 适合做标准 OAuth 2.0 client
   - 可作为 GitHub provider 接入基础
2. `github.com/go-oauth2/oauth2/v4`
   - 适合参考 OAuth 服务端建模和 token 流程
   - 对本项目“外部平台登录后再签发本系统 token”的思路有参考价值

### 13.2 综合身份系统参考

1. Casdoor
   - 可借鉴 provider 管理、社交登录接入、绑定配置方式
   - 但其体量较大，不适合整套迁入

### 13.3 微信生态 Go SDK

1. PowerWeChat
   - 可借鉴微信/小程序接口封装方式
   - 更适合作为平台 SDK，而不是业务认证框架

## 14. 可借鉴的前端界面参考

### 14.1 当前最适合直接复用的参考

1. `admin-ui` 自身的 `AuthenticationLogin`
2. `admin-ui` 自身的 `AuthenticationQrCodeLogin`
3. `packages/effects/common-ui/src/ui/authentication/third-party-login.vue`

这三处应被视为首要参考，而不是另找一套视觉体系。

### 14.2 可借鉴的页面组合

建议按以下三屏组织：

1. 登录页
   - 账号密码
   - 验证码登录
   - 第三方登录
   - 扫码入口
2. 第三方未绑定落地页
   - 绑定已有账号
   - 创建新账号
3. 个人中心第三方绑定页
   - 绑定列表
   - 绑定/解绑

### 14.3 扫码和授权页视觉要点

建议：

1. 二维码区与状态区并列。
2. 状态变化明显：
   - 待扫码
   - 已扫码
   - 待确认
   - 已确认
3. 第三方头像与昵称用于提升“我正在绑定哪个账号”的可见性。

## 15. 需要优先确认的业务决策

后续正式开发前，建议先确认以下问题：

1. 第三方首次登录是否允许自动注册。
2. 微信是否只做小程序，还是同时做网页扫码登录。
3. 支付宝是否只做小程序。
4. 第三方绑定唯一性是“租户内唯一”还是“全局唯一”。
5. 是否允许一个用户绑定多个同 provider 账号。
6. 注册成功后是直接登录，还是进入待激活状态。

## 16. 推荐结论

本次规划的核心结论如下：

1. 后端以 `sys_user_credentials` 为核心继续演进，不另起一套用户认证主模型。
2. 第三方绑定关系放在 `user_credential`，第三方资料和 token 元数据建议拆到单独 profile 表。
3. 登录、注册、绑定三条流程统一收敛到“用户主体 + 多凭据”模型。
4. 认证流程短期状态采用 `cache-based auth flow store`，并与现有 token store 分层复用同一套 `xkitpkg/cache` 抽象。
5. `oauth.proto` 主要承载“已登录用户的第三方绑定管理”，而“第三方首次登录 / 扫码登录 / 小程序 code 交换”建议新建更偏认证入口的服务族。
6. 前端登录页基于现有 Vben 认证页面体系增量改造。
7. 技术实施顺序建议先 GitHub，再微信网页登录，再微信/支付宝小程序。

## 17. 参考资料

### 官方文档

1. GitHub OAuth App 授权流程  
   https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps
2. 微信网站应用微信登录  
   https://developers.weixin.qq.com/doc/oplatform/Website_App/WeChat_Login/Wechat_Login.html
3. 微信小程序 `code2Session`  
   https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-login/code2Session.html
4. 支付宝小程序用户授权 / `authCode` 说明  
   https://opendocs.alipay.com/mini/introduce/authcode
5. 抖音开放平台小程序登录/授权  
   https://developer.open-douyin.com/docs/resource/zh-CN/thirdparty/API/smallprogram/auth-app-manage/login/

### 开源参考

1. go-oauth2/oauth2  
   https://github.com/go-oauth2/oauth2
2. golang/oauth2  
   https://github.com/golang/oauth2
3. casdoor/casdoor  
   https://github.com/casdoor/casdoor
4. ArtisanCloud/PowerWeChat  
   https://github.com/ArtisanCloud/PowerWeChat
