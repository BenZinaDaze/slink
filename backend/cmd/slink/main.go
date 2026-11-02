package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/benzi/slink/backend/internal/app"
	"github.com/benzi/slink/backend/internal/config"
	"github.com/benzi/slink/backend/internal/version"
)

func main() {
	cfgPath := flag.String("c", "config.yaml", "配置文件路径")
	flag.Parse()
	log.Printf("slink %s 启动中...", version.Get())

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("初始化失败: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := application.Run(ctx); err != nil {
		log.Fatalf("运行异常: %v", err)
	}
	log.Println("slink 已退出")
}
