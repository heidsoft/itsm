package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processdeployment"
	"itsm-backend/ent/processinstance"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// bpmn_process_definition_service.go — 流程定义 CRUD 与发布
//
// 职责：流程定义的创建、版本管理、查询、更新、发布、删除、激活/停用。
// 包含发布前的 BPMN lint 校验和候选策略校验。
//
// 社区贡献者只需理解此文件即可掌握流程定义的生命周期。
// ---------------------------------------------------------------------------

// Request DTOs

type CreateProcessDefinitionRequest struct {
	Key              string                 `json:"key" binding:"required"`
	Name             string                 `json:"name" binding:"required"`
	Description      string                 `json:"description"`
	Category         string                 `json:"category"`
	BPMNXML          string                 `json:"bpmnXml" binding:"required"`
	ProcessVariables map[string]interface{} `json:"processVariables"`
	TenantID         int                    `json:"tenantId" binding:"required"`
	Publish          bool                   `json:"publish"`
}

type UpdateProcessDefinitionRequest struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Category         string                 `json:"category"`
	BPMNXML          string                 `json:"bpmnXml"`
	ProcessVariables map[string]interface{} `json:"processVariables"`
	IsActive         *bool                  `json:"isActive"`
	// CandidateDefinition is the AI-generated business contract. It is kept
	// alongside the BPMN draft so publishing can enforce the same safety gate
	// after a human edits the diagram.
	CandidateDefinition map[string]interface{} `json:"candidateDefinition"`
}

type ListProcessDefinitionsRequest struct {
	Key      string `json:"key"`
	Category string `json:"category"`
	IsActive *bool  `json:"isActive"`
	TenantID int    `json:"tenantId"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

// Service struct

type bpmnProcessDefinitionService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// CRUD

func (s *bpmnProcessDefinitionService) CreateProcessDefinition(ctx context.Context, req *CreateProcessDefinitionRequest) (*ent.ProcessDefinition, error) {
	if req.Publish {
		if err := validateActivatableBPMNXML(req.BPMNXML); err != nil {
			return nil, err
		}
	} else if _, err := NewBPMNParser().ParseXML([]byte(req.BPMNXML)); err != nil {
		return nil, fmt.Errorf("BPMN XML 校验失败: %w", err)
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开始流程定义事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	client := tx.Client()

	var deployment *ent.ProcessDeployment
	existingDeployments, err := client.ProcessDeployment.Query().
		Where(processdeployment.TenantID(req.TenantID)).
		Order(ent.Desc("created_at")).
		Limit(1).
		All(ctx)

	if err == nil && len(existingDeployments) > 0 {
		deployment = existingDeployments[0]
	} else {
		deployment, err = client.ProcessDeployment.Create().
			SetDeploymentID(fmt.Sprintf("deploy-%d", time.Now().UnixNano())).
			SetDeploymentName(req.Name + "-deployment").
			SetDeploymentSource("api").
			SetTenantID(req.TenantID).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("创建部署记录失败: %w", err)
		}
	}

	nextVersion := s.getNextVersionWithClient(ctx, client, req.Key, req.TenantID)

	existing, err := client.ProcessDefinition.Query().
		Where(processdefinition.Key(req.Key)).
		Where(processdefinition.IsLatest(true)).
		Where(processdefinition.TenantID(req.TenantID)).
		First(ctx)

	if err == nil && existing != nil {
		_, err = client.ProcessDefinition.UpdateOne(existing).
			SetIsLatest(false).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("更新旧版本失败: %w", err)
		}
	}

	definition, err := client.ProcessDefinition.Create().
		SetKey(req.Key).
		SetName(req.Name).
		SetDescription(req.Description).
		SetCategory(req.Category).
		SetBpmnXML([]byte(req.BPMNXML)).
		SetProcessVariables(req.ProcessVariables).
		SetVersion(nextVersion).
		SetIsActive(req.Publish).
		SetIsLatest(true).
		SetTenantID(req.TenantID).
		SetDeploymentID(deployment.ID).
		SetDeploymentName(deployment.DeploymentName).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建流程定义失败: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交流程定义事务失败: %w", err)
	}
	return definition, nil
}

func (s *bpmnProcessDefinitionService) GetProcessDefinition(ctx context.Context, key string, version string) (*ent.ProcessDefinition, error) {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	definition, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(key),
			processdefinition.Version(version),
			processdefinition.TenantID(tenantID),
		).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	return definition, nil
}

func (s *bpmnProcessDefinitionService) GetProcessDefinitionByID(ctx context.Context, id int) (*ent.ProcessDefinition, error) {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	definition, err := s.client.ProcessDefinition.Query().
		Where(processdefinition.ID(id), processdefinition.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	return definition, nil
}

func (s *bpmnProcessDefinitionService) GetLatestProcessDefinition(ctx context.Context, key string) (*ent.ProcessDefinition, error) {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	definition, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(key),
			processdefinition.IsLatest(true),
			processdefinition.TenantID(tenantID),
		).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取最新流程定义失败: %w", err)
	}

	return definition, nil
}

func (s *bpmnProcessDefinitionService) UpdateProcessDefinition(ctx context.Context, key string, version string, req *UpdateProcessDefinitionRequest) (*ent.ProcessDefinition, error) {
	definition, err := s.GetProcessDefinition(ctx, key, version)
	if err != nil {
		return nil, err
	}

	willBeActive := definition.IsActive
	if req.IsActive != nil {
		willBeActive = *req.IsActive
	}
	if willBeActive {
		bpmnXML := req.BPMNXML
		if bpmnXML == "" {
			bpmnXML = string(definition.BpmnXML)
		}
		if err := validateActivatableBPMNXML(bpmnXML); err != nil {
			return nil, err
		}
	}

	update := s.client.ProcessDefinition.UpdateOne(definition)

	if req.Name != "" {
		update.SetName(req.Name)
	}
	if req.Description != "" {
		update.SetDescription(req.Description)
	}
	if req.Category != "" {
		update.SetCategory(req.Category)
	}
	if req.BPMNXML != "" {
		update.SetBpmnXML([]byte(req.BPMNXML))
	}
	if req.ProcessVariables != nil {
		update.SetProcessVariables(req.ProcessVariables)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新流程定义失败: %w", err)
	}

	return updated, nil
}

// PublishProcessDefinition atomically persists the draft contents and makes the
// selected immutable version the sole active/latest version for the key.
func (s *bpmnProcessDefinitionService) PublishProcessDefinition(ctx context.Context, key string, version string, req *UpdateProcessDefinitionRequest) (*ent.ProcessDefinition, error) {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.BPMNXML) == "" {
		return nil, fmt.Errorf("发布流程必须包含 BPMN XML")
	}
	if err := validateActivatableBPMNXML(req.BPMNXML); err != nil {
		return nil, err
	}
	if err := validateCandidateDefinition(req.CandidateDefinition); err != nil {
		return nil, fmt.Errorf("候选流程策略校验失败: %w", err)
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开始发布事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	definition, err := tx.ProcessDefinition.Query().Where(
		processdefinition.Key(key), processdefinition.Version(version), processdefinition.TenantID(tenantID),
	).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取待发布流程定义失败: %w", err)
	}
	if _, err = tx.ProcessDefinition.Update().Where(
		processdefinition.Key(key), processdefinition.TenantID(tenantID),
	).SetIsActive(false).SetIsLatest(false).Save(ctx); err != nil {
		return nil, fmt.Errorf("停用旧流程版本失败: %w", err)
	}
	update := tx.ProcessDefinition.UpdateOne(definition).
		SetBpmnXML([]byte(req.BPMNXML)).SetIsActive(true).SetIsLatest(true).SetDeployedAt(time.Now())
	if req.Name != "" {
		update.SetName(req.Name)
	}
	if req.Description != "" {
		update.SetDescription(req.Description)
	}
	if req.Category != "" {
		update.SetCategory(req.Category)
	}
	if req.ProcessVariables != nil {
		update.SetProcessVariables(req.ProcessVariables)
	}
	published, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("保存发布版本失败: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交发布事务失败: %w", err)
	}
	return published, nil
}

func (s *bpmnProcessDefinitionService) DeleteProcessDefinition(ctx context.Context, key string, version string) error {
	definition, err := s.GetProcessDefinition(ctx, key, version)
	if err != nil {
		return err
	}

	runningCount, err := s.client.ProcessInstance.
		Query().
		Where(processinstance.ProcessDefinitionID(definition.ID)).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("检查流程实例失败: %w", err)
	}
	if runningCount > 0 {
		return fmt.Errorf("该流程定义有 %d 个运行中的实例，请先关闭后再删除", runningCount)
	}

	return s.client.ProcessDefinition.DeleteOne(definition).Exec(ctx)
}

func (s *bpmnProcessDefinitionService) ListProcessDefinitions(ctx context.Context, req *ListProcessDefinitionsRequest) ([]*ent.ProcessDefinition, int, error) {
	ctxTenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	tenantID := ctxTenantID
	if req.TenantID > 0 && req.TenantID != ctxTenantID {
		return nil, 0, fmt.Errorf("请求租户 %d 与上下文租户 %d 不一致，已拒绝", req.TenantID, ctxTenantID)
	}

	query := s.client.ProcessDefinition.Query().
		Where(processdefinition.TenantID(tenantID))

	if req.Key != "" {
		query = query.Where(processdefinition.Key(req.Key))
	}
	if req.Category != "" {
		query = query.Where(processdefinition.Category(req.Category))
	}
	if req.IsActive != nil {
		query = query.Where(processdefinition.IsActive(*req.IsActive))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取流程定义总数失败: %w", err)
	}

	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	definitions, err := query.Order(ent.Desc(processdefinition.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取流程定义列表失败: %w", err)
	}

	return definitions, total, nil
}

func (s *bpmnProcessDefinitionService) SetProcessDefinitionActive(ctx context.Context, key string, version string, active bool) error {
	definition, err := s.GetProcessDefinition(ctx, key, version)
	if err != nil {
		return err
	}
	if active {
		if err := validateActivatableBPMNXML(string(definition.BpmnXML)); err != nil {
			return err
		}
	}

	_, err = s.client.ProcessDefinition.UpdateOne(definition).
		SetIsActive(active).
		Save(ctx)

	return err
}

// Version helpers

// getNextVersion 获取下一个版本号（major.minor.0）。约定与部署服务一致：递增 minor、patch 归零、minor 不封顶。
// 旧实现用 Order(Desc("version")) 对字符串版本号做字典序排序，多位数版本（如 1.10.0 vs 1.9.0）会被误判大小；
// 此处改为在 Go 侧解析取最大 minor 后递增，避免字典序陷阱。
//
//lint:ignore U1000 Deprecated: see function comment
func (s *bpmnProcessDefinitionService) getNextVersion(ctx context.Context, key string, tenantID int) string {
	return s.getNextVersionWithClient(ctx, s.client, key, tenantID)
}

func (s *bpmnProcessDefinitionService) getNextVersionWithClient(ctx context.Context, client *ent.Client, key string, tenantID int) string {
	defs, err := client.ProcessDefinition.Query().
		Where(processdefinition.Key(key)).
		Where(processdefinition.TenantID(tenantID)).
		All(ctx)
	if err != nil || len(defs) == 0 {
		return "1.0.0"
	}

	maxMaj, maxMin := 0, -1
	for _, d := range defs {
		maj, min, _ := parseSemver(d.Version)
		if maj > maxMaj || (maj == maxMaj && min > maxMin) {
			maxMaj, maxMin = maj, min
		}
	}
	return fmt.Sprintf("%d.%d.0", maxMaj, maxMin+1)
}

// parseSemver 解析 "major.minor.patch" / "major" / "major.minor" 为三段整数。
func parseSemver(v string) (int, int, int) {
	var maj, min, pat int
	parts := strings.Split(v, ".")
	if len(parts) > 0 {
		fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &maj)
	}
	if len(parts) > 1 {
		fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &min)
	}
	if len(parts) > 2 {
		fmt.Sscanf(strings.TrimSpace(parts[2]), "%d", &pat)
	}
	return maj, min, pat
}

// Validation helpers

func validateActivatableBPMNXML(bpmnXML string) error {
	result, err := NewBPMNLintService().LintBPMNXML([]byte(bpmnXML))
	if err != nil {
		return fmt.Errorf("BPMN 发布校验失败: %w", err)
	}
	if !result.HasErrors {
		return nil
	}

	messages := make([]string, 0, result.ErrorCount)
	for _, issue := range result.Issues {
		if issue.Severity != "error" {
			continue
		}
		if issue.ElementID != "" {
			messages = append(messages, fmt.Sprintf("[%s] %s", issue.ElementID, issue.Message))
			continue
		}
		messages = append(messages, issue.Message)
	}
	if len(messages) == 0 {
		return fmt.Errorf("BPMN 发布校验失败")
	}
	return fmt.Errorf("BPMN 发布校验失败: %s", strings.Join(messages, "；"))
}

// validateCandidateDefinition is deliberately deterministic. AI output is
// advisory; publishing requires typed form fields and unambiguous approval
// rules, while an omitted candidate remains valid for legacy BPMN definitions.
func validateCandidateDefinition(candidate map[string]interface{}) error {
	if len(candidate) == 0 {
		return nil
	}
	for _, section := range []string{"domain", "formSchema", "approvalPolicy", "ontologyBindings", "slaConfig"} {
		if _, ok := candidate[section]; !ok {
			return fmt.Errorf("缺少 %s", section)
		}
	}
	form, ok := candidate["formSchema"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("formSchema 格式错误")
	}
	fields, ok := form["fields"].([]interface{})
	if !ok || len(fields) == 0 {
		return fmt.Errorf("formSchema.fields 不能为空")
	}
	for i, raw := range fields {
		f, ok := raw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("字段 %d 格式错误", i)
		}
		if strings.TrimSpace(fmt.Sprint(f["name"])) == "" || strings.TrimSpace(fmt.Sprint(f["type"])) == "" {
			return fmt.Errorf("字段 %d 缺少 name/type", i)
		}
	}
	policy, ok := candidate["approvalPolicy"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("approvalPolicy 格式错误")
	}
	if rules, exists := policy["rules"]; exists {
		if _, ok := rules.([]interface{}); !ok {
			return fmt.Errorf("approvalPolicy.rules 格式错误")
		}
	}
	return nil
}
