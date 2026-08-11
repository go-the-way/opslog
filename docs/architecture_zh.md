# OpsLog 架构说明

[English](architecture.md)

## 1. 目标

OpsLog 面向运维排障场景，同时采集：

1. **日志**：业务与错误事件
2. **系统配置**：主机/进程/版本等快照
3. **资源占用**：CPU / 内存 / 磁盘 / FD / Goroutine 等
4. **连通性**：对依赖与 OpsLog 自身的探测结果

通过统一 `Signal` 模型进入管道，再扇出到多个 **Output**。

## 2. 总览

```text
┌──────────────────────────────────────────┐
│ Go 业务进程                               │
│  sdk.Agent                                │
│   ├─ Logger                               │
│   ├─ Collector[]  (runtime/host/probe...) │
│   ├─ Hook / Formatter                     │
│   ├─ Policy（选择 Transport）              │
│   └─ Transport[]  udp/tcp/http/ws/grpc    │
└─────────────────┬────────────────────────┘
                  │ payload bytes
                  ▼
┌──────────────────────────────────────────┐
│ OpsLog Server                             │
│  Input[]  ←── 与 Transport 协议对称        │
│    └─ Decoder                             │
│         └─ Dispatcher                     │
│              └─ Processor / Pipeline      │
│                   └─ Output[]             │
│                        ├─ filesystem      │
│                        ├─ mysql           │
│                        └─ clickhouse      │
│  Query API / Live（后续）                  │
└──────────────────────────────────────────┘
```

## 3. 分层职责

| 层           | 包/位置                           | 职责                                 |
| ------------ | --------------------------------- | ------------------------------------ |
| 共享模型     | `pkg/signal`                      | `Signal` / `Event` / Kind / Attr     |
| 传输契约     | `pkg/transport`                   | 客户端 `Transport`                   |
| 查询类型     | `pkg/query`                       | Query / Page / Archive               |
| SDK          | `sdk`                             | Agent、Logger、Collector、Policy…    |
| Server 契约  | `server/contract`                 | Input、Dispatcher、Output、Registry… |
| 实现（后续） | `sdk/internal`、`server/internal` | 具体协议与存储实现                   |

## 4. 传输层

客户端只依赖 `transport.Transport`，可按配置启用一种或多种：

| Type        | 定位                                                 |
| ----------- | ---------------------------------------------------- |
| `http`      | **默认 / 推荐** — 可靠，失败可感知可重试             |
| `websocket` | 持续上报，可双向                                     |
| `tcp`       | 长连接可靠主通道（长度前缀帧）                       |
| `grpc`      | 强契约、多语言、内网高性能                           |
| `udp`       | 可选，默认关闭；诊断大包易超过 OS UDP 上限被静默丢弃 |

建议策略（实现阶段可配）：

- 默认 → `http`（或 `tcp` / `grpc`）
- 含 environ / sys / stack 的信号不要走 `udp`

Server 侧为每种协议提供对称 **Input**（Ingress），解码后进入同一条 `Dispatcher` 管道。

## 5. Server 处理路径

```text
Input.Start(dispatcher)
  → 收包
  → Decoder.Decode(remote, payload) => Signal
  → Dispatcher.Dispatch(signal)
  → Pipeline.Process（Normalize / Filter / Enrich / Sample…）
  → 按 Kind 或原样扇出到 MultiOutput
  → Output.Write(batch)
```

关键命名：

- **Dispatcher**：分发进管道（不用 Emitter）
- **Output**：输出目的地（不用 Sink）

## 6. 输出与排障平面

| Output                | 优先级 | 能力                                   |
| --------------------- | ------ | -------------------------------------- |
| filesystem            | P0     | 热数据、自动归档、回档；可选 Queryable |
| mysql                 | P0     | 结构化检索                             |
| clickhouse            | P0     | 高吞吐分析 / 指标曲线                  |
| kafka / elasticsearch | 预留   | 仅类型常量，暂不实现                   |

排障关联（产品语义）：

```text
ERROR 日志
  → 同 service/host 时间窗 metrics
  → probes 是否失败/变慢
  → config 是否刚变更
  → SDK 发送队列丢弃是否上升
```

## 7. 扩展原则

1. 核心依赖 **interface**，不依赖具体协议或存储
2. 通过 **Registry** 注册 Factory，配置驱动装配
3. 可选能力用小接口（`Queryable` / `Archiver` / `Restorer`），避免大而全
4. 新增能力优先加实现 + 注册，而不是改主干

## 8. 相关文档

- [接口约定](interfaces_zh.md)
- [SDK 说明](sdk_zh.md)
- [输出说明](outputs_zh.md)
