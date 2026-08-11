# 输出（Output）说明

[English](outputs.md)

Server 管道末端使用 **Output** 接口（不再使用 Sink 命名）。

## 接口

```go
type Output interface {
    Name() string
    Type() OutputType
    Write(ctx context.Context, batch []signal.Signal) error
    Flush(ctx context.Context) error
    Close(ctx context.Context) error
}
```

可选能力（类型断言）：

| 接口             | 说明        |
| ---------------- | ----------- |
| `Queryable`      | 检索        |
| `Archiver`       | 热 → 冷归档 |
| `Restorer`       | 回档        |
| `LiveSubscriber` | 实时订阅    |

## 实现优先级

| Type            | 优先级 | 说明                         |
| --------------- | ------ | ---------------------------- |
| `filesystem`    | P0     | 本地热数据 + 自动归档 + 回档 |
| `mysql`         | P0     | 结构化存储与 SQL 检索        |
| `clickhouse`    | P0     | 高吞吐分析、指标曲线         |
| `kafka`         | 预留   | 常量已定义，暂不实现         |
| `elasticsearch` | 预留   | 常量已定义，暂不实现         |

## filesystem

建议目录：

```text
data/
  hot/
    log/
    metric/
    config/
    probe/
  archive/
    *.ndjson.gz
    *.meta.json
```

| 能力 | 行为                                              |
| ---- | ------------------------------------------------- |
| 写入 | NDJSON，按日/大小滚动                             |
| 归档 | 超过 N 天的热数据压缩移入 archive                 |
| 回档 | 按 archiveID 或时间范围恢复到 hot，或只读挂载查询 |
| 清理 | 归档保留 M 天后删除（可配）                       |

实现时应同时满足：`Output` + `Archiver` + `Restorer`，并尽量实现 `Queryable`。

## MySQL

建议分表（或等价设计）：

- `logs`
- `metrics`
- `host_configs`
- `probes`

常见索引：`ts`、`(service, ts)`、`(level, ts)`、`trace_id`。

批量插入；实现 `Output` + `Queryable`。

## ClickHouse

建议 MergeTree，按天分区，`ORDER BY (service, level, ts)`（可视情况调整）。

日志与指标分表更利于分析；实现 `Output` + `Queryable`。

## 扇出

同一批 Signal 可写入多个 Output：

```text
Pipeline → MultiOutput
             ├─ filesystem
             ├─ mysql
             └─ clickhouse
```

单个 Output 失败不应拖死其它 Output（独立重试/降级，实现阶段细化）。

## 配置示意

```yaml
outputs:
  - name: fs
    type: filesystem
    path: ./data
    rotate: daily
    archive_after_days: 7
    archive_retain_days: 90

  - name: mysql
    type: mysql
    dsn: "user:pass@tcp(127.0.0.1:3306)/opslog"

  - name: ch
    type: clickhouse
    dsn: "clickhouse://127.0.0.1:9000/opslog"

  # 预留，第一期不要启用
  # - name: kafka
  #   type: kafka
```

未实现类型必须在启动时返回明确错误。
