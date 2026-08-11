# Outputs

[简体中文](outputs_zh.md)

The end of the Server pipeline uses the **Output** interface (not "Sink").

## Interface

```go
type Output interface {
    Name() string
    Type() OutputType
    Write(ctx context.Context, batch []signal.Signal) error
    Flush(ctx context.Context) error
    Close(ctx context.Context) error
}
```

Optional capabilities (type assert):

| Interface        | Description        |
| ---------------- | ------------------ |
| `Queryable`      | Search             |
| `Archiver`       | Hot → cold archive |
| `Restorer`       | Restore            |
| `LiveSubscriber` | Realtime subscribe |

## Implementation priority

| Type            | Priority | Notes                                     |
| --------------- | -------- | ----------------------------------------- |
| `filesystem`    | P0       | Local hot data + auto-archive + restore   |
| `mysql`         | P0       | Structured storage and SQL search         |
| `clickhouse`    | P0       | High-throughput analytics / metric charts |
| `kafka`         | Reserved | Constant defined; not implemented yet     |
| `elasticsearch` | Reserved | Constant defined; not implemented yet     |

## filesystem

Suggested layout:

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

| Capability | Behavior                                                                 |
| ---------- | ------------------------------------------------------------------------ |
| Write      | NDJSON, rotate by day/size                                               |
| Archive    | Compress hot data older than N days into `archive/`                      |
| Restore    | Restore by archive ID or time range to hot, or mount read-only for query |
| Cleanup    | Delete archives older than M days (configurable)                         |

Implementations should provide `Output` + `Archiver` + `Restorer`, and preferably `Queryable`.

## MySQL

Suggested tables (or equivalent):

- `logs`
- `metrics`
- `host_configs`
- `probes`

Common indexes: `ts`, `(service, ts)`, `(level, ts)`, `trace_id`.

Use batch inserts; implement `Output` + `Queryable`.

## ClickHouse

Prefer MergeTree, daily partitions, `ORDER BY (service, level, ts)` (tune as needed).

Separate log and metric tables for analytics; implement `Output` + `Queryable`.

## Fan-out

The same batch may be written to multiple Outputs:

```text
Pipeline → MultiOutput
             ├─ filesystem
             ├─ mysql
             └─ clickhouse
```

Failure in one Output must not take down the others (independent retry/degrade; details in implementation).

## Example config

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

  # reserved — do not enable in phase 1
  # - name: kafka
  #   type: kafka
```

Unimplemented types must return a clear startup error.
