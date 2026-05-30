# XAdmin — Backend Service

> [中文版本](README.zh.md)

XAdmin backend is a Go microservice providing authentication, authorization, RBAC permission management, auditing, and dual-protocol (REST + gRPC) APIs. It is built on the xkit framework and uses Ent ORM for data access.

## Features

- **Authentication** — JWT/OIDC based login with password credential, refresh token flow, and CAPTCHA support
- **RBAC Authorization** — Role-based access control with permission points, menu resources, and API resource binding
- **Dual Protocol** — REST API (`:7788`) and gRPC (`:7790`) served from the same business logic layer
- **API Documentation** — Embedded Swagger UI served at `/docs/`
- **Audit Logging** — Login, API, data access, operation, and permission audit trails with configurable DB logging
- **Multi-Tenant** — Tenant management with membership, organization hierarchy, and position management
- **Async Tasks** — Redis-backed task queue via Asynq
- **Real-time Notifications** — Server-Sent Events (SSE) on `:7789`
- **Unified Observability** — OpenTelemetry distributed tracing, structured logging (Zap/Fluentd/Logrus), and gopsutil system metrics with OTLP export, providing end-to-end request visibility across service boundaries
- **Multi-Tenant** — Full tenant isolation with membership management, hierarchical organization units (company/division/department/team), position management with headcount tracking, and tenant-scoped data access via Ent interceptors
- **Multi-Database** — PostgreSQL, MySQL, and SQLite support via Ent ORM with auto-migration

## Observability

The service integrates three pillars of observability through the xkit framework:

**Logging** — Configurable in `configs/logger.yaml`. Supports multiple backends:
- **Zap** — High-performance structured logging (default, development mode)
- **Fluentd** — Log aggregation for cluster deployments
- **Logrus** — Compatible with legacy logging pipelines
- **Aliyun SLS / Tencent CLS** — Cloud-native log services

Log levels, output paths, rotation, and OTLP log export are fully configurable. Every HTTP/gRPC request is automatically logged with operation name, status code, latency, and arguments via the middleware chain.

**Tracing** — Configurable in `configs/trace.yaml`. Built on OpenTelemetry:
- **OTLP gRPC** — Default exporter to collector at `localhost:4317`
- **Jaeger** — Thrift-based exporter for Jaeger backends
- **Zipkin** — HTTP-based exporter
- **Stdout** — Development debugging

Sampling rate, batch size, and exporter endpoint are configurable. All incoming requests propagate trace context automatically.

**Metrics** — gopsutil-based system metrics collection:
- CPU, memory, and process-level resource usage
- Exported via OTLP alongside traces
- Integrated into the middleware chain for request-level measurement

**Service Discovery** — Configurable in `configs/registry.yaml`. Supports etcd and consul for multi-instance registration and health checking, enabling horizontal scaling.

## Prerequisites

- Go 1.26+
- Docker (for development dependencies)
- PostgreSQL / MySQL / SQLite

## Development Environment

The project includes a Docker Compose file (`../xkit/developer/docker-compose.yaml`) that starts all required infrastructure services:

```bash
# Start all development dependencies (from the admin directory)
docker compose -f ./docker-compose.yaml up -d
```

| Service | Port | Purpose |
|---------|------|---------|
| **PostgreSQL** | `5432` | Primary database (`admin` database, user `postgres`) |
| Redis | `6379` | Session cache, async task queue (Asynq), SSE |
| MinIO | `9000` / `9001` | S3-compatible object storage for file uploads (console at `:9001`) |
| OTel Collector | `4317` / `4318` | OpenTelemetry trace/metric/log ingestion (gRPC & HTTP) |
| etcd | `2379` / `2380` | Service discovery and remote configuration |
| etcd Browser | `9996` | Web UI for inspecting etcd keys |

**To switch to MySQL** instead of PostgreSQL, uncomment the `mysql` service in the compose file and set `data.database.driver: mysql` in `configs/data.yaml`.

After the services are healthy, update the database connection in `configs/data.yaml`:

```yaml
# configs/data.yaml
data:
  database:
    driver: postgres       # or mysql, sqlite3
    source: "host=localhost port=5432 user=postgres password=~Xyz4321Ab dbname=admin sslmode=disable"
    migrate: true           # auto-create tables on startup
  redis:
    addr: "localhost:6379"
    password: "~Xyz4321Ab"
```

## Quick Start

```bash
# 1. Start dependencies
docker compose -f ../xkit/developer/docker-compose.yaml up -d

# 2. Verify services are healthy, then run
go run ./cmd/server server -c ./configs

# 3. Build binary (optional)
go build -o admin.exe ./cmd/server
./admin.exe server -c ./configs
```

## Project Structure

```
admin/
├── cmd/server/              # Application entry point
│   ├── main.go              # CLI: parses "server" subcommand
│   ├── server.go            # Bootstrap wiring
│   └── assets/              # Embedded assets (OpenAPI spec, OPA policies)
├── api/
│   ├── protos/              # Protobuf definitions (10 domains)
│   │   ├── admin/           # Core admin services
│   │   ├── audit/           # Audit log services
│   │   ├── authentication/  # Auth services (login, credential, OAuth)
│   │   ├── dict/            # Dictionary & i18n services
│   │   ├── identity/        # User, org-unit, position, tenant, role
│   │   ├── internal_message/# In-app messaging
│   │   ├── permission/      # Permission & permission group
│   │   ├── resource/        # API resource & menu
│   │   ├── storage/         # File & OSS storage
│   │   └── task/            # Async task management
│   └── gen/                 # Generated Go code (pb, gRPC, HTTP, validate)
├── internal/
│   ├── bootstrap/           # DI composition root
│   │   ├── app.go           # Initialize(): wires logger → tracer → DB → repos → services → servers
│   │   ├── infra.go         # Logger, registrar, tracer factories
│   │   ├── hooks.go         # Lifecycle hooks & manual transport servers
│   │   └── generated_servers.gen.go  # Generated service/data wiring
│   ├── data/
│   │   ├── ent/             # Ent ORM: client, schema, migrations, generated CRUD
│   │   │   └── schema/      # 39 entity schema definitions
│   │   ├── repo/            # Repository layer (*.gen.go = generated, *_ext.go = hand-written)
│   │   └── bootstrap/       # Data initialization (ent client, seed data)
│   ├── service/             # Business logic layer (*.gen.go generated, *_ext.go extensions)
│   └── server/              # Transport layer
│       ├── http.go / grpc.go        # HTTP & gRPC server construction
│       ├── manual_http.go           # Auth, profile, portal, menu sync handlers
│       ├── viewer_auth.go           # JWT auth middleware
│       ├── sse.go / asynq.go        # SSE & async task servers
│       └── rest_register.gen.go     # Generated route registration
├── configs/                 # YAML configuration
│   ├── server.yaml          # gRPC/REST addresses, CORS, middleware, Asynq, SSE
│   ├── data.yaml            # Database (Postgres/MySQL/SQLite) & Redis
│   ├── auth.yaml            # AuthN (JWT/OIDC) & AuthZ (Casbin/OPA/Keto/OpenFGA)
│   ├── logger.yaml          # Logging (Zap, Fluentd, Logrus)
│   ├── trace.yaml           # OpenTelemetry tracing exporters
│   ├── registry.yaml        # Service discovery (etcd/consul)
│   ├── client.yaml          # gRPC client options
│   ├── oss.yaml             # Object storage (MinIO)
│   └── remote_config.yaml   # Remote config source
└── sql/                     # Seed data SQL scripts (Postgres & MySQL)
```

## Architecture

```
cmd/server/main.go
    │
    ▼
internal/bootstrap/app.go       (DI composition root)
    │
    ├── config.LoadServerConfig(configs/*.yaml)
    ├── NewLogger / NewRegistrar / NewTracer
    ├── NewEntClient             (ORM — Postgres / MySQL / SQLite)
    ├── NewGeneratedData         (all repositories)
    ├── NewGeneratedServices     (all services)
    ├── NewHTTPServer / NewGRPCServer / NewSSEServer / NewAsynqServer
    └── app.NewApp(...servers...) → app.Run()
```

The backend follows **Clean Architecture** with code generation:

- **Transport** (`internal/server/`) — HTTP REST + gRPC + SSE + Asynq protocols
- **Service** (`internal/service/`) — business logic, composition of repo operations
- **Repository** (`internal/data/repo/`) — data access layer, CRUD, filtering, sorting
- **Entity** (`internal/data/ent/`) — Ent ORM with 39 entity types

**Code generation ownership:**

| File Pattern | Source | Editable? |
|-------------|--------|-----------|
| `*_gen.go` | `xkit gen` tool | No — overwritten on regeneration |
| `*_ext.go` | Created once by xkit, filled by hand | Yes — safe to edit |
| `internal/server/manual_http.go` | Hand-written (Codex) | Yes |
| `internal/bootstrap/*_ext.go` | Hand-written (Codex) | Yes |
| Other `.go` | Check header comment | Depends on generator |

## Configuration

Key environment variables (override YAML configs):

| Variable | Purpose |
|----------|---------|
| `ADMIN_DB_DSN` | Database connection string |
| `ADMIN_REDIS_ADDR` | Redis address |
| `ADMIN_REDIS_PASSWORD` | Redis password |
| `ADMIN_JWT_KEY` | JWT signing secret |
| `ADMIN_OTEL_COLLECTOR` | OpenTelemetry collector endpoint |
| `ADMIN_ETCD_ENDPOINT` | etcd service discovery endpoint |

Set `data.database.migrate: true` in `configs/data.yaml` to auto-migrate the database schema on startup.

## Multi-Tenant Architecture

The service provides tenant-scoped data isolation through multiple layers:

- **Tenant Entity** — Each tenant has a name, code, domain, subscription plan, and audit status (pending/approved/rejected)
- **Organization Hierarchy** — OrgUnit supports company, division, department, team, project, committee, region, subsidiary, and branch types, organized in a tree structure with parent-child relationships
- **Position System** — Position management with job family, job grade, headcount tracking, and key position flags
- **Membership** — Users are linked to tenants, org-units, and positions via membership tables, supporting multiple assignments
- **Data Isolation** — Ent interceptors automatically scope queries by tenant ID, ensuring data separation without application-level boilerplate
- **Role Templates** — System roles, template roles, and tenant roles provide flexible permission delegation across the tenant hierarchy

## Frontend

The companion frontend is `admin-ui/`, a Vue.js single-page application built on **Vben Admin v5.7** (pnpm monorepo + Turborepo). It communicates with this backend via REST API and SSE:

| App | Port | Description |
|-----|------|-------------|
| `admin-ui` (dev server) | `:5666` | Vite dev server, proxies `/api` → `:7788` |
| `admin` (REST) | `:7788` | Backend HTTP API |
| `admin` (gRPC) | `:7790` | Backend gRPC API |
| `admin` (SSE) | `:7789` | Server-Sent Events stream |

The frontend features API-driven routing (menus fetched from backend), OAuth2 password + refresh token flow, dynamic watermarking, and real-time notification via SSE. See `admin-ui/README.md` for frontend setup and development instructions.

## API Endpoints

### Authentication
| Method | Path | Description |
|--------|------|-------------|
| POST | `/admin/v1/login` | Login with username/password + CAPTCHA |
| POST | `/admin/v1/logout` | Logout |
| POST | `/admin/v1/refresh-token` | Refresh access token |
| GET  | `/admin/v1/captcha` | Get CAPTCHA |

### User Profile (self-service)
| Method | Path | Description |
|--------|------|-------------|
| GET  | `/admin/v1/me` | Get current user profile |
| PUT  | `/admin/v1/me` | Update profile fields |
| POST | `/admin/v1/me/password` | Change password |
| POST | `/admin/v1/me/avatar` | Upload avatar |

### Portal
| Method | Path | Description |
|--------|------|-------------|
| GET  | `/admin/v1/routes` | Get navigation menus |
| GET  | `/admin/v1/perm-codes` | Get user permission codes |
| GET  | `/admin/v1/initial-context` | Get menus + permissions in one call |
| GET  | `/admin/v1/dashboard/analytics` | Analytics dashboard data |

### Resource CRUD
All domain resources (users, roles, menus, APIs, tenants, org-units, positions, permissions, dicts, files, tasks, messages, audit logs) are accessible via REST and gRPC with standard List/Get/Create/Update/Delete/Count/Exists operations.

Full API docs available at `http://localhost:7788/docs/` after starting the service.

## Code Generation

```bash
# Generate Ent ORM code from schemas
go run -mod=mod entgo.io/ent/cmd/ent generate ./internal/data/ent/schema

# Generate proto code
buf generate api/protos

# xkit resource generation (service + repo + register + bootstrap)
cd admin
go run github.com/chnxq/xkit/cmd/xkit gen all <resource-name>
```

## Verification

```bash
# REST API
curl http://localhost:7788/admin/v1/captcha | jq

# gRPC
grpcurl -plaintext localhost:7790 list

# Swagger UI
open http://localhost:7788/docs/

# Run tests
go test ./...
```

## Related Projects

| Project | Purpose |
|---------|---------|
| `admin-ui/` | Vue.js frontend (Vben Admin v5.7) |
| `xkit/` | Code generation CLI tool |
| `xkit-template/` | Project skeleton template |
| `x-crud/` | Multi-backend CRUD library |
| `xkitpkg/` | Application framework |
| `xkitmod/` | Low-level modules |
| `x-utils/` | General utilities |
