package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ziway/api"
	"ziway/internal/config"
	"ziway/internal/handler"
	"ziway/internal/xbus"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	log.Printf("[ziway] starting env=%s", cfg.Env)

	// 基础设施
	bus := xbus.NewPublisher()
	defer bus.Close()

	// 业务 handler（注入依赖）
	h := handler.NewSC001Handler(bus)

	// Gin 引擎
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "ver": "A1.3"})
	})

	// 注册 SC-001 路由（按 OpenAPI 路径约定）
	registerSC001(r, h)

	// 优雅关停
	srv := &http.Server{Addr: ":8080", Handler: r}
	go func() {
		log.Printf("[ziway] listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[ziway] shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// registerSC001 按 OpenAPI 约定挂载 SC-001 端点（tenantId 在 path 中）
func registerSC001(r *gin.Engine, h *handler.SC001Handler) {
	// HOS：①注册
	r.POST("/api/hos/:tenantId/resources", func(c *gin.Context) {
		h.RegisterEnterprise(c, c.Param("tenantId"))
	})
	// EOS：②④商品/店铺 / ⑦订单状态
	r.POST("/api/eos/:tenantId/resources", func(c *gin.Context) {
		h.CreateProduct(c, c.Param("tenantId"))
	})
	r.PUT("/api/eos/:tenantId/resources", func(c *gin.Context) {
		h.UpsertShop(c, c.Param("tenantId"))
	})
	r.PATCH("/api/eos/:tenantId/resources", func(c *gin.Context) {
		h.UpdateOrder(c, c.Param("tenantId"))
	})
	// COS：⑤下单 / ⑧评价
	r.POST("/api/cos/:tenantId/resources", func(c *gin.Context) {
		h.CreateOrder(c, c.Param("tenantId"))
	})
	r.POST("/api/cos/:tenantId/reviews", func(c *gin.Context) {
		h.Review(c, c.Param("tenantId"))
	})
	// FOS：⑩提现
	r.POST("/api/fos/:tenantId/withdrawals", func(c *gin.Context) {
		h.Withdraw(c, c.Param("tenantId"))
	})
	// VOS：⑫看板
	r.GET("/api/vos/:tenantId/resources", func(c *gin.Context) {
		h.Metrics(c, c.Param("tenantId"))
	})
	// EOS：②入驻申请（独立路径，语义更清晰）
	r.POST("/api/eos/:tenantId/apply", func(c *gin.Context) {
		h.ApplyMarket(c, c.Param("tenantId"))
	})

	// 满足 ServerInterface 编译期断言（其余端点桩实现见 handler 包扩展）
	var _ api.ServerInterface = h
}
