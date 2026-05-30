package ai

import (
	"fmt"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
)

const maxContentLength = 20000

// fileParser 文件解析策略接口
type fileParser interface {
	Parse(filePath string) (string, error)
}

// textParser 纯文本文件解析
type textParser struct{}

func (textParser) Parse(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	content := string(data)
	if len(content) > maxContentLength {
		content = content[:maxContentLength]
	}
	return content, nil
}

// pdfParser PDF 文件解析
type pdfParser struct{}

func (pdfParser) Parse(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开PDF文件失败: %w", err)
	}
	defer f.Close()

	var builder strings.Builder
	totalPages := r.NumPage()

	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		if builder.Len() >= maxContentLength {
			break
		}
		p := r.Page(pageNum)
		if p.V.IsNull() {
			continue
		}

		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}

		remaining := maxContentLength - builder.Len()
		if len(text) > remaining {
			builder.WriteString(text[:remaining])
		} else {
			builder.WriteString(text)
		}
	}

	return builder.String(), nil
}

// getParser 根据文件类型返回对应的解析器
func getParser(fileType string) (fileParser, error) {
	switch strings.ToLower(fileType) {
	case "txt":
		return textParser{}, nil
	case "md", "markdown":
		return textParser{}, nil
	case "pdf":
		return pdfParser{}, nil
	case "text/plain":
		return textParser{}, nil
	case "text/markdown":
		return textParser{}, nil
	case "application/pdf":
		return pdfParser{}, nil
	default:
		return nil, fmt.Errorf("暂不支持该文件类型 AI 分析")
	}
}

// ParseFileContent 解析文件内容为纯文本
// filePath: 文件路径
// fileType: 文件类型（支持文件扩展名或 MIME 类型）
func ParseFileContent(filePath string, fileType string) (string, error) {
	parser, err := getParser(fileType)
	if err != nil {
		return "", err
	}
	return parser.Parse(filePath)
}
