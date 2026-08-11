# OpsLog 文档索引

英文为默认文档；中文使用 `*_zh.md` 后缀。

| 文档                                     | 内容                               |
| ---------------------------------------- | ---------------------------------- |
| [../README_zh.md](../README_zh.md)       | 项目总览与快速导航                 |
| [architecture_zh.md](architecture_zh.md) | 总体架构与数据流                   |
| [interfaces_zh.md](interfaces_zh.md)     | 模块与接口约定                     |
| [sdk_zh.md](sdk_zh.md)                   | Go SDK 嵌入与扩展                  |
| [outputs_zh.md](outputs_zh.md)           | Output 设计（filesystem/MySQL/CH） |

英文索引：[README.md](README.md)

## 包内 Godoc

- `pkg/signal` — 统一信号模型
- `pkg/transport` — 传输接口
- `pkg/query` — 查询/归档类型
- `sdk` — 客户端 SDK 接口
- `server/contract` — Server 契约

```bash
go doc github.com/go-the-way/opslog/sdk
go doc github.com/go-the-way/opslog/server/contract
```
