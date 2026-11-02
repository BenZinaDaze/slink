# slink 开发文档

面向开发者与贡献者：架构设计、API、本地开发、测试与发布。使用与部署请看 [README](../README.md)。

## 总体架构

```
浏览器 ──► slink (Go/Gin 单二进制 :8080)
            ├── GET /{code}     302 跳转（热路径）
            ├── /api/v1/*       管理 API（token 鉴权）
            └── /*              内嵌 React SPA
           ├── SQLite (data/slink.db, WAL)   业务数据 + 统计聚合
           └── Redis                           跳转缓存 / 布隆过滤器 / token / UV 去重 / Stream 统计管道
```

**后端 Golang 单体单二进制 · 前端 React + TailwindCSS + shadcn/ui · SQLite（嵌入式，零 SQL 服务）· Redis**

## 技术要点

- **跳转热路径**：布隆过滤器（Redis bitmap，0.1% 误判）拦截不存在的短码 → 缓存命中直出（TTL 跟随有效期）→ 空值缓存防穿透 → singleflight 合并并发回源；创建即缓存预热。多实例部署时 singleflight 可平滑替换为 Redis `SET NX` 分布式锁
- **短码生成**：秒级雪花 ID（时间戳+节点+序列，46 位）经 Base62 编码为 7-8 位短码，趋势递增、单节点无冲突（`pkg/shortgen`）
- **统计管道**：跳转事件非阻塞写入 Redis Stream → 消费者组批量拉取（50 条/秒级）→ UA 解析（mssola/useragent）+ ip2region 地域 + Redis SET 去重（UV/UIP）→ 聚合 upsert 落库 → Ack；重启自动重投 pending 消息（`internal/stats`）
- **回收站语义**：`UNIQUE(short_uri, deleted_at)`（删除时间戳参与唯一键），软删后可重建同码、恢复时冲突检测
- **单二进制**：go:embed 内嵌前端产物与建表脚本（`backend/assets.go`），`make build` 产出约 33MB 二进制

### 认证机制

采用**不透明随机 token + 服务端会话态**（UUID 存 Redis，值格式 `username#凭证版本号`），而非 JWT：可随时吊销、无需管理签名密钥。配套安全设计：

- 密码 bcrypt 哈希存储；登录失败限速 10 次/分/IP
- 登录失败统一文案（防账号枚举），且两种失败路径均执行 bcrypt 比较（防时序探测）
- token 携带凭证版本号，修改用户名/密码后版本自增 → 全部旧会话立即下线，当前会话凭返回的新 token 无缝续期

## API 概览

统一响应 `{code, message, data, success, requestId}`，鉴权头 `Authorization: Bearer <token>`。

```
POST   /api/v1/auth/login            登录（限速 10 次/分/IP）
GET    /api/v1/auth/me               当前账号信息
DELETE /api/v1/auth/logout           登出
PUT    /api/v1/auth/account          修改用户名/密码（需原密码，吊销全部旧会话并返回新 token）

GET    /api/v1/groups               分组列表（含计数）
POST   /api/v1/groups               新增分组
PUT    /api/v1/groups/{gid}         修改分组
PUT    /api/v1/groups/sort          分组排序
DELETE /api/v1/groups/{gid}         删除分组（组内有链禁止）

GET    /api/v1/links?page&size&gid&key&order   分页
POST   /api/v1/links                创建（originUrl/gid/validDateType/validDate/customUri/describe/title）
PUT    /api/v1/links/{id}           修改（含启用/停用）
GET    /api/v1/links/{id}           详情
DELETE /api/v1/links/{id}           软删除 → 回收站
GET    /api/v1/links/title?url=     抓取目标页标题（3s 超时，禁内网防 SSRF）

GET    /api/v1/recycle-bin          回收站分页
POST   /api/v1/recycle-bin/{id}/restore   恢复（短码被占用时报错）

GET    /api/v1/links/{id}/stats?start&end       汇总 + 按天趋势（自动补零）
GET    /api/v1/links/{id}/stats/hourly|weekday  时段分布
GET    /api/v1/links/{id}/stats/dim?dim=        维度分布（locale/browser/os/device）
GET    /api/v1/links/{id}/stats/records?page=   访问明细分页

GET    /api/v1/dashboard            仪表盘汇总
GET    /{code}                      302 跳转（404 不存在 / 410 过期 / 403 停用）
```

## 目录结构

```
backend/
├── cmd/slink/main.go        入口
├── internal/
│   ├── app/                 装配与路由、SPA 兜底
│   ├── config/              viper 配置（环境变量覆盖，含单测）
│   ├── handler/             HTTP 层（auth/group/link/stats/redirect）
│   ├── service/             业务（跳转热路径、缓存、singleflight、账号）
│   ├── repo/                GORM/SQLite 数据访问
│   ├── stats/               Stream 消费者、UA 解析、ip2region
│   ├── middleware/          鉴权 / requestId
│   ├── version/             构建版本注入
│   └── model/ dto/
├── pkg/                     result / errcode / shortgen / bloom
├── migrations/schema.sql    建表（启动自动执行，幂等）
├── web/dist/                前端产物（go:embed）
└── config.yaml
frontend/
├── src/pages/               login / dashboard / links / link-stats / recycle-bin / settings
├── src/components/          布局、命令面板、创建编辑弹窗、二维码 + shadcn/ui
└── src/api/                 类型化 API 封装
docs/ · Makefile · docker-compose.yml · backend/Dockerfile · .github/workflows/
```

## 本地开发

依赖：Go 1.27+、Node 22+、一个可达的 Redis（默认 `127.0.0.1:6379`，用 `SLINK_REDIS_ADDR` 覆盖）。

```bash
# 终端1：后端（:8080）
make dev-be

# 终端2：前端（:5173，/api 代理到 :8080，热更新）
make dev-fe
```

默认账号 `admin / admin123`（`backend/config.yaml`）。

## 构建与发布

```bash
make test                                   # 单测（短码/标题提取/UA 解析/配置覆盖）
make ipdb                                   # 可选：下载 ip2region 离线库（缺失时地域降级为"未知"）
make build                                  # 前端构建 → go:embed → backend/slink 单二进制
make build VERSION=$(git describe --tags)   # 注入版本号
```

### 已知坑（改代码前必读）

- **GORM 列名**：三个连续大写字母的字段（`UIP`/`GID`）会被转成 `ui_p`/`gi_d` 列名，任何从 SQL 扫描的结构体必须加显式 `gorm:"column:..."` tag
- **版本注入**：版本变量放在 `internal/version` 包，不要挪回 main 包（本机 Go 1.27 下 main 包变量 `-X` 不生效）
- **shadcn（radix-vega 预设）**：`CommandDialog` 不内嵌 `<Command>` 根组件需手动包裹；Sidebar 的 Tooltip 需应用层包 `TooltipProvider`；`CardHeader` 是 grid 布局，固定宽度元素要 `justify-self-center`
- **统计事件发布**：必须 `context.WithoutCancel` 脱离请求 ctx，否则 handler 返回后异步 XAdd 被取消

### CI / 镜像发布

`.github/workflows/docker-publish.yml`：推送 `v*` tag 触发双平台（amd64/arm64）构建并推送 Docker Hub（`environment: Dockerhub` 读取 `DOCKERHUB_USERNAME`/`DOCKERHUB_TOKEN`），registry 缓存 `buildcache` tag；PR 仅构建验证。
