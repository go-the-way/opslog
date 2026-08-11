# OpsLog

基于可插拔传输与接口化管道的日志运维平台。

支持采集 **日志 / 系统配置 / 资源占用 / 连通性探测**，经 HTTP/WebSocket/TCP/gRPC 上报，输出到 filesystem / MySQL / ClickHouse，并内置 **Web 控制台**。推荐使用 **HTTP ingest**（诊断字段较大时 UDP 易被系统静默丢弃）。

[English](README.md)

## Docker 快速部署

```bash
./deploy.sh build
./deploy.sh up
./deploy.sh url
# 浏览器打开 http://127.0.0.1:8600/
```

常用命令（风格对齐 cloudsystem/auth 的 `all_in_one.sh`）：

```bash
./deploy.sh help
./deploy.sh logs -f
./deploy.sh status
./deploy.sh restart
./deploy.sh down
COMPOSE_PROFILES=full ./deploy.sh up   # 同时拉起 mysql + clickhouse
```

## 本地运行

```bash
go run ./cmd/opslog-server
# 或显式指定配置
go run ./cmd/opslog-server -config configs/opslog.yml
# 或
./deploy.sh run
```

控制台：http://127.0.0.1:8600/（浏览器会弹出 HTTP Basic Auth）

默认账号（请在 `configs/opslog.yml` 中修改）：**opslog / opslog**

```yaml
web:
  basic_auth:
    enabled: true      # false 关闭认证
    username: opslog
    password: opslog   # 空密码也会关闭认证
```

受保护：仅静态控制台（`/`）。查询 API（`/api/*`）与接入路径（`/ingest`、`/stream` 以及 TCP/gRPC）默认开放。

可选的接入 Bearer（仅 HTTP/WS）：在 `configs/opslog.yml` 中设置 `http.token`；客户端发送 `Authorization: Bearer <token>`。与 `web.basic_auth` 无关。

## 业务 Go 项目集成 SDK

```bash
go get github.com/go-the-way/opslog@latest
```

```go
import (
    "github.com/go-the-way/opslog/sdk/agent"
    "github.com/go-the-way/opslog/sdk/collectors"
    "github.com/go-the-way/opslog/sdk/transport"
)

httpTransport, _ := transport.NewHTTPTransport("http", "http://127.0.0.1:8600/ingest")

a, _ := agent.NewAgent(
    agent.WithService("order-api"),
    agent.WithTransport(httpTransport),
    agent.WithCollector(collectors.NewRuntime()),
)
defer a.Close()
_ = a.Start(context.Background())
a.Logger().Info("hello")
```

完整示例见 `examples/go-app`。

## 端口

| 端口       | 说明                           |
|----------|------------------------------|
| 8600/tcp | 控制台 + Query API + HTTP/WS 接入 |
| 8141/tcp | TCP 接入                       |
| 8900/tcp | gRPC 接入                      |

数据卷：`opslog-data` → 容器内 `/app/data`。

## 文档

中文设计文档：

- [架构](docs/architecture_zh.md)
- [接口](docs/interfaces_zh.md)
- [SDK](docs/sdk_zh.md)
- [输出](docs/outputs_zh.md)

## License

Apache License 2.0 — 见 [LICENSE](LICENSE)。
