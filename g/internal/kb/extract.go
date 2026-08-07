package kb

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// extractAndSaveText 提取文本并保存为 {原文件名}.txt，方便查阅和后续复用。
// fileStore 用于读写缓存文件；remotePath 是对象存储中的原始路径（S3 模式下缓存也放 S3）。
func extractAndSaveText(localPath string, fileStore FileStore, remotePath string) (string, error) {
	// 尝试读取缓存的 txt 文件
	txtPath := remotePath + ".txt"
	if cached, err := fileStore.ReadAll(txtPath); err == nil {
		text := string(cached)
		if strings.TrimSpace(text) != "" {
			return text, nil
		}
	}

	// 提取文本（始终用本地路径）
	text, err := extractText(localPath)
	if err != nil {
		return "", err
	}

	// 保存缓存（失败不影响主流程）
	if _, err := saveString(fileStore, txtPath, text); err != nil {
		_ = err
	}

	return text, nil
}

// saveString 保存字符串到文件存储
func saveString(fs FileStore, path, content string) (int64, error) {
	return fs.Save(path, strings.NewReader(content))
}

// extractText 根据文件后缀提取文本内容
func extractText(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pdf":
		return extractPDF(filePath)
	case ".docx":
		return extractDOCX(filePath)
	case ".xlsx", ".xls":
		return extractXLSX(filePath)
	default:
		return "", fmt.Errorf("不支持的文件格式: %s", ext)
	}
}

// extractPDF 从 PDF 文件中提取文本
func extractPDF(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开 PDF 失败: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	numPages := r.NumPage()
	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(text)
		buf.WriteString("\n")
	}

	result := strings.TrimSpace(buf.String())
	if result == "" {
		return "", fmt.Errorf("PDF 文件无文本内容，可能是扫描件")
	}
	return result, nil
}

// extractDOCX 从 DOCX 文件中提取文本
// DOCX 本质上是 zip 包，文本在 word/document.xml 的 <w:t> 标签中
func extractDOCX(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("打开 DOCX 失败: %w", err)
	}
	defer r.Close()

	var documentXML *zip.File
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			documentXML = f
			break
		}
	}
	if documentXML == nil {
		return "", fmt.Errorf("无效的 DOCX 文件: 缺少 word/document.xml")
	}

	rc, err := documentXML.Open()
	if err != nil {
		return "", fmt.Errorf("读取 document.xml 失败: %w", err)
	}
	defer rc.Close()

	// 解析 XML，提取所有 <w:t> 标签的文本
	type Text struct {
		Content string `xml:",chardata"`
	}
	type Run struct {
		Text []Text `xml:"t"`
	}
	type Paragraph struct {
		Runs []Run `xml:"r"`
	}
	type Body struct {
		Paragraphs []Paragraph `xml:"p"`
	}
	type Document struct {
		Body Body `xml:"body"`
	}

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("读取 XML 数据失败: %w", err)
	}

	var doc Document
	if err := xml.Unmarshal(data, &doc); err != nil {
		return "", fmt.Errorf("解析 DOCX XML 失败: %w", err)
	}

	var paragraphs []string
	for _, p := range doc.Body.Paragraphs {
		var line string
		for _, r := range p.Runs {
			for _, t := range r.Text {
				line += t.Content
			}
		}
		if strings.TrimSpace(line) != "" {
			paragraphs = append(paragraphs, line)
		}
	}

	result := strings.Join(paragraphs, "\n")
	if result == "" {
		return "", fmt.Errorf("DOCX 文件无文本内容")
	}
	return result, nil
}

// extractXLSX 从 XLSX/XLS 文件中提取文本
// XLSX 本质上是 zip 包，文本在 xl/sharedStrings.xml 和各 sheet 中
func extractXLSX(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".xls" {
		return "", fmt.Errorf("旧版 .xls 格式不支持，请转换为 .xlsx 格式后再上传")
	}

	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("打开 XLSX 失败: %w", err)
	}
	defer r.Close()

	// 1. 读取共享字符串表
	var sharedStrings []string
	for _, f := range r.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("读取 sharedStrings.xml 失败: %w", err)
			}
			defer rc.Close()

			type Si struct {
				T string `xml:"t"`
			}
			type SST struct {
				Si []Si `xml:"si"`
			}

			data, err := io.ReadAll(rc)
			if err != nil {
				return "", fmt.Errorf("读取共享字符串数据失败: %w", err)
			}

			var sst SST
			if err := xml.Unmarshal(data, &sst); err != nil {
				// 如果解析失败，继续处理，只是字符串可能为空
			} else {
				for _, si := range sst.Si {
					sharedStrings = append(sharedStrings, si.T)
				}
			}
			break
		}
	}

	// 2. 读取第一个 sheet
	var sheetFile *zip.File
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			sheetFile = f
			break
		}
	}
	if sheetFile == nil {
		return "", fmt.Errorf("无效的 XLSX 文件: 缺少 sheet 数据")
	}

	rc, err := sheetFile.Open()
	if err != nil {
		return "", fmt.Errorf("读取 sheet 失败: %w", err)
	}
	defer rc.Close()

	// 解析单元格
	type Cell struct {
		Ref string `xml:"r,attr"`
		T   string `xml:"t,attr"` // "s" = shared string, "inlineStr" = inline, 空 = number
		V   string `xml:"v"`
		Is  struct {
			T string `xml:"t"`
		} `xml:"is"`
	}
	type Row struct {
		Cells []Cell `xml:"c"`
	}
	type SheetData struct {
		Rows []Row `xml:"row"`
	}
	type Worksheet struct {
		SheetData SheetData `xml:"sheetData"`
	}

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", fmt.Errorf("读取 sheet 数据失败: %w", err)
	}

	var ws Worksheet
	if err := xml.Unmarshal(data, &ws); err != nil {
		return "", fmt.Errorf("解析 XLSX XML 失败: %w", err)
	}

	var rows []string
	for _, row := range ws.SheetData.Rows {
		var cells []string
		for _, c := range row.Cells {
			val := ""
			switch c.T {
			case "s":
				// 共享字符串引用
				idx := 0
				fmt.Sscanf(c.V, "%d", &idx)
				if idx >= 0 && idx < len(sharedStrings) {
					val = sharedStrings[idx]
				}
			case "inlineStr":
				val = c.Is.T
			default:
				val = c.V
			}
			if val != "" {
				cells = append(cells, val)
			}
		}
		if len(cells) > 0 {
			rows = append(rows, strings.Join(cells, "\t"))
		}
	}

	result := strings.Join(rows, "\n")
	if result == "" {
		return "", fmt.Errorf("XLSX 文件无内容")
	}
	return result, nil
}
