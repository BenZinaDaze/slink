// Package app 应用装配：初始化各组件、注册路由、管理生命周期
package app

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	slink "github.com/benzi/slink/backend"
	"github.com/benzi/slink/backend/internal/config"
	"github.com/benzi/slink/backend/internal/handler"
	"github.com/benzi/slink/backend/internal/middleware"
	"github.com/benzi/slink/backend/internal/repo"
	"github.com/benzi/slink/backend/internal/service"
	"github.com/benzi/slink/backend/internal/stats"
	"github.com/benzi/slink/backend/pkg/bloom"
	"github.com/benzi/slink/backend/pkg/shortgen"
)

type App struct {
	cfg      *config.Config
	engine   *gin.Engine
	rdb      *redis.Client
	consumer *stats.Consumer
	srv      *http.Server
}

// New 装配所有组件
func New(cfg *config.Config) (*App, error) {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 存储
	db, err := repo.Open(cfg.Database.Path)
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	// 仓储与服务
	settingRepo := repo.NewSettingRepo(db)
	groupRepo := repo.NewGroupRepo(db)
	linkRepo := repo.NewLinkRepo(db)
	statsRepo := repo.NewStatsRepo(db)

	authSvc := service.NewAuthService(cfg, rdb, settingRepo)
	groupSvc := service.NewGroupService(groupRepo)

	bf := bloom.New(rdb, "bloom:short:uri")
	cache := service.NewLinkCache(rdb, cfg)
	gen := shortgen.New(1)
	linkSvc := service.NewLinkService(cfg, linkRepo, groupRepo, gen, bf, cache)
	redirectSvc := service.NewRedirectService(linkRepo, bf, cache)
	events := service.NewEventProducer(rdb, cfg.Stats.Stream)
	statsSvc := service.NewStatsService(linkRepo, statsRepo)
	dashSvc := service.NewDashboardService(linkRepo, statsRepo)

	// 启动时重建布隆过滤器（全量短码回灌）
	if uris, err := linkRepo.AllActiveURIs(); err != nil {
		return nil, err
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		for _, u := range uris {
			_ = bf.Add(ctx, u)
		}
		cancel()
		log.Printf("[app] 布隆过滤器已加载 %d 个短码", len(uris))
	}

	// 统计消费者
	locale := stats.NewLocaleResolver(ip2regionPath(cfg))
	consumer := stats.NewConsumer(cfg, rdb, statsRepo, linkRepo, locale)

	// 路由
	engine := buildRouter(cfg, authSvc, groupSvc, linkSvc, redirectSvc, events, statsSvc, dashSvc)

	return &App{
		cfg: cfg, engine: engine, rdb: rdb, consumer: consumer,
	}, nil
}

// ip2regionPath xdb 离线库路径（默认数据目录，缺省自动降级）
func ip2regionPath(cfg *config.Config) string {
	return filepath.Join(filepath.Dir(cfg.Database.Path), "ip2region.xdb")
}

func buildRouter(
	cfg *config.Config,
	authSvc *service.AuthService,
	groupSvc *service.GroupService,
	linkSvc *service.LinkService,
	redirectSvc *service.RedirectService,
	events *service.EventProducer,
	statsSvc *service.StatsService,
	dashSvc *service.DashboardService,
) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/health"}, // 健康检查不打日志
	}), gin.Recovery(), middleware.RequestID(), middleware.SecurityHeaders())
	// 受信反代列表决定 ClientIP 是否采信 X-Forwarded-For（默认空 = 不信任，杜绝伪造头绕过限速）
	if err := engine.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		log.Printf("[app] trusted_proxies 配置无效: %v", err)
	}

	authH := handler.NewAuthHandler(authSvc)
	groupH := handler.NewGroupHandler(groupSvc)
	linkH := handler.NewLinkHandler(linkSvc)
	redirectH := handler.NewRedirectHandler(redirectSvc, events)
	statsH := handler.NewStatsHandler(statsSvc, dashSvc)

	engine.GET("/health", func(c *gin.Context) { c.String(200, "ok") })

	api := engine.Group("/api/v1")
	api.Use(middleware.APIGate(cfg)) // 短链域下管理 API 按路由不存在处理
	api.POST("/auth/login", authH.Login)
	// 登录态接口
	authed := api.Group("", middleware.Auth(authSvc))
	authed.GET("/auth/me", authH.Me)
	authed.DELETE("/auth/logout", authH.Logout)
	authed.PUT("/auth/account", authH.ChangeAccount)
	authed.GET("/dashboard", statsH.Dashboard)

	authed.GET("/groups", groupH.List)
	authed.POST("/groups", groupH.Create)
	authed.PUT("/groups/sort", groupH.Sort)
	authed.PUT("/groups/:gid", groupH.Update)
	authed.DELETE("/groups/:gid", groupH.Delete)

	authed.GET("/links", linkH.Page)
	authed.POST("/links", linkH.Create)
	authed.GET("/links/title", linkH.FetchTitle)
	authed.GET("/links/:id", linkH.GetByID)
	authed.PUT("/links/:id", linkH.Update)
	authed.DELETE("/links/:id", linkH.Delete)
	authed.GET("/links/:id/stats", statsH.Overview)
	authed.GET("/links/:id/stats/hourly", statsH.Hourly)
	authed.GET("/links/:id/stats/weekday", statsH.Weekday)
	authed.GET("/links/:id/stats/dim", statsH.Dim)
	authed.GET("/links/:id/stats/records", statsH.Records)

	authed.GET("/recycle-bin", linkH.PageRecycle)
	authed.POST("/recycle-bin/:id/restore", linkH.Restore)

	// 兜底：短码跳转 or SPA
	engine.NoRoute(redirectH.Handle, spaFallback(cfg))

	return engine
}

// spaFallback SPA 静态资源服务：内嵌 web/dist（go:embed），
// 精确命中文件则直出，否则回退 index.html（前端路由接管）
func spaFallback(cfg *config.Config) gin.HandlerFunc {
	webFS := slink.WebFS()
	fileServer := http.FileServer(http.FS(webFS))
	shortDomain := cfg.ShortLink.NormalizedDomain()
	return func(c *gin.Context) {
		// 短链域不服务任何管理面：SPA 与静态资源一律按不存在的短链处理
		if middleware.IsShortDomain(c, shortDomain) {
			handler.ErrorPage(c, http.StatusNotFound, "短链接不存在", "该短链接不存在或已被删除")
			c.Abort()
			return
		}
		// API 未匹配 → 404 JSON
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(404, gin.H{"code": "B0002", "message": "接口不存在", "success": false})
			return
		}
		p := strings.TrimPrefix(c.Request.URL.Path, "/")
		if p != "" {
			if st, err := fs.Stat(webFS, p); err == nil && !st.IsDir() {
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		// 前端产物未嵌入（仓库里 web/dist 只有 .gitkeep）时给出构建提示
		if _, err := fs.Stat(webFS, "index.html"); err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"code": "B0003",
				"message": "前端资源未构建：请在项目根目录执行 make build 生成单二进制，或使用 make dev 进入开发模式",
				"success": false,
			})
			return
		}
		c.Header("Cache-Control", "no-cache")
		http.ServeFileFS(c.Writer, c.Request, webFS, "index.html")
	}
}

// Run 启动 HTTP 服务与统计消费者，阻塞直至 ctx 取消
func (a *App) Run(ctx context.Context) error {
	go func() {
		if err := a.consumer.Run(ctx); err != nil {
			log.Printf("[app] 统计消费者退出: %v", err)
		}
	}()

	a.srv = &http.Server{
		Addr:    ":" + strconv.Itoa(a.cfg.Server.Port),
		Handler: a.engine,
	}
	go func() {
		log.Printf("[app] slink 已启动 :%d (mode=%s)", a.cfg.Server.Port, a.cfg.Server.Mode)
		if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[app] HTTP 服务异常: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = a.srv.Shutdown(shutdownCtx)
	_ = a.rdb.Close()
	return nil
}
