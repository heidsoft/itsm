package service

import (
	"archive/zip"
	"bytes"
	"html"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/dto"
)

// R4 回归：预测导出的 model/category/priority 是请求可控字符串（DTO 无 oneof），
// 此前 CSV 用 fmt.Sprintf 手拼、Excel 直接 SetCellValue——传 "=HYPERLINK(...)"
// 或 ",IMPORT" 时轻则撕裂列、重则被 Excel/WPS 当公式/DDE 执行。
// 以下用例锁死「用户可控单元格必须中性化 + CSV 必须走 encoding/csv」。

func newPredictionExportFixture() *dto.TrendPredictionResponse {
	return &dto.TrendPredictionResponse{
		Model:       "=HYPERLINK(\"http://evil.example\",\"点我\")",
		Confidence:  0.9,
		GeneratedAt: time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC),
		Predictions: []dto.PredictionDataPoint{
			{
				Date:     "2026-11-07",
				Category: "=cmd|' /C calc'!A0",
				Priority: "+7060045",
			},
		},
	}
}

func TestPredictionCSVExport_NeutralizesUserControlledCells(t *testing.T) {
	svc := &PredictionService{}
	resp := newPredictionExportFixture()

	data, filename, err := svc.exportPredictionToCSV(resp)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(filename, ".csv"))

	text := string(data)

	// 公式载荷被前置撇号中性化（Excel 显示为文本）
	require.Contains(t, text, "'=HYPERLINK", "Model 元数据必须中性化")

	// 原始载荷不得以「裸公式」形态出现（撇号前缀是唯一允许形态）
	assert.NotContains(t, text, "Model,=HYPERLINK", "公式载荷必须被中性化")

	// CSV 结构：含 BOM、表头行、数据行（encoding/csv 转义逗号/换行）
	require.True(t, strings.HasPrefix(text, "\xEF\xBB\xBF"), "需带 utf-8-sig BOM")
	require.Contains(t, text, "Date,Predicted value,Lower bound,Upper bound,Confidence")
	require.Contains(t, text, "Prediction information")
}

func TestPredictionCSVExport_NeutralizesCategoryAndPriority(t *testing.T) {
	svc := &PredictionService{}
	resp := newPredictionExportFixture()
	// CSV 主表只导出日期/数值列，category/priority 不落 CSV——
	// 但 Excel 路径落。这里锁「CSV 不引入新列、也不回归」。
	data, _, err := svc.exportPredictionToCSV(resp)
	require.NoError(t, err)
	text := string(data)
	assert.NotContains(t, text, "=cmd|", "CSV 不得出现未中性化的公式载荷")
	assert.NotContains(t, text, "+7060045", "CSV 不得出现未中性化的公式载荷")
}

func TestPredictionCSVExport_QuotesCommasAndNewlines(t *testing.T) {
	svc := &PredictionService{}
	resp := newPredictionExportFixture()
	// Model 含逗号与换行：手拼 CSV 会撕裂列/伪造行，encoding/csv 会加引号转义。
	resp.Model = "linear,regression\nv2"

	data, _, err := svc.exportPredictionToCSV(resp)
	require.NoError(t, err)

	text := string(data)
	assert.Contains(t, text, "\"linear,regression", "含逗号字段必须被引号包裹")
	assert.Contains(t, text, "v2", "换行内容不得截断")
	// 引号包裹的字段内换行不产生新的数据行：数据行数 = 1 表头 + 1 数据 + 空行 + 4 元数据
	lines := strings.Split(strings.TrimPrefix(text, "\xEF\xBB\xBF"), "\n")
	nonEmpty := 0
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty++
		}
	}
	assert.Equal(t, 7, nonEmpty, "换行必须留在引号字段内，不得伪造行")
}

func TestPredictionExcelExport_NeutralizesMetadataAndCells(t *testing.T) {
	exporter := NewReportExportService()
	resp := newPredictionExportFixture()

	data, _, err := exporter.ExportPredictionToExcel(nil, resp)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// xlsx 是 zip 容器，共享字符串以 XML 明文存储在 xl/sharedStrings.xml；
	// 整体字节流是压缩流，直接 Contains 不可靠——解包后校验。
	text := html.UnescapeString(sharedStringsXML(t, data))
	require.Contains(t, text, "'=HYPERLINK", "Model 元数据必须中性化")
	require.Contains(t, text, "'=cmd|", "Category 单元格必须中性化")
	require.Contains(t, text, "'+7060045", "Priority 单元格必须中性化")
	assert.NotContains(t, text, ">=HYPERLINK", "裸公式值不得出现（XML 转义后 = 前不带撇号）")
}

// sharedStringsXML 从 xlsx 字节流解出共享字符串表（小工具，测试专用）。
func sharedStringsXML(t *testing.T, xlsx []byte) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(xlsx), int64(len(xlsx)))
	require.NoError(t, err)
	for _, f := range zr.File {
		if f.Name == "xl/sharedStrings.xml" {
			rc, oErr := f.Open()
			require.NoError(t, oErr)
			defer rc.Close()
			raw, rErr := io.ReadAll(rc)
			require.NoError(t, rErr)
			return string(raw)
		}
	}
	t.Fatal("xl/sharedStrings.xml not found in xlsx")
	return ""
}
