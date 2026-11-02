.PHONY: dev dev-be dev-fe build test build-fe build-be ipdb up down clean

# 开发模式：后端(热重载可选) + 前端 vite dev server
dev:
	@echo "终端1: make dev-be   终端2: make dev-fe"
	@$(MAKE) --no-print-directory dev-be

dev-be:
	cd backend && go run ./cmd/slink -c config.yaml

dev-fe:
	cd frontend && npm run dev

# 构建单二进制：前端产物 → 内嵌 → go build
# 可选注入版本号：make build VERSION=$(git describe --tags --always)
VERSION ?= dev

build: build-fe build-be
	@ls -lh backend/slink | awk '{print "✓ 单二进制: backend/slink ("$$5")"}'

build-fe:
	cd frontend && npm run build
	rm -rf backend/web/dist
	cp -r frontend/dist backend/web/dist

build-be:
	cd backend && go build \
		-ldflags "-s -w -X github.com/benzi/slink/backend/internal/version.Version=$(VERSION)" \
		-o slink ./cmd/slink

# 下载 ip2region 离线库（地域统计用，缺省自动降级为"未知"）
# 注意：运行时按数据目录下的 ip2region.xdb 查找，文件名不能带 _v4 后缀
ipdb:
	mkdir -p backend/data
	curl -L -o backend/data/ip2region.xdb \
		https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region_v4.xdb
	@ls -lh backend/data/ip2region.xdb

test:
	go test ./backend/...

up:
	docker compose up -d --build

down:
	docker compose down

clean:
	rm -f backend/slink
	rm -rf frontend/dist backend/web/dist
