package seeder

// 业务记录播种（演示数据）：Incidents / Problems / Changes / KnowledgeArticles。
//
// 设计约束：
//   - 仅当 SeedConfig 中对应数组非空时生效。生产默认配置（getProductDefaultConfig）
//     会清空这四类数组，因此常规部署不预置任何虚构业务记录（与 GA readiness 的
//     "product_seed 只提供功能模板" 语义保持一致）。
//   - 幂等：按编号（incident_number / problem_number / change_number）或
//     标题（知识库）+ 租户去重，重复执行不会产生重复数据。
//   - 失败语义：与其他 seed helper 一致，记录 warn 日志并继续，不中断启动。
//   - 演示数据使用固定编号（如 INC-DEMO-0001），避免日期化编号在次日重跑时产生重复。

import (
	"context"

	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/tenant"
	"itsm-backend/ent/user"
)

// seedBusinessRecords 在 default 租户下播种演示业务记录。
// 依赖 seedDefaultTenant / seedAdmin 先行执行（SeedAll 顺序保证）。
func (s *Seeder) seedBusinessRecords(ctx context.Context) {
	hasRecords := len(s.config.Incidents) > 0 ||
		len(s.config.Problems) > 0 ||
		len(s.config.Changes) > 0 ||
		len(s.config.KnowledgeArticles) > 0
	if !hasRecords {
		return
	}

	t, err := s.client.Tenant.Query().Where(tenant.CodeEQ("default")).First(ctx)
	if err != nil {
		s.sugar.Warnw("default tenant not found; skip business records seed", "error", err)
		return
	}
	admin, err := s.client.User.Query().Where(user.UsernameEQ("admin"), user.TenantIDEQ(t.ID)).First(ctx)
	if err != nil {
		s.sugar.Warnw("admin user not found; skip business records seed (run with ADMIN_PASSWORD first)", "error", err)
		return
	}

	s.seedIncidentRecords(ctx, t.ID, admin.ID)
	s.seedProblemRecords(ctx, t.ID, admin.ID)
	s.seedChangeRecords(ctx, t.ID, admin.ID)
	s.seedKnowledgeArticleRecords(ctx, t.ID, admin.ID)
}

func (s *Seeder) seedIncidentRecords(ctx context.Context, tenantID, adminID int) {
	for _, seed := range s.config.Incidents {
		exists, err := s.client.Incident.Query().
			Where(incident.IncidentNumberEQ(seed.IncidentNumber), incident.TenantIDEQ(tenantID)).
			Exist(ctx)
		if err != nil {
			s.sugar.Warnw("check demo incident failed", "number", seed.IncidentNumber, "error", err)
			continue
		}
		if exists {
			s.sugar.Infow("demo incident already exists; skip", "number", seed.IncidentNumber)
			continue
		}
		builder := s.client.Incident.Create().
			SetTitle(seed.Title).
			SetIncidentNumber(seed.IncidentNumber).
			SetStatus(seed.Status).
			SetPriority(seed.Priority).
			SetTenantID(tenantID).
			SetReporterID(adminID)
		if seed.Description != "" {
			builder.SetDescription(seed.Description)
		}
		if seed.Severity != "" {
			builder.SetSeverity(seed.Severity)
		}
		if seed.Category != "" {
			builder.SetCategory(seed.Category)
		}
		if _, err := builder.Save(ctx); err != nil {
			if !ent.IsConstraintError(err) {
				s.sugar.Warnw("seed demo incident failed", "number", seed.IncidentNumber, "error", err)
			}
			continue
		}
		s.sugar.Infow("demo incident created", "number", seed.IncidentNumber, "title", seed.Title)
	}
}

func (s *Seeder) seedProblemRecords(ctx context.Context, tenantID, adminID int) {
	for _, seed := range s.config.Problems {
		exists, err := s.client.Problem.Query().
			Where(problem.ProblemNumberEQ(seed.ProblemNumber), problem.TenantIDEQ(tenantID)).
			Exist(ctx)
		if err != nil {
			s.sugar.Warnw("check demo problem failed", "number", seed.ProblemNumber, "error", err)
			continue
		}
		if exists {
			s.sugar.Infow("demo problem already exists; skip", "number", seed.ProblemNumber)
			continue
		}
		builder := s.client.Problem.Create().
			SetTitle(seed.Title).
			SetStatus(seed.Status).
			SetPriority(seed.Priority).
			SetTenantID(tenantID).
			SetCreatedBy(adminID)
		if seed.ProblemNumber != "" {
			builder.SetProblemNumber(seed.ProblemNumber)
		}
		if seed.Description != "" {
			builder.SetDescription(seed.Description)
		}
		if seed.Category != "" {
			builder.SetCategory(seed.Category)
		}
		if seed.RootCause != "" {
			builder.SetRootCause(seed.RootCause)
		}
		if seed.Impact != "" {
			builder.SetImpact(seed.Impact)
		}
		if _, err := builder.Save(ctx); err != nil {
			if !ent.IsConstraintError(err) {
				s.sugar.Warnw("seed demo problem failed", "number", seed.ProblemNumber, "error", err)
			}
			continue
		}
		s.sugar.Infow("demo problem created", "number", seed.ProblemNumber, "title", seed.Title)
	}
}

func (s *Seeder) seedChangeRecords(ctx context.Context, tenantID, adminID int) {
	for _, seed := range s.config.Changes {
		exists, err := s.client.Change.Query().
			Where(change.ChangeNumberEQ(seed.ChangeNumber), change.TenantIDEQ(tenantID)).
			Exist(ctx)
		if err != nil {
			s.sugar.Warnw("check demo change failed", "number", seed.ChangeNumber, "error", err)
			continue
		}
		if exists {
			s.sugar.Infow("demo change already exists; skip", "number", seed.ChangeNumber)
			continue
		}
		builder := s.client.Change.Create().
			SetTitle(seed.Title).
			SetType(seed.Type).
			SetStatus(seed.Status).
			SetPriority(seed.Priority).
			SetTenantID(tenantID).
			SetCreatedBy(adminID)
		if seed.ChangeNumber != "" {
			builder.SetChangeNumber(seed.ChangeNumber)
		}
		if seed.Description != "" {
			builder.SetDescription(seed.Description)
		}
		if seed.ImpactScope != "" {
			builder.SetImpactScope(seed.ImpactScope)
		}
		if seed.RiskLevel != "" {
			builder.SetRiskLevel(seed.RiskLevel)
		}
		if seed.Justification != "" {
			builder.SetJustification(seed.Justification)
		}
		if _, err := builder.Save(ctx); err != nil {
			if !ent.IsConstraintError(err) {
				s.sugar.Warnw("seed demo change failed", "number", seed.ChangeNumber, "error", err)
			}
			continue
		}
		s.sugar.Infow("demo change created", "number", seed.ChangeNumber, "title", seed.Title)
	}
}

func (s *Seeder) seedKnowledgeArticleRecords(ctx context.Context, tenantID, adminID int) {
	for _, seed := range s.config.KnowledgeArticles {
		exists, err := s.client.KnowledgeArticle.Query().
			Where(knowledgearticle.TitleEQ(seed.Title), knowledgearticle.TenantIDEQ(tenantID)).
			Exist(ctx)
		if err != nil {
			s.sugar.Warnw("check demo knowledge article failed", "title", seed.Title, "error", err)
			continue
		}
		if exists {
			s.sugar.Infow("demo knowledge article already exists; skip", "title", seed.Title)
			continue
		}
		builder := s.client.KnowledgeArticle.Create().
			SetTitle(seed.Title).
			SetTenantID(tenantID).
			SetAuthorID(adminID)
		if seed.Content != "" {
			builder.SetContent(seed.Content)
		}
		if seed.Category != "" {
			builder.SetCategory(seed.Category)
		}
		builder.SetIsPublished(seed.IsPublished)
		builder.SetViewCount(seed.ViewCount)
		if _, err := builder.Save(ctx); err != nil {
			if !ent.IsConstraintError(err) {
				s.sugar.Warnw("seed demo knowledge article failed", "title", seed.Title, "error", err)
			}
			continue
		}
		s.sugar.Infow("demo knowledge article created", "title", seed.Title)
	}
}
