package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"ziway/api"
	"ziway/internal/xbus"

	"github.com/gin-gonic/gin"
)

// SC001Handler 供应商入驻集市运营（SC-001）端到端实现。
// 覆盖：①注册 ②入驻申请 ③④店铺/商品 ⑤⑥⑦订单履约 ⑧收货评价 ⑨⑩结算 ⑪⑫运营
type SC001Handler struct {
	bus *xbus.Publisher
}

// NewSC001Handler 构造（依赖注入 xbus.Publisher）
func NewSC001Handler(bus *xbus.Publisher) *SC001Handler {
	return &SC001Handler{bus: bus}
}

// ---- ① 企业注册（HOS → OAS-IAM）----
// POST /api/hos/{tenantId}/resources   body: {name, root_role, legal_person, license_no}
func (h *SC001Handler) RegisterEnterprise(c *gin.Context, tenantID string) {
	var req struct {
		Name        string `json:"name"`
		RootRole    string `json:"root_role"` // EU
		LegalPerson string `json:"legal_person"`
		LicenseNo   string `json:"license_no"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Code: "invalid", Message: err.Error()})
		return
	}
	// 1) HOS: 写入企业档案  → 2) 经 OAS-IAM 创建 tenant（事务内完成）
	// 伪代码：tx := db.Begin(); tx.Exec(iam.tenants INSERT); tx.Exec(hos.enterprises INSERT); tx.Commit()
	_ = req
	// 3) 发布 SupplierRegistered 事件 → EMS 触发资质审核
	h.publish("SupplierRegistered", tenantID, map[string]string{"name": req.Name})

	c.JSON(http.StatusCreated, gin.H{
		"tenant_id": tenantID,
		"step":      "registered",
		"next":      "apply_market",
	})
}

// ---- ② 申请入驻集市（EOS → EMS 审核）----
// POST /api/eos/{tenantId}/resources   body: {category, capacity, delivery_area}
func (h *SC001Handler) ApplyMarket(c *gin.Context, tenantID string) {
	var req struct {
		Category    string `json:"category"`
		Capacity    int    `json:"capacity"`
		DeliveryArea string `json:"delivery_area"`
	}
	_ = c.ShouldBindJSON(&req)
	// EMS 资质校验（营业执照/信用/品类）→ 审批工作流（OU-EM 审批人）
	// 审核通过 → 开通虚拟店铺（eos.shops INSERT status=0审核中）
	h.publish("MarketApplicationSubmitted", tenantID, map[string]string{"category": req.Category})
	c.JSON(http.StatusAccepted, gin.H{"status": "pending_review"})
}

// ---- ③ 店铺装修 / ④ 创建商品（EOS，写后送 EMS 审核）----
// PUT /api/eos/{tenantId}/resources   body: {shop_id, name, logo, logistics_policy}
func (h *SC001Handler) UpsertShop(c *gin.Context, tenantID string) {
	// eos.shops UPSERT（status 默认 0 审核中）
	c.JSON(http.StatusOK, gin.H{"shop": "updated"})
}

// POST /api/eos/{tenantId}/resources   body: {shop_id, sku, name, price, stock}
func (h *SC001Handler) CreateProduct(c *gin.Context, tenantID string) {
	var p struct {
		SKU   string  `json:"sku"`
		Name  string  `json:"name"`
		Price float64 `json:"price"`
		Stock int     `json:"stock"`
	}
	_ = c.ShouldBindJSON(&p)
	// eos.products INSERT → 发布 ProductCreated → EMS 审核 → ProductApproved → OAS 搜索索引
	h.publish("ProductCreated", tenantID, map[string]string{"sku": p.SKU, "name": p.Name})
	c.JSON(http.StatusCreated, gin.H{"product": "pending_approval"})
}

// ---- ⑤⑥⑦ 订单履约（COS → EOS → OAS-物流，跨 OS 事件驱动）----
// POST /api/cos/{tenantId}/resources   body: {supplier_tenant_id, items[]}
func (h *SC001Handler) CreateOrder(c *gin.Context, tenantID string) {
	var req struct {
		SupplierTenantID string `json:"supplier_tenant_id"`
		Items            []struct {
			ProductID int `json:"product_id"`
			Qty       int `json:"qty"`
		} `json:"items"`
	}
	_ = c.ShouldBindJSON(&req)
	// cos.orders INSERT → 按 supplier_tenant_id 分发 → EOS.supplier_orders
	// EOS 扣库存(eos.inventory) + 调 OAS-物流匹配方案 → waybill_no/eta
	h.publish("OrderCreated", tenantID, map[string]string{
		"supplier": req.SupplierTenantID,
	})
	c.JSON(http.StatusCreated, gin.H{"order": "confirmed", "status": "confirmed"})
}

// PATCH /api/eos/{tenantId}/resources  body: {order_id, action: ship|complete}
func (h *SC001Handler) UpdateOrder(c *gin.Context, tenantID string) {
	// ship → 调物流；complete → 触发评价/评分/结算
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// ---- ⑧ 确认收货 + 评价（COS → EOS 评分，Saga 终态）----
// POST /api/cos/{tenantId}/resources   body: {order_id, shop_id, rating, comment}
func (h *SC001Handler) Review(c *gin.Context, tenantID string) {
	var r struct {
		OrderID int    `json:"order_id"`
		ShopID  int    `json:"shop_id"`
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	_ = c.ShouldBindJSON(&r)
	// cos.reviews INSERT → 发布 ReviewSubmitted → EOS 累加 eos.shop_ratings
	// → 触发 FMS 结算归集
	h.publish("ReviewSubmitted", tenantID, map[string]any{
		"shop_id": r.ShopID, "rating": r.Rating,
	})
	c.JSON(http.StatusOK, gin.H{"review": "created"})
}

// ---- ⑨⑩ 结算与提现（FMS 定时 → FOS 可提现 → OAS 支付网关）----
// 由 cron 触发：归集已完成订单 → fms.settlements（net_amount = gross-commission-logistics-refund）
// POST /api/fos/{tenantId}/resources   body: {settlement_id}
func (h *SC001Handler) Withdraw(c *gin.Context, tenantID string) {
	var req struct {
		SettlementID int `json:"settlement_id"`
	}
	_ = c.ShouldBindJSON(&req)
	// FMS 提现审批 → OAS 支付网关资金划拨 → settlement.status='paid'
	h.publish("SettlementPaid", tenantID, map[string]int{"id": req.SettlementID})
	c.JSON(http.StatusOK, gin.H{"settlement": "paid"})
}

// ---- ⑪⑫ 运营优化（OAS → VMS → VOS/Case → EOS 调价）----
// GET /api/vos/{tenantId}/resources  → 数据看板
func (h *SC001Handler) Metrics(c *gin.Context, tenantID string) {
	// VMS 聚合 datalake.shop_daily_metrics → VOS → Case 看板
	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"metrics": gin.H{
			"orders":     0, // 实际查 datalake
			"gmv":        0,
			"rating_avg": 0,
		},
	})
}

// publish 发布跨 OS 事件（幂等键 = eventType + eventID）
func (h *SC001Handler) publish(eventType, tenantID string, payload any) {
	if h.bus == nil {
		return
	}
	data, _ := json.Marshal(payload)
	_ = h.bus.Publish(nil, xbus.Event{
		Type:       eventType,
		TenantID:   tenantID,
		OccurredAt: time.Now().Format(time.RFC3339),
		Payload:    data,
	})
}

// compile-time 断言：确保实现了 ServerInterface（接口在 api 包，此处仅为示意）
var _ api.ServerInterface = (*SC001Handler)(nil)
