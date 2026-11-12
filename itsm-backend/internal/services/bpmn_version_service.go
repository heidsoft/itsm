package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processversionchangelog"

	// "itsm-backend/ent/processdeployment" // 暂时不使用，因为ProcessDeployment没有ProcessDefinitionID字段
	"go.uber.org/zap"
)

// BPMNVersionService BPMN流程版本管理服务
type BPMNVersionService struct {
	client     *ent.Client
	logger     *zap.SugaredLogger
	timerStore TimerStore
}

// NewBPMNVersionService 创建BPMN版本管理服务实例
func NewBPMNVersionService(client *ent.Client, logger *zap.SugaredLogger) *BPMNVersionService {
	return &BPMNVersionService{client: client, logger: logger}
}

// SetTimerStore 注入 start timer 时间表存储（bootstrap 在 timerStore 创建后装配）。
// 未注入时激活/回滚版本只切换定义行，不同步时间表。
func (s *BPMNVersionService) SetTimerStore(store TimerStore) {
	s.timerStore = store
}

// ProcessVersion 已下沉到 dto；此处保留别名，保证共享层方法签名与域内窄契约是同一类型身份。
type ProcessVersion = dto.ProcessVersion

// CreateVersionRequest 已下沉到 dto；此处保留别名，保证共享层方法签名与域内窄契约是同一类型身份。
type CreateVersionRequest = dto.CreateVersionRequest

// UpdateVersionRequest 已下沉到 dto；此处保留别名，保证共享层方法签名与域内窄契约是同一类型身份。
type UpdateVersionRequest = dto.UpdateVersionRequest

// VersionComparison 已下沉到 dto；此处保留别名，保证共享层方法签名与域内窄契约是同一类型身份。
type VersionComparison = dto.VersionComparison

// ChangeDetail 已下沉到 dto；此处保留别名，保证共享层方法签名与域内窄契约是同一类型身份。
type ChangeDetail = dto.ChangeDetail

// CreateVersion 创建新版本
func (s *BPMNVersionService) CreateVersion(ctx context.Context, req *CreateVersionRequest) (*ProcessVersion, error) {
	// 获取当前最高版本号
	currentVersion, err := s.getCurrentVersion(ctx, req.ProcessDefinitionKey, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("获取当前版本失败: %w", err)
	}

	newVersionNumber := currentVersion + 1

	// 先创建部署记录（因为ProcessDefinition需要deployment_id）
	deployment, err := s.client.ProcessDeployment.Create().
		SetDeploymentID(fmt.Sprintf("%s-v%d", req.ProcessDefinitionKey, newVersionNumber)).
		SetDeploymentName(fmt.Sprintf("%s v%d", req.Name, newVersionNumber)).
		SetDeploymentTime(time.Now()).
		SetTenantID(req.TenantID).
		SetDeployedBy(req.CreatedBy).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建部署记录失败: %w", err)
	}

	// 使用部署ID创建流程定义
	processDef, err := s.client.ProcessDefinition.Create().
		SetKey(req.ProcessDefinitionKey).
		SetName(req.Name).
		SetDescription(req.Description).
		SetBpmnXML([]byte(req.BPMNXML)).
		SetVersion(fmt.Sprintf("%d", newVersionNumber)).
		SetTenantID(req.TenantID).
		SetIsActive(false).
		SetIsLatest(false).
		SetDeploymentID(deployment.ID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建流程定义失败: %w", err)
	}

	// 记录版本变更日志 - processDef.ID是int类型
	if err := s.recordVersionChangeLog(ctx, fmt.Sprintf("%d", processDef.ID), fmt.Sprintf("%d", newVersionNumber), req.ChangeLog, req.CreatedBy, req.TenantID); err != nil {
		// 记录失败不影响主流程，只记录警告
		s.logger.Warnw("记录版本变更日志失败", "error", err)
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
	versionNumber, err := ParseVersionNumber(processDef.Version)
	if err != nil {
		return nil, fmt.Errorf("解析版本号失败: %w", err)
	}

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
		versionNumber, err := ParseVersionNumber(processDef.Version)
		if err != nil {
			continue
		}

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
func (s *BPMNVersionService) ActivateVersion(ctx context.Context, processKey string, version int, tenantID int) (err error) {
	// 开始事务
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer func() {
		if v := recover(); v != nil {
			tx.Rollback()
			err = fmt.Errorf("panic recovered in ActivateVersion: %v", v)
			s.logger.Errorw("Panic recovered in ActivateVersion", "error", v)
		}
	}()

	// 停用所有版本并取消 latest 标记
	_, err = tx.ProcessDefinition.Update().
		Where(
			processdefinition.Key(processKey),
			processdefinition.TenantID(tenantID),
		).
		SetIsActive(false).
		SetIsLatest(false).
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
		SetIsLatest(true).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("激活指定版本失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 激活即切换生产时间表：按目标版本 XML 整体替换 start timer（先取消旧
	// pending 再重建）。不同步会让激活后的版本按旧版本（或空）时间表启动实例，
	// 回滚路径经 ActivateVersion 同样继承该语义。
	if s.timerStore != nil {
		def, err := s.client.ProcessDefinition.Query().
			Where(
				processdefinition.Key(processKey),
				processdefinition.Version(fmt.Sprintf("%d", version)),
				processdefinition.TenantID(tenantID),
			).
			First(ctx)
		if err != nil {
			return fmt.Errorf("版本已激活但加载目标版本定义失败: %w", err)
		}
		if _, err := SyncStartTimers(ctx, s.client, s.timerStore, def.BpmnXML, processKey, tenantID); err != nil {
			return fmt.Errorf("版本已激活但同步 start timer 时间表失败: %w", err)
		}
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

// ParseVersionNumber 将版本字符串解析为整数版本号。
// 支持纯整数（"1","2"）和 semver（"1.0.0","1.1.0"）两种格式，semver 取主版本号。
func ParseVersionNumber(versionStr string) (int, error) {
	return dto.ParseVersionNumber(versionStr)
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

	return ParseVersionNumber(processDef.Version)
}

// recordVersionChangeLog 记录版本变更日志
func (s *BPMNVersionService) recordVersionChangeLog(ctx context.Context, processDefID string, version string, changeLog, createdBy string, tenantID int) error {
	if changeLog == "" {
		// 空变更描述不记录（change_log 字段 NotEmpty 校验会拒绝，避免每次告警）
		return nil
	}

	// 解析 processDefID 为 int
	processDefIDInt, err := strconv.Atoi(processDefID)
	if err != nil {
		s.logger.Warnw("Failed to parse processDefID", "process_def_id", processDefID, "error", err)
		return fmt.Errorf("解析流程定义 ID 失败：%w", err)
	}

	// 解析 createdBy 为 int (用户 ID)
	createdByInt, err := strconv.Atoi(createdBy)
	if err != nil {
		// 如果解析失败，使用 0 表示系统创建
		createdByInt = 0
		s.logger.Warnw("Failed to parse createdBy, using 0", "created_by", createdBy, "error", err)
	}

	// 创建版本变更日志记录
	_, err = s.client.ProcessVersionChangelog.Create().
		SetProcessDefinitionID(processDefIDInt).
		SetVersion(version). // 版本号（如 "1" / "1.0.0"），区别于变更描述
		SetChangeLog(changeLog).
		SetChangeType("update").
		SetCreatedBy(createdByInt).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create version changelog",
			"process_def_id", processDefID,
			"change_log", changeLog,
			"created_by", createdBy,
			"tenant_id", tenantID,
			"error", err)
		return fmt.Errorf("创建版本变更日志失败：%w", err)
	}

	s.logger.Infow("Version change logged successfully",
		"process_def_id", processDefID,
		"change_log", changeLog,
		"created_by", createdBy,
		"tenant_id", tenantID)
	return nil
}

// compareBPMNXML 比较BPMN XML内容
func (s *BPMNVersionService) compareBPMNXML(baseXML, targetXML string) ([]ChangeDetail, []string) {
	// 简单比较：检查XML长度是否变化
	baseLen := len(baseXML)
	targetLen := len(targetXML)

	if baseLen == targetLen && baseXML == targetXML {
		// 完全相同
		return []ChangeDetail{}, []string{}
	}

	var changes []ChangeDetail

	// 解析基础XML
	type Process struct {
		ID    string `xml:"id,attr"`
		Name  string `xml:"name,attr"`
		Tasks []struct {
			ID   string `xml:"id,attr"`
			Name string `xml:"name,attr"`
		} `xml:"userTask"`
	}
	type Definitions struct {
		Process Process `xml:"process"`
	}

	var baseDef Definitions
	if err := xml.Unmarshal([]byte(baseXML), &baseDef); err == nil {
		var targetDef Definitions
		if err := xml.Unmarshal([]byte(targetXML), &targetDef); err == nil {
			// 比较Task数量
			if len(baseDef.Process.Tasks) != len(targetDef.Process.Tasks) {
				changes = append(changes, ChangeDetail{
					Type:        "modified",
					ChangeType:  "structure",
					ElementType: "process",
					ElementID:   baseDef.Process.ID,
					ElementName: baseDef.Process.Name,
					Description: fmt.Sprintf("Task count changed from %d to %d", len(baseDef.Process.Tasks), len(targetDef.Process.Tasks)),
					Impact:      "high",
				})
			}
		}
	}

	// 如果无法详细解析或没有发现特定差异但内容不同，添加通用变更记录
	if len(changes) == 0 {
		changeDetail := ChangeDetail{
			ElementID:   "root",
			ElementType: "process",
			ChangeType:  "modified",
			OldValue:    fmt.Sprintf("length:%d", baseLen),
			NewValue:    fmt.Sprintf("length:%d", targetLen),
			Description: "BPMN XML content has changed",
			Impact:      "medium",
		}
		changes = append(changes, changeDetail)
	}

	return changes, []string{}
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

// GetChangeLogsByProcessKey 根据流程定义key获取版本变更日志列表
// 同一 key 可能存在多个版本的定义（每个版本一条 ProcessDefinition 记录），
// 变更日志挂在各版本定义的 ID 上，因此按 key 查询需聚合所有版本的日志。
func (s *BPMNVersionService) GetChangeLogsByProcessKey(ctx context.Context, processKey string, tenantID int) ([]*ent.ProcessVersionChangelog, error) {
	// 先通过 processdefinition 找到该 key 下所有版本的定义 ID
	pds, err := s.client.ProcessDefinition.Query().
		Where(processdefinition.Key(processKey), processdefinition.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("流程定义不存在: %w", err)
	}
	if len(pds) == 0 {
		return nil, fmt.Errorf("流程定义不存在: key=%s tenant=%d", processKey, tenantID)
	}

	ids := make([]int, 0, len(pds))
	for _, pd := range pds {
		ids = append(ids, pd.ID)
	}

	// 获取 changelogs
	changelogs, err := s.client.ProcessVersionChangelog.Query().
		Where(processversionchangelog.ProcessDefinitionIDIn(ids...)).
		Order(ent.Desc(processversionchangelog.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取变更日志失败: %w", err)
	}

	return changelogs, nil
}

// GetChangeLogsByProcessDefinitionID 根据流程定义ID获取版本变更日志；定义 ID 全局可枚举，查询必须带租户谓词。
func (s *BPMNVersionService) GetChangeLogsByProcessDefinitionID(ctx context.Context, processDefID, tenantID int) ([]*ent.ProcessVersionChangelog, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("缺少租户上下文")
	}
	changelogs, err := s.client.ProcessVersionChangelog.Query().
		Where(
			processversionchangelog.ProcessDefinitionIDEQ(processDefID),
			processversionchangelog.TenantIDEQ(tenantID),
		).
		Order(ent.Desc(processversionchangelog.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取变更日志失败: %w", err)
	}

	return changelogs, nil
}
