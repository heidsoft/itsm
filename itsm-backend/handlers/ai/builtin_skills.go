package ai

import (
	"fmt"

	"go.uber.org/zap"
	"itsm-backend/service"
)

// RegisterBuiltinSkills 将 ITSM 内置 AI Skill 注入到指定 SkillRegistry。
//
// 调用方必须在 ai.Service 装配完成后调用（本函数只持有 *Service 引用）。
//
// 本期为"薄包装"形态：每个 Skill.Execute() 复用 service 已有的方法，未替换
// 任何 HTTP 入口的逻辑。这样后续 Sprint 可以渐进迁移 endpoint 到 SkillRegistry。
//
// 返回首次注册失败的错误以触发启动期 fail-fast；后续调度不会阻断启动。
func RegisterBuiltinSkills(reg *service.SkillRegistry, svc *Service, logger *zap.SugaredLogger) error {
	if reg == nil {
		return fmt.Errorf("RegisterBuiltinSkills: nil registry")
	}
	if svc == nil {
		return fmt.Errorf("RegisterBuiltinSkills: nil service")
	}
	builtins := []service.Skill{
		NewTriageSkill(svc),
		NewChatSkill(svc),
		NewKnowledgeSearchSkill(svc),
		NewSummarizeSkill(svc),
		NewAnalyzeSkill(svc),
		NewAnalyticsSkill(svc),
		NewTrendPredictionSkill(svc),
		NewCreateTicketSkill(svc),
		NewAgentToolSkill(svc),
		NewMetricsSkill(svc),
		NewFeedbackSkill(svc),
	}
	var failed int
	for _, s := range builtins {
		if err := reg.Register(s); err != nil {
			if logger != nil {
				logger.Errorw("register builtin skill failed", "code", s.Code(), "error", err)
			}
			failed++
			continue
		}
		if logger != nil {
			logger.Infow("registered builtin skill", "code", s.Code(), "name", s.Name(), "category", s.Manifest().Category)
		}
	}
	if logger != nil {
		logger.Infow("builtin skills registered", "total", len(builtins), "failed", failed)
	}
	if failed > 0 {
		return fmt.Errorf("RegisterBuiltinSkills: %d/%d skills failed to register", failed, len(builtins))
	}
	return nil
}
