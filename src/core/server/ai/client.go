package ai

import (
	"fmt"
	"myobj/src/config"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func newHTTPClient(timeout time.Duration) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if proxyURL := strings.TrimSpace(config.CONFIG.AI.Proxy); proxyURL != "" {
		parsedURL, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("AI 代理地址配置错误: %w", err)
		}
		transport.Proxy = http.ProxyURL(parsedURL)
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}, nil
}

func chatCompletionsEndpoint() string {
	return strings.TrimRight(config.CONFIG.AI.Endpoint, "/") + "/chat/completions"
}
