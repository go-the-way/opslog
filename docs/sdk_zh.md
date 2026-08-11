# Go SDK 说明

[English](sdk.md)

根包 `sdk` 定义可嵌入业务的客户端**契约**（接口 + Attr 助手）。
实现位于子包——按需导入。

## 包布局

| 导入路径                     | 角色                                                                 |
| ---------------------------- | -------------------------------------------------------------------- |
| `sdk`                        | 接口（`Agent`、`Logger`、`Collector` 等）与 Attr 助手                |
| `sdk/agent`                  | `NewAgent`、选项、Agent/Logger 运行时                                |
| `sdk/formatter`              | JSON 格式化器                                                        |
| `sdk/policy`                 | 按级别选择 Transport 的策略                                          |
| `sdk/transport`              | 传输构造助手（UDP/TCP/HTTP/WS/gRPC）                                 |
| `sdk/collectors`             | Runtime / host / config / probe 采集器                               |
| `sdk/enricher`               | 诊断上下文 Hook（Agent 默认开启）                                    |
| `sdk/middleware/panicopt`    | 共享 panic 选项（`WithContinuePanic` / `WithAttrs` / `WithMessage`） |
| `sdk/middleware/opslog4gin`  | Gin 恢复中间件                                                       |
| `sdk/middleware/opslog4proc` | `Recover` / `Guard` / `Do` / `ReportPanic`（进程级）                 |
| `sdk/integration`            | 环境变量启动 + 进程级 `GinRecovery` / `Guard` / `Main`               |

## 定位

| 组件                               | 角色                          |
| ---------------------------------- | ----------------------------- |
| OpsLog Server                      | 接收、存储、检索、告警        |
| `sdk.Agent`（经 `agent.NewAgent`） | 运行在业务进程内：采集 + 发送 |

## 快速开始（integration 助手）

```go
integration.MustInitFromEnv(integration.WithService("order-api"), integration.WithVersion("1.0.0"))
defer integration.Close()
r.Use(integration.GinRecovery())
defer integration.Guard()()
```

设置 `OPSLOG_ENABLE=T` 启动 Agent（默认 HTTP ingest）。详见 `sdk/integration`。

## 快速开始（手动组装 Agent）

```go
package main

import (
    "context"

    "github.com/go-the-way/opslog/sdk"
    "github.com/go-the-way/opslog/sdk/agent"
    "github.com/go-the-way/opslog/sdk/collectors"
    "github.com/go-the-way/opslog/sdk/transport"
)

func main() {
    httpTransport, _ := transport.NewHTTPTransport("http", "http://127.0.0.1:8600/ingest")

    a, err := agent.NewAgent(
        agent.WithService("order-api"),
        agent.WithVersion("1.0.0"), // 可选；也会读 VERSION / APP_VERSION
        agent.WithTransport(httpTransport), // 推荐 HTTP；UDP 对大包诊断字段易静默丢弃
        agent.WithCollector(collectors.NewRuntime()),
        // 诊断 enricher 默认开启。关闭：
        // agent.WithoutDiagnosticEnricher(),
        // 或调参：agent.WithDiagnosticEnricher(enricher.WithIncludeEnviron(true)),
    )
    if err != nil {
        panic(err)
    }
    defer a.Close()

    if err := a.Start(context.Background()); err != nil {
        panic(err)
    }
    defer a.Sync()

    log := a.Logger()
    log.Info("order created", sdk.String("id", "42"))
}
```

## 迁移（扁平 `sdk` → 子包）

| 旧（`sdk`）                                    | 新                                                                   |
| ---------------------------------------------- | -------------------------------------------------------------------- |
| `sdk.NewAgent` / `sdk.With*`                   | `agent.NewAgent` / `agent.With*`                                     |
| `sdk.NewUDPTransport` / `NewHTTPTransport` / … | `sdk/transport` 同名函数                                             |
| `sdk.NewLevelPolicy`                           | `policy.NewLevelPolicy`                                              |
| `sdk.NewJSONFormatter`                         | `formatter.NewJSONFormatter`                                         |
| `sdk.Recover` / `ReportPanic` / `WithRepanic`  | `opslog4proc.Recover` / `ReportPanic` / `panicopt.WithContinuePanic` |
| `sdk.WithRecoverAttrs` / `WithRecoverMessage`  | `panicopt.WithAttrs` / `panicopt.WithMessage`                        |
| `sdk/ginmw` / `sdk/middleware/ginmw`           | `sdk/middleware/opslog4gin`                                          |
| `sdk/recovery` / `sdk/opslog4proc`             | `sdk/middleware/opslog4proc`                                         |
| 接口 / `sdk.String` / …                        | 仍在根包 `sdk`                                                       |

## 核心接口（`sdk`）

### Agent

- `Logger() Logger`
- `Start(ctx) / Sync() / Close()`
- `Transports() []transport.Transport`
- `Collectors() []Collector`

### Logger

- `Debug/Info/Warn/Error(msg, attrs...)`
- `With(attrs...) Logger`
- `Sync() error` — 进程退出前刷队列

### Collector / Probe

- `Collector.Collect(ctx) ([]signal.Signal, error)`
- 内置实现见 `sdk/collectors`：Runtime、Host、Config、Probe

### Formatter / Hook / Policy

| 接口        | 作用                         | 默认实现                               |
| ----------- | ---------------------------- | -------------------------------------- |
| `Formatter` | Signal → 报文                | `formatter.NewJSONFormatter`           |
| `Hook`      | 发送前补字段或过滤           | `enricher.NewDiagnostic`（Agent 默认） |
| `Policy`    | 按 level/kind 选择 Transport | `policy.NewLevelPolicy`                |

### 诊断 enricher（默认开启）

每条出站信号（日志、panic、采集器）都会经前置 Hook 附上诊断字段；可用 `agent.WithoutDiagnosticEnricher()` 关闭。

附加字段（已有同名 attr **不会**被覆盖）：

| 字段                           | 来源                                                                 |
| ------------------------------ | -------------------------------------------------------------------- |
| `service`                      | `WithService`                                                        |
| `version`                      | `WithVersion` / `VERSION` / `APP_VERSION`                            |
| `git_sha`                      | `WithGitSHA` / `GIT_SHA` / `GIT_COMMIT` / `COMMIT_SHA`               |
| `hostname`、`pid`、`cwd`       | 运行时                                                               |
| `go_version`、`goos`、`goarch` | 运行时                                                               |
| `profile`                      | `APP_ENV` / `ENV` / `GO_ENV` / `PROFILE` 中第一个非空                |
| `env`                          | 白名单键（`enricher.DefaultEnvKeys`）；疑似密钥键跳过                |
| `environ`                      | 可选：经过滤的 `os.Environ()`（`enricher.WithIncludeEnviron(true)`） |

`collectors.NewConfig` 复用同一套白名单 / 密钥过滤（`enricher.CollectAllowListedEnv`）。未传 env keys 时使用 `enricher.DefaultEnvKeys`。

## 发送路径

```text
Logger / Collector
  → Async Queue（背压 / 丢弃策略）
  → Hook.BeforeSend（默认诊断 enricher，再是用户 Hook）
  → Formatter.Format
  → Policy.Select
  → Transport.Send / SendBatch
```

## Panic 恢复

共享选项在 `sdk/middleware/panicopt`（`WithContinuePanic` / `WithAttrs` / `WithMessage`）。
`opslog4proc` 与 `opslog4gin` 共用上报核心（`sdk/middleware/internal/panicx`）；业务代码请勿直接导入 `panicx`。

Panic 上报走 Agent Logger，因此除 `panic` / `stack` / `recovered` 外，会自动带上诊断字段（`profile`、`cwd`、`go_version` 等）。

### defer 助手（`sdk/middleware/opslog4proc`）

默认：recover、带 stack 属性打日志、**不**再次 panic。需要继续 panic 时使用 `panicopt.WithContinuePanic(true)`。

```go
import (
    "github.com/go-the-way/opslog/sdk"
    "github.com/go-the-way/opslog/sdk/middleware/opslog4proc"
    "github.com/go-the-way/opslog/sdk/middleware/panicopt"
)

func handle() {
    defer opslog4proc.Recover(logger)
    defer opslog4proc.Guard(logger, panicopt.WithAttrs(sdk.String("component", "worker")))()
    // 或: defer opslog4proc.GuardAgent(agent)()
    // opslog4proc.Do(logger, func() { /* ... */ }, panicopt.WithContinuePanic(true))
    panic("boom")
}
```

若已自行 recover，可调用 `opslog4proc.ReportPanic(logger, recovered, opts...)`。

### Gin 中间件（`sdk/middleware/opslog4gin`）

panic 选项来自 `panicopt`；Gin 专属响应形态用 `Response` / `RecoveryResponse`。

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/go-the-way/opslog/sdk"
    "github.com/go-the-way/opslog/sdk/middleware/opslog4gin"
    "github.com/go-the-way/opslog/sdk/middleware/panicopt"
)

r := gin.New()
r.Use(opslog4gin.Recovery(logger, panicopt.WithAttrs(sdk.String("component", "http"))))
r.Use(opslog4gin.RecoveryResponse(logger, opslog4gin.Response{PlainText: true}, panicopt.WithContinuePanic(true)))
```

发生 panic 时：经 OpsLog 记录堆栈（与 `opslog4proc` 同一路径），并返回 `500` JSON `{"error":"internal server error"}`（可用 `RecoveryResponse` 覆盖）。
