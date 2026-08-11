# OpsLog

OpsLog is a pluggable log and ops-telemetry platform.

It collects **logs, host configuration, resource metrics, and connectivity probes**, ships them over selectable transports, and writes them to destinations such as the filesystem, MySQL, and ClickHouse. A built-in **Web Console** is embedded in the server.

[简体中文](README_zh.md)

## Quick start (Docker)

```bash
./deploy.sh build
./deploy.sh up
./deploy.sh url
# open http://127.0.0.1:8600/
```

Useful commands (styled after cloudsystem/auth `all_in_one.sh`):

```bash
./deploy.sh help
./deploy.sh logs -f
./deploy.sh status
./deploy.sh restart
./deploy.sh down
COMPOSE_PROFILES=full ./deploy.sh up   # also start mysql + clickhouse
```

## Quick start (local Go)

```bash
go run ./cmd/opslog-server
# or with an explicit config
go run ./cmd/opslog-server -config configs/opslog.yml
# or
./deploy.sh run
```

Console: [http://127.0.0.1:8600/](http://127.0.0.1:8600/)

## Integrate SDK into a Go service

```bash
go get github.com/go-the-way/opslog@latest
```

```go
package main

import (
    "context"
    "time"

    "github.com/go-the-way/opslog/sdk"
    "github.com/go-the-way/opslog/sdk/agent"
    "github.com/go-the-way/opslog/sdk/collectors"
    "github.com/go-the-way/opslog/sdk/transport"
)

func main() {
    httpTransport, _ := transport.NewHTTPTransport("http", "http://127.0.0.1:8600/ingest")

    a, err := agent.NewAgent(
        agent.WithService("order-api"),
        agent.WithTransport(httpTransport),
        agent.WithCollector(collectors.NewRuntime()),
        agent.WithCollector(collectors.NewHost()),
        agent.WithCollectInterval(15*time.Second),
    )
    if err != nil { panic(err) }
    defer a.Close()

    _ = a.Start(context.Background())
    a.Logger().Info("hello opslog", sdk.String("k", "v"))
    _ = a.Sync()
}
```

See `examples/go-app` for a runnable sample. Prefer **HTTP ingest** — diagnostic payloads (env/sys/stack) often exceed OS UDP limits and are dropped silently.

## Ports

| Port | Protocol | Purpose                                  |
| ---- | -------- | ---------------------------------------- |
| 8600 | TCP      | Web Console + Query API + HTTP/WS ingest |
| 8141 | TCP      | TCP length-prefixed ingest               |
| 8900 | TCP      | gRPC ingest                              |

## Docker details

- `Dockerfile` — multi-stage build of `opslog-server` (embedded console)
- `docker-compose.yml` — `opslog` service; optional profiles `mysql` / `clickhouse` / `full`
- Config mount: `configs/opslog.yml`
- Data volume: `opslog-data` → `/app/data` (filesystem output)

```bash
docker compose config   # validate
./deploy.sh config      # same via wrapper
```

Point the SDK at HTTP ingest: `http://127.0.0.1:8600/ingest`.

## Web Console

Open `http://127.0.0.1:8600/` after server start (browser will prompt for HTTP Basic Auth):

- Log search (time / level / service / keyword)
- Live Tail (WebSocket)
- Metrics / probes / config snapshots
- Filesystem archive list + restore

Default credentials (change in `configs/opslog.yml`): **opslog / opslog**

```yaml
web:
  basic_auth:
    enabled: true      # false disables auth
    username: opslog
    password: opslog   # empty password also disables auth
```

Protected: static UI (`/`) only. Query APIs (`/api/*`) and ingest (`/ingest`, `/stream`, TCP/gRPC) stay open for SDKs/tools by default.

Optional ingest Bearer (HTTP/WS only): set `http.token` in `configs/opslog.yml`; clients send `Authorization: Bearer <token>`. This is separate from `web.basic_auth`.

## Docs

| Doc                                  | Description                     |
| ------------------------------------ | ------------------------------- |
| [Architecture](docs/architecture.md) | System architecture             |
| [Interfaces](docs/interfaces.md)     | Module and interface contracts  |
| [SDK](docs/sdk.md)                   | SDK embedding                   |
| [Outputs](docs/outputs.md)           | filesystem / MySQL / ClickHouse |

## Module layout

```text
opslog/
├── cmd/opslog-server/     # server entry
├── sdk/                   # embeddable Go SDK
├── server/                # server public Run() + contracts/internal
├── pkg/                   # signal / transport / query / codec
├── opslog.yml             # default server config
├── configs/example.yaml   # optional example copy
├── deploy.sh              # Docker/local ops entry
├── Dockerfile
└── docker-compose.yml
```

## Status

| Area                                           | Status               |
| ---------------------------------------------- | -------------------- |
| SDK (Agent/Logger/Collectors/Transports)       | Implemented          |
| Server inputs (http/ws/tcp/grpc; udp optional) | Implemented          |
| Outputs filesystem / mysql / clickhouse        | Implemented          |
| Web Console + Query API                        | Implemented          |
| Docker + deploy.sh                             | Implemented          |
| kafka / elasticsearch outputs                  | Reserved (fail-fast) |

```bash
go build ./...
```

## License

Apache License 2.0 — see [LICENSE](LICENSE).
