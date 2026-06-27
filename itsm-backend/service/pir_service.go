package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/changepir"

	"go.uber.org/zap"
)

// ChangePIRService 变更实施后审查服务
type ChangePIRService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewChangePIRService 创建PIR服务
func NewChangePIRService(client *ent.Client, logger *zap.SugaredLogger) *ChangePIRService {
	return &ChangePIRService{
		client: client,
		logger: logger,
	}
}

// CreatePIR 创建实施后审查
func (s *ChangePIRService) CreatePIR(ctx context.Context, req *dto.CreateChangePIRRequest, reviewerID, tenantID int) (*dto.ChangePIRResponse, error) {
	s.logger.Infow("Creating PIR", "change_id", req.ChangeID, "reviewer_id", reviewerID, "tenant_id", tenantID)

	// 验证变更存在且属于该租户
	changeEntity, err := s.client.Change.Query().
		Where(change.ID(req.ChangeID), change.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("变更不存在")
		}
		return nil, fmt.Errorf("获取变更失败: %w", err)
	}

	// 检查该变更是否已有PIR (使用HasChangeWith查询)
	existingPIR, err := s.client.ChangePIR.Query().
		Where(changepir.HasChangeWith(change.ID(req.ChangeID))).
		Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("检查PIR失败: %w", err)
	}
	if existingPIR {
		return nil, fmt.Errorf("该变更已存在PIR，请使用更新接口")
	}

	// 计算实际持续时间
	var durationMinutes int
	if req.ActualStartTime != nil && req.ActualEndTime != nil {
		durationMinutes = int(req.ActualEndTime.Sub(*req.ActualStartTime).Minutes())
	}

	// 创建PIR记录并关联变更
	pir, err := s.client.ChangePIR.Create().
		SetChange(changeEntity).
		SetOverallResult(req.OverallResult).
		SetObjectivesAchieved(req.ObjectivesAchieved).
		SetTenantID(tenantID).
		SetReviewDate(time.Now()).
		SetActualDurationMinutes(durationMinutes).
		SetRollbackPerformed(req.RollbackPerformed).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create PIR", "error", err, "change_id", req.ChangeID)
		return nil, fmt.Errorf("创建PIR失败: %w", err)
	}

	// 设置可选字段
	update := pir.Update()
	if req.SuccessSummary != nil {
		update.SetSuccessSummary(*req.SuccessSummary)
	}
	if req.IssuesEncountered != nil {
		update.SetIssuesEncountered(*req.IssuesEncountered)
	}
	if req.LessonsLearned != nil {
		update.SetLessonsLearned(*req.LessonsLearned)
	}
	if req.ImprovementRecommendations != nil {
		update.SetImprovementRecommendations(*req.ImprovementRecommendations)
	}
	if req.ActualStartTime != nil {
		update.SetActualStartTime(*req.ActualStartTime)
	}
	if req.ActualEndTime != nil {
		update.SetActualEndTime(*req.ActualEndTime)
	}
	if req.RollbackReason != nil {
		update.SetRollbackReason(*req.RollbackReason)
	}

	pir, err = update.Save(ctx)
	if err != nil {
		s.logger.Warnw("Failed to update optional fields", "error", err)
	}

	s.logger.Infow("PIR created successfully", "pir_id", pir.ID, "change_id", req.ChangeID)
	return s.buildPIRResponse(ctx, pir, changeEntity.Title, tenantID, req.ChangeID)
}

// GetPIR 获取PIR详情
func (s *ChangePIRService) GetPIR(ctx context.Context, id, tenantID int) (*dto.ChangePIRResponse, error) {
	pir, err := s.client.ChangePIR.Query().
		Where(changepir.ID(id), changepir.TenantID(tenantID)).
		WithChange().
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("PIR不存在")
		}
		return nil, fmt.Errorf("获取PIR失败: %w", err)
	}

	changeTitle := ""
	changeID := 0
	if pir.Edges.Change != nil {
		changeTitle = pir.Edges.Change.Title
		changeID = pir.Edges.Change.ID
	}

	return s.buildPIRResponseFull(ctx, pir, changeTitle, changeID)
}

// GetPIRByChange 获取变更关联的PIR
func (s *ChangePIRService) GetPIRByChange(ctx context.Context, changeID, tenantID int) (*dto.ChangePIRResponse, error) {
	pir, err := s.client.ChangePIR.Query().
		Where(changepir.TenantID(tenantID), changepir.HasChangeWith(change.ID(changeID))).
		WithChange().
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("该变更无PIR记录")
		}
		return nil, fmt.Errorf("获取PIR失败: %w", err)
	}

	changeTitle := ""
	if pir.Edges.Change != nil {
		changeTitle = pir.Edges.Change.Title
	}

	return s.buildPIRResponseFull(ctx, pir, changeTitle, changeID)
}

// ListPIRs 获取PIR列表
func (s *ChangePIRService) ListPIRs(ctx context.Context, tenantID int, page, pageSize int, result string) (*dto.ChangePIRListResponse, error) {
	query := s.client.ChangePIR.Query().
		Where(changepir.TenantID(tenantID))

	// 结果筛选
	if result != "" && result != "全部" {
		query = query.Where(changepir.OverallResult(result))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取PIR总数失败: %w", err)
	}

	// 分页查询
	pirs, err := query.
		WithChange().
		Order(ent.Desc(changepir.FieldReviewDate)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取PIR列表失败: %w", err)
	}

	// 构建响应列表
	items := make([]*dto.ChangePIRResponse, 0, len(pirs))
	for _, pir := range pirs {
		changeTitle := ""
		changeID := 0
		if pir.Edges.Change != nil {
			changeTitle = pir.Edges.Change.Title
			changeID = pir.Edges.Change.ID
		}
		response, err := s.buildPIRResponseFull(ctx, pir, changeTitle, changeID)
		if err != nil {
			s.logger.Warnw("Failed to build PIR response", "error", err, "pir_id", pir.ID)
			continue
		}
		items = append(items, response)
	}

	return &dto.ChangePIRListResponse{
		Total: total,
		Items: items,
	}, nil
}

// UpdatePIR 更新PIR
func (s *ChangePIRService) UpdatePIR(ctx context.Context, id int, req *dto.UpdateChangePIRRequest, tenantID int) (*dto.ChangePIRResponse, error) {
	// 获取PIR记录
	pir, err := s.client.ChangePIR.Query().
		Where(changepir.ID(id), changepir.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("PIR不存在")
		}
		return nil, fmt.Errorf("获取PIR失败: %w", err)
	}

	// 构建更新
	update := pir.Update()
	if req.OverallResult != nil {
		update.SetOverallResult(*req.OverallResult)
	}
	if req.ObjectivesAchieved != nil {
		update.SetObjectivesAchieved(*req.ObjectivesAchieved)
	}
	if req.SuccessSummary != nil {
		update.SetSuccessSummary(*req.SuccessSummary)
	}
	if req.IssuesEncountered != nil {
		update.SetIssuesEncountered(*req.IssuesEncountered)
	}
	if req.LessonsLearned != nil {
		update.SetLessonsLearned(*req.LessonsLearned)
	}
	if req.ImprovementRecommendations != nil {
		update.SetImprovementRecommendations(*req.ImprovementRecommendations)
	}

	pir, err = update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update PIR", "error", err, "pir_id", id)
		return nil, fmt.Errorf("更新PIR失败: %w", err)
	}

	s.logger.Infow("PIR updated successfully", "pir_id", id)

	// 获取变更标题
	changeTitle := ""
	changeID := 0
	if pir.Edges.Change != nil {
		changeTitle = pir.Edges.Change.Title
		changeID = pir.Edges.Change.ID
	}

	return s.buildPIRResponseFull(ctx, pir, changeTitle, changeID)
}

// DeletePIR 删除PIR
func (s *ChangePIRService) DeletePIR(ctx context.Context, id, tenantID int) error {
	// 检查PIR存在
	pir, err := s.client.ChangePIR.Query().
		Where(changepir.ID(id), changepir.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return fmt.Errorf("PIR不存在")
		}
		return fmt.Errorf("获取PIR失败: %w", err)
	}

	err = s.client.ChangePIR.DeleteOne(pir).Exec(ctx)
	if err != nil {
		return fmt.Errorf("删除PIR失败: %w", err)
	}

	s.logger.Infow("PIR deleted successfully", "pir_id", id)
	return nil
}

// buildPIRResponse 构建PIR响应（基础信息）
func (s *ChangePIRService) buildPIRResponse(ctx context.Context, pir *ent.ChangePIR, changeTitle string, tenantID, changeID int) (*dto.ChangePIRResponse, error) {
	// 转换指针字段
	var successSummary, issuesEncountered, lessonsLearned, improvementRecs, rollbackReason *string
	if pir.SuccessSummary != "" {
		successSummary = &pir.SuccessSummary
	}
	if pir.IssuesEncountered != "" {
		issuesEncountered = &pir.IssuesEncountered
	}
	if pir.LessonsLearned != "" {
		lessonsLearned = &pir.LessonsLearned
	}
	if pir.ImprovementRecommendations != "" {
		improvementRecs = &pir.ImprovementRecommendations
	}
	if pir.RollbackReason != "" {
		rollbackReason = &pir.RollbackReason
	}

	var actualStartTime, actualEndTime *time.Time
	if !pir.ActualStartTime.IsZero() {
		actualStartTime = &pir.ActualStartTime
	}
	if !pir.ActualEndTime.IsZero() {
		actualEndTime = &pir.ActualEndTime
	}

	return &dto.ChangePIRResponse{
		ID:                         pir.ID,
		ChangeID:                   changeID,
		ChangeTitle:                changeTitle,
		ReviewerID:                 pir.ReviewerID,
		ReviewerName:               "", // 从reviewer_id字段获取
		OverallResult:              pir.OverallResult,
		ObjectivesAchieved:         pir.ObjectivesAchieved,
		SuccessSummary:             successSummary,
		IssuesEncountered:          issuesEncountered,
		LessonsLearned:             lessonsLearned,
		ImprovementRecommendations: improvementRecs,
		ActualStartTime:            actualStartTime,
		ActualEndTime:              actualEndTime,
		ActualDurationMinutes:      pir.ActualDurationMinutes,
		RollbackPerformed:          pir.RollbackPerformed,
		RollbackReason:             rollbackReason,
		TenantID:                   tenantID,
		ReviewDate:                 pir.ReviewDate,
		CreatedAt:                  pir.CreatedAt,
		UpdatedAt:                  pir.UpdatedAt,
	}, nil
}

// buildPIRResponseFull 构建完整的PIR响应（包含关联信息）
func (s *ChangePIRService) buildPIRResponseFull(ctx context.Context, pir *ent.ChangePIR, changeTitle string, changeID int) (*dto.ChangePIRResponse, error) {
	// 转换指针字段
	var successSummary, issuesEncountered, lessonsLearned, improvementRecs, rollbackReason *string
	if pir.SuccessSummary != "" {
		successSummary = &pir.SuccessSummary
	}
	if pir.IssuesEncountered != "" {
		issuesEncountered = &pir.IssuesEncountered
	}
	if pir.LessonsLearned != "" {
		lessonsLearned = &pir.LessonsLearned
	}
	if pir.ImprovementRecommendations != "" {
		improvementRecs = &pir.ImprovementRecommendations
	}
	if pir.RollbackReason != "" {
		rollbackReason = &pir.RollbackReason
	}

	var actualStartTime, actualEndTime *time.Time
	if !pir.ActualStartTime.IsZero() {
		actualStartTime = &pir.ActualStartTime
	}
	if !pir.ActualEndTime.IsZero() {
		actualEndTime = &pir.ActualEndTime
	}

	return &dto.ChangePIRResponse{
		ID:                         pir.ID,
		ChangeID:                   changeID,
		ChangeTitle:                changeTitle,
		ReviewerID:                 pir.ReviewerID,
		ReviewerName:               "", // 从reviewer_id字段获取
		OverallResult:              pir.OverallResult,
		ObjectivesAchieved:         pir.ObjectivesAchieved,
		SuccessSummary:             successSummary,
		IssuesEncountered:          issuesEncountered,
		LessonsLearned:             lessonsLearned,
		ImprovementRecommendations: improvementRecs,
		ActualStartTime:            actualStartTime,
		ActualEndTime:              actualEndTime,
		ActualDurationMinutes:      pir.ActualDurationMinutes,
		RollbackPerformed:          pir.RollbackPerformed,
		RollbackReason:             rollbackReason,
		TenantID:                   pir.TenantID,
		ReviewDate:                 pir.ReviewDate,
		CreatedAt:                  pir.CreatedAt,
		UpdatedAt:                  pir.UpdatedAt,
	}, nil
}
