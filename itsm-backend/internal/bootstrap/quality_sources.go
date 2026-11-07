package bootstrap

import (
	"context"

	"itsm-backend/dto"
	changeHandler "itsm-backend/handlers/change"
	problemHandler "itsm-backend/handlers/problem"
	"itsm-backend/service"
)

// 统一服务质量视图（/reports/service-quality）对问题/变更域统计的装配：
// 聚合器只依赖窄接口（service.QualityProblemStatsSource 等，返回 dto），
// 不反向 import handlers/<domain>；映射复用各域 GetStats 端点的同一份导出
// 映射函数，保证统一视图与单域报表的口径逐字段一致。

type problemStatsSource struct{ svc *problemHandler.Service }

func (s problemStatsSource) GetProblemStats(ctx context.Context, tenantID int) (*dto.ProblemStatsResponse, error) {
	stats, err := s.svc.GetStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return problemHandler.ToStatsResponse(stats), nil
}

type changeStatsSource struct{ svc *changeHandler.Service }

func (s changeStatsSource) GetChangeStats(ctx context.Context, tenantID int) (*dto.ChangeStatsResponse, error) {
	stats, err := s.svc.GetStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return changeHandler.ToStatsDTO(stats), nil
}

// wireQualityDomainStats 把问题/变更域统计注入统一服务质量视图。
// 任一依赖缺失时静默跳过——聚合器对缺省块的降级是显式可见的（响应里块缺省）。
func wireQualityDomainStats(q *service.ServiceQualityReportService, problems *problemHandler.Service, changes *changeHandler.Service) {
	if q == nil || problems == nil || changes == nil {
		return
	}
	q.SetProblemStatsSource(problemStatsSource{svc: problems})
	q.SetChangeStatsSource(changeStatsSource{svc: changes})
}
