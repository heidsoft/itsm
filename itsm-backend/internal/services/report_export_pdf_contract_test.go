package service

import (
	"context"
	"testing"

	"itsm-backend/dto"

	"github.com/stretchr/testify/require"
)

func TestExportToPDFIsExplicitlyUnsupported(t *testing.T) {
	data, name, err := NewReportExportService().ExportToPDF(context.Background(), &dto.DeepAnalyticsResponse{})
	require.Error(t, err, "伪 PDF 不得再伪装成可下载报告")
	require.Contains(t, err.Error(), "暂未开放")
	require.Nil(t, data)
	require.Empty(t, name)
}
