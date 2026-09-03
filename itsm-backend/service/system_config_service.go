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
		Where(systemconfig.KeyEQ(req.Key), systemconfig.DeletedAtIsNil()).
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
		Where(systemconfig.ID(id), systemconfig.DeletedAtIsNil()).
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
		Where(systemconfig.KeyEQ(key), systemconfig.DeletedAtIsNil()).
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

// ListSystemConfigs 获取系统配置列表。
// 额外保证：当前租户没有任何配置时，自动补齐全部默认配置键（懒加载）
// 后再返回，避免前端首次进入 /admin/system-config 看到空表单。
func (s *SystemConfigService) ListSystemConfigs(ctx context.Context, tenantID int, category string, page, pageSize int) ([]*ent.SystemConfig, int, error) {
	total, err := s.client.SystemConfig.Query().
		Where(systemconfig.TenantIDEQ(tenantID), systemconfig.DeletedAtIsNil()).
		Count(ctx)
	if err != nil {
		s.logger.Errorf("获取配置总数失败: %v", err)
		return nil, 0, fmt.Errorf("获取配置总数失败: %w", err)
	}

	// 懒加载首次默认配置（与 InitDefaultConfigs 端点等价，但不要求 UI 手动触发）
	if total == 0 {
		if err := s.InitDefaultConfigs(ctx, tenantID); err != nil {
			s.logger.Warnf("懒加载默认配置失败: %v", err)
		}
		total, err = s.client.SystemConfig.Query().
			Where(systemconfig.TenantIDEQ(tenantID), systemconfig.DeletedAtIsNil()).
			Count(ctx)
		if err != nil {
			s.logger.Errorf("重新统计配置总数失败: %v", err)
			return nil, 0, fmt.Errorf("重新统计配置总数失败: %w", err)
		}
	}

	query := s.client.SystemConfig.Query().
		Where(systemconfig.TenantIDEQ(tenantID), systemconfig.DeletedAtIsNil())

	if category != "" {
		query = query.Where(systemconfig.CategoryEQ(category))
	}

	total, err = query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count filtered system configs: %w", err)
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
		Where(systemconfig.ID(id), systemconfig.DeletedAtIsNil()).
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
		Where(systemconfig.TenantIDEQ(tenantID), systemconfig.DeletedAtIsNil()).
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
			Where(systemconfig.KeyEQ(cfg.Key), systemconfig.DeletedAtIsNil()).
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
		Where(systemconfig.ID(id), systemconfig.DeletedAtIsNil()).
		Where(systemconfig.TenantIDEQ(tenantID)).
		Exist(ctx)
	if err != nil {
		s.logger.Errorf("检查配置失败: %v", err)
		return fmt.Errorf("检查配置失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("配置不存在: %d", id)
	}

	_, err = s.client.SystemConfig.UpdateOneID(id).
		Where(systemconfig.TenantIDEQ(tenantID), systemconfig.DeletedAtIsNil()).
		SetDeletedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorf("删除系统配置失败: %v", err)
		return fmt.Errorf("删除系统配置失败: %w", err)
	}

	return nil
}

// InitDefaultConfigs 初始化默认配置。
// 覆盖前端 /admin/system-config 表单的 24 个键，避免首次进入页面看到空表单。
// 新增配置键必须同时在本表与前端 SystemConfiguration 表单中声明。
func (s *SystemConfigService) InitDefaultConfigs(ctx context.Context, tenantID int) error {
	defaultConfigs := []dto.SystemConfigRequest{
		// —— 通用设置 ——
		{Key: "systemName", Value: "ITSM系统", ValueType: "string", Category: "general", Description: "系统名称"},
		{Key: "systemUrl", Value: "http://localhost:3000", ValueType: "string", Category: "general", Description: "系统URL"},
		{Key: "timezone", Value: "Asia/Shanghai", ValueType: "string", Category: "general", Description: "时区"},
		{Key: "language", Value: "zh-CN", ValueType: "string", Category: "general", Description: "语言"},
		{Key: "dateFormat", Value: "YYYY-MM-DD", ValueType: "string", Category: "general", Description: "日期格式"},
		{Key: "timeFormat", Value: "24h", ValueType: "string", Category: "general", Description: "时间格式"},
		// —— 会话设置 ——
		{Key: "sessionTimeout", Value: "30", ValueType: "number", Category: "session", Description: "会话超时时间(分钟)"},
		// —— 上传设置 ——
		{Key: "maxFileSize", Value: "10", ValueType: "number", Category: "upload", Description: "最大文件大小(MB)"},
		{Key: "allowedFileTypes", Value: ".pdf,.doc,.docx,.xls,.xlsx,.jpg,.png,.gif", ValueType: "string", Category: "upload", Description: "允许的文件类型"},
		// —— 安全设置：密码策略 ——
		{Key: "passwordMinLength", Value: "8", ValueType: "number", Category: "security", Description: "密码最小长度"},
		{Key: "passwordRequireUppercase", Value: "true", ValueType: "boolean", Category: "security", Description: "需要大写字母"},
		{Key: "passwordRequireLowercase", Value: "true", ValueType: "boolean", Category: "security", Description: "需要小写字母"},
		{Key: "passwordRequireNumbers", Value: "true", ValueType: "boolean", Category: "security", Description: "需要数字"},
		{Key: "passwordRequireSpecialChars", Value: "false", ValueType: "boolean", Category: "security", Description: "需要特殊字符"},
		// —— 安全设置：账户安全 ——
		{Key: "loginMaxAttempts", Value: "5", ValueType: "number", Category: "security", Description: "登录失败次数限制"},
		{Key: "accountLockoutDuration", Value: "30", ValueType: "number", Category: "security", Description: "账户锁定时间(分钟)"},
		{Key: "enable2FA", Value: "false", ValueType: "boolean", Category: "security", Description: "启用双因素认证"},
		// —— 邮件设置：SMTP ——
		{Key: "smtpHost", Value: "smtp.example.com", ValueType: "string", Category: "email", Description: "SMTP服务器"},
		{Key: "smtpPort", Value: "465", ValueType: "number", Category: "email", Description: "SMTP端口"},
		{Key: "smtpUsername", Value: "noreply@example.com", ValueType: "string", Category: "email", Description: "SMTP用户名"},
		{Key: "smtpPassword", Value: "", ValueType: "string", Category: "email", Description: "SMTP密码"},
		{Key: "smtpEnableSSL", Value: "true", ValueType: "boolean", Category: "email", Description: "启用SSL/TLS"},
		// —— 邮件设置：模板 ——
		{Key: "emailFrom", Value: "noreply@example.com", ValueType: "string", Category: "email", Description: "发件人邮箱"},
		{Key: "systemNotificationTemplate", Value: "您好 {username}：\n您有一条新的系统通知：{message}\n— ITSM 团队", ValueType: "string", Category: "email", Description: "系统通知模板"},
	}

	for _, cfg := range defaultConfigs {
		// 检查是否已存在
		exists, err := s.client.SystemConfig.Query().
			Where(systemconfig.KeyEQ(cfg.Key), systemconfig.DeletedAtIsNil()).
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
