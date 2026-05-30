# API Layer

本目录负责维护 XAdmin 的 Protobuf API 定义与代码生成配置，包含：

- 统一的 `.proto` 契约（gRPC + HTTP 注解）
- Go 服务端/客户端代码生成
- OpenAPI 文档生成
- 前端 TypeScript HTTP 客户端生成

## 目录结构

```text
api/
├── protos/                                  # Protobuf 源文件
│   ├── admin/v1
│   ├── audit/v1
│   ├── authentication/v1
│   ├── dict/v1
│   ├── identity/v1
│   ├── internal_message/v1
│   ├── permission/v1
│   ├── resource/v1
│   ├── storage/v1
│   └── task/v1
├── gen/                                     # 生成的 Go 代码
├── buf.yaml                                 # Buf 模块/lint/breaking 配置
├── buf.gen.yaml                             # Go 代码生成配置
├── buf.admin.openapi.gen.yaml               # OpenAPI 生成配置（admin）
└── buf.vue.admin.typescript.gen.yaml        # Vue TS 客户端生成配置
```

## Proto 概览（基于当前 `protos/`）

- 总计：`87` 个 `.proto` 文件
- 总计：`59` 个 `service`
- 总计：`307` 个 `message`
- 总计：`22` 个 `enum`

### 各模块统计

| 模块 | Proto 文件 | Service | RPC |
| --- | ---: | ---: | ---: |
| admin | 30 | 28 | 130 |
| audit | 9 | 5 | 15 |
| authentication | 8 | 5 | 47 |
| dict | 4 | 3 | 19 |
| identity | 12 | 5 | 33 |
| internal_message | 4 | 3 | 24 |
| permission | 11 | 4 | 26 |
| resource | 3 | 2 | 14 |
| storage | 4 | 3 | 13 |
| task | 2 | 1 | 11 |

## 主要模块说明

- `admin/v1`：管理后台聚合接口（包含多领域服务接口与管理侧错误定义）
- `audit/v1`：审计日志领域（API、数据访问、登录、操作、权限）
- `authentication/v1`：认证与登录策略、MFA、OAuth、凭据等
- `dict/v1`：字典类型、字典项、多语言
- `identity/v1`：租户、用户、组织、岗位及关联关系
- `internal_message/v1`：站内消息、消息分类、接收人
- `permission/v1`：角色、权限、策略、评估日志及关系模型
- `resource/v1`：API 与菜单资源
- `storage/v1`：文件与文件传输、对象存储能力
- `task/v1`：任务领域

## 代码生成

在 `api` 目录执行以下命令。

### 1) 生成 Go 代码（默认）

```bash
buf generate --template buf.gen.yaml
```

输出到：`api/gen/`

包括：
- protobuf Go 结构体
- gRPC 代码
- HTTP 映射代码（`protoc-gen-go-http`）
- 错误码相关代码（`protoc-gen-go-errors`）
- 校验代码（`protoc-gen-validate`）
- 脱敏代码（`protoc-gen-redact`）

### 2) 生成 OpenAPI（admin）

```bash
buf generate --template buf.admin.openapi.gen.yaml
```

输出到：`../cmd/server/assets`

### 3) 生成前端 TypeScript HTTP 客户端

Vue：

```bash
buf generate --template buf.vue.admin.typescript.gen.yaml
```

输出路径会在执行 `xkit init source` 时按目标前端工程自动改写。

## Buf 配置与依赖

### `buf.yaml`

- 使用 `v2` 配置
- 模块根目录：`protos`
- 开启 lint 与 breaking change 检查

### 远程依赖

- `buf.build/googleapis/googleapis`
- `buf.build/gnostic/gnostic`
- `buf.build/chnxq/x-crud`
- `buf.build/menta2k-org/redact`
- `buf.build/envoyproxy/protoc-gen-validate`

## 开发流程

### 新增 API

1. 在对应模块目录新增 `.proto`（例如 `protos/identity/v1/*.proto`）
2. 定义 `message`、`service`、`rpc` 与 HTTP 注解
3. 执行代码生成命令
4. 处理实现层编译与测试

### 修改 API

1. 修改已有 `.proto`
2. 优先保持向后兼容（避免破坏性变更）
3. 执行 lint 与 breaking check
4. 重新生成代码并联调

### 质量检查

```bash
buf lint
buf breaking --against .
```
