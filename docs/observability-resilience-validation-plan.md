# Admin 项目链路追踪、熔断、限流现状与验证计划

## 1. 目标

本文档用于回答两个问题：

1. 当前 `admin` 项目对链路追踪、熔断、限流是否已经真实接入运行链路。
2. 如何设计一轮可执行的验证/测试计划，确认这些能力的实际有效性与性能影响。

本文基于 `2026-06-12` 的源码与本轮真实联调结果整理，不按 README 宣传口径下结论。

## 2. 当前现状结论

### 2.1 链路追踪

结论：`HTTP/gRPC tracing` 已接入且默认开启，`DB tracing` 也已接入实际运行链路，并已通过 `otelsql` span 观测确认有效；但“审计日志与 trace_id 的自动关联”仍需单独验证与补强。

证据：

- Tracer 初始化：
  - [internal/bootstrap/infra.go](../internal/bootstrap/infra.go)
  - `NewTracer(appCtx)` 在应用启动时执行，调用 `tracer.NewTracerProviderWithShutdown(...)`
- Server middleware 挂载：
  - [internal/server/transport_config.go](../internal/server/transport_config.go)
  - `enable_tracing=true` 时挂载 `tracing.Server()`
- HTTP/gRPC server 实际使用该 middleware：
  - [internal/server/http_options.go](../internal/server/http_options.go)
  - [internal/server/grpc_options.go](../internal/server/grpc_options.go)
  - [internal/server/http.go](../internal/server/http.go)
  - [internal/server/grpc.go](../internal/server/grpc.go)
- 默认配置开启 tracing：
  - [configs/server.yaml](../configs/server.yaml)
  - [configs/client.yaml](../configs/client.yaml)
  - [configs/trace.yaml](../configs/trace.yaml)
- 数据库配置声明了 `enable_trace: true`：
  - [configs/data.yaml](../configs/data.yaml)
- `admin` 当前实际使用的 ent 初始化链路已经走 `CreateDriver(...)`，会按配置接通 tracing / metrics：
  - [internal/data/bootstrap/ent_client.gen.go](../internal/data/bootstrap/ent_client.gen.go)
- 本轮已在 OTel 侧观察到 `otelsql` 相关 span，说明 DB trace 已经进入真实运行链路。

判断：

- “请求级 tracing middleware 生效”成立。
- “数据库 tracing 已接通”成立。
- “审计日志 trace_id 已稳定落库并可用于链路关联”当前不能视为已验证成立。

### 2.2 熔断

结论：Server 侧熔断 middleware 已接入 HTTP/gRPC 入口，默认开启，并且本轮已经通过真实故障注入验证“确实会打开”；但其外部观测效果会受到下游阻塞时长、客户端超时与 limiter 介入时机的影响。

证据：

- 熔断 middleware 定义：
  - [internal/server/circuitbreaker.go](../internal/server/circuitbreaker.go)
- 挂载点：
  - [internal/server/transport_config.go](../internal/server/transport_config.go)
  - `enable_circuit_breaker=true` 时挂载 `serverCircuitBreakerMiddleware(cfg.GetCircuitBreaker())`
- HTTP/gRPC server 实际使用：
  - [internal/server/http_options.go](../internal/server/http_options.go)
  - [internal/server/grpc_options.go](../internal/server/grpc_options.go)
- 默认配置开启：
  - [configs/server.yaml](../configs/server.yaml)
  - [configs/client.yaml](../configs/client.yaml)

代码行为：

- breaker key 为 `transportKind + operation`
  - 同一路由/同一 RPC 独立统计，不会全局串扰。
- 被视为失败的错误类型：
  - `INTERNAL_SERVER_ERROR`
  - `SERVICE_UNAVAILABLE`
  - `GATEWAY_TIMEOUT`
  - `context.Canceled`
  - `context.DeadlineExceeded`
- `BAD_REQUEST`、`UNAUTHORIZED` 这类错误不会推动熔断器打开。

判断：

- “代码已挂上”成立。
- “在真实故障场景中 breaker 会打开”成立。
- “故障窗口内是否一定立刻对外表现为稳定 503”不成立，这取决于请求是否先被下游阻塞或被 limiter 抢先拒绝。

### 2.3 限流

结论：Server 侧限流 middleware 已有实际接入链路，且支持从配置中读取 `window`、`bucket`、`cpu_threshold`、`cpu_quota` 等参数；但 `admin` 当前默认运行配置并未启用 limiter，且当前使用的是 BBR 自适应策略，不适合按“固定阈值、固定并发”方式做直观验证。

证据：

- 限流 middleware 挂载条件：
  - [internal/server/transport_config.go](../internal/server/transport_config.go)
  - 只有 `cfg.GetLimiter() != nil` 才挂载 `ratelimit.Server()`
- 限流构造已支持读取配置：
  - [internal/server/transport_config.go](../internal/server/transport_config.go)
  - `newServerRateLimiter(cfg.GetLimiter())`
- 当前支持的参数来自配置 proto：
  - [../../xkitpkg/conf/protos/v1/middleware.proto](../../xkitpkg/conf/protos/v1/middleware.proto)
  - [../../xkitpkg/conf/v1/middleware.pb.go](../../xkitpkg/conf/v1/middleware.pb.go)
- 当前默认 `admin/configs/server.yaml` 中没有 `middleware.limiter` 配置块，因此默认运行态不会启用 limiter：
  - [configs/server.yaml](../configs/server.yaml)

判断：

- “具备基础限流能力代码”成立。
- “可按配置启用并调节核心参数”成立。
- “当前 admin 默认运行已开启限流”不成立。
- “BBR limiter 可用固定轻压测稳定打出阈值型 429”当前不成立。

## 3. 实际有效性评估

按当前源码与本轮验证结果，三项能力分层判断如下：

| 能力 | 代码支持 | 默认配置 | 当前运行链路已挂载 | 实际有效性判断 |
|---|---|---|---|---|
| HTTP/gRPC tracing | 是 | 是 | 是 | 已验证有效 |
| DB tracing | 是 | 是 | 是 | 已验证有效 |
| 审计日志 trace_id 落库 | 表结构支持 | 无自动保证 | 不明确 | 仍需专项验证 |
| Server 熔断 | 是 | 是 | 是 | 已验证 breaker 可真实打开 |
| Client 熔断 | 配置有 | 是 | `admin` 中未见专项验证 | 不能视为已验证 |
| Server 限流 | 是 | 否 | 默认未挂 | 代码与挂载链有效，默认运行态未启用 |
| Client 限流 | 代码层未见实际使用链 | 否 | 否 | 当前无效 |

## 4. 本轮已完成验证

### 4.1 Tracing 验证结果

已确认：

- `HTTP/gRPC` 请求 span 可以正常产生。
- `DB tracing` 已进入实际运行链路。
- OTel 侧已观察到 `otelsql` span。

当前未完成：

- 审计日志中的 `trace_id` / `span_id` 是否与本次请求 trace 自动关联。
- Redis / 其他外部依赖是否也具备同等级别的 tracing 观测。

### 4.2 Breaker 验证结果

已采用“临时降低 breaker 门槛 + 暂时放开 `GET /admin/v1/tenants` + pause/unpause postgres”的方式做了两轮验证。

联合验证结果：

- 当 breaker 与 limiter 同时启用时：
  - pause postgres 后，前几次请求先表现为超时；
  - 随后开始出现连续 `429`；
  - 说明 breaker 已感知失败，但 limiter 比 breaker 更早接管了外部拒绝流量。

纯 breaker 验证结果：

- 临时关闭 limiter，仅保留低门槛 breaker 后：
  - 故障窗口内外部请求仍以客户端超时为主；
  - 数据库恢复后的第一下请求直接返回 `503`；
  - 说明 breaker 已经真实打开，只是故障窗口内请求先被下游阻塞和客户端超时吃掉，没有及时走到对外“快速拒绝”的阶段。

结论：

- server breaker 已验证真实生效。
- 本轮补的两处逻辑已验证有效：
  - 失败判定补进 `context.Canceled` / `context.DeadlineExceeded`
  - breaker 参数可配置化

### 4.3 Limiter 验证结果

已确认：

- limiter 代码链可启用。
- 在与 breaker 联合的故障场景中，BBR limiter 会介入并拒绝后续请求。

未确认：

- 用“轻接口 + 固定并发”方式稳定打出 `429` 并不能证明 limiter 无效，因为 BBR 属于自适应策略，不是固定配额限流。

结论：

- limiter 的“运行链路有效”成立。
- limiter 的“易于直观压测展示固定阈值效果”不成立。

## 5. 验证环境建议

### 5.1 基础环境

- `admin` 本地启动
- PostgreSQL / Redis 正常可用
- OTel Collector 正常启动
  - 可直接使用 [docker-compose.yaml](../docker-compose.yaml) 中的 `otel-collector`
- 至少准备一个可稳定调用的轻量接口作为基线接口
  - 推荐：
    - `GET /admin/v1/portal:captcha`
    - `POST /admin/v1/login`
    - `GET /admin/v1/me`

### 5.2 压测工具

建议任选一种：

- `k6`
- `wrk`
- `vegeta`

推荐 `k6`，因为更容易组织分阶段场景、错误注入和指标输出。

## 6. 后续验证计划

### 6.1 阶段一：审计日志 trace 关联验证

目标：确认请求 trace 是否能稳定落到 `sys_api_audit_logs` / `sys_login_audit_logs` 的 `trace_id`、`span_id` 字段。

步骤：

1. 发起一条公开接口请求和一条鉴权接口请求。
2. 在 OTel 后端确认对应 trace 已生成。
3. 查询最近产生的审计日志记录。
4. 对比：
   - `trace_id` 是否非空
   - `trace_id` 是否与 OTel 中同一请求一致
   - `span_id` 是否可用于回溯当前入口 span 或关键子 span

通过标准：

- 审计日志中 `trace_id`、`span_id` 非空。
- 至少能通过 `trace_id` 唯一关联到对应请求。

### 6.2 阶段二：Limiter 专项验证

目标：在不与 breaker 干扰的前提下，确认 limiter 的启用、拒绝行为与参数敏感性。

建议方式：

1. 临时在 `server.yaml` 中显式开启 `middleware.limiter`。
2. 先关闭 breaker，只保留 limiter。
3. 选择一个轻接口和一个中等 DB 接口分别测试。
4. 记录：
   - 是否出现 `429`
   - 请求延迟分布
   - CPU 变化
   - 不同 `window` / `bucket` / `cpu_threshold` 下的行为差异

说明：

- 当前 limiter 为 BBR，自适应特征较强。
- 更适合验证“过载保护是否介入”，不适合验证“每秒固定限制 N 次”。

### 6.3 阶段三：性能影响验证

目标：量化 tracing / breaker / limiter 对核心接口性能的影响。

建议做四组对照：

1. 基线：三者都关闭
2. 只开 tracing
3. tracing + circuit breaker
4. tracing + circuit breaker + limiter

测试接口建议：

- 公共轻接口：验证码
- 鉴权接口：登录
- DB 查询接口：用户列表 / 角色列表

核心指标：

- 吞吐量（RPS / QPS）
- 平均延迟
- P95 / P99 延迟
- 错误率
- CPU / 内存
- OTel 导出延迟或堆积情况

建议负载：

- 低负载：20 VUs / 2 min
- 中负载：100 VUs / 5 min
- 突发负载：200~500 VUs / 1 min

## 7. 推荐验证顺序

建议按以下顺序执行：

1. 审计日志 trace_id / span_id 落库验证
2. limiter 专项验证
3. 做一轮性能对照测试

这样可以先补齐“观测可追溯性”，再评估保护策略行为和性能影响。

## 8. 当前已知问题清单

基于当前源码与本轮验证，当前仍需关注这些问题：

1. 审计日志的 `trace_id` / `span_id` 关联状态尚未确认
2. limiter 默认未开启，生产/联调环境是否启用取决于具体配置
3. BBR limiter 的观测方式需要按其自适应特性设计，不能用固定阈值思维判断有效性
4. client 侧 tracing / breaker / limiter 在 `admin` 中尚未做专项实证

## 9. 后续改进建议

### 9.1 高优先级

1. 在审计日志落库链路中显式补齐 `trace_id` / `span_id` 传递与写入验证
2. 为 limiter 增加一组专门的压测脚本和标准化验证步骤
3. 为 breaker 增加状态观测出口或诊断日志，降低排障成本

### 9.2 中优先级

1. 为 tracing / breaker / limiter 增加专门的集成测试或压测脚本
2. 对关键接口建立固定压测基线
3. 视后续项目需要，再评估 client 侧 resilience 能力的实装与验证

## 10. 建议产出物

完成后续计划后，建议形成以下落地物：

1. 一份压测脚本目录，例如 `admin/test/perf/`
2. 一份测试记录文档，保存每轮测试参数与结果
3. 一份问题清单，区分：
   - 已有效
   - 有代码但未接通
   - 已接通但缺少验证
   - 需要重构

## 11. 本轮结论摘要

当前 `admin` 的真实状态可以概括为：

- `Tracing`：`HTTP/gRPC` 与 `DB tracing` 均已接入并已验证有效；审计日志 trace 关联仍需单独验证。
- `Circuit Breaker`：server breaker 已验证真实生效；但其外部表现会受到下游阻塞、客户端超时与 limiter 介入顺序影响。
- `Rate Limit`：代码与运行链路有效，且已支持核心配置项；但默认运行态未开启，且 BBR 不适合按固定阈值思路做验证。

## 附录 A：本轮实施与代码变更

本轮已完成的代码侧工作：

- `xkitpkg/conf/protos/v1/middleware.proto` 补充了：
  - `Middleware.RateLimiter.window`
  - `Middleware.RateLimiter.bucket`
  - `Middleware.RateLimiter.cpu_threshold`
  - `Middleware.RateLimiter.cpu_quota`
  - `Middleware.CircuitBreaker.window`
  - `Middleware.CircuitBreaker.request`
  - `Middleware.CircuitBreaker.bucket`
  - `Middleware.CircuitBreaker.success`
- 本地生成了：
  - [../../xkitpkg/conf/v1/middleware.pb.go](../../xkitpkg/conf/v1/middleware.pb.go)
- `admin` 已接入新配置：
  - [internal/server/circuitbreaker.go](../internal/server/circuitbreaker.go)
  - [internal/server/transport_config.go](../internal/server/transport_config.go)
- `xkit-template` 已同步同样的 server 侧接法：
  - [../../xkit-template/internal/server/circuitbreaker.go](../../xkit-template/internal/server/circuitbreaker.go)
  - [../../xkit-template/internal/server/transport_config.go](../../xkit-template/internal/server/transport_config.go)
- `admin/configs/server.yaml` 与 `xkit-template/configs/server.yaml` 已补 breaker 参数样例。

编译验证结果：

- `admin`：`go test ./internal/server` 通过
- `xkit-template`：`go test ./internal/server` 通过

说明：

- 本轮 `conf` 代码生成使用的是本地 `protoc`，因为当时 `buf generate` 未跑通。

## 附录 B：Limiter + Breaker 联合验证结果

验证方式：

- 临时调低 breaker 门槛
- 临时放开 `GET /admin/v1/tenants`
- pause postgres 注入故障

结果：

- 前几次请求先超时
- 随后开始返回连续 `429`
- 恢复 postgres 后，接口恢复正常

结论：

- breaker 已感知失败
- limiter 比 breaker 更早接管了外部拒绝流量
- 因此这一轮没有直接观察到对外稳定 `503`

## 附录 C：纯 Breaker 验证结果

验证方式：

- 临时关闭 limiter
- 仅保留低门槛 breaker
- 再次执行 `tenants + pause postgres`

结果：

- 故障窗口内外部请求仍以客户端超时为主
- 数据库恢复后的第一下请求直接返回 `503`

结论：

- server breaker 已真实打开
- 在当前故障模型下，故障窗口内的多数请求先被下游阻塞和客户端超时吃掉，因此不一定能直接观察到 breaker 的快速拒绝响应

## 附录 D：当前运行态与回滚确认

本轮所有“验证用临时改动”均已回滚：

- `admin/internal/server/auth_support_ext.go` 中临时 `"/admin/v1/tenants"` 白名单已删除
- `admin/configs/server.yaml` 已恢复到当前默认样例参数
- 运行中的 `admin` 已重启恢复
- 当前 `GET /admin/v1/tenants` 未放白名单，访问结果为 `401`

当前保留的、属于正式代码能力的一部分：

- breaker 参数配置化
- limiter 参数配置化
- `context.Canceled` / `context.DeadlineExceeded` 纳入 breaker failure 判定
- `CreateDriver(...)` 方式接通 DB tracing
