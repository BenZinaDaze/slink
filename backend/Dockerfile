# ---- 前端构建（产物平台无关：跑在构建机原生架构，只构建一次，禁用 QEMU）----
FROM --platform=$BUILDPLATFORM node:22-alpine AS fe
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---- 后端构建（跑在构建机原生架构，GOARCH 交叉编译出目标平台二进制，禁用 QEMU）----
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS be
ARG TARGETARCH
ARG APP_VERSION=dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY backend/ ./backend/
COPY --from=fe /app/dist ./backend/web/dist
RUN cd backend && CGO_ENABLED=0 GOARCH=${TARGETARCH} go build \
    -ldflags "-s -w -X github.com/benzi/slink/backend/internal/version.Version=${APP_VERSION}" \
    -o /slink ./cmd/slink

# ---- ip2region 离线库 ----
FROM alpine:3.20 AS ipdb
RUN apk add --no-cache curl && \
    mkdir -p /data && \
    curl -L -o /data/ip2region_v4.xdb \
    https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region_v4.xdb

# ---- 运行镜像 ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=be /slink /app/slink
COPY backend/config.yaml /app/config.yaml
COPY --from=ipdb /data/ip2region_v4.xdb /app/data/ip2region.xdb
ENV TZ=Asia/Shanghai
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["/app/slink", "-c", "config.yaml"]
