package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Cfg 全局配置（与 docker-compose 环境变量对齐）
type Cfg struct {
	Env   string
	Port  int
	DBDSN string
	KafkaBrokers string
	XBusTopic    string
}

// Load 加载配置，支持环境变量覆盖。
func Load() *Cfg {
	v := viper.New()
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("PORT", 8080)
	v.SetDefault("DB_DSN", "host=localhost user=ziway password=ziway123 dbname=ziway port=5432 sslmode=disable")
	v.SetDefault("KAFKA_BROKERS", "localhost:9092")
	v.SetDefault("XBUS_TOPIC", "ziway-xbus")
	v.AutomaticEnv()

	return &Cfg{
		Env:          v.GetString("APP_ENV"),
		Port:         v.GetInt("PORT"),
		DBDSN:        v.GetString("DB_DSN"),
		KafkaBrokers: v.GetString("KAFKA_BROKERS"),
		XBusTopic:    v.GetString("XBUS_TOPIC"),
	}
}

// DSN 格式化（供 database/sql 使用）
func (c *Cfg) String() string {
	return fmt.Sprintf("env=%s port=%d db=%s", c.Env, c.Port, c.DBDSN)
}
