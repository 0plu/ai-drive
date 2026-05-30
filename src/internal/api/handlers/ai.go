package handlers

import (
	"encoding/json"
	"fmt"
	"myobj/src/core/domain/request"
	"myobj/src/core/domain/response"
	serverai "myobj/src/core/server/ai"
	"myobj/src/internal/api/middleware"
	"myobj/src/internal/repository/impl"
	"myobj/src/pkg/cache"
	"myobj/src/pkg/logger"
	"myobj/src/pkg/models"

	"github.com/gin-gonic/gin"
)

type AiHandler struct {
	summarizer *serverai.Summarizer
	chater     *serverai.Chater
	factory    *impl.RepositoryFactory
	cache      cache.Cache
}

func NewAiHandler(factory *impl.RepositoryFactory, cacheLocal cache.Cache) *AiHandler {
	return &AiHandler{
		summarizer: serverai.NewSummarizer(factory),
		chater:     serverai.NewChater(factory),
		factory:    factory,
		cache:      cacheLocal,
	}
}

func (h *AiHandler) Router(c *gin.RouterGroup) {
	aiGroup := c.Group("/ai")
	auth := middleware.NewAuthMiddleware(
		h.cache,
		h.factory.ApiKey(),
		h.factory.User(),
		h.factory.GroupPower(),
		h.factory.Power(),
	)
	aiGroup.Use(auth.Verify())
	{
		aiGroup.POST("/summarize", h.Summarize)
		aiGroup.POST("/summarize/stream", h.SummarizeStream)
		aiGroup.POST("/chat", h.Chat)
		aiGroup.POST("/chat/stream", h.ChatStream)
	}
	logger.LOG.Info("[路由] AI 路由注册完成✔️")
}

func setupSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(200)
}

func writeSSEData(c *gin.Context, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

func writeSSEError(c *gin.Context, err error) {
	data, _ := json.Marshal(map[string]string{"error": err.Error()})
	_, _ = fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", data)
	c.Writer.Flush()
}

func writeSSEDone(c *gin.Context) {
	_, _ = fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
}

// Summarize godoc
// @Summary AI 文件总结
// @Description 对文件内容进行 AI 智能总结
// @Tags AI
// @Accept json
// @Produce json
// @Param request body request.SummarizeRequest true "总结请求"
// @Success 200 {object} models.JsonResponse{data=response.SummarizeResponse} "总结成功"
// @Failure 400 {object} models.JsonResponse "参数错误"
// @Failure 500 {object} models.JsonResponse "总结失败"
// @Router /ai/summarize [post]
func (h *AiHandler) Summarize(c *gin.Context) {
	req := new(request.SummarizeRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(200, models.NewJsonResponse(400, "参数错误", err.Error()))
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(200, models.NewJsonResponse(401, "未授权", nil))
		return
	}

	summary, err := h.summarizer.Summarize(userID.(string), req.FileID, req.FileName, req.FileType)
	if err != nil {
		c.JSON(200, models.NewJsonResponse(500, "总结失败", err.Error()))
		return
	}

	c.JSON(200, models.NewJsonResponse(200, "success", response.SummarizeResponse{
		Summary: summary,
	}))
}

func (h *AiHandler) SummarizeStream(c *gin.Context) {
	setupSSE(c)

	req := new(request.SummarizeRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		writeSSEError(c, err)
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		writeSSEError(c, fmt.Errorf("unauthorized"))
		return
	}

	err := h.summarizer.SummarizeStream(
		c.Request.Context(),
		userID.(string),
		req.FileID,
		req.FileName,
		req.FileType,
		func(content string) error {
			return writeSSEData(c, map[string]string{"content": content})
		},
	)
	if err != nil {
		writeSSEError(c, err)
		return
	}
	writeSSEDone(c)
}

// Chat godoc
// @Summary AI 文件问答
// @Description 基于文件内容进行 AI 智能问答
// @Tags AI
// @Accept json
// @Produce json
// @Param request body request.ChatRequest true "问答请求"
// @Success 200 {object} models.JsonResponse{data=response.ChatResponse} "问答成功"
// @Failure 400 {object} models.JsonResponse "参数错误"
// @Failure 500 {object} models.JsonResponse "问答失败"
// @Router /ai/chat [post]
func (h *AiHandler) Chat(c *gin.Context) {
	req := new(request.ChatRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(200, models.NewJsonResponse(400, "参数错误", err.Error()))
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(200, models.NewJsonResponse(401, "未授权", nil))
		return
	}

	reply, err := h.chater.Chat(userID.(string), req.FileID, req.FileName, req.FileType, req.Message)
	if err != nil {
		c.JSON(200, models.NewJsonResponse(500, "问答失败", err.Error()))
		return
	}

	c.JSON(200, models.NewJsonResponse(200, "success", response.ChatResponse{
		Reply: reply,
	}))
}

func (h *AiHandler) ChatStream(c *gin.Context) {
	setupSSE(c)

	req := new(request.ChatRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		writeSSEError(c, err)
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		writeSSEError(c, fmt.Errorf("unauthorized"))
		return
	}

	history := make([]serverai.ChatHistoryMessage, 0, len(req.History))
	for _, item := range req.History {
		history = append(history, serverai.ChatHistoryMessage{
			Role:    item.Role,
			Content: item.Content,
		})
	}

	err := h.chater.ChatStream(
		c.Request.Context(),
		userID.(string),
		req.FileID,
		req.FileName,
		req.FileType,
		req.Message,
		history,
		func(content string) error {
			return writeSSEData(c, map[string]string{"content": content})
		},
	)
	if err != nil {
		writeSSEError(c, err)
		return
	}
	writeSSEDone(c)
}
