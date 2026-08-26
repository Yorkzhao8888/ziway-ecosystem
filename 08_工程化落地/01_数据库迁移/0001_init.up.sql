-- ============================================================================
-- 知味生态 A1.2 → A1.3 数据库初始化迁移
-- 基线：07 场景范例 / SC-001 供应商入驻运营（权威源 DDL）
-- 命名：所有业务表带 tenant_id（多租户隔离），与 OpenAPI X-Tenant-Id 对齐
-- 执行：psql -d ziway -f 0001_init.up.sql   （或 make migrate）
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 0. 命名空间（schema）
-- ----------------------------------------------------------------------------
CREATE SCHEMA IF NOT EXISTS iam;        -- OAS 身份基座
CREATE SCHEMA IF NOT EXISTS hos;        -- 人力运营 (HOS)
CREATE SCHEMA IF NOT EXISTS eos;        -- 供给运营 (EOS)
CREATE SCHEMA IF NOT EXISTS cos;        -- 商业运营 (COS)
CREATE SCHEMA IF NOT EXISTS fos;        -- 财务运营 (FOS)
CREATE SCHEMA IF NOT EXISTS fms;        -- 财务管控 (FMS)
CREATE SCHEMA IF NOT EXISTS vms;        -- 价值运营管控 (VMS)
CREATE SCHEMA IF NOT EXISTS logistics;  -- OAS 物流 (底座能力)
CREATE SCHEMA IF NOT EXISTS xbus;       -- 跨 OS 事件总线
CREATE SCHEMA IF NOT EXISTS datalake;   -- 数据湖 / 运营聚合

-- ----------------------------------------------------------------------------
-- 1. OAS-IAM：身份基座（全生态唯一权威源）
-- ----------------------------------------------------------------------------
CREATE TABLE iam.tenants (
    tenant_id   BIGSERIAL PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,        -- 张记农场
    root_role   CHAR(2)      NOT NULL,         -- 'OU'/'HU'/'EU'/'GU' 四根身份
    status      SMALLINT     NOT NULL DEFAULT 1, -- 1启用 0禁用
    created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenants_name UNIQUE (name),
    CONSTRAINT chk_root_role CHECK (root_role IN ('OU','HU','EU','GU'))
);

CREATE TABLE iam.users (
    user_id   BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES iam.tenants(tenant_id) ON DELETE CASCADE,
    name      VARCHAR(64) NOT NULL,
    hats      TEXT[]      NOT NULL DEFAULT '{}', -- 戴帽：{EU, DU} 可叠加
    iam_key   VARCHAR(64) UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_users_tenant ON iam.users(tenant_id);
CREATE INDEX idx_users_hats   ON iam.users USING GIN(hats);

-- 权限/角色（四根身份 + 派生角色的访问控制）
CREATE TABLE iam.permissions (
    role       CHAR(2)      PRIMARY KEY, -- OU/HU/EU/GU + 派生 CU/DU/PU/SU/VU/FU/IU/AU
    can_switch BOOLEAN      NOT NULL DEFAULT FALSE, -- 是否允许"戴帽切换"
    description VARCHAR(128)
);

-- ----------------------------------------------------------------------------
-- 2. HOS：人力运营（企业档案，本场景写入）
-- ----------------------------------------------------------------------------
CREATE TABLE hos.enterprises (
    tenant_id     BIGINT PRIMARY KEY REFERENCES iam.tenants(tenant_id) ON DELETE CASCADE,
    legal_person  VARCHAR(64),      -- 法人
    license_no    VARCHAR(64),      -- 营业执照
    business_scope TEXT,            -- 经营范围
    credit_level  SMALLINT DEFAULT 3, -- 信用评级 1-5
    audited_at    TIMESTAMP
);
CREATE INDEX idx_enterprises_license ON hos.enterprises(license_no);

-- ----------------------------------------------------------------------------
-- 3. EOS：供给运营（供给侧核心：店铺/商品/库存/供应方订单/评分）
-- ----------------------------------------------------------------------------
CREATE TABLE eos.shops (
    shop_id           BIGSERIAL PRIMARY KEY,
    tenant_id         BIGINT NOT NULL REFERENCES iam.tenants(tenant_id),
    name              VARCHAR(128) NOT NULL,
    status            SMALLINT NOT NULL DEFAULT 0, -- 0审核中 1营业 2冻结
    logistics_policy  JSONB,
    created_at        TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_shops_tenant ON eos.shops(tenant_id);
CREATE INDEX idx_shops_status ON eos.shops(status);

CREATE TABLE eos.products (
    product_id  BIGSERIAL PRIMARY KEY,
    shop_id     BIGINT NOT NULL REFERENCES eos.shops(shop_id) ON DELETE CASCADE,
    tenant_id   BIGINT NOT NULL REFERENCES iam.tenants(tenant_id),
    sku_code    VARCHAR(64),
    name        VARCHAR(128) NOT NULL,
    price       DECIMAL(12,2) NOT NULL DEFAULT 0,
    stock       INT NOT NULL DEFAULT 0,
    status      SMALLINT NOT NULL DEFAULT 0  -- 0待审 1在售
);
CREATE INDEX idx_products_shop ON eos.products(shop_id);
CREATE INDEX idx_products_tenant ON eos.products(tenant_id);
CREATE INDEX idx_products_status ON eos.products(status);

CREATE TABLE eos.inventory (
    product_id BIGINT PRIMARY KEY REFERENCES eos.products(product_id) ON DELETE CASCADE,
    stock      INT NOT NULL DEFAULT 0,
    locked     INT NOT NULL DEFAULT 0  -- 下单锁定库存
);

CREATE TABLE eos.supplier_orders (
    order_id    BIGSERIAL PRIMARY KEY,
    product_id  BIGINT NOT NULL REFERENCES eos.products(product_id),
    qty         INT NOT NULL,
    status      VARCHAR(16) NOT NULL DEFAULT 'created', -- created/confirmed/shipped/done
    waybill_no  VARCHAR(64),
    eta         TIMESTAMP,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_supplier_orders_status ON eos.supplier_orders(status);

CREATE TABLE eos.shop_ratings (
    shop_id    BIGINT PRIMARY KEY REFERENCES eos.shops(shop_id) ON DELETE CASCADE,
    rating_avg  DECIMAL(3,2) NOT NULL DEFAULT 0,
    review_cnt  INT NOT NULL DEFAULT 0
);

-- ----------------------------------------------------------------------------
-- 4. COS：商业运营（消费者订单 + 评价，主权威源）
-- ----------------------------------------------------------------------------
CREATE TABLE cos.orders (
    order_id            BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL,                 -- 消费者 CU
    supplier_tenant_id  BIGINT NOT NULL REFERENCES iam.tenants(tenant_id), -- 分发至 EOS 依据
    total               DECIMAL(12,2) NOT NULL DEFAULT 0,
    status              VARCHAR(16) NOT NULL DEFAULT 'created',
    created_at          TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_orders_supplier ON cos.orders(supplier_tenant_id);
CREATE INDEX idx_orders_status   ON cos.orders(status);

CREATE TABLE cos.reviews (
    review_id   BIGSERIAL PRIMARY KEY,
    order_id    BIGINT UNIQUE NOT NULL REFERENCES cos.orders(order_id) ON DELETE CASCADE,
    shop_id     BIGINT NOT NULL REFERENCES eos.shops(shop_id),
    rating      SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment     TEXT
);

-- ----------------------------------------------------------------------------
-- 5. FMS：财务管控（结算单，生成净结算额）+ FOS 运营视图
-- ----------------------------------------------------------------------------
CREATE TABLE fms.settlements (
    settlement_id BIGSERIAL PRIMARY KEY,
    tenant_id     BIGINT NOT NULL REFERENCES iam.tenants(tenant_id),
    period        DATE   NOT NULL,                        -- T+1
    gross_amount  DECIMAL(14,2) NOT NULL DEFAULT 0,       -- 订单金额
    commission    DECIMAL(14,2) NOT NULL DEFAULT 0,       -- 平台佣金
    logistics_fee DECIMAL(14,2) NOT NULL DEFAULT 0,
    refund        DECIMAL(14,2) NOT NULL DEFAULT 0,
    net_amount    DECIMAL(14,2) GENERATED ALWAYS AS (gross_amount - commission - logistics_fee - refund) STORED,
    status        VARCHAR(16) NOT NULL DEFAULT 'pending'  -- pending/paid
);
CREATE INDEX idx_settlements_tenant ON fms.settlements(tenant_id);

-- FOS 运营视图（基于 FMS 结算单，供 Market APP 提现）
CREATE VIEW fos.settlement_view AS
SELECT s.settlement_id, s.tenant_id, s.period, s.net_amount, s.status,
       t.name AS tenant_name
  FROM fms.settlements s
  JOIN iam.tenants t ON t.tenant_id = s.tenant_id
 WHERE s.status = 'pending';

-- ----------------------------------------------------------------------------
-- 6. 物流（OAS 底座能力）
-- ----------------------------------------------------------------------------
CREATE TABLE logistics.waybills (
    waybill_no  VARCHAR(64) PRIMARY KEY,
    order_id    BIGINT NOT NULL REFERENCES cos.orders(order_id),
    carrier     VARCHAR(64),
    eta         TIMESTAMP,
    status      VARCHAR(16) DEFAULT 'created'
);

-- ----------------------------------------------------------------------------
-- 7. XBUS：跨 OS 事件总线（幂等 + 死信）
-- ----------------------------------------------------------------------------
CREATE TABLE xbus.outbox (
    event_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type   VARCHAR(64)  NOT NULL,  -- 如 SupplierRegistered / ProductApproved
    tenant_id    VARCHAR(64)  NOT NULL,
    payload      JSONB         NOT NULL DEFAULT '{}'::jsonb,
    occurred_at  TIMESTAMP     NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP
);
CREATE INDEX idx_outbox_unpublished ON xbus.outbox(published_at) WHERE published_at IS NULL;

CREATE TABLE xbus.idempotency (
    -- 幂等键 = event_type + 业务 ID（约定由生产者保证唯一）
    idempotency_key VARCHAR(128) PRIMARY KEY,
    handled_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE xbus.dlq (
    event_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original   JSONB NOT NULL,
    error_msg  TEXT,
    enqueued_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- ----------------------------------------------------------------------------
-- 8. 数据湖（VMS 聚合，供 VOS/Case 看板）
-- ----------------------------------------------------------------------------
CREATE TABLE datalake.shop_daily_metrics (
    shop_id    BIGINT,
    stat_date  DATE,
    orders     INT NOT NULL DEFAULT 0,
    gmv        DECIMAL(14,2) NOT NULL DEFAULT 0,
    rating_avg DECIMAL(3,2),
    PRIMARY KEY (shop_id, stat_date)
);

COMMIT;

-- ============================================================================
-- 回滚：DROP SCHEMA ... CASCADE（见 0001_init.down.sql）
-- ============================================================================
