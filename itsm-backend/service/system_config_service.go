package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/systemconfig"

	"go.uber.org/zap"
)

type SystemConfigService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func NewSystemConfigService(client *ent.Client, logger *zap.SugaredLogger) *SystemConfigService {
	return &SystemConfigService{
		client: client,
		logger: logger,
	}
}

// CreateSystemConfig 创建系统配置
func (s *SystemConfigService) CreateSystemConfig(ctx context.Context, req *dto.SystemConfigRequest, tenantID int) (*ent.SystemConfig, error) {
	// 检查key是否已存在
	exists, err := s.client.SystemConfig.Query().
		Where(systemconfig.KeyEQ(req.Key)).
		Where(systemconfig.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorf("检查配置key失败: %v", err)
		return nil, fmt.Errorf("检查配置key失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("配置key已存在: %s", req.Key)
	}

	config, err := s.client.SystemConfig.Create().
		SetKey(req.Key).
		SetValue(req.Value).
		SetValueType(req.ValueType).
		SetCategory(req.Category).
		SetDescription(req.Description).
		SetTenantID(tenantID).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorf("创建系统配置失败: %v", err)
		return nil, fmt.Errorf("创建系统配置失败: %w", err)
	}

	return config, nil
}

// GetSystemConfig 获取系统配置
func (s *SystemConfigService) GetSystemConfig(ctx context.Context, id int, tenantID int) (*ent.SystemConfig, error) {
	config, err := s.client.SystemConfig.Query().
		Where(systemconfig.ID(id)).
		Where(systemconfig.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("配置不存在: %d", id)
		}
		s.logger.Errorf("获取系统配置失败: %v", err)
		return nil, fmt.Errorf("获取系统配置失败: %w", err)
	}
	return config, nil
}

// GetSystemConfigByKey 根据key获取配置
func (s *SystemConfigService) GetSystemConfigByKey(ctx context.Context, key string, tenantID int) (*ent.SystemConfig, error) {
	config, err := s.client.SystemConfig.Query().
		Where(systemconfig.KeyEQ(key)).
		Where(systemconfig.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("配置不存在: %s", key)
		}
		s.logger.Errorf("获取系统配置失败: %v", err)
		return nil, fmt.Errorf("获取系统配置失败: %w", err)
	}
	return config, nil
}

// ListSystemConfigs 获取系统配置列表
func (s *SystemConfigService) ListSystemConfigs(ctx context.Context, tenantID int, category string, page, pageSize int) ([]*ent.SystemConfig, int, error) {
	query := s.client.SystemConfig.Query().
		Where(systemconfig.TenantIDEQ(tenantID))

	if category != "" {
		query = query.Where(systemconfig.CategoryEQ(category))
	}

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorf("获取配置总数失败: %v", err)
		return nil, 0, fmt.Errorf("获取配置总数失败: %w", err)
	}

	configs, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(ent.Desc(systemconfig.FieldUpdatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorf("获取配置列表失败: %v", err)
		return nil, 0, fmt.Errorf("获取配置列表失败: %w", err)
	}

	return configs, total, nil
}

// UpdateSystemConfig 更新系统配置
func (s *SystemConfigService) UpdateSystemConfig(ctx context.Context, id int, req *dto.UpdateSystemConfigRequest, tenantID int) (*ent.SystemConfig, error) {
	// 检查配置是否存在
	exists, err := s.client.SystemConfig.Query().
		Where(systemconfig.ID(id)).
		Where(systemconfig.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorf("检查配置失败: %v", err)
		return nil, fmt.Errorf("检查配置失败: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("配置不存在: %d", id)
	}

	update := s.client.SystemConfig.UpdateOneID(id).
		SetUpdatedAt(time.Now())

	if req.Value != "" {
		update = update.SetValue(req.Value)
	}
	if req.ValueType != "" {
		update = update.SetValueType(req.ValueType)
	}
	if req.Description != "" {
		update = update.SetDescription(req.Description)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorf("更新系统配置失败: %v", err)
		return nil, fmt.Errorf("更新系统配置失败: %w", err)
	}

	return updated, nil
}

// BatchUpdateSystemConfigs 批量更新系统配置
func (s *SystemConfigService) BatchUpdateSystemConfigs(ctx context.Context, configs []dto.UpdateSystemConfigRequest, tenantID int) ([]*ent.SystemConfig, error) {
	results := make([]*ent.SystemConfig, 0, len(configs))

	for _, cfg := range configs {
		// 尝试查找现有配置
		existing, err := s.client.SystemConfig.Query().
			Where(systemconfig.KeyEQ(cfg.Key)).
			Where(systemconfig.TenantIDEQ(tenantID)).
			First(ctx)

		if err == nil && existing != nil {
			// 更新现有配置
			updated, err := s.client.SystemConfig.UpdateOneID(existing.ID).
				SetValue(cfg.Value).
				SetValueType(cfg.ValueType).
				SetDescription(cfg.Description).
				SetUpdatedAt(time.Now()).
				Save(ctx)
			if err != nil {
				s.logger.Errorf("更新配置失败: %v", err)
				continue
			}
			results = append(results, updated)
		} else {
			// 创建新配置
			created, err := s.client.SystemConfig.Create().
				SetKey(cfg.Key).
				SetValue(cfg.Value).
				SetValueType(cfg.ValueType).
				SetDescription(cfg.Description).
				SetCategory("general").
				SetTenantID(tenantID).
				SetCreatedAt(time.Now()).
				SetUpdatedAt(time.Now()).
				Save(ctx)
			if err != nil {
				s.logger.Errorf("创建配置失败: %v", err)
				continue
			}
			results = append(results, created)
		}
	}

	return results, nil
}

// DeleteSystemConfig 删除系统配置
func (s *SystemConfigService) DeleteSystemConfig(ctx context.Context, id int, tenantID int) error {
	exists, err := s.client.SystemConfig.Query().
		Where(systemconfig.ID(id)).
		Where(systemconfig.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorf("检查配置失败: %v", err)
		return fmt.Errorf("检查配置失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("配置不存在: %d", id)
	}

	err = s.client.SystemConfig.DeleteOneID(id).Exec(ctx)
	if err != nil {
		s.logger.Errorf("删除系统配置失败: %v", err)
		return fmt.Errorf("删除系统配置失败: %w", err)
	}

	return nil
}

// InitDefaultConfigs 初始化默认配置
func (s *SystemConfigService) InitDefaultConfigs(ctx context.Context, tenantID int) error {
	defaultConfigs := []dto.SystemConfigRequest{
		{Key: "systemName", Value: "ITSM系统", ValueType: "string", Category: "general", Description: "系统名称"},
		{Key: "systemUrl", Value: "http://localhost:3000", ValueType: "string", Category: "general", Description: "系统URL"},
		{Key: "timezone", Value: "Asia/Shanghai", ValueType: "string", Category: "general", Description: "时区"},
		{Key: "language", Value: "zh-CN", ValueType: "string", Category: "general", Description: "语言"},
		{Key: "dateFormat", Value: "YYYY-MM-DD", ValueType: "string", Category: "general", Description: "日期格式"},
		{Key: "sessionTimeout", Value: "30", ValueType: "number", Category: "session", Description: "会话超时时间(分钟)"},
		{Key: "maxFileSize", Value: "10", ValueType: "number", Category: "upload", Description: "最大文件大小(MB)"},
		{Key: "passwordMinLength", Value: "6", ValueType: "number", Category: "security", Description: "密码最小长度"},
		{Key: "loginMaxAttempts", Value: "5", ValueType: "number", Category: "security", Description: "登录失败次数限制"},
	}

	for _, cfg := range defaultConfigs {
		// 检查是否已存在
		exists, err := s.client.SystemConfig.Query().
			Where(systemconfig.KeyEQ(cfg.Key)).
			Where(systemconfig.TenantIDEQ(tenantID)).
			Exist(ctx)
		if err != nil {
			s.logger.Errorf("检查配置失败: %v", err)
			continue
		}
		if exists {
			continue
		}

		// 创建配置
		_, err = s.client.SystemConfig.Create().
			SetKey(cfg.Key).
			SetValue(cfg.Value).
			SetValueType(cfg.ValueType).
			SetCategory(cfg.Category).
			SetDescription(cfg.Description).
			SetTenantID(tenantID).
			SetCreatedAt(time.Now()).
			SetUpdatedAt(time.Now()).
			Save(ctx)
		if err != nil {
			s.logger.Errorf("创建默认配置失败: %v", err)
		}
	}

	return nil
}
