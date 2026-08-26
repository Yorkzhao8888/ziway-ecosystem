package xbus

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// Event 跨 OS 事件（与 YAML XBusEvent schema 对齐：幂等键 = Type+ID）
type Event struct {
	Type      string          `json:"eventType"`
	ID        string          `json:"eventId"`
	TenantID  string          `json:"tenantId"`
	OccurredAt string         `json:"occurredAt"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// Publisher 事件生产者（写入 outbox + 投递 Kafka）
type Publisher struct {
	writer *kafka.Writer
	topic  string
}

// NewPublisher 从环境变量构造；未配置 Kafka 时返回 nil-safe（仅日志）。
func NewPublisher() *Publisher {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		return &Publisher{} // noop
	}
	return &Publisher{
		topic: os.Getenv("XBUS_TOPIC"),
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers),
			Topic:    os.Getenv("XBUS_TOPIC"),
			Balancer: &kafka.Hash{},
		},
	}
}

// Publish 发布事件。幂等键 = eventType + eventID（由调用方保证 ID 唯一）。
func (p *Publisher) Publish(ctx context.Context, ev Event) error {
	if p.writer == nil {
		fmt.Printf("[xbus][noop] %s %s\n", ev.Type, ev.ID)
		return nil
	}
	if ev.ID == "" {
		ev.ID = uuid.New().String()
	}
	data, _ := json.Marshal(ev)
	return p.writer.WriteMessages(ctx, kafka.Message{Value: data})
}

// Close 释放资源。
func (p *Publisher) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
