# 08 · 工程化落地（A1.3）

> 从"可阅读的架构文档"到"可运行的工程系统"——本卷把 A1.2 的架构契约落地为代码、数据库与本地环境。

## 目录结构

```
08_工程化落地/
├── 00_README.md                      ← 本文件
├── 01_数据库迁移/
│   ├── 0001_init.up.sql               ← T1：完整 CREATE TABLE（基于 SC-001 DDL）
│   └── 0001_init.down.sql             ← 回滚
├── 02_本地开发环境/
│   └── docker-compose.yml             ← T2：Go + Postgres + Kafka + 自动迁移
└── 03_API代码骨架/
    ├── Makefile                        ← gen / run / migrate / build
    ├── go-server/
    │   ├── go.mod
    │   ├── api/{types.go,server.go}   ← oapi-codegen 产物（不手改）
    │   ├── cmd/server/main.go          ← 入口 + 依赖注入 + 优雅关停
    │   ├── internal/
    │   │   ├── config/config.go
    │   │   ├── xbus/bus.go            ← Kafka 生产者 + 幂等
    │   │   └── handler/sc001.go        ← T3：SC-001 端到端业务流程
    │   ├── .air.toml                   ← 热重载
    │   └── Dockerfile                  ← 多阶段构建
    └── ts-client/
        ├── src/{index.ts, api/client.ts}  ← TS SDK（与 Go types 1:1）
        └── examples/sc-001-flow.html      ← T3：SC-001 前端单页演示
```

## 三步跑通 SC-001

```bash
# 1. 起本地环境（Postgres + Kafka + 自动执行迁移）
docker compose -f 02_本地开发环境/docker-compose.yml up -d

# 2. 迁移数据库（如未走自动初始化）
make -C 03_API代码骨架 migrate

# 3. 启动 API + 打开前端演示
make -C 03_API代码骨架 run
# 浏览器打开 ts-client/examples/sc-001-flow.html → 点击"一键走完全流程"
```

## 与 A1.2 契约的对齐

| 契约要素 | 落地位置 | 状态 |
|:--|:--|:--|
| `X-OS` / `X-MS` 命名 | SQL schema + OpenAPI 路径 | ✅ |
| `tenant_id` 多租户隔离 | 所有表 + `X-Tenant-Id` 头 | ✅ |
| 统一事件 schema | `xbus.Event` + 71 个事件类型 | ✅ |
| 幂等键 = eventType + eventID | `xbus.outbox` / `idempotency` | ✅ |
| SC-001 数据所有权（权威源）| SQL 表 `iam/cos/eos/fms/datalake` | ✅ |
| Saga + 事件驱动 | handler 步骤 ①②③⑤⑧⑨⑩ | ✅ |

## 下一步

- [ ] 用 oapi-codegen 从 `07/00_OpenAPI.yaml` 重新生成 `api/` 覆盖手写桩
- [ ] 补全其余 13 个场景（SC-002~014）的 handler
- [ ] 接入 OAS 底座：IAM（JWT）、支付网关 Mock、物流 Mock
- [ ] 为 SC-001 写集成测试（testcontainers + 真实 PG/Kafka）
