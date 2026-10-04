package incident

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/common"
)

// GetReport 返回事件趋势报表读模型。
//
// 窗口解析放在 service 而不是 handler，是为了让 HTTP、导出与后续 CLI 复用同一套
// 区间规则（缺省窗口、成对校验、跨度上限），handler 只做参数透传与错误映射。
// dateFrom/dateTo 任一为空表示使用缺省窗口；只传一端是参数错误，不会静默补另一端。
func (s *Service) GetReport(ctx context.Context, tenantID int, dateFrom, dateTo string) (*IncidentReport, error) {
	if tenantID <= 0 {
		return nil, common.NewBusinessError(common.AuthFailedCode, "缺少租户上下文", "incident report requires tenant")
	}
	period, err := ResolveReportPeriod(dateFrom, dateTo, time.Now())
	if err != nil {
		return nil, err
	}
	report, err := s.repo.GetReport(ctx, tenantID, period)
	if err != nil {
		return nil, fmt.Errorf("get incident report: %w", err)
	}
	return report, nil
}
