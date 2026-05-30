package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"myobj/src/config"
	"net/http"
	"strings"
	"time"
)

const streamTimeout = 5 * time.Minute

type streamRequestBody struct {
	Model       string               `json:"model"`
	Messages    []chatRequestMessage `json:"messages"`
	Temperature float64              `json:"temperature"`
	Stream      bool                 `json:"stream"`
}

type streamDelta struct {
	Content string `json:"content"`
}

type streamChoice struct {
	Delta streamDelta `json:"delta"`
}

type streamResponseBody struct {
	Choices []streamChoice `json:"choices"`
}

func callChatAIStreamMessages(ctx context.Context, messages []chatRequestMessage, onDelta func(string) error) error {
	ctx, cancel := context.WithTimeout(ctx, streamTimeout)
	defer cancel()

	cfg := config.CONFIG.AI
	if !cfg.Enable {
		return fmt.Errorf("AI service is disabled")
	}
	if cfg.Endpoint == "" || cfg.ApiKey == "" || cfg.Model == "" {
		return fmt.Errorf("AI service config is incomplete")
	}
	if len(messages) == 0 {
		return fmt.Errorf("AI request messages is empty")
	}

	body := streamRequestBody{
		Model:       cfg.Model,
		Messages:    messages,
		Temperature: 0.3,
		Stream:      true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal AI request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatCompletionsEndpoint(), bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create AI request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+cfg.ApiKey)

	client, err := newHTTPClient(0)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("call AI API failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("AI API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil
		}

		var streamResp streamResponseBody
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue
		}
		for _, choice := range streamResp.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			if err := onDelta(choice.Delta.Content); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read AI stream failed: %w", err)
	}
	return nil
}

func callChatAIStream(ctx context.Context, systemPrompt, prompt string, onDelta func(string) error) error {
	return callChatAIStreamMessages(ctx, []chatRequestMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}, onDelta)
}
