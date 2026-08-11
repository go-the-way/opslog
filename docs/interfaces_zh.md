# 接口约定

[English](interfaces.md)

本文描述当前已定义的模块与接口，对应源码包。实现代码后续补齐，**契约以本仓库 interface 为准**。

## 包一览

| 包        | 路径              | 内容                                |
| --------- | ----------------- | ----------------------------------- |
| signal    | `pkg/signal`      | `Signal`、`Event`、`Kind`、`Attr`   |
| transport | `pkg/transport`   | `Transport`、`Type`、`Factory`      |
| query     | `pkg/query`       | `Query`、`Filter`、`Page`、归档类型 |
| sdk       | `sdk`             | Agent / Logger / Collector / …      |
| contract  | `server/contract` | Server 侧全部扩展接口               |

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

默认实现：`Event`（可 JSON 序列化）。可选能力：`Cloneable`。

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

用于 **SDK 客户端发送**。Server 监听侧对应 `contract.Input`。

## 3. `sdk`

| 接口        | 职责                                          |
| ----------- | --------------------------------------------- |
| `Agent`     | 嵌入入口：持有 Logger、Collectors、Transports |
| `Logger`    | Debug/Info/Warn/Error/With/Sync               |
| `Collector` | 周期或按需采集 metric/config/probe            |
| `Probe`     | 单个连通性检查                                |
| `Formatter` | Signal → `[]byte`                             |
| `Hook`      | 发送前改写/过滤                               |
| `Policy`    | 按信号选择一个或多个 Transport                |

组合关系：

```text
Agent
  ├─ Logger()
  ├─ Collectors()
  └─ Transports()
       ▲
       │ Policy.Select(signal, available)
Formatter / Hook 作用于发送前
```

## 4. `server/contract`

### 主路径

```text
Input → Decoder → Dispatcher → Processor/Pipeline → Output
```

| 接口              | 职责                                       |
| ----------------- | ------------------------------------------ |
| `Input`           | 协议接入（监听），`Start(ctx, Dispatcher)` |
| `Decoder`         | 原始字节 → `Signal`                        |
| `Dispatcher`      | `Dispatch(ctx, Signal)` 进入管道           |
| `BatchDispatcher` | 可选批量分发                               |
| `Processor`       | 变换/过滤；`keep=false` 表示丢弃           |
| `Pipeline`        | 有序 Processor 链                          |
| `Output`          | `Write/Flush/Close` 到目的地               |
| `Server`          | 进程级 Start/Stop                          |
| `Registry`        | 注册与 Build 各类 Factory                  |

### Output 可选能力

| 接口             | 用途                    |
| ---------------- | ----------------------- |
| `Queryable`      | 检索分页                |
| `Archiver`       | 热数据归档              |
| `Restorer`       | 回档                    |
| `LiveSubscriber` | 实时订阅（控制台 Tail） |

### OutputType

| 常量            | 状态     |
| --------------- | -------- |
| `filesystem`    | 优先实现 |
| `mysql`         | 优先实现 |
| `clickhouse`    | 优先实现 |
| `kafka`         | 预留     |
| `elasticsearch` | 预留     |

## 5. `pkg/query`

- `Query` / `Filter` / `Page`：检索与 Live 过滤
- `ArchiveInfo` / `RestoreOptions`：filesystem 归档回档

## 6. Registry 扩展步骤

```text
1. 实现 interface（例如 Output）
2. registry.RegisterOutput(type, factory)
3. 配置：
   outputs:
     - name: fs
       type: filesystem
       ...
4. BuildOutput → 注入 Server
```

同样模式适用于 Input / Decoder / Processor / Transport。

## 7. 设计约束

1. **依赖接口，不依赖实现**
2. **小接口 + 类型断言** 表达可选能力
3. **配置使用 `map[string]any` 进入 Factory**（具体 Typed Config 由各实现自行解析）
4. **预留类型必须显式失败**，不可静默忽略
