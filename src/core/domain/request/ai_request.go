package request

// SummarizeRequest AI 文件总结请求
type SummarizeRequest struct {
	FileID   string `json:"fileId" binding:"required"`
	FileName string `json:"fileName" binding:"required"`
	FileType string `json:"fileType" binding:"required"`
}

// ChatHistoryMessage 短期对话上下文（由前端传入，不落库）
type ChatHistoryMessage struct {
	Role    string `json:"role" binding:"required"`    // user | assistant
	Content string `json:"content" binding:"required"` // message content
}

// ChatRequest AI 文件问答请求
type ChatRequest struct {
	FileID   string `json:"fileId"`
	FileName string `json:"fileName"`
	FileType string `json:"fileType"`
	Message  string `json:"message" binding:"required"`
	History  []ChatHistoryMessage `json:"history"`
}

