package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/knownerror"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

// KnownErrorService 已知错误服务
type KnownErrorService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewKnownErrorService 创建已知错误服务
func NewKnownErrorService(client *ent.Client, logger *zap.SugaredLogger) *KnownErrorService {
	return &KnownErrorService{client: client, logger: logger}
}

// CreateKnownError 创建已知错误
func (s *KnownErrorService) CreateKnownError(ctx context.Context, input dto.CreateKnownErrorRequest) (*ent.KnownError, error) {
	// 验证 creator 有效性：用户必须存在、同租户、且处于激活状态
	creator, err := s.client.User.Query().
		Where(user.IDEQ(input.CreatedBy), user.TenantIDEQ(input.TenantID), user.ActiveEQ(true)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("creator not found or inactive")
		}
		return nil, fmt.Errorf("failed to validate creator: %w", err)
	}
	_ = creator // creator 已通过校验

	return s.client.KnownError.Create().
		SetTitle(input.Title).
		SetDescription(input.Description).
		SetSymptoms(input.Symptoms).
		SetRootCause(input.RootCause).
		SetWorkaround(input.Workaround).
		SetResolution(input.Resolution).
		SetStatus(input.Status).
		SetCategory(input.Category).
		SetSeverity(input.Severity).
		SetAffectedProducts(input.AffectedProducts).
		SetAffectedCis(input.AffectedCIs).
		SetKeywords(input.Keywords).
		SetOccurrenceCount(0).
		SetCreatedBy(input.CreatedBy).
		SetTenantID(input.TenantID).
		Save(ctx)
}

// GetKnownErrorByID 获取已知错误
func (s *KnownErrorService) GetKnownErrorByID(ctx context.Context, id int) (*ent.KnownError, error) {
	return s.client.KnownError.Get(ctx, id)
}

// QueryKnownErrors 查询已知错误列表
func (s *KnownErrorService) QueryKnownErrors(ctx context.Context, tenantID int) ([]*ent.KnownError, error) {
	return s.client.KnownError.Query().
		Where(knownerror.TenantID(tenantID)).
		All(ctx)
}

// GetKnownErrorsByTenant 获取租户的已知错误
func (s *KnownErrorService) GetKnownErrorsByTenant(ctx context.Context, tenantID int) ([]*ent.KnownError, error) {
	return s.client.KnownError.Query().
		Where(knownerror.TenantID(tenantID)).
		All(ctx)
}

// SearchKnownErrors 搜索已知错误
func (s *KnownErrorService) SearchKnownErrors(ctx context.Context, tenantID int, keyword string) ([]*ent.KnownError, error) {
	all, err := s.client.KnownError.Query().
		Where(knownerror.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*ent.KnownError
	for _, ke := range all {
		if ke.Title != "" && contains(ke.Title, keyword) {
			result = append(result, ke)
			continue
		}
		if ke.Description != "" && contains(ke.Description, keyword) {
			result = append(result, ke)
		}
	}
	return result, nil
}

// GetActiveKnownErrors 获取所有激活的已知错误
func (s *KnownErrorService) GetActiveKnownErrors(ctx context.Context, tenantID int) ([]*ent.KnownError, error) {
	return s.client.KnownError.Query().
		Where(knownerror.TenantID(tenantID), knownerror.Status("active")).
		All(ctx)
}

// UpdateKnownError 更新已知错误
func (s *KnownErrorService) UpdateKnownError(ctx context.Context, id int, input dto.UpdateKnownErrorRequest) (*ent.KnownError, error) {
	// 先验证记录存在且属于目标租户
	existing, err := s.client.KnownError.Query().
		Where(knownerror.ID(id)).
		Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("known error not found: %w", err)
	}
	update := s.client.KnownError.UpdateOne(existing)
	if input.Title != nil {
		update.SetTitle(*input.Title)
	}
	if input.Description != nil {
		update.SetDescription(*input.Description)
	}
	if input.Symptoms != nil {
		update.SetSymptoms(*input.Symptoms)
	}
	if input.RootCause != nil {
		update.SetRootCause(*input.RootCause)
	}
	if input.Workaround != nil {
		update.SetWorkaround(*input.Workaround)
	}
	if input.Resolution != nil {
		update.SetResolution(*input.Resolution)
	}
	if input.Status != nil {
		update.SetStatus(*input.Status)
	}
	if input.Category != nil {
		update.SetCategory(*input.Category)
	}
	if input.Severity != nil {
		update.SetSeverity(*input.Severity)
	}
	if input.AffectedProducts != nil {
		update.SetAffectedProducts(*input.AffectedProducts)
	}
	if input.AffectedCIs != nil {
		update.SetAffectedCis(*input.AffectedCIs)
	}
	if input.Keywords != nil {
		update.SetKeywords(*input.Keywords)
	}
	return update.Save(ctx)
}

// DeleteKnownError 删除已知错误
func (s *KnownErrorService) DeleteKnownError(ctx context.Context, id int) error {
	deleted, err := s.client.KnownError.Delete().Where(knownerror.ID(id)).Exec(ctx)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return fmt.Errorf("known error not found")
	}
	return nil
}

// ApproveKnownError 审批已知错误
func (s *KnownErrorService) ApproveKnownError(ctx context.Context, id, approverID int) error {
	_, err := s.client.KnownError.Update().
		Where(knownerror.ID(id)).
		SetApprovedBy(approverID).
		SetApprovedAt(time.Now()).
		SetStatus("active").
		Save(ctx)
	return err
}

// IncrementOccurrenceCount 增加发生次数
func (s *KnownErrorService) IncrementOccurrenceCount(ctx context.Context, id int) error {
	errEnt, err := s.client.KnownError.Get(ctx, id)
	if err != nil {
		return err
	}

	_, err = s.client.KnownError.UpdateOneID(id).
		SetOccurrenceCount(errEnt.OccurrenceCount + 1).
		Save(ctx)
	return err
}

// MatchKnownErrorBySymptoms 根据症状匹配已知错误
func (s *KnownErrorService) MatchKnownErrorBySymptoms(ctx context.Context, tenantID int, symptoms string) (*ent.KnownError, error) {
	candidates, err := s.client.KnownError.Query().
		Where(knownerror.TenantID(tenantID), knownerror.Status("active")).
		All(ctx)
	if err != nil {
		return nil, err
	}

	for _, e := range candidates {
		if e.Symptoms != "" && contains(e.Symptoms, symptoms) {
			if err := s.IncrementOccurrenceCount(ctx, e.ID); err != nil {
				s.logger.Warnw("failed to increment occurrence count", "knownErrorID", e.ID, "error", err)
			}
			return e, nil
		}
	}

	return nil, nil
}

// LinkKnownErrorToProblem 关联已知错误到问题
func (s *KnownErrorService) LinkKnownErrorToProblem(ctx context.Context, knownErrorID, problemID, tenantID int) error {
	problemExists, err := s.client.Problem.Query().
		Where(problem.IDEQ(problemID), problem.TenantIDEQ(tenantID), problem.DeletedAtIsNil()).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !problemExists {
		return fmt.Errorf("problem not found")
	}
	updated, err := s.client.KnownError.Update().
		Where(knownerror.IDEQ(knownErrorID), knownerror.TenantIDEQ(tenantID)).
		AddProblemIDs(problemID).
		Save(ctx)
	if err != nil {
		return err
	}
	if updated != 1 {
		return fmt.Errorf("known error not found")
	}
	return nil
}

// CreateKnowledgeArticleFromKnownError 从已知错误创建知识库文章
func (s *KnownErrorService) CreateKnowledgeArticleFromKnownError(ctx context.Context, knownErrorID, authorID int) (*ent.KnowledgeArticle, error) {
	knownError, err := s.client.KnownError.Get(ctx, knownErrorID)
	if err != nil {
		return nil, err
	}

	content := fmt.Sprintf(
		`## 症状
%s

## 根本原因
%s

## 临时解决方案
%s

## 永久解决方案
%s`,
		knownError.Symptoms,
		knownError.RootCause,
		knownError.Workaround,
		knownError.Resolution,
	)

	return s.client.KnowledgeArticle.Create().
		SetTitle(knownError.Title).
		SetContent(content).
		SetCategory(knownError.Category).
		SetIsPublished(false).
		SetAuthorID(authorID).
		SetTenantID(knownError.TenantID).
		Save(ctx)
}
