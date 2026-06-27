package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/sladefinition"

	"go.uber.org/zap"
)

// SLATemplate SLA模板定义（开箱即用）
type SLATemplate struct {
	Key             string                 `json:"key"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	ServiceType     string                 `json:"serviceType"`
	Priority        string                 `json:"priority"`
	ResponseTime    int                    `json:"responseTime"`   // 分钟
	ResolutionTime  int                    `json:"resolutionTime"` // 分钟
	BusinessHours   map[string]interface{} `json:"businessHours"`
	EscalationRules map[string]interface{} `json:"escalationRules"`
	Conditions      map[string]interface{} `json:"conditions"`
	Industry        string                 `json:"industry"` // incident/change/service_request
	Recommended     bool                   `json:"recommended"`
}

// TemplateInstallResult 模板安装结果
type TemplateInstallResult struct {
	TemplateKey     string `json:"templateKey"`
	SLADefinitionID int    `json:"slaDefinitionId"`
	Created         bool   `json:"created"`
	WasAlreadyExist bool   `json:"wasAlreadyExist"`
	Message         string `json:"message"`
}

// SLATemplateService SLA模板服务
type SLATemplateService struct {
	client *ent.Client
	logger *zap.SugaredLogger

	// 模板目录（不可变）
	catalog map[string]SLATemplate
	order   []string

	// 安装状态缓存（key=tenantID:templateKey）
	mu        sync.RWMutex
	installed map[string]int // tenantID + ":" + templateKey -> SLADefinitionID
}

// NewSLATemplateService 创建SLA模板服务
func NewSLATemplateService(client *ent.Client, logger *zap.SugaredLogger) *SLATemplateService {
	s := &SLATemplateService{
		client:    client,
		logger:    logger,
		catalog:   make(map[string]SLATemplate),
		installed: make(map[string]int),
	}
	s.initCatalog()
	return s
}

// initCatalog 初始化模板目录（6 个开箱即用模板）
func (s *SLATemplateService) initCatalog() {
	templates := []SLATemplate{
		{
			Key:            "incident_p1_critical",
			Name:           "事件 P1 紧急",
			Description:    "适用于业务完全中断的紧急事件，要求 15 分钟内响应、4 小时内解决",
			ServiceType:    "incident",
			Priority:       "critical",
			ResponseTime:   15,
			ResolutionTime: 240,
			BusinessHours: map[string]interface{}{
				"timezone":        "Asia/Shanghai",
				"workdays":        []int{1, 2, 3, 4, 5},
				"work_hour_start": 0,
				"work_hour_end":   24,
				"is_24_7":         true,
			},
			EscalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{"level": 1, "threshold_minutes": 15, "notify_roles": []string{"team_lead"}},
					{"level": 2, "threshold_minutes": 30, "notify_roles": []string{"manager"}},
					{"level": 3, "threshold_minutes": 60, "notify_roles": []string{"director"}},
				},
			},
			Conditions: map[string]interface{}{
				"ticket_type":   []string{"incident"},
				"priority":      []string{"critical"},
				"customer_tier": []string{"vip", "enterprise"},
			},
			Industry:    "incident",
			Recommended: true,
		},
		{
			Key:            "incident_p2_high",
			Name:           "事件 P2 高",
			Description:    "适用于影响部分用户的高优先级事件，要求 1 小时内响应、8 小时内解决",
			ServiceType:    "incident",
			Priority:       "high",
			ResponseTime:   60,
			ResolutionTime: 480,
			BusinessHours: map[string]interface{}{
				"timezone":        "Asia/Shanghai",
				"workdays":        []int{1, 2, 3, 4, 5},
				"work_hour_start": 9,
				"work_hour_end":   18,
				"is_24_7":         false,
			},
			EscalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{"level": 1, "threshold_minutes": 60, "notify_roles": []string{"team_lead"}},
					{"level": 2, "threshold_minutes": 240, "notify_roles": []string{"manager"}},
				},
			},
			Conditions: map[string]interface{}{
				"ticket_type": []string{"incident"},
				"priority":    []string{"high"},
			},
			Industry:    "incident",
			Recommended: true,
		},
		{
			Key:            "incident_p3_medium",
			Name:           "事件 P3 中",
			Description:    "适用于影响少数用户的中优先级事件，要求 4 小时内响应、24 小时内解决",
			ServiceType:    "incident",
			Priority:       "medium",
			ResponseTime:   240,
			ResolutionTime: 1440,
			BusinessHours: map[string]interface{}{
				"timezone":        "Asia/Shanghai",
				"workdays":        []int{1, 2, 3, 4, 5},
				"work_hour_start": 9,
				"work_hour_end":   18,
				"is_24_7":         false,
			},
			EscalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{"level": 1, "threshold_minutes": 240, "notify_roles": []string{"team_lead"}},
				},
			},
			Conditions: map[string]interface{}{
				"ticket_type": []string{"incident"},
				"priority":    []string{"medium"},
			},
			Industry:    "incident",
			Recommended: true,
		},
		{
			Key:            "change_normal",
			Name:           "变更 - 普通",
			Description:    "适用于风险较低的常规变更，要求 4 小时内审批、24 小时内实施",
			ServiceType:    "change",
			Priority:       "medium",
			ResponseTime:   240,
			ResolutionTime: 1440,
			BusinessHours: map[string]interface{}{
				"timezone":        "Asia/Shanghai",
				"workdays":        []int{1, 2, 3, 4, 5},
				"work_hour_start": 9,
				"work_hour_end":   18,
				"is_24_7":         false,
			},
			EscalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{"level": 1, "threshold_minutes": 240, "notify_roles": []string{"change_manager"}},
				},
			},
			Conditions: map[string]interface{}{
				"ticket_type": []string{"change"},
				"risk_level":  []string{"low", "medium"},
			},
			Industry:    "change",
			Recommended: true,
		},
		{
			Key:            "change_emergency",
			Name:           "变更 - 紧急",
			Description:    "适用于紧急修复变更，要求 30 分钟内审批、4 小时内实施",
			ServiceType:    "change",
			Priority:       "critical",
			ResponseTime:   30,
			ResolutionTime: 240,
			BusinessHours: map[string]interface{}{
				"timezone": "Asia/Shanghai",
				"is_24_7":  true,
			},
			EscalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{"level": 1, "threshold_minutes": 30, "notify_roles": []string{"change_manager", "team_lead"}},
					{"level": 2, "threshold_minutes": 60, "notify_roles": []string{"director"}},
				},
			},
			Conditions: map[string]interface{}{
				"ticket_type": []string{"change"},
				"risk_level":  []string{"critical", "high"},
			},
			Industry:    "change",
			Recommended: true,
		},
		{
			Key:            "service_request_standard",
			Name:           "服务请求 - 标准",
			Description:    "适用于标准服务请求（账号开通、权限申请等），要求 8 小时内响应、72 小时内解决",
			ServiceType:    "service_request",
			Priority:       "low",
			ResponseTime:   480,
			ResolutionTime: 4320,
			BusinessHours: map[string]interface{}{
				"timezone":        "Asia/Shanghai",
				"workdays":        []int{1, 2, 3, 4, 5},
				"work_hour_start": 9,
				"work_hour_end":   18,
				"is_24_7":         false,
			},
			EscalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{"level": 1, "threshold_minutes": 480, "notify_roles": []string{"team_lead"}},
					{"level": 2, "threshold_minutes": 2880, "notify_roles": []string{"manager"}},
				},
			},
			Conditions: map[string]interface{}{
				"ticket_type": []string{"service_request"},
			},
			Industry:    "service_request",
			Recommended: false,
		},
	}

	for _, t := range templates {
		s.catalog[t.Key] = t
		s.order = append(s.order, t.Key)
	}
}

// ListTemplates 列出所有预置模板
func (s *SLATemplateService) ListTemplates() []SLATemplate {
	out := make([]SLATemplate, 0, len(s.order))
	for _, key := range s.order {
		if t, ok := s.catalog[key]; ok {
			out = append(out, t)
		}
	}
	return out
}

// GetTemplate 获取指定模板
func (s *SLATemplateService) GetTemplate(key string) (*SLATemplate, error) {
	t, ok := s.catalog[key]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", key)
	}
	return &t, nil
}

// InstallTemplate 将模板安装到租户（生成 SLADefinition）
//
// 幂等设计：同一租户对同一模板多次 install 不会产生重复记录；
// 已存在则返回 wasAlreadyExist=true。
func (s *SLATemplateService) InstallTemplate(ctx context.Context, key string, tenantID int) (*TemplateInstallResult, error) {
	tmpl, err := s.GetTemplate(key)
	if err != nil {
		return nil, err
	}

	// 检查是否已存在同 key 的 SLA Definition（按 name 精确匹配）
	existing, err := s.client.SLADefinition.Query().
		Where(
			sladefinition.TenantIDEQ(tenantID),
			sladefinition.NameEQ(fmt.Sprintf("[Template] %s", tmpl.Name)),
		).
		First(ctx)
	if err == nil && existing != nil {
		s.mu.Lock()
		s.installed[fmt.Sprintf("%d:%s", tenantID, key)] = existing.ID
		s.mu.Unlock()
		s.logger.Infow("SLA template already installed", "template", key, "tenant_id", tenantID, "sla_definition_id", existing.ID)
		return &TemplateInstallResult{
			TemplateKey:     key,
			SLADefinitionID: existing.ID,
			Created:         false,
			WasAlreadyExist: true,
			Message:         "该模板已安装过，已返回现有 SLA Definition",
		}, nil
	}
	if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to check existing template installation: %w", err)
	}

	// 创建 SLADefinition
	created, err := s.client.SLADefinition.Create().
		SetTenantID(tenantID).
		SetName(fmt.Sprintf("[Template] %s", tmpl.Name)).
		SetDescription(fmt.Sprintf("[Template] %s - %s", tmpl.Key, tmpl.Description)).
		SetServiceType(tmpl.ServiceType).
		SetPriority(tmpl.Priority).
		SetResponseTime(tmpl.ResponseTime).
		SetResolutionTime(tmpl.ResolutionTime).
		SetBusinessHours(tmpl.BusinessHours).
		SetEscalationRules(tmpl.EscalationRules).
		SetConditions(tmpl.Conditions).
		SetIsActive(true).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to install SLA template", "template", key, "tenant_id", tenantID, "error", err)
		return nil, fmt.Errorf("failed to install SLA template: %w", err)
	}

	s.mu.Lock()
	s.installed[fmt.Sprintf("%d:%s", tenantID, key)] = created.ID
	s.mu.Unlock()

	s.logger.Infow(
		"SLA template installed successfully",
		"template", key,
		"tenant_id", tenantID,
		"sla_definition_id", created.ID,
	)
	return &TemplateInstallResult{
		TemplateKey:     key,
		SLADefinitionID: created.ID,
		Created:         true,
		WasAlreadyExist: false,
		Message:         fmt.Sprintf("模板 %s 已成功安装", tmpl.Name),
	}, nil
}

// IsInstalled 检查模板是否已安装到租户
func (s *SLATemplateService) IsInstalled(tenantID int, key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.installed[fmt.Sprintf("%d:%s", tenantID, key)]
	return ok
}
