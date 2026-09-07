package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"itsm-backend/dto"
)

var (
	ErrWorkflowTemplateNotFound = errors.New("workflow template not found")
	ErrWorkflowTemplateConflict = errors.New("workflow template conflict")
	ErrWorkflowTemplateInvalid  = errors.New("workflow template invalid")
)

type BPMNWorkflowTemplateCatalog struct {
	db *sql.DB
}

func NewBPMNWorkflowTemplateCatalog(db *sql.DB) *BPMNWorkflowTemplateCatalog {
	return &BPMNWorkflowTemplateCatalog{db: db}
}

func (s *BPMNWorkflowTemplateCatalog) List(ctx context.Context, tenantID int, keyword, domain, status string, page, pageSize int) (*dto.WorkflowTemplateListResponse, error) {
	if s == nil || s.db == nil || tenantID <= 0 {
		return nil, fmt.Errorf("模板目录数据库未配置")
	}
	page, pageSize = normalizePage(page, pageSize)
	where := []string{"tenant_id = $1"}
	args := []interface{}{tenantID}
	arg := 2
	if keyword != "" {
		where = append(where, fmt.Sprintf("(key ILIKE $%d OR name ILIKE $%d OR description ILIKE $%d)", arg, arg, arg))
		args = append(args, "%"+keyword+"%")
		arg++
	}
	if domain != "" {
		where = append(where, fmt.Sprintf("domain = $%d", arg))
		args = append(args, domain)
		arg++
	}
	if status != "" {
		where = append(where, fmt.Sprintf("status = $%d", arg))
		args = append(args, status)
		arg++
	}
	base := " FROM workflow_templates WHERE " + strings.Join(where, " AND ")
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*)"+base, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("查询模板总数失败: %w", err)
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := s.db.QueryContext(ctx, "SELECT id,key,name,description,domain,form_schema,approval_policy,ontology_bindings,sla_config,bpmn_xml,version,status,is_public,created_by,created_at,updated_at"+base+fmt.Sprintf(" ORDER BY updated_at DESC, id DESC LIMIT $%d OFFSET $%d", arg, arg+1), args...)
	if err != nil {
		return nil, fmt.Errorf("查询模板列表失败: %w", err)
	}
	defer rows.Close()
	items := make([]*dto.WorkflowTemplate, 0)
	for rows.Next() {
		item, err := scanWorkflowTemplate(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取模板列表失败: %w", err)
	}
	return &dto.WorkflowTemplateListResponse{Items: items, Total: total, Page: page, PageSize: pageSize, TotalPages: int(math.Ceil(float64(total) / float64(pageSize)))}, nil
}

func (s *BPMNWorkflowTemplateCatalog) Get(ctx context.Context, tenantID int, key, version string) (*dto.WorkflowTemplate, error) {
	if s == nil || s.db == nil || tenantID <= 0 {
		return nil, fmt.Errorf("模板目录数据库未配置")
	}
	query := "SELECT id,key,name,description,domain,form_schema,approval_policy,ontology_bindings,sla_config,bpmn_xml,version,status,is_public,created_by,created_at,updated_at FROM workflow_templates WHERE tenant_id=$1 AND key=$2"
	args := []interface{}{tenantID, key}
	if version == "" {
		query += " ORDER BY CASE WHEN status='draft' THEN 0 WHEN status='published' THEN 1 ELSE 2 END, updated_at DESC, id DESC LIMIT 1"
	} else {
		query += " AND version=$3 LIMIT 1"
		args = append(args, version)
	}
	row := s.db.QueryRowContext(ctx, query, args...)
	item, err := scanWorkflowTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWorkflowTemplateNotFound
	}
	return item, err
}

func (s *BPMNWorkflowTemplateCatalog) CreateDraft(ctx context.Context, tenantID, userID int, req *dto.CreateWorkflowTemplateRequest) (*dto.WorkflowTemplate, error) {
	if err := validateTemplateRequest(req); err != nil {
		return nil, err
	}
	if s == nil || s.db == nil || tenantID <= 0 || userID <= 0 {
		return nil, fmt.Errorf("模板目录上下文未配置")
	}
	var draftExists bool
	if err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM workflow_templates WHERE tenant_id=$1 AND key=$2 AND status='draft')", tenantID, req.Key).Scan(&draftExists); err != nil {
		return nil, fmt.Errorf("检查模板草稿失败: %w", err)
	}
	if draftExists {
		return nil, ErrWorkflowTemplateConflict
	}
	version, err := s.nextVersion(ctx, tenantID, req.Key)
	if err != nil {
		return nil, err
	}
	form, approval, bindings, sla, err := marshalTemplateJSON(req.FormSchema, req.ApprovalPolicy, req.OntologyBindings, req.SLAConfig)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO workflow_templates (key,name,description,domain,form_schema,approval_policy,ontology_bindings,sla_config,bpmn_xml,version,status,is_public,tenant_id,created_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,to_jsonb($9::text),$10,'draft',$11,$12,$13)`, req.Key, req.Name, req.Description, req.Domain, form, approval, bindings, sla, req.BPMNXML, version, req.IsPublic, tenantID, userID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, ErrWorkflowTemplateConflict
		}
		return nil, fmt.Errorf("创建模板草稿失败: %w", err)
	}
	return s.Get(ctx, tenantID, req.Key, version)
}

func (s *BPMNWorkflowTemplateCatalog) UpdateDraft(ctx context.Context, tenantID int, key string, req *dto.UpdateWorkflowTemplateRequest) (*dto.WorkflowTemplate, error) {
	current, err := s.Get(ctx, tenantID, key, "")
	if err != nil {
		return nil, err
	}
	if current.Status != "draft" {
		return nil, ErrWorkflowTemplateConflict
	}
	sets, args := make([]string, 0), []interface{}{tenantID, key, current.Version}
	arg := 4
	add := func(column string, value interface{}) {
		assignment := fmt.Sprintf("%s=$%d", column, arg)
		if column == "bpmn_xml" {
			assignment = fmt.Sprintf("bpmn_xml=to_jsonb($%d::text)", arg)
		}
		sets = append(sets, assignment)
		args = append(args, value)
		arg++
	}
	if req.Name != nil {
		add("name", *req.Name)
	}
	if req.Description != nil {
		add("description", *req.Description)
	}
	if req.Domain != nil {
		add("domain", *req.Domain)
	}
	if req.FormSchema != nil {
		value, _ := json.Marshal(*req.FormSchema)
		add("form_schema", value)
	}
	if req.ApprovalPolicy != nil {
		value, _ := json.Marshal(*req.ApprovalPolicy)
		add("approval_policy", value)
	}
	if req.OntologyBindings != nil {
		value, _ := json.Marshal(*req.OntologyBindings)
		add("ontology_bindings", value)
	}
	if req.SLAConfig != nil {
		value, _ := json.Marshal(*req.SLAConfig)
		add("sla_config", value)
	}
	if req.BPMNXML != nil {
		if strings.TrimSpace(*req.BPMNXML) == "" {
			return nil, fmt.Errorf("%w: bpmnXml 不能为空", ErrWorkflowTemplateInvalid)
		}
		add("bpmn_xml", *req.BPMNXML)
	}
	if req.IsPublic != nil {
		add("is_public", *req.IsPublic)
	}
	if len(sets) == 0 {
		return current, nil
	}
	sets = append(sets, "updated_at=NOW()")
	_, err = s.db.ExecContext(ctx, "UPDATE workflow_templates SET "+strings.Join(sets, ", ")+" WHERE tenant_id=$1 AND key=$2 AND version=$3 AND status='draft'", args...)
	if err != nil {
		return nil, fmt.Errorf("更新模板草稿失败: %w", err)
	}
	return s.Get(ctx, tenantID, key, current.Version)
}

func (s *BPMNWorkflowTemplateCatalog) Publish(ctx context.Context, tenantID int, key string) (*dto.WorkflowTemplate, error) {
	draft, err := s.Get(ctx, tenantID, key, "")
	if err != nil {
		return nil, err
	}
	if draft.Status != "draft" {
		return nil, ErrWorkflowTemplateConflict
	}
	result, err := NewBPMNLintService().LintBPMNXML([]byte(draft.BPMNXML))
	if err != nil || result.HasErrors {
		if err != nil {
			return nil, fmt.Errorf("%w: BPMN 校验失败", ErrWorkflowTemplateInvalid)
		}
		return nil, fmt.Errorf("%w: BPMN 存在 %d 个错误", ErrWorkflowTemplateInvalid, result.ErrorCount)
	}
	updated, err := s.db.ExecContext(ctx, "UPDATE workflow_templates SET status='published', updated_at=NOW() WHERE tenant_id=$1 AND key=$2 AND version=$3 AND status='draft'", tenantID, key, draft.Version)
	if err != nil {
		return nil, fmt.Errorf("发布模板失败: %w", err)
	}
	if count, _ := updated.RowsAffected(); count != 1 {
		return nil, ErrWorkflowTemplateConflict
	}
	return s.Get(ctx, tenantID, key, draft.Version)
}

func (s *BPMNWorkflowTemplateCatalog) Archive(ctx context.Context, tenantID int, key string) error {
	var version string
	err := s.db.QueryRowContext(ctx, "SELECT version FROM workflow_templates WHERE tenant_id=$1 AND key=$2 AND status='published' ORDER BY id DESC LIMIT 1", tenantID, key).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrWorkflowTemplateNotFound
	}
	if err != nil {
		return fmt.Errorf("查询待停用模板失败: %w", err)
	}
	result, err := s.db.ExecContext(ctx, "UPDATE workflow_templates SET status='archived', updated_at=NOW() WHERE tenant_id=$1 AND key=$2 AND version=$3 AND status='published'", tenantID, key, version)
	if err != nil {
		return fmt.Errorf("停用模板失败: %w", err)
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return ErrWorkflowTemplateNotFound
	}
	return nil
}

// Reload 重新部署已发布的模板为 BPMN 流程定义。
//
// 适用场景：管理员在 workflow_templates 中修改并发布了一个新版本，但当前
// process_definitions 表中仍是旧版本（is_latest=true）。此方法会：
//  1. 取该 key 最新的已发布（status='published'）版本；
//  2. 用其 bpmn_xml 在 process_deployments + process_definitions 中创建新版本；
//  3. 把该 key 之前 is_latest=true 的流程定义全部降级为 is_latest=false。
//
// 返回新部署的 deployment id、process_definition id 和版本号。
//
// 实现使用原生 SQL：与 catalog 其他方法保持一致；process_definitions 上的
// softdelete 拦截器会强制 DeletedAtIsNil，但 process_definitions 并不走软删
// （schema 未声明 soft_delete_fields），使用原生 SQL 与 Ent 视图保持一致。
func (s *BPMNWorkflowTemplateCatalog) Reload(ctx context.Context, tenantID int, key string) (*dto.ReloadWorkflowTemplateResponse, error) {
	if s == nil || s.db == nil || tenantID <= 0 {
		return nil, fmt.Errorf("模板目录数据库未配置")
	}
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("%w: key 不能为空", ErrWorkflowTemplateInvalid)
	}

	// 1. 取最新已发布版本（status='published'）
	var template dto.WorkflowTemplate
	var bpmnXML []byte
	var description sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT id, key, name, description, version, bpmn_xml FROM workflow_templates WHERE tenant_id=$1 AND key=$2 AND status='published' ORDER BY updated_at DESC, id DESC LIMIT 1`, tenantID, key).Scan(
		&template.ID, &template.Key, &template.Name, &description, &template.Version, &bpmnXML,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWorkflowTemplateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询已发布模板失败: %w", err)
	}
	if description.Valid {
		template.Description = description.String
	}

	// bpmn_xml 是 jsonb 列，存储时是 JSON 字符串（catalog CreateDraft 用 to_jsonb($9::text)）
	// 取出来时 json.Marshal 把它包成 `"..."`，需要解一次。
	if len(bpmnXML) > 0 {
		var encoded string
		if json.Unmarshal(bpmnXML, &encoded) == nil && encoded != "" {
			template.BPMNXML = encoded
		} else {
			template.BPMNXML = string(bpmnXML)
		}
	}
	if strings.TrimSpace(template.BPMNXML) == "" {
		return nil, fmt.Errorf("%w: 已发布版本没有 BPMN XML，无法重新部署", ErrWorkflowTemplateInvalid)
	}

	// 2. 计算下一个版本号：取该 key 当前 process_definitions 中最新版本 + minor+1
	var latestVersion sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT version FROM process_definitions WHERE tenant_id=$1 AND key=$2 ORDER BY updated_at DESC, id DESC LIMIT 1`, tenantID, key).Scan(&latestVersion); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("查询当前部署版本失败: %w", err)
	}
	nextVersion := bumpMinorVersion(latestVersion.String)

	// 3. 创建部署记录 + 新的 process_definition 并降级旧版本（事务保证原子性）
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("开始重载事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	deploymentIDStr := fmt.Sprintf("RELOAD-%s-%d", key, time.Now().Unix())
	deploymentName := fmt.Sprintf("%s reload v%s", template.Name, nextVersion)
	var deploymentID int
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO process_deployments (deployment_id, deployment_name, deployment_source, deployment_comment, deployment_time, is_active, tenant_id) VALUES ($1,$2,$3,$4,NOW(),true,$5) RETURNING id`,
		deploymentIDStr, deploymentName, template.BPMNXML, "Reload from workflow_templates catalog", tenantID,
	).Scan(&deploymentID); err != nil {
		return nil, fmt.Errorf("创建部署记录失败: %w", err)
	}

	// 把同 key 旧 is_latest=true 的降级
	if _, err := tx.ExecContext(ctx,
		`UPDATE process_definitions SET is_latest=false, updated_at=NOW() WHERE tenant_id=$1 AND key=$2 AND is_latest=true`,
		tenantID, key,
	); err != nil {
		return nil, fmt.Errorf("降级旧版本失败: %w", err)
	}

	var definitionID int
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO process_definitions (key, name, description, version, category, bpmn_xml, process_variables, is_active, is_latest, deployment_id, deployment_name, deployed_at, tenant_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,'{}',true,true,$7,$8,NOW(),$9,NOW(),NOW()) RETURNING id`,
		key, template.Name, template.Description, nextVersion, "default", template.BPMNXML, deploymentID, deploymentName, tenantID,
	).Scan(&definitionID); err != nil {
		return nil, fmt.Errorf("创建新流程定义失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交重载事务失败: %w", err)
	}

	return &dto.ReloadWorkflowTemplateResponse{
		Key:               key,
		Name:              template.Name,
		PreviousVersion:   latestVersion.String,
		NewVersion:        nextVersion,
		DeploymentID:      deploymentIDStr,
		ProcessDefinitionID: definitionID,
		Source:            "workflow_templates",
		ReloadedAt:        time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// bumpMinorVersion 按 major.minor.0 约定递增：major 不变，minor+1，patch 归零。
// 空字符串返回 "1.0.0"；major 段无法解析时也回退 "1.0.0"，避免和 (tenant_id,key,version)
// 唯一索引冲突；minor 段无法解析则按 0 处理。
func bumpMinorVersion(current string) string {
	trimmed := strings.TrimSpace(current)
	if trimmed == "" {
		return "1.0.0"
	}
	parts := strings.Split(trimmed, ".")
	major, err := strconv.Atoi(parts[0])
	if err != nil || major <= 0 {
		return "1.0.0"
	}
	minor := 0
	if len(parts) > 1 {
		if n, perr := strconv.Atoi(strings.TrimSpace(parts[1])); perr == nil && n >= 0 {
			minor = n
		}
	}
	minor++
	return fmt.Sprintf("%d.%d.0", major, minor)
}

func (s *BPMNWorkflowTemplateCatalog) Versions(ctx context.Context, tenantID int, key string) ([]*dto.WorkflowTemplate, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id,key,name,description,domain,form_schema,approval_policy,ontology_bindings,sla_config,bpmn_xml,version,status,is_public,created_by,created_at,updated_at FROM workflow_templates WHERE tenant_id=$1 AND key=$2 ORDER BY updated_at DESC, id DESC", tenantID, key)
	if err != nil {
		return nil, fmt.Errorf("查询模板版本失败: %w", err)
	}
	defer rows.Close()
	items := make([]*dto.WorkflowTemplate, 0)
	for rows.Next() {
		item, scanErr := scanWorkflowTemplate(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *BPMNWorkflowTemplateCatalog) nextVersion(ctx context.Context, tenantID int, key string) (string, error) {
	var current sql.NullString
	err := s.db.QueryRowContext(ctx, "SELECT version FROM workflow_templates WHERE tenant_id=$1 AND key=$2 ORDER BY id DESC LIMIT 1", tenantID, key).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) || !current.Valid {
		return "1.0.0", nil
	}
	parts := strings.Split(current.String, ".")
	minor := 0
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	major := 1
	if len(parts) > 0 {
		major, _ = strconv.Atoi(parts[0])
		if major <= 0 {
			major = 1
		}
	}
	return fmt.Sprintf("%d.%d.0", major, minor+1), nil
}

func validateTemplateRequest(req *dto.CreateWorkflowTemplateRequest) error {
	if req == nil || strings.TrimSpace(req.Key) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Domain) == "" || strings.TrimSpace(req.BPMNXML) == "" {
		return fmt.Errorf("%w: 模板基本字段不能为空", ErrWorkflowTemplateInvalid)
	}
	return nil
}

func marshalTemplateJSON(values ...map[string]interface{}) ([]byte, []byte, []byte, []byte, error) {
	result := make([][]byte, 4)
	for i, value := range values {
		if value == nil {
			value = map[string]interface{}{}
		}
		data, err := json.Marshal(value)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("%w: JSON 配置无效", ErrWorkflowTemplateInvalid)
		}
		result[i] = data
	}
	return result[0], result[1], result[2], result[3], nil
}

type workflowTemplateScanner interface {
	Scan(dest ...interface{}) error
}

func scanWorkflowTemplate(row workflowTemplateScanner) (*dto.WorkflowTemplate, error) {
	var item dto.WorkflowTemplate
	var form, approval, bindings, sla, xml []byte
	var created, updated time.Time
	if err := row.Scan(&item.ID, &item.Key, &item.Name, &item.Description, &item.Domain, &form, &approval, &bindings, &sla, &xml, &item.Version, &item.Status, &item.IsPublic, &item.CreatedBy, &created, &updated); err != nil {
		return nil, fmt.Errorf("读取模板记录失败: %w", err)
	}
	for _, entry := range []struct {
		data   []byte
		target *map[string]interface{}
	}{{form, &item.FormSchema}, {approval, &item.ApprovalPolicy}, {bindings, &item.OntologyBindings}, {sla, &item.SLAConfig}} {
		if len(entry.data) > 0 && string(entry.data) != "null" {
			if err := json.Unmarshal(entry.data, entry.target); err != nil {
				return nil, fmt.Errorf("解析模板配置失败: %w", err)
			}
		}
	}
	item.BPMNXML = string(xml)
	var encodedXML string
	if len(xml) > 0 && json.Unmarshal(xml, &encodedXML) == nil {
		item.BPMNXML = encodedXML
	}
	item.CreatedAt, item.UpdatedAt = created.Format(time.RFC3339), updated.Format(time.RFC3339)
	return &item, nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
