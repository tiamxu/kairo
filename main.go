package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tiamxu/kairo/logic/api"
	"github.com/tiamxu/kairo/logic/service"
	httpkit "github.com/tiamxu/kit/http"
	"github.com/tiamxu/kit/log"
)

var (
	cfg *Config
	// name = "kairo"
)

func init() {
	loadConfig()
	if err := cfg.Initial(); err != nil {
		log.Fatalf("Config initialization failed: %v", err)
	}
}

func main() {
	// 初始化 service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	modelService, err := service.NewLLMService(ctx, cfg.LLMConfig, cfg.VectorStoreConfig, cfg.DB)
	if err != nil {
		log.Fatalf("Model service initialization failed: %v", err)
	}

	// 创建 handler
	handler := api.NewHandler(modelService)

	// 创建 gin 路由
	router := httpkit.NewGin(cfg.HttpSrv)
	// 添加中间件
	router.Use(httpkit.TimeoutMiddleware(30 * time.Second))
	router.Use(httpkit.RateLimitMiddleware(100, time.Minute))

	// 设置路由
	router.POST("/api/query", handler.QueryHandler)
	router.POST("/api/store", handler.StoreQAHandler)
	router.GET("/api/questions", handler.GetQuestionsHandler)
	// router.Static("/static", "./static")
	// 启动服务器
	srv := httpkit.StartServer(router, cfg.HttpSrv)
	log.Infoln("Server listen: ", cfg.HttpSrv.Address)

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Infoln("Shutting down server...")
	httpkit.ShutdownServer(srv)
	log.Infoln("Server exited")
}
