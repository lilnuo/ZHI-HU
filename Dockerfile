FROM golang:alpine AS builder

WORKDIR /build

ENV GOPROXY=https://goproxy.cn,direct
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0  GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o go-zhihu .

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

WORKDIR /app

COPY  --from=builder /build/go-zhihu .

COPY  --from=builder /build/config ./config

EXPOSE 8080

ENTRYPOINT ["./go-zhihu"]