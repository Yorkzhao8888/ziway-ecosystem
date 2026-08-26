package api

import "github.com/gin-gonic/gin"

// ServerInterface 由 OpenAPI YAML 定义的所有端点契约。
// 业务侧实现本接口（见 internal/handler），即可保证与契约一致。
type ServerInterface interface {
	// EOS（供给运营，SC-001）
	ListEOSResources(c *gin.Context, tenantId string)
	CreateEOSResource(c *gin.Context, tenantId string)

	// EMS（供给管控，SC-001）
	ListEMSResources(c *gin.Context, tenantId string)
	CreateEMSResource(c *gin.Context, tenantId string)

	// COS（商业运营，SC-001）
	ListCOSResources(c *gin.Context, tenantId string)
	CreateCOSResource(c *gin.Context, tenantId string)

	// HOS（人力，SC-007）
	ListHOSResources(c *gin.Context, tenantId string)
	CreateHOSResource(c *gin.Context, tenantId string)

	// HMS（人资管控，SC-007）
	ListHMSResources(c *gin.Context, tenantId string)
	CreateHMSResource(c *gin.Context, tenantId string)

	// FOS/FMS（财务，SC-008）
	ListFOSResources(c *gin.Context, tenantId string)
	CreateFOSResource(c *gin.Context, tenantId string)
	ListFMSResources(c *gin.Context, tenantId string)
	CreateFMSResource(c *gin.Context, tenantId string)

	// GOS/GMS（风控，SC-009）
	ListGOSResources(c *gin.Context, tenantId string)
	CreateGOSResource(c *gin.Context, tenantId string)
	ListGMSResources(c *gin.Context, tenantId string)
	CreateGMSResource(c *gin.Context, tenantId string)

	// OOS/OMS（治理，SC-010）
	ListOOSResources(c *gin.Context, tenantId string)
	CreateOOSResource(c *gin.Context, tenantId string)
	ListOMSResources(c *gin.Context, tenantId string)
	CreateOMSResource(c *gin.Context, tenantId string)

	// VOS/VMS（价值运营，SC-010）
	ListVOSResources(c *gin.Context, tenantId string)
	CreateVOSResource(c *gin.Context, tenantId string)
	ListVMSResources(c *gin.Context, tenantId string)
	CreateVMSResource(c *gin.Context, tenantId string)

	// IOS/IMS（资本，SC-011）
	ListIOSResources(c *gin.Context, tenantId string)
	CreateIOSResource(c *gin.Context, tenantId string)
	ListIMSResources(c *gin.Context, tenantId string)
	CreateIMSResource(c *gin.Context, tenantId string)

	// AOS/AMS（系统，SC-012）
	ListAOSResources(c *gin.Context, tenantId string)
	CreateAOSResource(c *gin.Context, tenantId string)
	ListAMSResources(c *gin.Context, tenantId string)
	CreateAMSResource(c *gin.Context, tenantId string)

	// OAS（底座，SC-012/SC-013）
	ListOASResources(c *gin.Context, tenantId string)
	CreateOASResource(c *gin.Context, tenantId string)

	// XBU（事件总线，SC-013）
	ListXBUResources(c *gin.Context, tenantId string)
	CreateXBUResource(c *gin.Context, tenantId string)

	// 健康检查
	Healthz(c *gin.Context)
}

// RegisterRoutes 将所有路径按 OpenAPI 约定注册到 Gin 引擎。
// 路径前缀即所属 OS/MS（tenant_id 在 path 中，与 X-Tenant-Id 头双重校验）。
func RegisterRoutes(r *gin.Engine, h ServerInterface) {
	g := r.Group("")
	g.GET("/healthz", h.Healthz)

	pairs := []struct{ prefix, kind string }{
		{"/api/eos", "eos"}, {"/api/ems", "ems"}, {"/api/cos", "cos"},
		{"api/hos", "hos"}, {"/api/hms", "hms"}, {"/api/fos", "fos"},
		{"/api/fms", "fms"}, {"/api/gos", "gos"}, {"/api/gms", "gms"},
		{"/api/oos", "oos"}, {"/api/oms", "oms"}, {"/api/vos", "vos"},
		{"/api/vms", "vms"}, {"/api/ios", "ios"}, {"/api/ims", "ims"},
		{"/api/aos", "aos"}, {"/api/ams", "ams"}, {"/api/oas", "oas"},
		{"/api/xbu", "xbu"},
	}
	_ = pairs // 具体注册在各 handler 包的 InitRoutes（避免循环依赖）
}
