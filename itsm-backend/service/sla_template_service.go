package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

	// installLocks 串行化同租户同模板的并发安装。进程内互斥挡不住多副本，
	// 跨副本去重需要 sla_definitions 上的 (tenant_id, template_key) 唯一约束。
	lockMu       sync.Mutex
	installLocks map[string]*sync.Mutex
}

// NewSLATemplateService 创建SLA模板服务
func NewSLATemplateService(client *ent.Client, logger *zap.SugaredLogger) *SLATemplateService {
	s := &SLATemplateService{
		client:       client,
		logger:       logger,
		catalog:      make(map[string]SLATemplate),
		installLocks: make(map[string]*sync.Mutex),
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
				// 键名与 parseBusinessHoursConfig 解析器对齐（ticket_sla_service.go:366+）；
				// 旧键名 workdays/work_hour_start/is_24_7 被忽略导致 24×7 模板 fallback 9-18。
				"time_zone":  "Asia/Shanghai",
				"work_days":  []int{1, 2, 3, 4, 5},
				"start_time": "00:00",
				"end_time":   "23:59",
				"is_24_7":    true,
			},
			EscalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{"level": 1, "threshold_minutes": 15, "notify_roles": []string{"manager"}},
					{"level": 2, "threshold_minutes": 30, "notify_roles": []string{"it_admin"}},
					{"level": 3, "threshold_minutes": 60, "notify_roles": []string{"admin"}},
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
				"time_zone":  "Asia/Shanghai",
				"work_days":  []int{1, 2, 3, 4, 5},
				"start_time": "09:00",
				"end_time":   "18:00",
				"is_24_7":    false,
			},
			EscalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{"level": 1, "threshold_minutes": 60, "notify_roles": []string{"manager"}},
					{"level": 2, "threshold_minutes": 240, "notify_roles": []string{"it_admin"}},
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
				"time_zone":  "Asia/Shanghai",
				"work_days":  []int{1, 2, 3, 4, 5},
				"start_time": "09:00",
				"end_time":   "18:00",
				"is_24_7":    false,
			},
			EscalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{"level": 1, "threshold_minutes": 240, "notify_roles": []string{"manager"}},
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
				"time_zone":  "Asia/Shanghai",
				"work_days":  []int{1, 2, 3, 4, 5},
				"start_time": "09:00",
				"end_time":   "18:00",
				"is_24_7":    false,
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
				"time_zone": "Asia/Shanghai",
				"is_24_7":   true,
			},
			EscalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{"level": 1, "threshold_minutes": 30, "notify_roles": []string{"change_manager", "manager"}},
					{"level": 2, "threshold_minutes": 60, "notify_roles": []string{"it_admin"}},
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
				"time_zone":  "Asia/Shanghai",
				"work_days":  []int{1, 2, 3, 4, 5},
				"start_time": "09:00",
				"end_time":   "18:00",
				"is_24_7":    false,
			},
			EscalationRules: map[string]interface{}{
				"levels": []map[string]interface{}{
					{"level": 1, "threshold_minutes": 480, "notify_roles": []string{"manager"}},
					{"level": 2, "threshold_minutes": 2880, "notify_roles": []string{"it_admin"}},
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

// ErrSLATemplateNotFound 让 handler 能把「模板不存在」映射成 404，而不是把一切错误
// 都说成请求参数错误。
var ErrSLATemplateNotFound = errors.New("sla template not found")

const (
	// 安装生成的定义以此前缀命名，用来把模板产物与租户自建 SLA 区分开。
	slaTemplateNamePrefix = "[Template] "
	// templateKeyMarker 记在定义的 conditions 里：名称与描述都可能被管理员改掉，
	// 只靠字符串前缀去重会在改名后重复安装同一模板。
	templateKeyMarker = "templateKey"
)

// SLATemplateInstallInfo 是模板在某个租户的安装事实，来源是数据库而不是进程内存。
type SLATemplateInstallInfo struct {
	SLADefinitionID int
	InstalledAt     time.Time
}

// templateKeyOf 优先读 conditions 里的标记，回落到 description 的
// 「[Template] <key> - 」约定（标记引入之前装的历史数据）。
func templateKeyOf(def *ent.SLADefinition) string {
	if marker, ok := def.Conditions[templateKeyMarker].(string); ok && marker != "" {
		return marker
	}
	rest := strings.TrimPrefix(def.Description, slaTemplateNamePrefix)
	if rest == def.Description {
		return ""
	}
	if idx := strings.Index(rest, " - "); idx > 0 {
		return rest[:idx]
	}
	return ""
}

// withTemplateKey 复制 conditions 后写入标记。目录里的 map 是跨租户共享的只读数据，
// 直接塞标记会让后续安装读到上一个租户的值。
func withTemplateKey(src map[string]interface{}, key string) map[string]interface{} {
	out := make(map[string]interface{}, len(src)+1)
	for k, v := range src {
		out[k] = v
	}
	out[templateKeyMarker] = key
	return out
}

// lockInstall 串行化同租户同模板的并发安装。进程内互斥只挡得住单副本的重复点击，
// 跨副本去重需要 sla_definitions 上的 (tenant_id, template_key) 唯一约束。
func (s *SLATemplateService) lockInstall(tenantID int, key string) func() {
	lockKey := fmt.Sprintf("%d:%s", tenantID, key)
	s.lockMu.Lock()
	mu, ok := s.installLocks[lockKey]
	if !ok {
		mu = &sync.Mutex{}
		s.installLocks[lockKey] = mu
	}
	s.lockMu.Unlock()
	mu.Lock()
	return mu.Unlock
}

// GetTemplate 获取指定模板
func (s *SLATemplateService) GetTemplate(key string) (*SLATemplate, error) {
	t, ok := s.catalog[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrSLATemplateNotFound, key)
	}
	return &t, nil
}

// InstallStates 返回该租户每个在册模板的安装事实。列表展示与安装去重共用这套判定，
// 显示的状态和实际幂等行为不会分叉。
//
// 不能按名称前缀过滤：管理员改掉「[Template]」开头后，记录会从已安装视图里消失，
// 再点安装就产生第二份相同配置。
func (s *SLATemplateService) InstallStates(ctx context.Context, tenantID int) (map[string]SLATemplateInstallInfo, error) {
	states := make(map[string]SLATemplateInstallInfo, len(s.catalog))
	if s.client == nil {
		return states, nil
	}

	defs, err := s.client.SLADefinition.Query().
		Where(sladefinition.TenantIDEQ(tenantID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load installed sla templates: %w", err)
	}

	for _, def := range defs {
		key := templateKeyOf(def)
		if key == "" {
			continue
		}
		if _, ok := s.catalog[key]; !ok {
			continue
		}
		if existing, ok := states[key]; ok && !def.CreatedAt.After(existing.InstalledAt) {
			continue
		}
		states[key] = SLATemplateInstallInfo{SLADefinitionID: def.ID, InstalledAt: def.CreatedAt}
	}
	return states, nil
}

// InstallTemplate 将模板安装到租户（生成 SLADefinition）
//
// 幂等设计：同一租户对同一模板多次 install 不会产生重复记录；已存在则返回
// wasAlreadyExist=true。身份以写入 conditions 的 templateKey 标记为准，管理员改名
// 之后重装仍然会被识别成同一模板。
func (s *SLATemplateService) InstallTemplate(ctx context.Context, key string, tenantID int) (*TemplateInstallResult, error) {
	tmpl, err := s.GetTemplate(key)
	if err != nil {
		return nil, err
	}
	if s.client == nil {
		return nil, errors.New("sla template service is not wired to a database client")
	}

	release := s.lockInstall(tenantID, key)
	defer release()

	states, err := s.InstallStates(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if state, ok := states[key]; ok {
		s.logger.Infow("SLA template already installed",
			"template", key, "tenant_id", tenantID, "sla_definition_id", state.SLADefinitionID)
		return &TemplateInstallResult{
			TemplateKey:     key,
			SLADefinitionID: state.SLADefinitionID,
			Created:         false,
			WasAlreadyExist: true,
			Message:         "该模板已安装过，已返回现有 SLA 定义",
		}, nil
	}

	now := time.Now()
	created, err := s.client.SLADefinition.Create().
		SetTenantID(tenantID).
		SetName(slaTemplateNamePrefix + tmpl.Name).
		SetDescription(fmt.Sprintf("%s%s - %s", slaTemplateNamePrefix, tmpl.Key, tmpl.Description)).
		SetServiceType(tmpl.ServiceType).
		SetPriority(tmpl.Priority).
		SetResponseTime(tmpl.ResponseTime).
		SetResolutionTime(tmpl.ResolutionTime).
		SetBusinessHours(tmpl.BusinessHours).
		SetEscalationRules(tmpl.EscalationRules).
		SetConditions(withTemplateKey(tmpl.Conditions, tmpl.Key)).
		SetIsActive(true).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to install SLA template", "template", key, "tenant_id", tenantID, "error", err)
		return nil, fmt.Errorf("failed to install sla template: %w", err)
	}

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
