# XAdmin — 后端服务

> [English Version](README.md)

XAdmin 后端是一个 Go 微服务，提供认证、授权、RBAC 权限管理、审计以及双协议（REST + gRPC）API。基于 xkit 框架构建，使用 Ent ORM 进行数据访问。

## 主要功能

- **认证鉴权** — 基于 JWT/OIDC 的登录，支持密码凭证、refresh token 续期、验证码
- **RBAC 权限** — 基于角色的访问控制，支持权限点、菜单资源、API 资源三级绑定
- **双协议支持** — 同一业务逻辑层对外提供 REST API（`:7788`）和 gRPC（`:7790`）
- **API 文档** — 内嵌 Swagger UI，通过 `/docs/` 访问
- **审计日志** — 登录审计、API 审计、数据访问审计、操作审计、权限审计，可按需开启数据库日志中间件
- **多租户** — 租户管理、成员关系、组织架构、岗位管理
- **异步任务** — 基于 Redis 的 Asynq 任务队列
- **实时消息** — Server-Sent Events（SSE），监听端口 `:7789`
- **统一可观测性** — OpenTelemetry 分布式链路追踪、结构化日志（Zap/Fluentd/Logrus）、gopsutil 系统指标采集，支持 OTLP 导出，实现跨服务边界的端到端请求可观测
- **多租户** — 完整的租户数据隔离，支持成员关系管理、层级化组织架构（公司/事业部/部门/团队/项目组等）、带编制管理的岗位体系、Ent 拦截器自动租户范围过滤
- **多数据库** — 通过 Ent ORM 支持 PostgreSQL、MySQL、SQLite，启动时可自动迁移

## 可观测性

服务通过 xkit 框架集成了三大可观测性支柱：

**日志** — 通过 `configs/logger.yaml` 配置，支持多种后端：
- **Zap** — 高性能结构化日志（默认，开发模式）
- **Fluentd** — 适用于集群环境的日志聚合
- **Logrus** — 兼容传统日志流水线
- **阿里云 SLS / 腾讯云 CLS** — 云原生日志服务

日志级别、输出路径、轮转策略、OTLP 日志导出均可配置。所有 HTTP/gRPC 请求通过中间件链自动记录操作名称、状态码、延迟和参数。

**链路追踪** — 通过 `configs/trace.yaml` 配置，基于 OpenTelemetry：
- **OTLP gRPC** — 默认导出器，目标 `localhost:4317`
- **Jaeger** — 基于 Thrift 的 Jaeger 后端导出器
- **Zipkin** — 基于 HTTP 的导出器
- **Stdout** — 开发调试

采样率、批量大小和导出端点均可配置。所有入站请求自动传播追踪上下文。

**指标测量** — 基于 gopsutil 的系统指标采集：
- CPU、内存和进程级资源使用
- 通过 OTLP 与追踪数据一并导出
- 集成到中间件链，支持请求级测量

**服务发现** — 通过 `configs/registry.yaml` 配置。支持 etcd 和 consul 进行多实例注册与健康检查，支持水平扩展。

## 环境要求

- Go 1.26+
- Docker（开发环境依赖）
- PostgreSQL / MySQL / SQLite

## 开发环境搭建

项目包含 Docker Compose 文件（`../xkit/developer/docker-compose.yaml`），可一键启动所有基础设施服务：

```bash
# 从 admin 目录执行，启动所有开发依赖
docker compose -f ./docker-compose.yaml up -d
```

| 服务 | 端口 | 用途 |
|------|------|------|
| **PostgreSQL** | `5432` | 主数据库（数据库名 `admin`，用户 `postgres`） |
| Redis | `6379` | 会话缓存、异步任务队列（Asynq）、SSE |
| MinIO | `9000` / `9001` | S3 兼容对象存储，用于文件上传（管理控制台 `:9001`） |
| OTel Collector | `4317` / `4318` | OpenTelemetry 追踪/指标/日志采集（gRPC & HTTP） |
| etcd | `2379` / `2380` | 服务发现与远程配置 |
| etcd Browser | `9996` | etcd 键值 Web 管理界面 |

**如需使用 MySQL** 替代 PostgreSQL，请取消 compose 文件中 `mysql` 服务的注释，并将 `configs/data.yaml` 中的 `data.database.driver` 设为 `mysql`。

服务启动并健康运行后，编辑 `configs/data.yaml` 中的数据库连接信息：

```yaml
# configs/data.yaml
data:
  database:
    driver: postgres       # 可选 mysql、sqlite3
    source: "host=localhost port=5432 user=postgres password=~Xyz4321Ab dbname=admin sslmode=disable"
    migrate: true           # 启动时自动建表
  redis:
    addr: "localhost:6379"
    password: "~Xyz4321Ab"
```

## 快速入门

```bash
# 1. 启动依赖服务
docker compose -f ../xkit/developer/docker-compose.yaml up -d

# 2. 确认服务运行正常后，启动后端
go run ./cmd/server server -c ./configs

# 3. 构建二进制（可选）
go build -o admin.exe ./cmd/server
./admin.exe server -c ./configs
```

## 项目结构

```
admin/
├── cmd/server/              # 应用入口
│   ├── main.go              # 命令行入口：解析 "server" 子命令
│   ├── server.go            # 启动装配
│   └── assets/              # 内嵌资源（OpenAPI 文档、OPA 策略）
├── api/
│   ├── protos/              # Protobuf 定义（10 个业务域）
│   │   ├── admin/           # 核心管理服务
│   │   ├── audit/           # 审计日志服务
│   │   ├── authentication/  # 认证服务（登录、凭证、OAuth）
│   │   ├── dict/            # 字典与国际化
│   │   ├── identity/        # 用户、组织、岗位、租户、角色
│   │   ├── internal_message/# 站内消息
│   │   ├── permission/      # 权限点与权限组
│   │   ├── resource/        # API 资源与菜单
│   │   ├── storage/         # 文件与对象存储
│   │   └── task/            # 异步任务管理
│   └── gen/                 # 生成代码（pb、gRPC、HTTP、validate）
├── internal/
│   ├── bootstrap/           # 依赖注入装配层
│   │   ├── app.go           # Initialize()：装配 logger → tracer → DB → repos → services → servers
│   │   ├── infra.go         # 日志、注册中心、链路追踪工厂
│   │   ├── hooks.go         # 生命周期钩子与手动传输服务
│   │   └── generated_servers.gen.go  # 生成的 service/data 装配代码
│   ├── data/
│   │   ├── ent/             # Ent ORM：客户端、schema、迁移、生成 CRUD
│   │   │   └── schema/      # 39 个实体 schema 定义
│   │   ├── repo/            # 仓储层（*.gen.go 生成，*_ext.go 手写）
│   │   └── bootstrap/       # 数据初始化（ent 客户端、种子数据）
│   ├── service/             # 业务逻辑层（*.gen.go 生成，*_ext.go 扩展）
│   └── server/              # 传输层
│       ├── http.go / grpc.go        # HTTP & gRPC 服务构造
│       ├── manual_http.go           # 认证、个人中心、Portal、菜单同步处理器
│       ├── viewer_auth.go           # JWT 认证中间件
│       ├── sse.go / asynq.go        # SSE & 异步任务服务
│       └── rest_register.gen.go     # 生成的路由注册
├── configs/                 # YAML 配置文件
│   ├── server.yaml          # gRPC/REST 地址、CORS、中间件、Asynq、SSE
│   ├── data.yaml            # 数据库（Postgres/MySQL/SQLite）与 Redis
│   ├── auth.yaml            # 认证（JWT/OIDC）与授权（Casbin/OPA/Keto/OpenFGA）
│   ├── logger.yaml          # 日志（Zap、Fluentd、Logrus）
│   ├── trace.yaml           # OpenTelemetry 链路追踪导出器
│   ├── registry.yaml        # 服务发现（etcd/consul）
│   ├── client.yaml          # gRPC 客户端配置
│   ├── oss.yaml             # 对象存储（MinIO）
│   └── remote_config.yaml   # 远程配置源
└── sql/                     # 种子数据 SQL 脚本（Postgres & MySQL）
```

## 架构

```
cmd/server/main.go
    │
    ▼
internal/bootstrap/app.go       （DI 装配根）
    │
    ├── config.LoadServerConfig(configs/*.yaml)
    ├── NewLogger / NewRegistrar / NewTracer
    ├── NewEntClient             （ORM — Postgres / MySQL / SQLite）
    ├── NewGeneratedData         （全部 repository）
    ├── NewGeneratedServices     （全部 service）
    ├── NewHTTPServer / NewGRPCServer / NewSSEServer / NewAsynqServer
    └── app.NewApp(...servers...) → app.Run()
```

后端采用 **Clean Architecture**，结合代码生成：

- **传输层**（`internal/server/`） — HTTP REST + gRPC + SSE + Asynq 四协议
- **服务层**（`internal/service/`） — 业务逻辑，组合 repo 操作
- **仓储层**（`internal/data/repo/`） — 数据访问，CRUD、过滤、排序
- **实体层**（`internal/data/ent/`） — Ent ORM，共 39 个实体类型

**代码生成归属规则：**

| 文件模式 | 来源 | 是否可编辑 |
|---------|------|-----------|
| `*_gen.go` | `xkit gen` 工具生成 | 否 — 重新生成时会被覆盖 |
| `*_ext.go` | xkit 首次创建，后续手写 | 是 — 不会被覆盖 |
| `internal/server/manual_http.go` | 手写（Codex） | 是 |
| `internal/bootstrap/*_ext.go` | 手写（Codex） | 是 |
| 其他 `.go` 文件 | 检查文件头注释 | 取决于生成器 |

## 配置说明

支持通过环境变量覆盖 YAML 配置：

| 变量 | 用途 |
|------|------|
| `ADMIN_DB_DSN` | 数据库连接串 |
| `ADMIN_REDIS_ADDR` | Redis 地址 |
| `ADMIN_REDIS_PASSWORD` | Redis 密码 |
| `ADMIN_JWT_KEY` | JWT 签名密钥 |
| `ADMIN_OTEL_COLLECTOR` | OpenTelemetry 收集器地址 |
| `ADMIN_ETCD_ENDPOINT` | etcd 服务发现地址 |

将 `configs/data.yaml` 中的 `data.database.migrate` 设为 `true` 可在启动时自动迁移数据库结构。

## 多租户架构

服务通过多个层面实现租户级数据隔离：

- **租户实体** — 每个租户拥有名称、编码、域名、订阅计划和审核状态（待审核/通过/拒绝）
- **组织架构** — OrgUnit 支持公司、事业部、部门、团队、项目组、委员会、区域、子公司、分支机构等类型，以树形结构组织父子关系
- **岗位体系** — Position 管理支持岗位族、岗位等级、编制跟踪和关键岗位标识
- **成员关系** — 用户通过 membership 表关联到租户、组织、岗位，支持多重归属
- **数据隔离** — Ent 拦截器自动按租户 ID 过滤查询，无需应用层额外代码即可确保数据分离
- **角色模板** — 系统角色、模板角色、租户角色提供灵活的权限委派，贯穿租户层级

## 前端

配合的前端仓库为 `admin-ui/`，是基于 **Vben Admin v5.7** 构建的 Vue.js 单页应用（pnpm monorepo + Turborepo）。通过 REST API 和 SSE 与后端通信：

| 应用 | 端口 | 说明 |
|------|------|------|
| `admin-ui`（开发服务器） | `:5666` | Vite 开发服务器，代理 `/api` → `:7788` |
| `admin`（REST） | `:7788` | 后端 HTTP API |
| `admin`（gRPC） | `:7790` | 后端 gRPC API |
| `admin`（SSE） | `:7789` | Server-Sent Events 推送流 |

前端支持 API 驱动的动态路由（从后端获取菜单）、OAuth2 password + refresh token 认证流程、动态水印和 SSE 实时通知。前端搭建和开发说明请参见 `admin-ui/README.md`。

## API 端点

### 认证
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/admin/v1/login` | 用户名/密码 + 验证码登录 |
| POST | `/admin/v1/logout` | 登出 |
| POST | `/admin/v1/refresh-token` | 刷新令牌 |
| GET  | `/admin/v1/captcha` | 获取验证码 |

### 个人中心
| 方法 | 路径 | 说明 |
|------|------|------|
| GET  | `/admin/v1/me` | 获取当前用户信息 |
| PUT  | `/admin/v1/me` | 更新个人信息 |
| POST | `/admin/v1/me/password` | 修改密码 |
| POST | `/admin/v1/me/avatar` | 上传头像 |

### Portal
| 方法 | 路径 | 说明 |
|------|------|------|
| GET  | `/admin/v1/routes` | 获取导航菜单 |
| GET  | `/admin/v1/perm-codes` | 获取用户权限码 |
| GET  | `/admin/v1/initial-context` | 一次获取菜单 + 权限 |
| GET  | `/admin/v1/dashboard/analytics` | 分析仪表盘数据 |

### 资源 CRUD
所有领域资源（用户、角色、菜单、API、租户、组织、岗位、权限、字典、文件、任务、消息、审计日志）均可通过 REST 和 gRPC 进行标准 List/Get/Create/Update/Delete/Count/Exists 操作。

完整 API 文档请启动服务后访问 `http://localhost:7788/docs/`。

## 代码生成

```bash
# 从 schema 生成 Ent ORM 代码
go run -mod=mod entgo.io/ent/cmd/ent generate ./internal/data/ent/schema

# 生成 proto 代码
buf generate api/protos

# xkit 资源生成（service + repo + register + bootstrap）
cd admin
go run github.com/chnxq/xkit/cmd/xkit gen all <resource-name>
```

## 验证

```bash
# REST API
curl http://localhost:7788/admin/v1/captcha | jq

# gRPC
grpcurl -plaintext localhost:7790 list

# Swagger UI
open http://localhost:7788/docs/

# 运行测试
go test ./...
```

## 相关项目

| 项目 | 说明 |
|------|------|
| `admin-ui/` | Vue.js 前端（Vben Admin v5.7） |
| `xkit/` | 代码生成 CLI 工具 |
| `xkit-template/` | 项目骨架模板 |
| `x-crud/` | 多后端 CRUD 抽象库 |
| `xkitpkg/` | 应用框架 |
| `xkitmod/` | 底层模块 |
| `x-utils/` | 通用工具库 |
