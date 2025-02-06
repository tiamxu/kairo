package api

import (
	"context"
	"net/http"
	"time"

	"github.com/tiamxu/kairo/logic/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	modelService *service.Service
}

func NewHandler(service *service.Service) *Handler {
	return &Handler{
		modelService: service,
	}
}

type QueryRequest struct {
	Query string `json:"query" binding:"required"`
	TopK  int    `json:"top_k,omitempty"`
}

type QAPairRequest struct {
	Question string `json:"question" binding:"required"`
	Answer   string `json:"answer" binding:"required"`
}

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (h *Handler) Query(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: "无效的请求体"})
		return
	}

	if req.TopK <= 0 {
		req.TopK = 5 // 默认值
	} else if req.TopK > 20 {
		req.TopK = 20 // 限制最大值
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	answer, err := h.modelService.QueryWithRetrieve(ctx, req.Query, req.TopK)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			c.JSON(http.StatusGatewayTimeout, Response{Success: false, Error: "请求超时"})
			return
		}
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, Response{Success: true, Data: answer})
}

func (h *Handler) StoreQA(c *gin.Context) {
	var req QAPairRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{Success: false, Error: "无效的请求体"})
		return
	}

	err := h.modelService.StoreQA(c.Request.Context(), req.Question, req.Answer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, Response{Success: true})
}

func (h *Handler) GetQuestions(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	questions, err := h.modelService.GetStoredQuestions(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, Response{Success: true, Data: questions})
}
