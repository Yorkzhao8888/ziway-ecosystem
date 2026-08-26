BINARY := ziway-api
MIGRATIONS := 01_数据库迁移

.PHONY: gen run migrate build clean tidy

## 从 OpenAPI 重新生成代码（需先安装 oapi-codegen）
gen:
	oapi-codegen -config=gen.yaml ../../07_场景范例/00_OpenAPI.yaml > api/types.go

## 启动服务（依赖 docker-compose 已起 PG + Kafka）
run:
	go run ./cmd/server

## 执行数据库迁移（创建所有 schema + 表）
migrate:
	psql "$$DB_DSN" -f $(MIGRATIONS)/0001_init.up.sql

## 编译二进制
build: tidy
	go build -o bin/$(BINARY) ./cmd/server

tidy:
	go mod tidy

clean:
	rm -rf bin
