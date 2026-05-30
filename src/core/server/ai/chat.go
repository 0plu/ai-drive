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
	"strings"
	"time"
)

const chatTimeout = 120 * time.Second

// Chater AI 文件问答器
type Chater struct {
	factory *impl.RepositoryFactory
}

func NewChater(factory *impl.RepositoryFactory) *Chater {
	return &Chater{factory: factory}
}

type chatRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatHistoryMessage is short-term chat context from the client (no DB persistence).
// Role must be "user" or "assistant".
type ChatHistoryMessage struct {
	Role    string
	Content string
}

type chatRequestBody struct {
	Model       string               `json:"model"`
	Messages    []chatRequestMessage `json:"messages"`
	Temperature float64              `json:"temperature"`
	Stream      bool                 `json:"stream,omitempty"`
}

type chatResponseChoice struct {
	Message chatRequestMessage `json:"message"`
}

type chatResponseBody struct {
	Choices []chatResponseChoice `json:"choices"`
}

// buildChatPrompt 构建问答 Prompt
func buildChatPrompt(content, message string) string {
	return fmt.Sprintf(`你是一个文件问答助手。

你必须基于以下文件内容回答问题。

如果答案不在文件中，请明确说明：
"文件中未提及该内容"。

文件内容：
%s

用户问题：
%s`, content, message)
}

// callChatAI 调用 OpenAI Compatible API 进行问答
// 后续可扩展为流式 SSE 模式：
// 设置 stream=true 并通过 Server-Sent Events 逐字返回
func callChatAI(prompt string) (string, error) {
	cfg := config.CONFIG.AI
	if !cfg.Enable {
		return "", fmt.Errorf("AI 服务未启用")
	}
	if cfg.Endpoint == "" || cfg.ApiKey == "" {
		return "", fmt.Errorf("AI 服务配置不完整")
	}

	body := chatRequestBody{
		Model: cfg.Model,
		Messages: []chatRequestMessage{
			{Role: "system", Content: "你是文件问答助手，严格基于文件内容回答。"},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		Stream:      false,
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

	client, err := newHTTPClient(chatTimeout)
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

	var chatResp chatResponseBody
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("AI 接口未返回有效结果")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// Chat 执行文件问答流程
// fileID: 文件 ID
// fileName: 文件名
// fileType: 文件类型（扩展名或 MIME）
// message: 用户提问内容
func (c *Chater) Chat(userID, fileID, fileName, fileType, message string) (string, error) {
	if fileID == "" {
		logger.LOG.Info("姝ｅ湪璋冪敤 AI 鎺ュ彛杩涜閫氱敤瀵硅瘽", "userID", userID)
		reply, err := callChatAI(message)
		if err != nil {
			return "", fmt.Errorf("AI 闂瓟澶辫触: %w", err)
		}
		return reply, nil
	}

	// 步骤1：根据 fileId 获取真实文件路径
	fileInfo, err := resolveUserFileInfo(context.Background(), c.factory, userID, fileID)
	if err != nil {
		logger.LOG.Error("获取文件信息失败", "fileID", fileID, "error", err)
		return "", fmt.Errorf("文件不存在或无法访问")
	}

	if fileInfo.Size == 0 {
		return "", fmt.Errorf("文件内容为空")
	}

	// 步骤2：调用 ParseFileContent 解析文件内容
	content, err := aiparser.ParseFileContent(fileInfo.Path, fileType)
	if err != nil {
		logger.LOG.Error("解析文件内容失败", "fileID", fileID, "error", err)
		return "", err
	}
	if content == "" {
		return "", fmt.Errorf("文件内容为空")
	}

	// 步骤3：拼接 Prompt
	prompt := buildChatPrompt(content, message)

	// 步骤4：调用 AI 接口
	logger.LOG.Info("正在调用 AI 接口进行文件问答", "fileID", fileID, "fileName", fileName)
	reply, err := callChatAI(prompt)
	if err != nil {
		return "", fmt.Errorf("AI 问答失败: %w", err)
	}

	return reply, nil
}

func (c *Chater) ChatStream(ctx context.Context, userID, fileID, fileName, fileType, message string, history []ChatHistoryMessage, onDelta func(string) error) error {
	if fileID == "" {
		logger.LOG.Info("streaming general AI chat", "userID", userID)
		messages := []chatRequestMessage{
			{Role: "system", Content: "You are the MyObj cloud drive assistant."},
		}
		messages = appendHistoryMessages(messages, pickRecentHistory(history, 5))
		messages = append(messages, chatRequestMessage{Role: "user", Content: message})
		return callChatAIStreamMessages(ctx, messages, onDelta)
	}

	fileInfo, err := resolveUserFileInfo(ctx, c.factory, userID, fileID)
	if err != nil {
		logger.LOG.Error("鑾峰彇鏂囦欢淇℃伅澶辫触", "fileID", fileID, "error", err)
		return fmt.Errorf("鏂囦欢涓嶅瓨鍦ㄦ垨鏃犳硶璁块棶")
	}
	if fileInfo.Size == 0 {
		return fmt.Errorf("鏂囦欢鍐呭涓虹┖")
	}

	content, err := aiparser.ParseFileContent(fileInfo.Path, fileType)
	if err != nil {
		logger.LOG.Error("瑙ｆ瀽鏂囦欢鍐呭澶辫触", "fileID", fileID, "error", err)
		return err
	}
	if content == "" {
		return fmt.Errorf("鏂囦欢鍐呭涓虹┖")
	}

	logger.LOG.Info("streaming AI file chat", "fileID", fileID, "fileName", fileName)
	systemPrompt := `你是一个文件问答助手。
你必须优先基于“文件内容”回答问题，可以参考“历史对话”理解上下文。
如果文件中没有相关信息，请明确回答“文件中未提及该内容”，不要编造。`

	messages := []chatRequestMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: "文件内容：\n" + content},
	}
	messages = appendHistoryMessages(messages, pickRecentHistory(history, 5))
	messages = append(messages, chatRequestMessage{Role: "user", Content: message})

	return callChatAIStreamMessages(ctx, messages, onDelta)
}

func pickRecentHistory(history []ChatHistoryMessage, maxRounds int) []ChatHistoryMessage {
	if maxRounds <= 0 || len(history) == 0 {
		return nil
	}
	limit := maxRounds * 2
	if limit <= 0 {
		return nil
	}
	if len(history) <= limit {
		return history
	}
	return history[len(history)-limit:]
}

func appendHistoryMessages(dst []chatRequestMessage, history []ChatHistoryMessage) []chatRequestMessage {
	for _, h := range history {
		role := strings.ToLower(strings.TrimSpace(h.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(h.Content)
		if content == "" {
			continue
		}
		dst = append(dst, chatRequestMessage{Role: role, Content: content})
	}
	return dst
}
