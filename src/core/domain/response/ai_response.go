package response

// SummarizeResponse AI 文件总结响应
type SummarizeResponse struct {
	Summary string `json:"summary"`
}

// ChatResponse AI 文件问答响应
type ChatResponse struct {
	Reply string `json:"reply"`
}
