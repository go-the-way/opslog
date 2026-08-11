# Interface Contracts

[简体中文](interfaces_zh.md)

This document describes the modules and interfaces defined today.
Implementations will follow; **the interfaces in this repository are the source of truth**.

## Packages

| Package   | Path              | Contents                                 |
| --------- | ----------------- | ---------------------------------------- |
| signal    | `pkg/signal`      | `Signal`, `Event`, `Kind`, `Attr`        |
| transport | `pkg/transport`   | `Transport`, `Type`, `Factory`           |
| query     | `pkg/query`       | `Query`, `Filter`, `Page`, archive types |
| sdk       | `sdk`             | Agent / Logger / Collector / …           |
| contract  | `server/contract` | All server extension interfaces          |

## 1. `pkg/signal`

### Kind

- `log` / `metric` / `config` / `probe`

### Signal

```go
type Signal interface {
    Kind() Kind
    Time() time.Time
    Level() string
    Service() string
    Host() string
    Message() string
    TraceID() string
    Attrs() map[string]any
    Raw() []byte
}
```

Default implementation: `Event` (JSON-serializable). Optional capability: `Cloneable`.

## 2. `pkg/transport`

```go
type Transport interface {
    Name() string
    Type() Type // udp|tcp|http|websocket|grpc
    Start(ctx context.Context) error
    Send(ctx context.Context, payload []byte) error
    SendBatch(ctx context.Context, payloads [][]byte) error
    Flush(ctx context.Context) error
    Close() error
    Healthy() bool
}
```

Used for **SDK client sending**. Server listeners are modeled as `contract.Input`.

## 3. `sdk`

| Interface   | Responsibility                                        |
| ----------- | ----------------------------------------------------- |
| `Agent`     | Embed entrypoint; owns Logger, Collectors, Transports |
| `Logger`    | Debug/Info/Warn/Error/With/Sync                       |
| `Collector` | Periodic or on-demand metric/config/probe collection  |
| `Probe`     | Single connectivity check                             |
| `Formatter` | Signal → `[]byte`                                     |
| `Hook`      | Mutate/filter before send                             |
| `Policy`    | Select one or more Transports per signal              |

Composition:

```text
Agent
  ├─ Logger()
  ├─ Collectors()
  └─ Transports()
       ▲
       │ Policy.Select(signal, available)
Formatter / Hook run before send
```

## 4. `server/contract`

### Main path

```text
Input → Decoder → Dispatcher → Processor/Pipeline → Output
```

| Interface         | Responsibility                              |
| ----------------- | ------------------------------------------- |
| `Input`           | Protocol listener; `Start(ctx, Dispatcher)` |
| `Decoder`         | Raw bytes → `Signal`                        |
| `Dispatcher`      | `Dispatch(ctx, Signal)` into the pipeline   |
| `BatchDispatcher` | Optional batch dispatch                     |
| `Processor`       | Transform/filter; `keep=false` drops        |
| `Pipeline`        | Ordered processor chain                     |
| `Output`          | `Write/Flush/Close` to a destination        |
| `Server`          | Process-level Start/Stop                    |
| `Registry`        | Register and Build factories                |

### Optional Output capabilities

| Interface        | Purpose                           |
| ---------------- | --------------------------------- |
| `Queryable`      | Search with pagination            |
| `Archiver`       | Hot → cold archive                |
| `Restorer`       | Restore from archive              |
| `LiveSubscriber` | Realtime subscribe (console tail) |

### OutputType

| Constant        | Status   |
| --------------- | -------- |
| `filesystem`    | Priority |
| `mysql`         | Priority |
| `clickhouse`    | Priority |
| `kafka`         | Reserved |
| `elasticsearch` | Reserved |

## 5. `pkg/query`

- `Query` / `Filter` / `Page` — search and live filters
- `ArchiveInfo` / `RestoreOptions` — filesystem archive/restore

## 6. Registry extension steps

```text
1. Implement the interface (e.g. Output)
2. registry.RegisterOutput(type, factory)
3. Configure:
   outputs:
     - name: fs
       type: filesystem
       ...
4. BuildOutput → inject into Server
```

The same pattern applies to Input / Decoder / Processor / Transport.

## 7. Design constraints

1. **Depend on interfaces, not implementations**
2. Use **small interfaces + type assertions** for optional capabilities
3. Factories receive **`map[string]any`** config; typed config is parsed by each implementation
4. **Reserved / unimplemented types must fail explicitly** — never silently ignore
