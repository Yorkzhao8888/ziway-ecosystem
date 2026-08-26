// Package api 由 oapi-codegen 从 07_场景范例/00_OpenAPI.yaml 生成
// 不手改本文件（重新生成会覆盖）。业务实现见 internal/handler。
package api

import (
	"encoding/json"
	"time"
)

// 通用事件载荷（对应 YAML AlertTriggeredEvent 等统一 schema）
type Event struct {
	EventType  string          `json:"eventType"`
	EventID    string          `json:"eventId"`
	TenantID   string          `json:"tenantId"`
	OccurredAt time.Time       `json:"occurredAt"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

// 错误响应
type ErrorResponse struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details,omitempty"`
	TraceID string          `json:"traceId,omitempty"`
}

// 统一资源请求/响应（各 OS/MS 通用）
type Resource struct {
	ID       string          `json:"id"`
	TenantID string          `json:"tenantId"`
	Kind     string          `json:"kind"` // eos/ems/cos/hos/...
	Data     json.RawMessage `json:"data,omitempty"`
}

type ResourceList struct {
	Items []Resource `json:"items"`
	Total  int       `json:"total"`
}
