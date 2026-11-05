# slink · 简单高效的短链接系统

自部署的个人短链接服务：短链管理、分组、访问统计、二维码、监控看板——一个 Docker 命令跑起来，数据全在自己手里。

**不需要单独安装数据库**：内置 SQLite，整个服务只依赖一个 Redis。

## 功能一览

| 模块 | 能做什么 |
|---|---|
| 短链接 | 自定义后缀或自动生成短码、设置有效期、启用/停用、批量搜索 |
| 分组 | 按用途归组（如"推广"、"文档"），一键筛选 |
| 回收站 | 删除的短链可恢复，也允许用相同短码重建新链接 |
| 访问统计 | 每条短链的 PV / UV / 独立 IP、访问趋势、时段分布、访客地域/浏览器/系统/设备、访问明细 |
| 仪表盘 | 短链总数、今日访问、累计点击、近 30 天趋势、热门 Top 10 |
| 体验 | 中文界面、深色模式、⌘K 快速搜索、短链二维码一键复制 |

## 快速开始

### 方式一：Docker Compose（推荐）

```bash
git clone https://github.com/BenZinaDaze/slink.git
cd slink

cp .env.example .env    # 打开改一下 Redis 密码和初始账号
docker compose up -d --build
```

打开 http://localhost:8080 ，用 `.env` 里设置的账号登录即可。

### 方式二：直接使用已发布的镜像

```bash
docker network create slink
docker run -d --name slink-redis --network slink \
  redis:7-alpine redis-server --requirepass 换成你的密码

docker run -d --name slink --network slink -p 8080:8080 \
  -e SLINK_REDIS_ADDR=slink-redis:6379 \
  -e SLINK_REDIS_PASSWORD=换成你的密码 \
  -e SLINK_AUTH_USERNAME=admin \
  -e SLINK_AUTH_PASSWORD=换成初始密码 \
  -v slink-data:/app/data \
  benz1/slink:latest
```

### 首次登录后建议做的事

1. 进入「设置」页**修改初始密码**——使用初始密码期间会强制停留在设置页，改完才能使用其他功能
2. 创建第一条短链接：短链接页 → 新建短链接，粘贴长链接即可；标题可自动抓取
3. 点短码旁的二维码图标可展示/复制二维码
4. 生产环境建议配置[域名分离](#域名分离部署)：短链公开域与管理后台分开

## 配置

容器部署用 `.env`（或环境变量），二进制直跑改 `backend/config.yaml`。所有配置均可通过 `SLINK_` 前缀的环境变量覆盖。

| 配置 | 环境变量 | 默认值 | 说明 |
|---|---|---|---|
| Redis 密码 | `REDIS_PASSWORD` | `slink123` | compose 中 slink 与 Redis 共用 |
| 初始账号 | `SLINK_USERNAME` | `admin` | 仅首次启动生效，之后以「设置」页修改的为准 |
| 初始密码 | `SLINK_PASSWORD` | `admin123` | 同上 |
| 服务端口 | compose 里 `8080:8080` 左侧 | 8080 | 对外访问端口 |
| 短链域名 | `SLINK_SHORTLINK_DOMAIN` | 请求 Host | 公开短链域名，如 `https://s.example.com`。**配置后该域只提供跳转服务**（`/api` 与管理界面一律 404），管理后台需走其他域名访问，见「域名分离部署」 |
| 受信代理网段 | config `server.trusted_proxies` | 不信任 | 前置 Nginx/Caddy 时必填（如 `172.16.0.0/12`），否则客户端 IP 只能看到代理地址，限速与地域统计失真；留空时伪造 `X-Forwarded-For` 也无法绕过登录限速 |
| 目标域名白名单 | config `shortlink.whitelist` | 不限制 | 限制只能给指定域名生成短链（如防滥用） |

> 保留字：`login` `links` `recycle-bin` `settings` `assets` 等前端路由不能用作自定义短码，创建时会被拦截。

## 数据与备份

- 全部业务数据在**一个 SQLite 文件**里，日志等无需关心
  - Docker：名为 `slink-data` 的卷（容器内 `/app/data/slink.db`）
  - 源码直跑：`backend/data/slink.db`
- **备份**：定期复制该文件即可（建议停机或低峰期复制）
- **重置全部数据**：停止服务 → 删除该文件及同名 `-wal`/`-shm` 文件 → 重启（账号密码恢复为初始值）
- Redis 中的数据（缓存/登录态/未落库统计）丢失不影响业务数据，最多损失几秒内的统计数据

## 常见问题

**忘记密码怎么办？**
「设置」页改过的密码无法找回。两种处理：
- 完全重置：删除数据文件（见上）
- 保留数据：停止服务后，在数据目录执行
  `sqlite3 slink.db "DELETE FROM t_settings WHERE k='password_hash';"`
  然后重启，即可用 config/.env 里的初始密码登录

**如何启用 HTTPS？**
在前面挂一层反向代理（Nginx / Caddy / Traefik）终结 TLS，并把 `SLINK_SHORTLINK_DOMAIN` 设为 `https://你的域名`。跳转统计依赖每次点击回源，务必使用 HTTPS。反代记得同时设置受信代理网段（见配置表）。HSTS 也建议在这一层加：`Strict-Transport-Security: max-age=31536000`。

**域名分离部署**

短链域名是公开的，任何人都访问得到；管理后台（登录、短链管理）不该暴露在同一个域上。给 slink 配置 `SLINK_SHORTLINK_DOMAIN` 后，**该域只提供跳转服务**——`/api`、登录页、静态资源一律返回 404，扫描器探测不到任何管理面，即使服务裸暴露也一样。管理后台走另一个域名反代到同一容器即可。

Caddy 示例（`s.example.com` 公开跳转，`admin.example.com` 管理，按需给管理域加 IP 白名单或 VPN）：

```caddyfile
s.example.com {
	reverse_proxy 127.0.0.1:8080
}

admin.example.com {
	reverse_proxy 127.0.0.1:8080
	# 更收紧的做法：只允许自己的 IP 访问管理域
	# @blocked not remote_ip 1.2.3.4
	# respond @blocked 404
}
```

对应环境变量：

```bash
SLINK_SHORTLINK_DOMAIN=https://s.example.com
SLINK_SERVER_TRUSTED_PROXIES=127.0.0.1   # Caddy/Nginx 所在网段
```

**短链会失效吗？**
永久短链不会过期；设置了自定义有效期的到点后返回"已过期"页（410），可在回收站外重新编辑续期。删除进回收站的链接返回 404，可随时恢复。

**升级版本**
```bash
git pull && docker compose up -d --build
# 或使用镜像：docker compose pull && docker compose up -d
```
数据自动沿用，无需迁移。

> 从旧版本升级：登录凭证从浏览器 localStorage 迁移为 HttpOnly Cookie，升级后需**重新登录一次**；使用初始密码的实例会在登录后被引导至设置页完成修改。

## 开发者

架构设计、API 文档、本地开发与测试见 [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)。

## 许可

个人使用与学习自由部署。
