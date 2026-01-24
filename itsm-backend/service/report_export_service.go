package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"

	"github.com/xuri/excelize/v2"
)

// ReportExportService 报告导出服务
type ReportExportService struct{}

// NewReportExportService 创建报告导出服务
func NewReportExportService() *ReportExportService {
	return &ReportExportService{}
}

// ExportToExcel 导出为Excel格式
func (s *ReportExportService) ExportToExcel(ctx context.Context, data *dto.DeepAnalyticsResponse) ([]byte, string, error) {
	f := excelize.NewFile()
	sheet := "Sheet1"

	// 设置标题样式
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#4472C4"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#D9E2F3"}},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	dataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "left"},
	})

	// 标题
	f.SetCellValue(sheet, "A1", "数据分析报告")
	f.MergeCell(sheet, "A1", "D1")
	f.SetCellStyle(sheet, "A1", "D1", titleStyle)
	f.SetRowHeight(sheet, 1, 25)

	// 生成时间
	f.SetCellValue(sheet, "A2", fmt.Sprintf("生成时间: %s", time.Now().Format("2006-01-02 15:04:05")))
	f.MergeCell(sheet, "A2", "D2")

	// 表头
	headers := []string{"指标名称", "数值", "数量", "平均时间(分钟)"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c%d", 'A'+i, 4)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// 数据
	row := 5
	for _, point := range data.Data {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), point.Name)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%.2f", point.Value))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), point.Count)
		avgTime := ""
		if point.AvgTime != nil {
			avgTime = fmt.Sprintf("%.2f", *point.AvgTime)
		}
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), avgTime)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row), dataStyle)
		row++
	}

	// 汇总信息
	row += 2
	f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "汇总信息")
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), headerStyle)
	row++

	summaryData := []struct {
		Label string
		Value interface{}
	}{
		{"总计", data.Summary.Total},
		{"已解决", data.Summary.Resolved},
		{"平均响应时间", fmt.Sprintf("%.2f", data.Summary.AvgResponseTime)},
		{"平均解决时间", fmt.Sprintf("%.2f", data.Summary.AvgResolutionTime)},
		{"SLA合规率", fmt.Sprintf("%.2f%%", data.Summary.SLACompliance)},
		{"客户满意度", fmt.Sprintf("%.2f", data.Summary.CustomerSatisfaction)},
	}

	for _, s := range summaryData {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), s.Label)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%v", s.Value))
		row++
	}

	// 设置列宽
	f.SetColWidth(sheet, "A", "A", 25)
	f.SetColWidth(sheet, "B", "B", 15)
	f.SetColWidth(sheet, "C", "C", 10)
	f.SetColWidth(sheet, "D", "D", 20)

	// 生成文件
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, "", fmt.Errorf("生成Excel失败: %w", err)
	}

	filename := fmt.Sprintf("analytics_report_%s.xlsx", time.Now().Format("20060102_150405"))
	return buf.Bytes(), filename, nil
}

// ExportToPDF 导出为PDF格式
func (s *ReportExportService) ExportToPDF(ctx context.Context, data *dto.DeepAnalyticsResponse) ([]byte, string, error) {
	// 使用简单的文本格式生成PDF兼容内容
	// 实际生产中应使用 gofpdf 或其他PDF库

	var content bytes.Buffer

	// PDF头
	content.WriteString("%PDF-1.4\n")
	content.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")
	content.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// 页面内容
	content.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>\nendobj\n")
	content.WriteString("4 0 obj\n<< /Length 5 0 R >>\nstream\n")

	// 标题
	content.WriteString("BT\n")
	content.WriteString("/F1 24 Tf\n")
	content.WriteString("100 750 Td\n")
	content.WriteString("(数据分析报告) Tj\n")
	content.WriteString("ET\n")

	// 生成时间
	content.WriteString("BT\n")
	content.WriteString("/F1 12 Tf\n")
	content.WriteString("100 720 Td\n")
	content.WriteString("(生成时间: " + time.Now().Format("2006-01-02 15:04:05") + ") Tj\n")
	content.WriteString("ET\n")

	// 数据表格
	y := 680
	content.WriteString("BT\n")
	content.WriteString("/F1 10 Tf\n")
	for _, point := range data.Data {
		line := fmt.Sprintf("%s - 数值: %.2f, 数量: %d", point.Name, point.Value, point.Count)
		content.WriteString(fmt.Sprintf("100 %d Td\n", y))
		content.WriteString(fmt.Sprintf("(%s) Tj\n", line))
		y -= 20
	}
	content.WriteString("ET\n")

	// 汇总信息
	y -= 30
	content.WriteString("BT\n")
	content.WriteString("/F1 14 Tf\n")
	content.WriteString(fmt.Sprintf("100 %d Td\n", y))
	content.WriteString("(汇总信息) Tj\n")
	y -= 25
	content.WriteString("ET\n")

	summaryData := []struct {
		Label string
		Value interface{}
	}{
		{"总计", data.Summary.Total},
		{"已解决", data.Summary.Resolved},
		{"平均响应时间", fmt.Sprintf("%.2f", data.Summary.AvgResponseTime)},
		{"平均解决时间", fmt.Sprintf("%.2f", data.Summary.AvgResolutionTime)},
		{"SLA合规率", fmt.Sprintf("%.2f%%", data.Summary.SLACompliance)},
		{"客户满意度", fmt.Sprintf("%.2f", data.Summary.CustomerSatisfaction)},
	}

	content.WriteString("/F1 10 Tf\n")
	for _, s := range summaryData {
		line := fmt.Sprintf("%s: %v", s.Label, s.Value)
		content.WriteString(fmt.Sprintf("100 %d Td\n", y))
		content.WriteString(fmt.Sprintf("(%s) Tj\n", line))
		y -= 15
	}

	content.WriteString("endstream\nendobj\n")
	content.WriteString("5 0 obj\n")
	content.WriteString(fmt.Sprintf("%d\n", content.Len()))
	content.WriteString("endobj\n")

	//xref
	content.WriteString("xref\n")
	content.WriteString("0 6\n")
	content.WriteString("0000000000 65535 f \n")
	content.WriteString("0000000009 00000 n \n")
	content.WriteString("0000000058 00000 n \n")
	content.WriteString("0000000115 00000 n \n")
	content.WriteString("0000000206 00000 n \n")
	content.WriteString("0000000245 00000 n \n")

	content.WriteString("trailer\n<< /Size 6 /Root 1 0 R >>\n")
	content.WriteString("startxref\n")
	content.WriteString(fmt.Sprintf("%d\n", content.Len()))
	content.WriteString("%%EOF")

	filename := fmt.Sprintf("analytics_report_%s.pdf", time.Now().Format("20060102_150405"))
	return content.Bytes(), filename, nil
}

// ExportPredictionToExcel 导出预测报告为Excel
func (s *ReportExportService) ExportPredictionToExcel(ctx context.Context, prediction *dto.TrendPredictionResponse) ([]byte, string, error) {
	f := excelize.NewFile()
	sheet := "Sheet1"

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#4472C4"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#D9E2F3"}},
	})

	// 标题
	f.SetCellValue(sheet, "A1", "趋势预测报告")
	f.MergeCell(sheet, "A1", "F1")
	f.SetCellStyle(sheet, "A1", "F1", titleStyle)

	// 元数据
	f.SetCellValue(sheet, "A2", fmt.Sprintf("预测模型: %s", prediction.Model))
	f.SetCellValue(sheet, "A3", fmt.Sprintf("置信度: %.2f%%", prediction.Confidence*100))
	f.SetCellValue(sheet, "A4", fmt.Sprintf("生成时间: %s", prediction.GeneratedAt.Format("2006-01-02 15:04:05")))

	// 表头
	headers := []string{"日期", "预测值", "下限", "上限", "置信度"}
	if len(prediction.Predictions) > 0 {
		p := prediction.Predictions[0]
		if p.Category != "" {
			headers = append(headers, "分类")
		}
		if p.Priority != "" {
			headers = append(headers, "优先级")
		}
	}

	for i, h := range headers {
		cell := fmt.Sprintf("%c%d", 'A'+i, 6)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// 数据
	row := 7
	for _, p := range prediction.Predictions {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), p.Date)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%.2f", p.PredictedValue))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%.2f", p.LowerBound))
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("%.2f", p.UpperBound))
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("%.2f%%", p.Confidence*100))
		col := 'F'
		if p.Category != "" {
			f.SetCellValue(sheet, fmt.Sprintf("%c%d", col, row), p.Category)
			col++
		}
		if p.Priority != "" {
			f.SetCellValue(sheet, fmt.Sprintf("%c%d", col, row), p.Priority)
		}
		row++
	}

	// 设置列宽
	f.SetColWidth(sheet, "A", "A", 15)
	f.SetColWidth(sheet, "B", "B", 12)
	f.SetColWidth(sheet, "C", "D", 12)
	f.SetColWidth(sheet, "E", "E", 10)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, "", fmt.Errorf("生成Excel失败: %w", err)
	}

	filename := fmt.Sprintf("prediction_report_%s.xlsx", time.Now().Format("20060102_150405"))
	return buf.Bytes(), filename, nil
}
