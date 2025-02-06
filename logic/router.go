package logic

import (
	"github.com/gin-gonic/gin"
	"github.com/tiamxu/kairo/logic/api"
)

func RegisterHttpRoute(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})
	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	r.POST("/api/query", api.Query)
	r.POST("/api/store", api.StoreQA)
	r.POST("/api/questions", api.GetQuestions)
}
