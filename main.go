package main

import (
	"github.com/gin-gonic/gin"
	"github.com/tiamxu/kairo/logic"
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
func newServer() {
	// e := httpkit.NewGin(cfg.HttpSrv)
	// logic.RegisterHttpRoute(e)
	// return httpkit.StartServer(e, cfg.HttpSrv)
	r := gin.Default()
	// r.Use(gin.Logger())
	// r.Use(gin.Recovery())
	logic.RegisterHttpRoute(r)
	r.Static("/static", "./static")

	r.Run(":8800")
}

func main() {

	// if err := modelService.Initialize(ctx, &cfg.LLMConfig, cfg.VectorStoreConfig); err != nil {
	// 	log.Fatalf("Model service initialization failed: %v", err)
	// }

	// 启动 HTTP 服务
	newServer()

}
