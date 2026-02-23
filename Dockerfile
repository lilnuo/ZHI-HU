# syntax=docker/dockerfile:1.4
# Builder
FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://goproxy.cn,direct

# 利用缓存加速模块下载
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .
# main.go 位于仓库根目录，所以直接构建当前目录
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/go-zhihu .

# Final - minimal
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /out/go-zhihu /app/go-zhihu
# 拷贝配置（你的 main.go 使用 config/config.yaml）
COPY --from=builder /src/config /app/config
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/app/go-zhihu"]