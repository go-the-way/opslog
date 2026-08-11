# OpsLog Architecture

[简体中文](architecture_zh.md)

## 1. Goals

OpsLog targets ops troubleshooting and collects:

1. **Logs** — application and error events
2. **Host / app config** — OS, process, version snapshots
3. **Resource usage** — CPU, memory, disk, FDs, goroutines, etc.
4. **Connectivity** — probes against dependencies and OpsLog itself

All of these enter the pipeline as a unified `Signal`, then fan out to one or more **Outputs**.

## 2. Overview

```text
┌──────────────────────────────────────────┐
│ Go service process                        │
│  sdk.Agent                                │
│   ├─ Logger                               │
│   ├─ Collector[]  (runtime/host/probe...) │
│   ├─ Hook / Formatter                     │
│   ├─ Policy (select Transport)            │
│   └─ Transport[]  udp/tcp/http/ws/grpc    │
└─────────────────┬────────────────────────┘
                  │ payload bytes
                  ▼
┌──────────────────────────────────────────┐
│ OpsLog Server                             │
│  Input[]  ←── symmetric to Transport      │
│    └─ Decoder                             │
│         └─ Dispatcher                     │
│              └─ Processor / Pipeline      │
│                   └─ Output[]             │
│                        ├─ filesystem      │
│                        ├─ mysql           │
│                        └─ clickhouse      │
│  Query API / Live (later)                 │
└──────────────────────────────────────────┘
```

## 3. Layer responsibilities

| Layer                   | Package                           | Responsibility                         |
| ----------------------- | --------------------------------- | -------------------------------------- |
| Shared model            | `pkg/signal`                      | `Signal` / `Event` / Kind / Attr       |
| Transport contract      | `pkg/transport`                   | Client `Transport`                     |
| Query types             | `pkg/query`                       | Query / Page / Archive                 |
| SDK                     | `sdk`                             | Agent, Logger, Collector, Policy, …    |
| Server contracts        | `server/contract`                 | Input, Dispatcher, Output, Registry, … |
| Implementations (later) | `sdk/internal`, `server/internal` | Protocol and storage implementations   |

## 4. Transport layer

Clients depend only on `transport.Transport` and may enable one or more types:

| Type        | Role                                                                                      |
| ----------- | ----------------------------------------------------------------------------------------- |
| `http`      | **Default / recommended** — reliable; failures visible and retryable                      |
| `websocket` | Continuous upload; optional bidirectional                                                 |
| `tcp`       | Reliable long-lived channel (length-prefixed frames)                                      |
| `grpc`      | Strong contract, polyglot, strong for private networks                                    |
| `udp`       | Optional only — not enabled by default; OS datagram limits drop large diagnostic payloads |

Suggested policy (configurable at implementation time):

- default → `http` (or `tcp` / `grpc`)
- avoid `udp` for signals carrying environ / sys / stack

The server exposes a matching **Input** per protocol; after decode, all paths share one `Dispatcher` pipeline.

## 5. Server processing path

```text
Input.Start(dispatcher)
  → receive
  → Decoder.Decode(remote, payload) => Signal
  → Dispatcher.Dispatch(signal)
  → Pipeline.Process (Normalize / Filter / Enrich / Sample …)
  → fan-out via MultiOutput
  → Output.Write(batch)
```

Naming:

- **Dispatcher** — dispatch into the pipeline (not Emitter)
- **Output** — destination (not Sink)

## 6. Outputs and troubleshooting plane

| Output                | Priority | Capabilities                                        |
| --------------------- | -------- | --------------------------------------------------- |
| filesystem            | P0       | Hot data, auto-archive, restore; optional Queryable |
| mysql                 | P0       | Structured search                                   |
| clickhouse            | P0       | High-throughput analytics / metric charts           |
| kafka / elasticsearch | Reserved | Type constants only for now                         |

Troubleshooting correlation (product semantics):

```text
ERROR log
  → metrics in the same service/host time window
  → whether probes failed or slowed down
  → whether config just changed
  → whether SDK send-queue drops increased
```

## 7. Extension principles

1. Depend on **interfaces**, not concrete protocols or storage
2. Register factories through **Registry**; assemble from config
3. Optional capabilities use small interfaces (`Queryable` / `Archiver` / `Restorer`)
4. Prefer new implementations + registration over changing the core

## 8. Related docs

- [Interfaces](interfaces.md)
- [SDK](sdk.md)
- [Outputs](outputs.md)
