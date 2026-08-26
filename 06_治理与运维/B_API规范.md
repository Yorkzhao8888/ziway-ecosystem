# API 规范

> 统一 API 设计规范，含直通 OS 路径 transform 规则。

---

## 一、URL 规范

### 业务 API（经 OS 编排）

```
/api/v1/{tenant_id}/{bos}/{resource}[/{id}]
```

示例：
```
GET  /api/v1/repair/cbos/orders
POST /api/v1/repair/cbos/orders
```

### 直通 API（Kong 路由直达 MS）

```
/api/v1/{tenant_id}/{mbs}/*
```

示例：
```
GET  /api/v1/repair/pms/products
POST /api/v1/repair/ems/suppliers
```

> Kong 按 `{mbs}` 前缀将请求转发到对应 MS 服务。

---

## 二、直通 OS 路由表

| 直通 OS | MS | Kong 路由 |
|:--------:|:---:|:----------|
| AOS | AMS | `/api/ams/*` |
| POS | PMS | `/api/pms/*` |
| HOS | HMS | `/api/hms/*` |
| EOS | EMS | `/api/ems/*` |
| SOS | SMS | `/api/sms/*` |
| FOS | FMS | `/api/fms/*` |
| GOS | GMS | `/api/gms/*` |
| OOS | OMS | `/api/oms/*` |

---

## 三、通用规范

| 项 | 规则 |
|:--:|:----|
| 认证 | JWT（含 tenant_id claim）|
| 版本 | URL 路径 v1 |
| 分页 | `?page=&size=`，默认 size=20 |
| 排序 | `?sort=created_at,desc` |
| 过滤 | `?status=active` |
| 响应 | 统一 `{ "code": 0, "data": {}, "msg": "" }` |

---

## 四、租户隔离

所有 SQL 自动追加 `WHERE tenant_id = ?`，由租户上下文管理器注入。

---

*LOCKED · 知味生态新全量文档集 A1.0 · 06_治理与运维/B*
