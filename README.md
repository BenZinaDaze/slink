# slink · 简单高效的短链接系统

单用户个人版短链接平台。

**后端 Golang 单体单二进制 · 前端 React + TailwindCSS + shadcn/ui · SQLite（零 SQL 服务）· Redis**

```
浏览器 ──► slink (Go/Gin 单二进制 :8080)
            ├── GET /{code}     302 跳转（热路径）
            ├── /api/v1/*       管理 API（token 鉴权）
            └── /*              内嵌 React SPA
           ├── SQLite (data/slink.db, WAL)   业务数据 + 统计聚合
           └── Redis                           跳转缓存 / 布隆过滤器 / token / UV 去重 / Stream 统计管道
```

## 功能清单

| 模块 | 功能 |
|---|---|
| 认证 | 单用户登录（config 配置账号）、修改密码（bcrypt）、登出踢下线、登录限速 |
| 分组 | 增删改查、排序、每组短链计数（默认分组不可删） |
| 短链接 | 创建（自定义后缀 / 雪花+Base62 自动短码）、编辑、分页搜索、标题自动抓取、启用/停用、永久/自定义有效期 |
| 回收站 | 软删除 → 回收站 → 恢复（同码冲突检测）→ 可重建同码 |
| 跳转 | 302 重定向；布隆过滤器 → 缓存 → 空值缓存 → singleflight 回源；404/410(过期)/403(停用) 页面 |
| 监控 | PV/UV/UIP 汇总、按天趋势、小时/星期分布、地域(ip2region)/浏览器/OS/设备分布、访问明细分页 |
| 仪表盘 | 短链总数、今日 PV/UV、累计点击、近 30 天趋势、Top10 短链 |
| 管理台 | shadcn 管理面板（可折叠侧边栏、⌘K 命令面板、深色模式、二维码、中文界面） |

## 技术要点

- **跳转热路径**：布隆过滤器（Redis bitmap，0.1% 误判）拦截不存在的短码 → 缓存命中直出（TTL 跟随有效期）→ 空值缓存防穿透 → singleflight 合并并发回源；创建即缓存预热
- **短码生成**：秒级雪花 ID（时间戳+节点+序列，46 位）经 Base62 编码为 7-8 位短码，趋势递增、单节点无冲突
- **统计管道**：跳转事件非阻塞写入 Redis Stream → 消费者组批量拉取 → UA 解析 + ip2region 地域 + Redis SET 去重（UV/UIP）→ 聚合 upsert 落库 → Ack；重启自动重投 pending 消息
- **回收站语义**：`UNIQUE(short_uri, deleted_at)`（删除时间戳参与唯一键），软删后可重建同码、恢复时冲突检测
- **单二进制**：go:embed 内嵌前端产物与建表脚本，33MB 二进制 + Redis 即完整服务

### 认证机制

采用**不透明随机 token + 服务端会话态**（UUID 存 Redis，30 天 TTL），而非 JWT：可随时吊销（登出/改密即失效）、无需管理签名密钥；代价是每次校验多一次 Redis 读，单机部署可忽略。配套安全设计：

- 密码 bcrypt 哈希存储；登录失败限速 10 次/分/IP
- 登录失败统一返回"用户名或密码错误"（防账号枚举），且两种失败路径均执行 bcrypt 比较（防时序探测）
- token 携带凭证版本号，修改用户名/密码后版本自增 → **全部旧会话立即下线**，当前会话凭返回的新 token 无缝续期
- 生产部署请置于 HTTPS 反向代理之后（`shortlink.domain` 同步配置）

## 快速开始

### 依赖

- Go 1.27+、Node 22+（构建）；运行期仅需 Redis

### 本地开发

```bash
# 终端1：后端（读取 backend/config.yaml）
make dev-be

# 终端2：前端（vite :5173，/api 代理到 :8080）
make dev-fe
```

打开 http://localhost:5173 ，默认账号 `admin / admin123`（见 `backend/config.yaml`）。

### 单二进制部署

```bash
make ipdb    # 可选：下载 ip2region 离线库（地域统计，缺失自动降级）
make build   # 前端构建 → go:embed → backend/slink 单二进制
cd backend && ./slink -c config.yaml
```

SQLite 数据文件在 `backend/data/slink.db`。

### Docker Compose

```bash
cp .env.example .env   # 按需修改 Redis 密码与初始账号（不复制则用默认值）
docker compose up -d --build
```

- **Redis 已启用密码**（`--requirepass`，默认 `slink123`，`.env` 中 `REDIS_PASSWORD` 覆盖），slink 通过环境变量自动接入
- slink 等 Redis 健康检查通过后才启动；初始账号同样可用 `.env` 的 `SLINK_USERNAME` / `SLINK_PASSWORD` 覆盖
- 数据持久化：SQLite 与 ip2region 库在 `slink-data` 卷，Redis 数据在 `redis-data` 卷

## 配置（`backend/config.yaml`，环境变量 `SLINK_*` 覆盖）

| 配置 | 说明 |
|---|---|
| `server.port / mode` | 端口 / debug·release |
| `auth.username / password / token_ttl` | 登录账号；密码为初始值，改密后存 SQLite |
| `database.path` | SQLite 文件路径（ip2region.xdb 放同目录生效） |
| `redis.addr` | Redis 地址 |
| `shortlink.domain / whitelist` | 短链域名（默认请求 Host）/ 目标域名白名单（空不限制） |
| `stats.*` | Stream / 消费者组 / 批量参数 |

## API 概览（统一响应 `{code, message, data, success, requestId}`）

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

GET    /api/v1/links?page&size&gid&key   分页（gid/关键词/排序）
POST   /api/v1/links                创建（originUrl/gid/validDate*/customUri/describe/title）
PUT    /api/v1/links/{id}           修改（含启用/停用）
GET    /api/v1/links/{id}           详情
DELETE /api/v1/links/{id}           软删除 → 回收站
GET    /api/v1/links/title?url=     抓取目标页标题

GET    /api/v1/recycle-bin          回收站分页
POST   /api/v1/recycle-bin/{id}/restore   恢复

GET    /api/v1/links/{id}/stats?start&end      汇总 + 按天趋势
GET    /api/v1/links/{id}/stats/hourly|weekday 时段分布
GET    /api/v1/links/{id}/stats/dim?dim=      维度分布（locale/browser/os/device）
GET    /api/v1/links/{id}/stats/records       访问明细

GET    /api/v1/dashboard            仪表盘
GET    /{code}                      302 跳转（404/410/403）
```

> 保留字：`login/links/recycle-bin/settings/assets/api/dashboard/link-error` 不作为短码解析（SPA 路由），创建自定义后缀时同样被拦截。

## 目录结构

```
backend/
├── cmd/slink/main.go        入口
├── internal/
│   ├── app/                 装配与路由、SPA 兜底
│   ├── config/              viper 配置
│   ├── handler/             HTTP 层（auth/group/link/stats/redirect）
│   ├── service/             业务（含跳转热路径、缓存、singleflight）
│   ├── repo/                GORM/SQLite 数据访问
│   ├── stats/               Stream 消费者、UA 解析、ip2region
│   ├── middleware/          鉴权 / requestId / 限速
│   └── model/ dto/
├── pkg/                     result / errcode / shortgen / bloom
├── migrations/schema.sql    建表（启动自动执行）
├── web/dist/                前端产物（go:embed）
└── config.yaml
frontend/                    Vite + React + TS + shadcn/ui
Makefile · docker-compose.yml · backend/Dockerfile
```

## 测试

```bash
make test    # 短码生成/Base62、标题提取、UA 解析单测
```
