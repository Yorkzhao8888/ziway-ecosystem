# SC-013 · 跨 OS 事件驱动数据同步（标准范例）

> **视角**：横切/事件｜**主角角色**：`—` ｜ **涉及平台**：COS↔EOS (XBUS)
> **对应 A1.1**：02_B 角色派生 / 03_B OS 十二平台 / 03_C MS 十二管理集座 / 03_D OAS 底座
> **格式**：标准范例（业务流 + 精简时序图 + 增量 DDL + 事件契约 + 一致性验证）


## 一、业务背景

验证跨 OS 协作经 XBUS 事件总线：权威源、幂等、Saga 补偿、72h 超时等架构约束。

## 二、前置准备

- **角色**：`—`（由四根身份 HU/EU/GU/OU 派生，戴帽机制见 SC-014）
- **数据**：复用通用表 `iam.tenants` / `iam.users`（见 `00_场景范例索引.md` 通用表清单），以下仅列**本场景增量表**
- **权限**：`tenant_id` 多租户隔离；跨 OS 调用经 XBUS 事件总线

## 三、关键流程步骤

| # | 步骤 | 参与者流向 |
|:--|:--|:--|
| 1 | 事件发布至 XBUS |
| 2 | 消费者拉取(幂等键去重) |
| 3 | 本地事务写入 |
| 4 | 补偿/重试机制 |
| 5 | 72h 超时告警 |
| 6 | 死信队列处置 |
| 7 | 数据一致性校验 |
| 8 | 链路追踪(Jaeger) |

## 四、时序图（精简）

```mermaid
sequenceDiagram
        participant System as Actor
        participant APP as APP终端
        participant OS as X-OS业务平台
        participant MS as X-MS管控总部
        participant OAS as OAS/datalake
        APP->>OS: 1. 事件发布至 XBUS
        OS->>OAS/datalake: 2. 消费者拉取(幂等键去重)
        APP->>OS: 3. 本地事务写入
        OS->>OAS/datalake: 4. 补偿/重试机制
        APP->>OS: 5. 72h 超时告警
        OS->>MS: 6. 死信队列处置
        APP->>OS: 7. 数据一致性校验
        OS->>OAS/datalake: 8. 链路追踪(Jaeger)
```

## 五、关键 API（RESTful）

| 方法 | 路径 | 说明 |
|:--|:--|:--|
| POST | `/api/{os}/...` | 创建类操作（写，走 Saga） |
| GET  | `/api/{os}/...` | 查询（同步 gRPC/HTTP） |
| POST | `/api/{ms}/...` | 管控规则/审批 |
| EVENT| `xbus.<事件类型>` | 跨 OS 异步事件（幂等） |

> 完整路径与字段见 `00_API_清单.md`（OpenAPI YAML）。

## 六、增量数据表 DDL

以下表为**本场景新增**，通用表（`iam.tenants`/`iam.users` 等）不重复定义：

- `xbus.event_log`
- `xbus.dead_letter_queue`
- `datalake.consistency_checks`

## 七、事件契约

复用统一事件类型（幂等键 = 事件类型 + 业务 ID，72h 超时，死信队列见 SC-013）：

`EventPublished`, `EventConsumed`, `EventIdempotent`, `CompensationTriggered`, `DLQRouting`

## 八、与 A1.1 一致性验证

| 验证点 | 结果 |
|:--|:--|
| X 在 OS、M 在 MS | ✅ 交互在 APP/OS，审核管控在 MS |
| 层间正交（APP 不经 MS）| ✅ 均经 OS 中转 |
| 单一权威源 | ✅ 每类数据仅一个 owner |
| 四根身份 / 戴帽 | ✅ 主角由根身份派生 |
| 跨 OS 协作经 XBUS | ✅ 多场景涉及（详见 SC-013）|
| EOS=供给运营 / EMS=供给管理 | ✅ 命名规范统一 |

## 九、一句话总结

> **跨 OS 事件驱动数据同步** 场景完整走通了「横切/事件」下的角色→平台→管控→底座闭环，
验证了 A1.1 「COS↔EOS (XBUS)」协作的正确性与职责边界清晰性。
