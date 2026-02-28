package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/approvalchain"
	"itsm-backend/ent/schema"

	"go.uber.org/zap"
)

type ApprovalChainService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewApprovalChainService(client *ent.Client, logger *zap.SugaredLogger) *ApprovalChainService {
	return &ApprovalChainService{
		client: client,
		logger: logger,
	}
}

// CreateApprovalChain 创建审批链
func (s *ApprovalChainService) CreateApprovalChain(ctx context.Context, req *dto.ApprovalChainRequest, tenantID int) (*ent.ApprovalChain, error) {
	// Convert DTO steps to schema steps
	chainSteps := make([]schema.ApprovalChainStep, len(req.Chain))
	for i, step := range req.Chain {
		chainSteps[i] = schema.ApprovalChainStep{
			Level:      step.Level,
			ApproverID: step.ApproverID,
			Role:       step.Role,
			Name:       step.Name,
			IsRequired: step.IsRequired,
		}
	}

	status := "active"
	if req.Status != "" {
		status = req.Status
	}

	entity, err := s.client.ApprovalChain.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetEntityType(req.EntityType).
		SetChain(chainSteps).
		SetStatus(status).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorf("创建审批链失败: %v", err)
		return nil, fmt.Errorf("创建审批链失败: %w", err)
	}

	return entity, nil
}

// GetApprovalChain 获取审批链
func (s *ApprovalChainService) GetApprovalChain(ctx context.Context, id int, tenantID int) (*ent.ApprovalChain, error) {
	entity, err := s.client.ApprovalChain.Query().
		Where(approvalchain.ID(id)).
		Where(approvalchain.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("审批链不存在: %d", id)
		}
		s.logger.Errorf("获取审批链失败: %v", err)
		return nil, fmt.Errorf("获取审批链失败: %w", err)
	}
	return entity, nil
}

// ListApprovalChains 获取审批链列表
func (s *ApprovalChainService) ListApprovalChains(ctx context.Context, tenantID int, entityType string, status string, page, pageSize int) ([]*ent.ApprovalChain, int, error) {
	query := s.client.ApprovalChain.Query().
		Where(approvalchain.TenantIDEQ(tenantID))

	if entityType != "" {
		query = query.Where(approvalchain.EntityTypeEQ(entityType))
	}
	if status != "" {
		query = query.Where(approvalchain.StatusEQ(status))
	}

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorf("获取审批链总数失败: %v", err)
		return nil, 0, fmt.Errorf("获取审批链总数失败: %w", err)
	}

	entities, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(ent.Desc(approvalchain.FieldUpdatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorf("获取审批链列表失败: %v", err)
		return nil, 0, fmt.Errorf("获取审批链列表失败: %w", err)
	}

	return entities, total, nil
}

// UpdateApprovalChain 更新审批链
func (s *ApprovalChainService) UpdateApprovalChain(ctx context.Context, id int, req *dto.ApprovalChainRequest, tenantID int) (*ent.ApprovalChain, error) {
	// 检查是否存在
	exists, err := s.client.ApprovalChain.Query().
		Where(approvalchain.ID(id)).
		Where(approvalchain.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorf("检查审批链失败: %v", err)
		return nil, fmt.Errorf("检查审批链失败: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("审批链不存在: %d", id)
	}

	update := s.client.ApprovalChain.UpdateOneID(id).
		SetUpdatedAt(time.Now())

	if req.Name != "" {
		update = update.SetName(req.Name)
	}
	if req.Description != "" {
		update = update.SetDescription(req.Description)
	}
	if req.EntityType != "" {
		update = update.SetEntityType(req.EntityType)
	}
	if len(req.Chain) > 0 {
		chainSteps := make([]schema.ApprovalChainStep, len(req.Chain))
		for i, step := range req.Chain {
			chainSteps[i] = schema.ApprovalChainStep{
				Level:      step.Level,
				ApproverID: step.ApproverID,
				Role:       step.Role,
				Name:       step.Name,
				IsRequired: step.IsRequired,
			}
		}
		update = update.SetChain(chainSteps)
	}
	if req.Status != "" {
		update = update.SetStatus(req.Status)
	}

	entity, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorf("更新审批链失败: %v", err)
		return nil, fmt.Errorf("更新审批链失败: %w", err)
	}

	return entity, nil
}

// DeleteApprovalChain 删除审批链
func (s *ApprovalChainService) DeleteApprovalChain(ctx context.Context, id int, tenantID int) error {
	exists, err := s.client.ApprovalChain.Query().
		Where(approvalchain.ID(id)).
		Where(approvalchain.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorf("检查审批链失败: %v", err)
		return fmt.Errorf("检查审批链失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("审批链不存在: %d", id)
	}

	err = s.client.ApprovalChain.DeleteOneID(id).Exec(ctx)
	if err != nil {
		s.logger.Errorf("删除审批链失败: %v", err)
		return fmt.Errorf("删除审批链失败: %w", err)
	}

	return nil
}

// GetApprovalChainStats 获取审批链统计
func (s *ApprovalChainService) GetApprovalChainStats(ctx context.Context, tenantID int) (*dto.ApprovalChainStats, error) {
	chains, err := s.client.ApprovalChain.Query().
		Where(approvalchain.TenantIDEQ(tenantID)).
		All(ctx)
	if err != nil {
		s.logger.Errorf("获取审批链列表失败: %v", err)
		return nil, fmt.Errorf("获取审批链列表失败: %w", err)
	}

	total := len(chains)
	active := 0
	inactive := 0
	totalSteps := 0

	for _, chain := range chains {
		if chain.Status == "active" {
			active++
		} else {
			inactive++
		}

		// 直接使用 chain 字段获取步骤数
		if chain.Chain != nil {
			totalSteps += len(chain.Chain)
		}
	}

	avgSteps := 0.0
	if total > 0 {
		avgSteps = float64(totalSteps) / float64(total)
	}

	return &dto.ApprovalChainStats{
		Total:            total,
		Active:           active,
		Inactive:         inactive,
		TotalSteps:       totalSteps,
		AvgStepsPerChain: avgSteps,
	}, nil
}
