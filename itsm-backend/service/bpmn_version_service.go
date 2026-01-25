package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	// "itsm-backend/ent/processdeployment" // 暂时不使用，因为ProcessDeployment没有ProcessDefinitionID字段
)

// BPMNVersionService BPMN流程版本管理服务
type BPMNVersionService struct {
	client *ent.Client
}

// NewBPMNVersionService 创建BPMN版本管理服务实例
func NewBPMNVersionService(client *ent.Client) *BPMNVersionService {
	return &BPMNVersionService{client: client}
}

// ProcessVersion 流程版本信息
type ProcessVersion struct {
	ID                   string    `json:"id"`
	ProcessDefinitionKey string    `json:"process_definition_key"`
	Version              int       `json:"version"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	BPMNXML              string    `json:"bpmn_xml"`
	DeploymentID         string    `json:"deployment_id"`
	IsActive             bool      `json:"is_active"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	CreatedBy            string    `json:"created_by"`
	TenantID             int       `json:"tenant_id"`
	ChangeLog            string    `json:"change_log"`
	CompatibilityNotes   string    `json:"compatibility_notes"`
}

// CreateVersionRequest 创建版本请求
type CreateVersionRequest struct {
	ProcessDefinitionKey string `json:"process_definition_key" binding:"required"`
	Name                 string `json:"name" binding:"required"`
	Description          string `json:"description"`
	BPMNXML              string `json:"bpmn_xml" binding:"required"`
	ChangeLog            string `json:"change_log"`
	CompatibilityNotes   string `json:"compatibility_notes"`
	TenantID             int    `json:"tenant_id" binding:"required"`
	CreatedBy            string `json:"created_by"`
}

// UpdateVersionRequest 更新版本请求
type UpdateVersionRequest struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	BPMNXML            string `json:"bpmn_xml"`
	ChangeLog          string `json:"change_log"`
	CompatibilityNotes string `json:"compatibility_notes"`
}

// VersionComparison 版本比较结果
type VersionComparison struct {
	BaseVersion     *ProcessVersion `json:"base_version"`
	TargetVersion   *ProcessVersion `json:"target_version"`
	Changes         []ChangeDetail  `json:"changes"`
	BreakingChanges []string        `json:"breaking_changes"`
	Compatibility   string          `json:"compatibility"`
}

// ChangeDetail 变更详情
type ChangeDetail struct {
	Type        string `json:"type"`         // "added", "removed", "modified"
	ElementType string `json:"element_type"` // "task", "gateway", "event", "flow"
	ElementID   string `json:"element_id"`
	ElementName string `json:"element_name"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // "low", "medium", "high", "critical"
}

// CreateVersion 创建新版本
func (s *BPMNVersionService) CreateVersion(ctx context.Context, req *CreateVersionRequest) (*ProcessVersion, error) {
	// 获取当前最高版本号
	currentVersion, err := s.getCurrentVersion(ctx, req.ProcessDefinitionKey, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("获取当前版本失败: %w", err)
	}

	newVersionNumber := currentVersion + 1

	// 创建新的流程定义
	processDef, err := s.client.ProcessDefinition.Create().
		SetKey(req.ProcessDefinitionKey).
		SetName(req.Name).
		SetDescription(req.Description).
		SetBpmnXML([]byte(req.BPMNXML)).                 // BPMNXML是[]byte类型
		SetVersion(fmt.Sprintf("%d", newVersionNumber)). // Version是string类型
		SetTenantID(req.TenantID).
		SetIsActive(false). // 新版本默认不激活
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建流程定义失败: %w", err)
	}

	// 创建部署记录 - ProcessDeployment没有ProcessDefinitionID字段
	deployment, err := s.client.ProcessDeployment.Create().
		SetDeploymentID(fmt.Sprintf("%s-v%d", req.ProcessDefinitionKey, newVersionNumber)). // 使用deployment_id字段
		SetDeploymentName(fmt.Sprintf("%s v%d", req.Name, newVersionNumber)).
		SetDeploymentTime(time.Now()).
		SetTenantID(req.TenantID).
		SetDeployedBy(req.CreatedBy).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建部署记录失败: %w", err)
	}

	// 记录版本变更日志 - processDef.ID是int类型
	if err := s.recordVersionChangeLog(ctx, fmt.Sprintf("%d", processDef.ID), req.ChangeLog, req.CreatedBy, req.TenantID); err != nil {
		// 记录失败不影响主流程，只记录警告
		fmt.Printf("警告: 记录版本变更日志失败: %v\n", err)
	}

	return &ProcessVersion{
		ID:                   fmt.Sprintf("%d", processDef.ID), // ID是int类型，转换为string
		ProcessDefinitionKey: processDef.Key,
		Version:              newVersionNumber, // 使用原始int值
		Name:                 processDef.Name,
		Description:          processDef.Description,
		BPMNXML:              string(processDef.BpmnXML), // BPMNXML是[]byte类型，转换为string
		DeploymentID:         deployment.DeploymentID,    // 使用DeploymentID字段
		IsActive:             processDef.IsActive,
		CreatedAt:            processDef.CreatedAt,
		UpdatedAt:            processDef.UpdatedAt,
		CreatedBy:            req.CreatedBy,
		TenantID:             processDef.TenantID,
		ChangeLog:            req.ChangeLog,
		CompatibilityNotes:   req.CompatibilityNotes,
	}, nil
}

// GetVersion 获取指定版本
func (s *BPMNVersionService) GetVersion(ctx context.Context, processKey string, version int, tenantID int) (*ProcessVersion, error) {
	processDef, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(processKey),
			processdefinition.Version(fmt.Sprintf("%d", version)), // Version是string类型
			processdefinition.TenantID(tenantID),
		).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	// ProcessDeployment没有ProcessDefinitionID字段，暂时跳过部署信息查询
	// 获取部署信息
	// deployment, err := s.client.ProcessDeployment.Query().
	// 	Where(processdeployment.ProcessDefinitionID(processDef.ID)).
	// 	First(ctx)
	// if err != nil {
	// 	return nil, fmt.Errorf("获取部署信息失败: %w", err)
	// }

	// 解析版本号
	versionNumber, _ := strconv.Atoi(processDef.Version)

	return &ProcessVersion{
		ID:                   fmt.Sprintf("%d", processDef.ID), // ID是int类型，转换为string
		ProcessDefinitionKey: processDef.Key,
		Version:              versionNumber, // 解析后的int版本号
		Name:                 processDef.Name,
		Description:          processDef.Description,
		BPMNXML:              string(processDef.BpmnXML), // BPMNXML是[]byte类型，转换为string
		DeploymentID:         "",                         // 暂时为空，因为无法查询部署信息
		IsActive:             processDef.IsActive,
		CreatedAt:            processDef.CreatedAt,
		UpdatedAt:            processDef.UpdatedAt,
		TenantID:             processDef.TenantID,
	}, nil
}

// ListVersions 获取流程的所有版本
func (s *BPMNVersionService) ListVersions(ctx context.Context, processKey string, tenantID int) ([]*ProcessVersion, error) {
	processDefs, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(processKey),
			processdefinition.TenantID(tenantID),
		).
		Order(ent.Desc(processdefinition.FieldVersion)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义列表失败: %w", err)
	}

	var versions []*ProcessVersion
	for _, processDef := range processDefs {
		// ProcessDeployment没有ProcessDefinitionID字段，暂时跳过部署信息查询
		// 获取部署信息
		// deployment, err := s.client.ProcessDeployment.Query().
		// 	Where(processdeployment.ProcessDefinitionID(processDef.ID)).
		// 	First(ctx)
		// if err != nil {
		// 	// 跳过没有部署信息的版本
		// 	continue
		// }

		// 解析版本号
		versionNumber, _ := strconv.Atoi(processDef.Version)

		version := &ProcessVersion{
			ID:                   fmt.Sprintf("%d", processDef.ID), // ID是int类型，转换为string
			ProcessDefinitionKey: processDef.Key,
			Version:              versionNumber, // 解析后的int版本号
			Name:                 processDef.Name,
			Description:          processDef.Description,
			BPMNXML:              string(processDef.BpmnXML), // BPMNXML是[]byte类型，转换为string
			DeploymentID:         "",                         // 暂时为空，因为无法查询部署信息
			IsActive:             processDef.IsActive,
			CreatedAt:            processDef.CreatedAt,
			UpdatedAt:            processDef.UpdatedAt,
			TenantID:             processDef.TenantID,
		}
		versions = append(versions, version)
	}

	return versions, nil
}

// ActivateVersion 激活指定版本
func (s *BPMNVersionService) ActivateVersion(ctx context.Context, processKey string, version int, tenantID int) error {
	// 开始事务
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer func() {
		if v := recover(); v != nil {
			tx.Rollback()
			panic(v)
		}
	}()

	// 停用所有版本
	_, err = tx.ProcessDefinition.Update().
		Where(
			processdefinition.Key(processKey),
			processdefinition.TenantID(tenantID),
		).
		SetIsActive(false).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("停用其他版本失败: %w", err)
	}

	// 激活指定版本 - Version是string类型
	_, err = tx.ProcessDefinition.Update().
		Where(
			processdefinition.Key(processKey),
			processdefinition.Version(fmt.Sprintf("%d", version)), // 转换为string
			processdefinition.TenantID(tenantID),
		).
		SetIsActive(true).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("激活指定版本失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// RollbackToVersion 回滚到指定版本
func (s *BPMNVersionService) RollbackToVersion(ctx context.Context, processKey string, targetVersion int, tenantID int, reason string) error {
	// 检查目标版本是否存在 - Version是string类型
	targetProcessDef, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(processKey),
			processdefinition.Version(fmt.Sprintf("%d", targetVersion)), // 转换为string
			processdefinition.TenantID(tenantID),
		).
		First(ctx)
	if err != nil {
		return fmt.Errorf("目标版本不存在: %w", err)
	}

	// 创建回滚版本 - BPMNXML是[]byte类型，转换为string
	rollbackReq := &CreateVersionRequest{
		ProcessDefinitionKey: processKey,
		Name:                 fmt.Sprintf("%s (回滚到v%d)", targetProcessDef.Name, targetVersion),
		Description:          fmt.Sprintf("回滚到版本 %d，原因: %s", targetVersion, reason),
		BPMNXML:              string(targetProcessDef.BpmnXML), // 转换为string
		ChangeLog:            fmt.Sprintf("回滚到版本 %d，原因: %s", targetVersion, reason),
		CompatibilityNotes:   "回滚版本，可能存在兼容性问题",
		TenantID:             tenantID,
		CreatedBy:            "系统回滚",
	}

	rollbackVersion, err := s.CreateVersion(ctx, rollbackReq)
	if err != nil {
		return fmt.Errorf("创建回滚版本失败: %w", err)
	}

	// 激活回滚版本
	if err := s.ActivateVersion(ctx, processKey, rollbackVersion.Version, tenantID); err != nil {
		return fmt.Errorf("激活回滚版本失败: %w", err)
	}

	return nil
}

// CompareVersions 比较两个版本
func (s *BPMNVersionService) CompareVersions(ctx context.Context, processKey string, baseVersion, targetVersion int, tenantID int) (*VersionComparison, error) {
	// 获取基础版本
	baseProcessDef, err := s.GetVersion(ctx, processKey, baseVersion, tenantID)
	if err != nil {
		return nil, fmt.Errorf("获取基础版本失败: %w", err)
	}

	// 获取目标版本
	targetProcessDef, err := s.GetVersion(ctx, processKey, targetVersion, tenantID)
	if err != nil {
		return nil, fmt.Errorf("获取目标版本失败: %w", err)
	}

	// 比较BPMN XML内容
	changes, breakingChanges := s.compareBPMNXML(baseProcessDef.BPMNXML, targetProcessDef.BPMNXML)

	// 评估兼容性
	compatibility := s.assessCompatibility(changes, breakingChanges)

	return &VersionComparison{
		BaseVersion:     baseProcessDef,
		TargetVersion:   targetProcessDef,
		Changes:         changes,
		BreakingChanges: breakingChanges,
		Compatibility:   compatibility,
	}, nil
}

// getCurrentVersion 获取当前最高版本号
func (s *BPMNVersionService) getCurrentVersion(ctx context.Context, processKey string, tenantID int) (int, error) {
	processDef, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(processKey),
			processdefinition.TenantID(tenantID),
		).
		Order(ent.Desc(processdefinition.FieldVersion)).
		First(ctx)
	if err != nil {
		return 0, fmt.Errorf("获取流程定义失败: %w", err)
	}

	// Version是string类型，需要解析为int
	version, err := strconv.Atoi(processDef.Version)
	if err != nil {
		return 0, fmt.Errorf("解析版本号失败: %w", err)
	}

	return version, nil
}

// recordVersionChangeLog 记录版本变更日志
func (s *BPMNVersionService) recordVersionChangeLog(ctx context.Context, processDefID string, changeLog, createdBy string, tenantID int) error {
	// 记录版本变更到审计日志
	s.logger.Infow("Version change logged",
		"process_def_id", processDefID,
		"change_log", changeLog,
		"created_by", createdBy,
		"tenant_id", tenantID)
	// TODO: 创建专门的版本变更日志表（未来版本）
	return nil
}

// compareBPMNXML 比较BPMN XML内容
func (s *BPMNVersionService) compareBPMNXML(baseXML, targetXML string) ([]ChangeDetail, []string) {
	// 简单比较：检查XML长度是否变化
	baseLen := len(baseXML)
	targetLen := len(targetXML)

	if baseLen == targetLen {
		// 长度相同，视为无变化（简化处理）
		return []ChangeDetail{}, []string{}
	}

	// 有变化但不确定具体内容
	changeDetail := ChangeDetail{
		ElementID:   "root",
		ElementType: "process",
		ChangeType:  "modified",
		OldValue:    fmt.Sprintf("length:%d", baseLen),
		NewValue:    fmt.Sprintf("length:%d", targetLen),
	}

	// TODO: 实现完整的XML解析比较（未来版本）
	return []ChangeDetail{changeDetail}, []string{}
}

// assessCompatibility 评估兼容性
func (s *BPMNVersionService) assessCompatibility(changes []ChangeDetail, breakingChanges []string) string {
	if len(breakingChanges) > 0 {
		return "incompatible"
	}

	if len(changes) == 0 {
		return "identical"
	}

	// 检查是否有高风险变更
	for _, change := range changes {
		if change.Impact == "critical" || change.Impact == "high" {
			return "risky"
		}
	}

	return "compatible"
}
