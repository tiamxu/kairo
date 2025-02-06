package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tiamxu/kairo/logic"
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
func newServer() *http.Server {
	e := httpkit.NewGin(cfg.HttpSrv)
	logic.RegisterHttpRoute(e)
	return httpkit.StartServer(e, cfg.HttpSrv)
}

func main() {
	// 初始化服务
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := logic.NewLLMService(ctx, &cfg.LLMConfig, cfg.VectorStoreConfig)
	if err != nil {
		log.Fatalf("Model service initialization failed: %v", err)

	}
	// if err := modelService.Initialize(ctx, &cfg.LLMConfig, cfg.VectorStoreConfig); err != nil {
	// 	log.Fatalf("Model service initialization failed: %v", err)
	// }

	// 启动 HTTP 服务
	srv := newServer()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	httpkit.ShutdownServer(srv)

	log.Println("Server exiting")

}
