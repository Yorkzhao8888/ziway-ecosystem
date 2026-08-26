# ---- 构建阶段 ----
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ziway-api ./cmd/server

# ---- 运行阶段 ----
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /build/ziway-api .
EXPOSE 8080
CMD ["./ziway-api"]
