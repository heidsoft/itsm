package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
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
		Font:      &excelize.Font{Bold: true, Size: 14},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#4472C4"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#D9E2F3"}},
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
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), neutralizeSpreadsheetFormula(point.Name))
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

// ExportToPDF 显式拒绝 PDF 导出。
//
// 旧实现手拼 PDF 字节：xref 偏移是硬编码常量、字体对象缺失，严格阅读器无法打开；
// 预测导出还把「预测置信度」塞进 SLACompliance 字段并打印成「SLA 合规率」。
// 这是伪装成成功的能力半成品，会直接误导容量/合规决策。在引入真实 PDF 渲染器
// （含中文字体与可校验的 cross-reference 表）之前，统一返回参数错误。
func (s *ReportExportService) ExportToPDF(ctx context.Context, data *dto.DeepAnalyticsResponse) ([]byte, string, error) {
	return nil, "", common.NewBadRequestError("PDF 导出暂未开放，请使用 Excel 或 CSV", nil)
}

func neutralizeSpreadsheetFormula(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}

// ExportPredictionToExcel 导出预测报告为Excel
func (s *ReportExportService) ExportPredictionToExcel(ctx context.Context, prediction *dto.TrendPredictionResponse) ([]byte, string, error) {
	f := excelize.NewFile()
	sheet := "Sheet1"

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#4472C4"}},
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
	// Model 完全来自请求参数（dto 无 oneof 约束），是公式注入的可达入口：
	// 传 "=HYPERLINK(...)" 时 Excel/WPS 打开即执行。所有用户可控字符串
	// 落单元格前必须过 neutralizeSpreadsheetFormula（R4）。
	f.SetCellValue(sheet, "A2", fmt.Sprintf("预测模型: %s", neutralizeSpreadsheetFormula(prediction.Model)))
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
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), neutralizeSpreadsheetFormula(p.Date))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("%.2f", p.PredictedValue))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("%.2f", p.LowerBound))
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("%.2f", p.UpperBound))
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("%.2f%%", p.Confidence*100))
		col := 'F'
		if p.Category != "" {
			f.SetCellValue(sheet, fmt.Sprintf("%c%d", col, row), neutralizeSpreadsheetFormula(p.Category))
			col++
		}
		if p.Priority != "" {
			f.SetCellValue(sheet, fmt.Sprintf("%c%d", col, row), neutralizeSpreadsheetFormula(p.Priority))
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
