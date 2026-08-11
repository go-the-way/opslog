# OpsLog Documentation

English is the default. Chinese translations use the `*_zh.md` suffix.

| Doc                                | Description                             |
| ---------------------------------- | --------------------------------------- |
| [../README.md](../README.md)       | Project overview                        |
| [architecture.md](architecture.md) | Architecture and data flow              |
| [interfaces.md](interfaces.md)     | Modules and interfaces                  |
| [sdk.md](sdk.md)                   | Go SDK embedding                        |
| [outputs.md](outputs.md)           | Output design (filesystem / MySQL / CH) |

Chinese index: [README_zh.md](README_zh.md)

## Package godoc

- `pkg/signal` — unified signal model
- `pkg/transport` — transport interface
- `pkg/query` — query / archive types
- `sdk` — client SDK interfaces
- `server/contract` — server contracts

```bash
go doc github.com/go-the-way/opslog/sdk
go doc github.com/go-the-way/opslog/server/contract
```
