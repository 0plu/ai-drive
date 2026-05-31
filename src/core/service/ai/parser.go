package ai

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
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

// docxParser Word 文档解析
type docxParser struct{}

func (docxParser) Parse(filePath string) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("打开DOCX文件失败: %w", err)
	}
	defer reader.Close()

	var documentFile *zip.File
	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			documentFile = file
			break
		}
	}
	if documentFile == nil {
		return "", fmt.Errorf("DOCX正文内容不存在")
	}

	rc, err := documentFile.Open()
	if err != nil {
		return "", fmt.Errorf("读取DOCX正文失败: %w", err)
	}
	defer rc.Close()

	content, err := parseDocxDocumentXML(rc)
	if err != nil {
		return "", err
	}
	if len(content) > maxContentLength {
		content = content[:maxContentLength]
	}
	return content, nil
}

func parseDocxDocumentXML(r io.Reader) (string, error) {
	decoder := xml.NewDecoder(r)
	var builder strings.Builder
	inText := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("解析DOCX正文失败: %w", err)
		}
		if builder.Len() >= maxContentLength {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "t":
				inText = true
			case "tab":
				builder.WriteString("\t")
			case "br", "cr":
				builder.WriteString("\n")
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inText = false
			case "p":
				builder.WriteString("\n")
			}
		case xml.CharData:
			if inText {
				remaining := maxContentLength - builder.Len()
				text := string(t)
				if len(text) > remaining {
					builder.WriteString(text[:remaining])
				} else {
					builder.WriteString(text)
				}
			}
		}
	}

	return strings.TrimSpace(builder.String()), nil
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
	case "docx":
		return docxParser{}, nil
	case "text/plain":
		return textParser{}, nil
	case "text/markdown":
		return textParser{}, nil
	case "application/pdf":
		return pdfParser{}, nil
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return docxParser{}, nil
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
