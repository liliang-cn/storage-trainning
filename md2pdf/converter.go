package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// convertToPDF 使用 wkhtmltopdf 将 markdown 文件转换为 PDF
func convertToPDF(mdPath string) error {
	fmt.Printf("Converting: %s\n", mdPath)

	// 1. 检查 wkhtmltopdf 是否存在
	if _, err := exec.LookPath("wkhtmltopdf"); err != nil {
		return fmt.Errorf("wkhtmltopdf not found in PATH. Please install it from https://wkhtmltopdf.org/")
	}

	// 2. 读取 Markdown 文件内容
	mdContent, err := os.ReadFile(mdPath)
	if err != nil {
		return fmt.Errorf("could not read markdown file: %w", err)
	}

	// 3. 将 Markdown 转换为 HTML
	var htmlBuffer bytes.Buffer
	mdParser := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)
	if err := mdParser.Convert(mdContent, &htmlBuffer); err != nil {
		return fmt.Errorf("could not convert markdown to html: %w", err)
	}

	// 添加一些样式，特别是针对中文字体
	htmlWithStyle := fmt.Sprintf(`
	<html>
	<head>
		<meta charset="UTF-8">
		<style>
			body { font-family: "Source Han Sans", "Noto Sans CJK SC", "Microsoft YaHei", sans-serif; }
			pre { background-color: #f0f0f0; padding: 10px; border-radius: 5px; }
			code { font-family: "Courier New", monospace; }
		</style>
	</head>
	<body>
		%s
	</body>
	</html>`, htmlBuffer.String())

	// 4. 创建一个临时的 HTML 文件
	tempHTMLFile, err := os.CreateTemp("", "md2pdf-*.html")
	if err != nil {
		return fmt.Errorf("could not create temporary html file: %w", err)
	}
	defer os.Remove(tempHTMLFile.Name()) // 确保临时文件被删除

	if _, err := tempHTMLFile.WriteString(htmlWithStyle); err != nil {
		return fmt.Errorf("could not write to temporary html file: %w", err)
	}
	tempHTMLFile.Close()

	// 5. 定义输出的 PDF 文件路径
	pdfPath := strings.TrimSuffix(mdPath, filepath.Ext(mdPath)) + ".pdf"

	// 6. 执行 wkhtmltopdf 命令
	cmd := exec.Command("wkhtmltopdf",
		"--load-error-handling", "ignore", // 忽略网络错误
		"--enable-local-file-access", // 允许访问本地文件 (例如图片)
		"--encoding", "utf-8",
		tempHTMLFile.Name(),
		pdfPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// 即使 wkhtmltopdf 返回错误，也检查一下 PDF 文件是否生成
		if fileInfo, statErr := os.Stat(pdfPath); statErr == nil && fileInfo.Size() > 0 {
			fmt.Printf("Successfully created with non-critical errors: %s\n", pdfPath)
			return nil // 文件已生成，忽略错误
		}
		return fmt.Errorf("wkhtmltopdf execution failed: %w\nStderr: %s", err, stderr.String())
	}

	fmt.Printf("Successfully created: %s\n", pdfPath)
	return nil
}