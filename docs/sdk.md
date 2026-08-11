# Go SDK

[简体中文](sdk_zh.md)

The root `sdk` package defines embeddable client **contracts** (interfaces + Attr helpers).
Implementations live in subpackages — import what you use.

## Package layout

| Import path                  | Role                                                                     |
| ---------------------------- | ------------------------------------------------------------------------ |
| `sdk`                        | Interfaces (`Agent`, `Logger`, `Collector`, …) and Attr helpers          |
| `sdk/agent`                  | `NewAgent`, options, Agent/Logger runtime                                |
| `sdk/formatter`              | JSON formatter                                                           |
| `sdk/policy`                 | Level-based transport policy                                             |
| `sdk/transport`              | Convenience constructors (UDP/TCP/HTTP/WS/gRPC)                          |
| `sdk/collectors`             | Runtime / host / config / probe collectors                               |
| `sdk/enricher`               | Diagnostic context hook (default-on for Agent)                           |
| `sdk/middleware/panicopt`    | Shared panic options (`WithContinuePanic` / `WithAttrs` / `WithMessage`) |
| `sdk/middleware/opslog4gin`  | Gin recovery middleware                                                  |
| `sdk/middleware/opslog4proc` | `Recover` / `Guard` / `Do` / `ReportPanic` (process-level)               |
| `sdk/integration`            | Env bootstrap + process-wide `GinRecovery` / `Guard` / `Main`            |

## Roles

| Component                          | Role                            |
| ---------------------------------- | ------------------------------- |
| OpsLog Server                      | Receive, store, search, alert   |
| `sdk.Agent` (via `agent.NewAgent`) | Runs in-process: collect + send |

## Quick start (integration helper)

```go
integration.MustInitFromEnv(integration.WithService("order-api"), integration.WithVersion("1.0.0"))
defer integration.Close()
r.Use(integration.GinRecovery())
defer integration.Guard()()
```

Set `OPSLOG_ENABLE=T` to start the Agent (HTTP ingest by default). See `sdk/integration`.

## Quick start (manual Agent)

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
        agent.WithVersion("1.0.0"), // optional; also reads VERSION / APP_VERSION
        agent.WithTransport(httpTransport), // prefer HTTP; UDP drops large diagnostic payloads
        agent.WithCollector(collectors.NewRuntime()),
        // Diagnostic enricher is on by default. Opt out:
        // agent.WithoutDiagnosticEnricher(),
        // Or tune: agent.WithDiagnosticEnricher(enricher.WithIncludeEnviron(true)),
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

## Migration (flat `sdk` → subpackages)

| Old (`sdk`)                                    | New                                                                  |
| ---------------------------------------------- | -------------------------------------------------------------------- |
| `sdk.NewAgent` / `sdk.With*`                   | `agent.NewAgent` / `agent.With*`                                     |
| `sdk.NewUDPTransport` / `NewHTTPTransport` / … | `sdk/transport` same names                                           |
| `sdk.NewLevelPolicy`                           | `policy.NewLevelPolicy`                                              |
| `sdk.NewJSONFormatter`                         | `formatter.NewJSONFormatter`                                         |
| `sdk.Recover` / `ReportPanic` / `WithRepanic`  | `opslog4proc.Recover` / `ReportPanic` / `panicopt.WithContinuePanic` |
| `sdk.WithRecoverAttrs` / `WithRecoverMessage`  | `panicopt.WithAttrs` / `panicopt.WithMessage`                        |
| `sdk/ginmw` / `sdk/middleware/ginmw`           | `sdk/middleware/opslog4gin`                                          |
| `sdk/recovery` / `sdk/opslog4proc`             | `sdk/middleware/opslog4proc`                                         |
| Interfaces / `sdk.String` / …                  | stay in root `sdk`                                                   |

## Core interfaces (`sdk`)

### Agent

- `Logger() Logger`
- `Start(ctx) / Sync() / Close()`
- `Transports() []transport.Transport`
- `Collectors() []Collector`

### Logger

- `Debug/Info/Warn/Error(msg, attrs...)`
- `With(attrs...) Logger`
- `Sync() error` — flush before process exit

### Collector / Probe

- `Collector.Collect(ctx) ([]signal.Signal, error)`
- Built-ins in `sdk/collectors`: Runtime, Host, Config, Probe

### Formatter / Hook / Policy

| Interface   | Role                              | Default impl                             |
| ----------- | --------------------------------- | ---------------------------------------- |
| `Formatter` | Signal → bytes                    | `formatter.NewJSONFormatter`             |
| `Hook`      | Enrich or filter before send      | `enricher.NewDiagnostic` (Agent default) |
| `Policy`    | Choose Transport(s) by level/kind | `policy.NewLevelPolicy`                  |

### Diagnostic enricher (default-on)

Every outbound signal (logs, panics, collectors) gets diagnostic attrs via a prepended Hook, unless disabled with `agent.WithoutDiagnosticEnricher()`.

Attached attrs (existing keys are never overwritten):

| Attr                           | Source                                                                   |
| ------------------------------ | ------------------------------------------------------------------------ |
| `service`                      | `WithService`                                                            |
| `version`                      | `WithVersion` / `VERSION` / `APP_VERSION`                                |
| `git_sha`                      | `WithGitSHA` / `GIT_SHA` / `GIT_COMMIT` / `COMMIT_SHA`                   |
| `hostname`, `pid`, `cwd`       | runtime                                                                  |
| `go_version`, `goos`, `goarch` | runtime                                                                  |
| `profile`                      | first of `APP_ENV` / `ENV` / `GO_ENV` / `PROFILE`                        |
| `env`                          | allow-listed keys (`enricher.DefaultEnvKeys`); secrets skipped           |
| `environ`                      | optional filtered `os.Environ()` via `enricher.WithIncludeEnviron(true)` |

`collectors.NewConfig` reuses the same allow-list / secret filter (`enricher.CollectAllowListedEnv`). When no env keys are passed, it uses `enricher.DefaultEnvKeys`.

## Send path

```text
Logger / Collector
  → Async Queue (backpressure / drop policy)
  → Hook.BeforeSend (default: diagnostic enricher, then user hooks)
  → Formatter.Format
  → Policy.Select
  → Transport.Send / SendBatch
```

## Panic recovery

Shared options live in `sdk/middleware/panicopt` (`WithContinuePanic` / `WithAttrs` / `WithMessage`).
`opslog4proc` and `opslog4gin` share the reporting core (`sdk/middleware/internal/panicx`); do not import `panicx` from application code.

Panic reports go through the Agent Logger, so they automatically include diagnostic attrs (`profile`, `cwd`, `go_version`, …) in addition to `panic` / `stack` / `recovered`.

### Defer helper (`sdk/middleware/opslog4proc`)

Default: recover, log with stack attrs, **do not** continue the panic. Pass `panicopt.WithContinuePanic(true)` to panic again after logging.

```go
import (
    "github.com/go-the-way/opslog/sdk"
    "github.com/go-the-way/opslog/sdk/middleware/opslog4proc"
    "github.com/go-the-way/opslog/sdk/middleware/panicopt"
)

func handle() {
    defer opslog4proc.Recover(logger)
    defer opslog4proc.Guard(logger, panicopt.WithAttrs(sdk.String("component", "worker")))()
    // or: defer opslog4proc.GuardAgent(agent)()
    // opslog4proc.Do(logger, func() { /* ... */ }, panicopt.WithContinuePanic(true))
    panic("boom")
}
```

Use `opslog4proc.ReportPanic(logger, recovered, opts...)` when you already recovered.

### Gin middleware (`sdk/middleware/opslog4gin`)

Panic options come from `panicopt`. Gin-specific response shape uses `Response` / `RecoveryResponse`.

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

On panic: logs stack via OpsLog (same path as `opslog4proc`), responds `500` JSON `{"error":"internal server error"}` (override via `RecoveryResponse`).
