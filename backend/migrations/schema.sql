-- slink SQLite schema（幂等，启动时自动执行）
PRAGMA journal_mode = WAL;

-- 分组表
CREATE TABLE IF NOT EXISTS t_group (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    gid         TEXT    NOT NULL UNIQUE,              -- 分组标识（UUID）
    name        TEXT    NOT NULL,                     -- 分组名称
    sort_order  INTEGER NOT NULL DEFAULT 0,           -- 排序值，越小越靠前
    del_flag    INTEGER NOT NULL DEFAULT 0,           -- 删除标识 0:未删 1:已删
    created_at  DATETIME NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_group_sort ON t_group (sort_order);

-- 短链接表
-- deleted_at = 0 表示未删除；软删时写入删除时间戳，与 short_uri 组成唯一键，
-- 支持"回收站中的短码可被重建、恢复时若短码已被占用则报错"的语义
CREATE TABLE IF NOT EXISTS t_link (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    short_uri        TEXT    NOT NULL,                -- 短码
    origin_url       TEXT    NOT NULL,                -- 原始链接
    gid              TEXT    NOT NULL DEFAULT 'default', -- 所属分组
    describe         TEXT    DEFAULT '',              -- 描述
    valid_date_type  INTEGER NOT NULL DEFAULT 0,      -- 有效期类型 0:永久 1:自定义
    valid_date       DATETIME,                        -- 自定义有效期截止
    enable_status    INTEGER NOT NULL DEFAULT 0,      -- 启用标识 0:启用 1:禁用
    total_pv         INTEGER NOT NULL DEFAULT 0,      -- 历史PV
    total_uv         INTEGER NOT NULL DEFAULT 0,      -- 历史UV
    total_uip        INTEGER NOT NULL DEFAULT 0,      -- 历史UIP
    title            TEXT    DEFAULT '',              -- 原始链接标题
    created_at       DATETIME NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at       DATETIME NOT NULL DEFAULT (datetime('now','localtime')),
    deleted_at       INTEGER NOT NULL DEFAULT 0,      -- 删除时间戳(秒)，0=未删
    CONSTRAINT uk_link_uri UNIQUE (short_uri, deleted_at)
);
CREATE INDEX IF NOT EXISTS idx_link_gid ON t_link (gid, deleted_at);

-- 访问统计表（按天/小时聚合 PV/UV/UIP）
CREATE TABLE IF NOT EXISTS t_link_access_stats (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    short_uri   TEXT    NOT NULL,
    date        TEXT    NOT NULL,                     -- YYYY-MM-DD
    hour        INTEGER NOT NULL DEFAULT -1,          -- 0-23，聚合到天时为 -1
    weekday     INTEGER NOT NULL DEFAULT 0,           -- 1-7（周一为1）
    pv          INTEGER NOT NULL DEFAULT 0,
    uv          INTEGER NOT NULL DEFAULT 0,
    uip         INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT uk_access_stats UNIQUE (short_uri, date, hour)
);
CREATE INDEX IF NOT EXISTS idx_access_stats_date ON t_link_access_stats (short_uri, date);

-- 维度统计表（locale/browser/os/device 四个维度合并存储）
CREATE TABLE IF NOT EXISTS t_link_dim_stats (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    short_uri   TEXT    NOT NULL,
    date        TEXT    NOT NULL,                     -- YYYY-MM-DD
    dim         TEXT    NOT NULL,                     -- locale | browser | os | device
    value       TEXT    NOT NULL,                     -- 维度值（如 Chrome / macOS / 中国 广东 深圳）
    cnt         INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT uk_dim_stats UNIQUE (short_uri, date, dim, value)
);
CREATE INDEX IF NOT EXISTS idx_dim_stats ON t_link_dim_stats (short_uri, dim, date);

-- 访问明细日志表
CREATE TABLE IF NOT EXISTS t_link_access_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    short_uri   TEXT    NOT NULL,
    ip          TEXT    DEFAULT '',
    browser     TEXT    DEFAULT '',
    os          TEXT    DEFAULT '',
    device      TEXT    DEFAULT '',
    locale      TEXT    DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_access_logs_uri ON t_link_access_logs (short_uri, created_at DESC);

-- 系统设置表（key-value，存放修改后的密码哈希等）
CREATE TABLE IF NOT EXISTS t_settings (
    k           TEXT PRIMARY KEY,
    v           TEXT NOT NULL,
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now','localtime'))
);

-- 初始数据：默认分组
INSERT OR IGNORE INTO t_group (gid, name, sort_order) VALUES ('default', '默认分组', 0);
