package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"myobj/src/config"
	aiparser "myobj/src/core/service/ai"
	"myobj/src/internal/repository/impl"
	"myobj/src/pkg/logger"
	"net/http"
	"time"
)

const summarizeTimeout = 60 * time.Second

// Summarizer AI 文件总结器
type Summarizer struct {
	factory *impl.RepositoryFactory
}

func NewSummarizer(factory *impl.RepositoryFactory) *Summarizer {
	return &Summarizer{factory: factory}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

// buildPrompt 构建精炼文件总结 Prompt
func buildPrompt(content string) string {
	return fmt.Sprintf(`你是一个文件分析助手。请基于文件内容生成精炼摘要。

输出要求：
1. 总字数控制在 180 字以内。
2. 最多输出 4 个要点。
3. 优先提炼：主题、关键内容、技术栈/模块、风险或待办。
4. 不要复述大段原文，不要输出完整代码。
5. 如果信息不足，只说明能确定的内容。
6. 使用简洁中文，适合在右侧 AI 面板中快速阅读。

输出格式：
### 摘要
- ...
- ...
- ...

文件内容：
%s`, content)
}

// callAI 调用 OpenAI Compatible API
func callAI(prompt string) (string, error) {
	cfg := config.CONFIG.AI
	if !cfg.Enable {
		return "", fmt.Errorf("AI 服务未启用")
	}
	if cfg.Endpoint == "" || cfg.ApiKey == "" {
		return "", fmt.Errorf("AI 服务配置不完整")
	}

	body := chatRequest{
		Model: cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: "你是一个文件分析助手，输出必须精炼。"},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		chatCompletionsEndpoint(),
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ApiKey)

	client, err := newHTTPClient(summarizeTimeout)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 AI 接口失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI 接口返回异常: %s", string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("AI 接口未返回有效结果")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// Summarize 执行文件总结流程
func (s *Summarizer) Summarize(userID, fileID, fileName, fileType string) (string, error) {
	fileInfo, err := resolveUserFileInfo(context.Background(), s.factory, userID, fileID)
	if err != nil {
		logger.LOG.Error("获取文件信息失败", "fileID", fileID, "error", err)
		return "", fmt.Errorf("文件不存在或无法访问")
	}

	if fileInfo.Size == 0 {
		return "", fmt.Errorf("文件内容为空")
	}

	content, err := aiparser.ParseFileContent(fileInfo.Path, fileType)
	if err != nil {
		logger.LOG.Error("解析文件内容失败", "fileID", fileID, "error", err)
		return "", err
	}
	if content == "" {
		return "", fmt.Errorf("文件内容为空")
	}

	prompt := buildPrompt(content)

	logger.LOG.Info("正在调用 AI 接口进行文件总结", "fileID", fileID, "fileName", fileName)
	summary, err := callAI(prompt)
	if err != nil {
		return "", fmt.Errorf("AI 总结失败: %w", err)
	}

	return summary, nil
}

func (s *Summarizer) SummarizeStream(ctx context.Context, userID, fileID, fileName, fileType string, onDelta func(string) error) error {
	fileInfo, err := resolveUserFileInfo(ctx, s.factory, userID, fileID)
	if err != nil {
		logger.LOG.Error("获取文件信息失败", "fileID", fileID, "error", err)
		return fmt.Errorf("文件不存在或无法访问")
	}
	if fileInfo.Size == 0 {
		return fmt.Errorf("文件内容为空")
	}

	content, err := aiparser.ParseFileContent(fileInfo.Path, fileType)
	if err != nil {
		logger.LOG.Error("解析文件内容失败", "fileID", fileID, "error", err)
		return err
	}
	if content == "" {
		return fmt.Errorf("文件内容为空")
	}

	logger.LOG.Info("streaming AI summary", "fileID", fileID, "fileName", fileName)
	return callChatAIStream(ctx, "你是一个文件分析助手，输出必须精炼。", buildPrompt(content), onDelta)
}
