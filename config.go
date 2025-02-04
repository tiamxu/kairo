package main

import (
	"github.com/sirupsen/logrus"
	"github.com/tiamxu/kit/llm"
	"github.com/tiamxu/kit/log"
	"github.com/tiamxu/kit/sql"

	"github.com/tiamxu/kit/vectorstore"

	"github.com/koding/multiconfig"
	httpkit "github.com/tiamxu/kit/http"
)

const configPath = "config/config.yaml"

// yaml文件内容映射到结构体
type Config struct {
	ENV               string                        `yaml:"env"`
	LogLevel          string                        `yaml:"log_level"`
	HttpSrv           httpkit.GinServerConfig       `yaml:"http_srv"`
	VectorStoreConfig vectorstore.VectorStoreConfig `yaml:"vector_store"`
	LLMConfig         llm.Config                    `yaml:"models"`
	DB                *sql.Config                   `yaml:"db" xml:"db" json:"db"`
}

// set log level
func (c *Config) Initial() (err error) {
	defer func() {
		if err == nil {
			log.Printf("config initialed, env: %s", cfg.ENV)
		}
	}()

	if level, err := logrus.ParseLevel(c.LogLevel); err != nil {
		return err
	} else {
		log.DefaultLogger().SetLevel(level)
	}

	if _, err := sql.Connect(cfg.DB); err != nil {
		return err
	}
	return nil
}

// 读取配置文件
func loadConfig() {
	cfg = new(Config)
	multiconfig.MustLoadWithPath(configPath, cfg)
}
