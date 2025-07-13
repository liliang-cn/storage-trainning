package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// convertToPDF 使用纯 Go 库将 markdown 文件转换为 PDF
func convertToPDF(mdPath string) error {
	fmt.Printf("Converting: %s\n", mdPath)

	// 1. 读取 Markdown 文件内容
	mdContent, err := os.ReadFile(mdPath)
	if err != nil {
		return fmt.Errorf("could not read markdown file: %w", err)
	}

	// 2. 初始化 PDF 对象
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 12)
	pdf.SetMargins(20, 20, 20)

	// 3. 解析 Markdown AST
	mdParser := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	).Parser()
	rootNode := mdParser.Parse(text.NewReader(mdContent))

	// 4. 遍历 AST 并渲染到 PDF
	err = walkAndRenderPDF(pdf, rootNode, mdContent)
	if err != nil {
		return fmt.Errorf("could not render pdf: %w", err)
	}

	// 5. 定义并保存 PDF 文件
	pdfPath := strings.TrimSuffix(mdPath, filepath.Ext(mdPath)) + ".pdf"
	err = pdf.OutputFileAndClose(pdfPath)
	if err != nil {
		return fmt.Errorf("could not save pdf: %w", err)
	}

	fmt.Printf("Successfully created: %s\n", pdfPath)
	return nil
}

// walkAndRenderPDF 遍历 AST 节点并将其内容写入 PDF
func walkAndRenderPDF(pdf *gofpdf.Fpdf, node ast.Node, source []byte) error {
	return ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			// 只在进入节点时处理
			return ast.WalkContinue, nil
		}

		switch n.Kind() {
		case ast.KindHeading:
			heading := n.(*ast.Heading)
			level := heading.Level
			content := string(n.Text(source))

			// 根据标题级别设置字体大小和样式
			switch level {
			case 1:
				pdf.SetFont("Arial", "B", 24)
				pdf.Ln(10)
			case 2:
				pdf.SetFont("Arial", "B", 18)
				pdf.Ln(8)
			case 3:
				pdf.SetFont("Arial", "B", 14)
				pdf.Ln(6)
			default:
				pdf.SetFont("Arial", "B", 12)
				pdf.Ln(4)
			}
			pdf.Cell(0, 10, content)
			pdf.Ln(6)
			pdf.SetFont("Arial", "", 12) // 恢复默认字体

		case ast.KindParagraph:
			content := string(n.Text(source))
			pdf.Write(5, content)
			pdf.Ln(8)

		case ast.KindList:
			list := n.(*ast.List)
			// 为列表项添加缩进
			pdf.SetLeftMargin(pdf.GetX() + 5)
			// 遍历子节点（列表项）
			for c, i := n.FirstChild(), 1; c != nil; c, i = c.NextSibling(), i+1 {
				item := c.(*ast.ListItem)
				itemText := string(item.Text(source))
				if list.IsOrdered() {
					pdf.Write(5, fmt.Sprintf("%d. %s", i, itemText))
				} else {
					pdf.Write(5, fmt.Sprintf("- %s", itemText))
				}
				pdf.Ln(5)
			}
			pdf.SetLeftMargin(20) // 恢复边距
			pdf.Ln(5)
			return ast.WalkSkipChildren, nil // 已经手动处理子节点

		case ast.KindCodeBlock, ast.KindFencedCodeBlock:
			pdf.SetFont("Courier", "", 10)
			pdf.SetFillColor(240, 240, 240) // 浅灰色背景
			var content string
			for i := 0; i < n.Lines().Len(); i++ {
				line := n.Lines().At(i)
				content += string(line.Value(source))
			}
			pdf.MultiCell(0, 5, content, "1", "L", true)
			pdf.SetFont("Arial", "", 12) // 恢复默认字体
			pdf.Ln(5)

		case ast.KindThematicBreak: // 正确的类型是 ThematicBreak
			pdf.Line(pdf.GetX(), pdf.GetY(), 210-pdf.GetX(), pdf.GetY())
			pdf.Ln(5)
		}

		return ast.WalkContinue, nil
	})
}