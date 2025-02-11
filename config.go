package main

import (
	"fmt"
	"os"

	"github.com/tiamxu/kairo/logic/model"

	"github.com/tiamxu/kit/llm"
	"github.com/tiamxu/kit/log"
	"github.com/tiamxu/kit/sql"

	"github.com/tiamxu/kit/vectorstore"

	"github.com/koding/multiconfig"
	httpkit "github.com/tiamxu/kit/http"
)

var configPath = "config/config.yaml"

// yaml文件内容映射到结构体
type Config struct {
	ENV               string                         `yaml:"env"`
	LogLevel          string                         `yaml:"log_level"`
	HttpSrv           httpkit.GinServerConfig        `yaml:"http_srv"`
	VectorStoreConfig *vectorstore.VectorStoreConfig `yaml:"vector_store"`
	LLMConfig         *llm.Config                    `yaml:"models"`
	DB                *sql.Config                    `yaml:"db" xml:"db" json:"db"`
}

// set log level
func (c *Config) Initial() (err error) {

	defer func() {
		if err == nil {
			log.Printf("config initialed, env: %s", cfg.ENV)
		}
	}()
	//日志
	// if level, err := logrus.ParseLevel(c.LogLevel); err != nil {
	// 	return err
	// } else {
	// 	log.DefaultLogger().SetLevel(level)
	// }
	err = log.InitLogger(&log.Config{
		Level:      "info",
		Type:       "stdout", // "file" 或 "stdout"
		Format:     "json",   // "json" 或 "text"
		FilePath:   "logs",
		FileName:   "kairo.log",
		MaxSize:    100, // 每个文件最大 100MB
		MaxAge:     7,   // 保留7天
		MaxBackups: 10,  // 保留10个备份
		Compress:   true,
	})
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	log.SetGlobalFields(log.Fields{
		"appname": "kairo",
		// "version": "1.0.0",
	})

	if err = model.Init(cfg.DB); err != nil {
		return fmt.Errorf("database initialization failed: %w", err)

	}

	return nil
}

// 读取配置文件
func loadConfig() {
	cfg = new(Config)

	// env := os.Getenv("ENV")
	env := "local"

	switch env {
	case "dev":
		configPath = "config/config-dev.yaml"
	case "test":
		configPath = "config/config-test.yaml"
	case "prod":
		configPath = "config/config-prod.yaml"
	default:
		configPath = "config/config.yaml"
	}

	multiconfig.MustLoadWithPath(configPath, cfg)
}
