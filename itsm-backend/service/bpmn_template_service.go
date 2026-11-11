package service

import (
	"bytes"
	"context"
	"embed"
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processdeployment"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

//go:embed bpmn/*.bpmn
var bpmnTemplates embed.FS

// BPMNTemplateService BPMN模板服务
type BPMNTemplateService struct {
	client *ent.Client
	logger *zap.Logger

	// deployTemplateFn 只供包内测试替换单次部署动作，用来验证外层对
	// ErrTemplateAlreadyDeployed 的幂等处理；生产路径恒等于 s.deployTemplate。
	// 保留 nil 兜底，避免绕过构造器直接建对象时出现空指针。
	deployTemplateFn func(context.Context, *TemplateInfo, int, string, []byte) error
}

// NewBPMNTemplateService 创建BPMN模板服务
func NewBPMNTemplateService(client *ent.Client) *BPMNTemplateService {
	s := &BPMNTemplateService{client: client, logger: zap.NewNop()}
	s.deployTemplateFn = s.deployTemplate
	return s
}

// TemplateInfo 模板信息
type TemplateInfo struct {
	ID          string
	Name        string
	Category    string
	SubCategory string
	Version     string
	Description string
	Filename    string
}

// LoadAndDeployTemplates 加载并部署所有内置模板，并把已有租户升级到模板声明的新版本。
//
// 「已存在同 key 的定义」不等于「模板已部署到位」：旧实现按 key 存在即跳过，
// 导致内置模板升级后只有全新租户能拿到新流程，存量租户永远停在最早版本。
// 现按模板 metaData 声明的版本比较：高于已部署最新版本才升级；低于时不回滚；
// 同版本但内容变化说明版本号漏提升，按补丁版本部署并告警。
func (s *BPMNTemplateService) LoadAndDeployTemplates(ctx context.Context, tenantID int) ([]*TemplateInfo, error) {
	templates, err := s.listTemplates()
	if err != nil {
		return nil, errors.Wrap(err, "列出模板失败")
	}

	deployed := make([]*TemplateInfo, 0, len(templates))
	deployFailures := make([]string, 0)
	deployTemplate := s.deployTemplate
	if s.deployTemplateFn != nil {
		deployTemplate = s.deployTemplateFn
	}

	for _, tmpl := range templates {
		latest, err := s.latestTemplateDefinition(ctx, tmpl.ID, tenantID)
		if err != nil {
			s.logger.Sugar().Errorw("检查模板部署状态失败，跳过该模板", "template", tmpl.ID, "error", err)
			deployFailures = append(deployFailures, tmpl.ID)
			continue
		}

		data, err := bpmnTemplates.ReadFile(filepath.Join("bpmn", tmpl.Filename))
		if err != nil {
			s.logger.Sugar().Errorw("读取内置模板失败，跳过该模板", "template", tmpl.ID, "error", err)
			deployFailures = append(deployFailures, tmpl.ID)
			continue
		}

		deployVersion, needed := s.resolveTemplateDeployVersion(ctx, tenantID, tmpl, latest, data)
		if !needed {
			deployed = append(deployed, tmpl)
			continue
		}

		// 部署模板（deployTemplate 内含 lint 门禁）。
		// 单个坏模板只告警并跳过，不阻断其余模板部署。
		if err := deployTemplate(ctx, tmpl, tenantID, deployVersion, data); err != nil {
			// 多副本同时启动时另一个实例可能已写入同一版本。deployTemplate
			// 内部事务已随冲突回滚，这里按 sentinel 判幂等命中，不是在事务内
			// 吞一个还需要善后的约束错误（静态门禁 5.13 的禁令针对后者）。
			if errors.Is(err, ErrTemplateAlreadyDeployed) {
				s.logger.Sugar().Infow("内置模板已由其他实例部署",
					"template", tmpl.ID, "version", deployVersion, "tenant_id", tenantID)
				deployed = append(deployed, tmpl)
				continue
			}
			s.logger.Sugar().Errorw("内置模板未部署（lint 门禁拦截）", "template", tmpl.ID, "version", deployVersion, "error", err)
			deployFailures = append(deployFailures, tmpl.ID)
			continue
		}

		deployed = append(deployed, tmpl)
	}

	if len(deployFailures) > 0 {
		return deployed, fmt.Errorf("%d 个内置模板未通过部署门禁: %s", len(deployFailures), strings.Join(deployFailures, ", "))
	}
	return deployed, nil
}

// resolveTemplateDeployVersion 判断是否需要部署以及部署哪个版本。
// 返回 needed=false 表示已是最新（或本地模板比线上旧，禁止降级）。
func (s *BPMNTemplateService) resolveTemplateDeployVersion(ctx context.Context, tenantID int, tmpl *TemplateInfo, latest *ent.ProcessDefinition, data []byte) (string, bool) {
	if latest == nil {
		return tmpl.Version, true
	}
	if bytes.Equal(latest.BpmnXML, data) {
		return "", false
	}

	embedded, embeddedOK := parseTemplateVersion(tmpl.Version)
	installed, installedOK := parseTemplateVersion(latest.Version)
	if !embeddedOK || !installedOK {
		s.logger.Sugar().Warnw("内置模板版本号不可比较，跳过自动升级",
			"template", tmpl.ID, "embedded_version", tmpl.Version, "installed_version", latest.Version, "tenant_id", tenantID)
		return "", false
	}

	switch {
	case embedded.greaterThan(installed):
		return tmpl.Version, true
	case embedded.equals(installed):
		// 内容变了但版本没提升：按补丁版本部署，避免与线上同 key+version 唯一约束冲突。
		s.logger.Sugar().Warnw("内置模板内容变化但版本未提升，按补丁版本部署",
			"template", tmpl.ID, "version", tmpl.Version, "tenant_id", tenantID)
		return s.nextPatchVersion(ctx, tenantID, tmpl.ID, embedded), true
	default:
		s.logger.Sugar().Warnw("内置模板版本低于已部署版本，跳过（不自动降级）",
			"template", tmpl.ID, "embedded_version", tmpl.Version, "installed_version", latest.Version, "tenant_id", tenantID)
		return "", false
	}
}

// listTemplates 列出所有内置模板
func (s *BPMNTemplateService) listTemplates() ([]*TemplateInfo, error) {
	templates := make([]*TemplateInfo, 0)

	// 读取嵌入的模板文件
	err := fs.WalkDir(bpmnTemplates, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".bpmn") {
			return nil
		}

		// 解析文件名获取模板信息
		filename := filepath.Base(path)
		key := strings.TrimSuffix(filename, ".bpmn")

		info := &TemplateInfo{
			ID:       key,
			Filename: filename,
		}

		// 根据文件名设置默认值
		switch key {
		case "ticket_general_flow":
			info.Name = "通用工单流程"
			info.Category = "ticket"
			info.Description = "通用工单处理流程"
		case "ticket_assignment_flow":
			info.Name = "工单分配流程"
			info.Category = "ticket"
			info.SubCategory = "assignment"
			info.Description = "工单自动分配处理流程"
		case "change_normal_flow":
			info.Name = "普通变更流程"
			info.Category = "change"
			info.SubCategory = "normal"
			info.Description = "普通变更管理流程"
		case "change_emergency_flow":
			info.Name = "紧急变更流程"
			info.Category = "change"
			info.SubCategory = "emergency"
			info.Description = "紧急变更快速处理流程"
		case "incident_emergency_flow":
			info.Name = "紧急事件流程"
			info.Category = "incident"
			info.Description = "紧急事件快速响应流程"
		case "service_request_flow":
			info.Name = "服务请求流程"
			info.Category = "service_request"
			info.Description = "标准服务请求处理流程"
		case "problem_management_flow":
			info.Name = "问题管理流程"
			info.Category = "problem"
			info.Description = "问题管理全流程"
		case "release_approval_flow":
			info.Name = "发布审批流程"
			info.Category = "release"
			info.Description = "软件发布审批管理流程"
		case "dev_approval_ops_flow":
			info.Name = "开发审批运维流程"
			info.Category = "ticket"
			info.SubCategory = "approval"
			info.Description = "开发提交→主管审批→运维操作三级流程"
		default:
			info.Name = key
			info.Category = "default"
		}

		info.Version = "1.0.0"
		if content, readErr := bpmnTemplates.ReadFile(filepath.Join("bpmn", filename)); readErr == nil {
			info.Version = declaredTemplateVersion(content, info.Version)
		}
		templates = append(templates, info)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "遍历模板目录失败")
	}

	return templates, nil
}

// latestTemplateDefinition 返回租户下该 key 的最新流程定义；未部署时返回 nil。
func (s *BPMNTemplateService) latestTemplateDefinition(ctx context.Context, templateID string, tenantID int) (*ent.ProcessDefinition, error) {
	definition, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(templateID),
			processdefinition.TenantID(tenantID),
			processdefinition.IsLatest(true),
		).
		Order(ent.Desc(processdefinition.FieldID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "查询流程定义失败")
	}
	return definition, nil
}

// nextPatchVersion 在模板声明版本的同 major.minor 下取下一个补丁号，
// 供「内容变了但版本号未提升」的升级使用；重试时不会与已写入的补丁版本冲突。
func (s *BPMNTemplateService) nextPatchVersion(ctx context.Context, tenantID int, templateID string, base templateVersion) string {
	definitions, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(templateID),
			processdefinition.TenantID(tenantID),
		).
		All(ctx)
	if err != nil {
		s.logger.Sugar().Warnw("查询已部署版本失败，按声明版本+1 处理",
			"template", templateID, "tenant_id", tenantID, "error", err)
	}
	next := base
	next.patch++
	if err == nil {
		for _, definition := range definitions {
			installed, ok := parseTemplateVersion(definition.Version)
			if !ok || installed.major != base.major || installed.minor != base.minor {
				continue
			}
			if installed.patch >= next.patch {
				next.patch = installed.patch + 1
			}
		}
	}
	return next.String()
}

type templateVersion struct {
	major int
	minor int
	patch int
}

func (v templateVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

func (v templateVersion) equals(other templateVersion) bool {
	return v.major == other.major && v.minor == other.minor && v.patch == other.patch
}

func (v templateVersion) greaterThan(other templateVersion) bool {
	if v.major != other.major {
		return v.major > other.major
	}
	if v.minor != other.minor {
		return v.minor > other.minor
	}
	return v.patch > other.patch
}

func parseTemplateVersion(raw string) (templateVersion, bool) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) == 0 || len(parts) > 3 {
		return templateVersion{}, false
	}
	segments := [3]int{}
	for i, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || value < 0 {
			return templateVersion{}, false
		}
		segments[i] = value
	}
	return templateVersion{major: segments[0], minor: segments[1], patch: segments[2]}, true
}

// declaredTemplateVersion 读取模板 process 下 extensionElements 里的
// metaData name="version"；缺失或不可解析时回退 fallback。
func declaredTemplateVersion(data []byte, fallback string) string {
	var definitions struct {
		Processes []struct {
			ExtensionElements struct {
				MetaData []struct {
					Name  string `xml:"name,attr"`
					Value string `xml:",chardata"`
				} `xml:"metaData"`
			} `xml:"extensionElements"`
		} `xml:"process"`
	}
	if err := xml.Unmarshal(data, &definitions); err != nil {
		return fallback
	}
	for _, process := range definitions.Processes {
		for _, meta := range process.ExtensionElements.MetaData {
			if !strings.EqualFold(strings.TrimSpace(meta.Name), "version") {
				continue
			}
			if value := strings.TrimSpace(meta.Value); value != "" {
				return value
			}
		}
	}
	return fallback
}

// deployTemplate 部署单个模板版本。部署记录按版本区分，定义写入与
// 「旧版本取消 latest」在同一事务内完成，避免中途失败留下两个 latest。
//
// 多副本并发部署同一版本时，唯一约束冲突会把本函数内部的事务连坐回滚
// （defer Rollback 已执行，事务不可能再提交）。这里在出口统一把这类错误
// 翻译为 ErrTemplateAlreadyDeployed——它对外层的语义是「幂等命中，本次
// 无事可做」，而不是一个还需要事务善后动作的失败。外层据此吞错是安全的；
// 直接在调用点摸 ent.IsConstraintError 无法表达「内部事务已回滚」这一事实，
// 也是静态门禁 5.13 禁止的写法。
var ErrTemplateAlreadyDeployed = errors.New("bpmn template version already deployed")

func (s *BPMNTemplateService) deployTemplate(ctx context.Context, tmpl *TemplateInfo, tenantID int, version string, data []byte) error {
	// 部署前 lint 门禁：内置模板与用户导入的模板走同一套发布校验，否则 XML 语法损坏或
	// 连线悬空的模板会直接入库。解析失败或存在 error 级 issue 时拒绝部署；
	// 由上层决定跳过或报错。
	lintSvc := NewBPMNLintService()
	lintResult, lintErr := lintSvc.LintBPMNXML(data)
	if lintErr != nil {
		return fmt.Errorf("模板 %s 未通过 lint（XML 不可解析）: %w", tmpl.ID, lintErr)
	}
	if lintResult != nil && lintResult.HasErrors {
		msgs := make([]string, 0, lintResult.ErrorCount)
		for _, is := range lintResult.Issues {
			if is.Severity == "error" {
				msgs = append(msgs, is.Message)
			}
		}
		return fmt.Errorf("模板 %s 未通过 lint（%d 个 error）: %s", tmpl.ID, lintResult.ErrorCount, strings.Join(msgs, "; "))
	}

	now := time.Now()

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return errors.Wrap(err, "开启模板部署事务失败")
	}
	defer func() { _ = tx.Rollback() }()

	// 事务体的所有错误在此统一出口翻译：走到这里时 defer Rollback 已把事务
	// 收干净（提交成功后 Rollback 是无害的幂等空操作），约束冲突因此可以
	// 安全地按「已回滚的幂等命中」对外报告。
	return translateConstraintAsAlreadyDeployed(
		s.deployTemplateTx(ctx, tx, tmpl, tenantID, version, data, now))
}

// deployTemplateTx 在调用方管理的事务内完成单个模板版本的全部写入与提交。
func (s *BPMNTemplateService) deployTemplateTx(ctx context.Context, tx *ent.Tx, tmpl *TemplateInfo, tenantID int, version string, data []byte, now time.Time) error {
	deployment, err := s.ensureTemplateDeployment(ctx, tx, tmpl, tenantID, version, now)
	if err != nil {
		return err
	}

	existing, err := tx.ProcessDefinition.Query().
		Where(
			processdefinition.Key(tmpl.ID),
			processdefinition.TenantID(tenantID),
			processdefinition.Version(version),
		).
		First(ctx)
	switch {
	case err == nil:
		if _, err := tx.ProcessDefinition.UpdateOneID(existing.ID).
			SetName(tmpl.Name).
			SetDescription(tmpl.Description).
			SetCategory(tmpl.Category).
			SetBpmnXML(data).
			SetIsActive(true).
			SetDeploymentID(deployment.ID).
			SetDeployedAt(now).
			Save(ctx); err != nil {
			return errors.Wrap(err, "更新流程定义失败")
		}
	case ent.IsNotFound(err):
		if _, err := tx.ProcessDefinition.Create().
			SetKey(tmpl.ID).
			SetName(tmpl.Name).
			SetDescription(tmpl.Description).
			SetVersion(version).
			SetCategory(tmpl.Category).
			SetBpmnXML(data).
			SetIsActive(true).
			SetTenantID(tenantID).
			SetDeploymentID(deployment.ID).
			SetDeployedAt(now).
			Save(ctx); err != nil {
			return errors.Wrap(err, "保存流程定义失败")
		}
	default:
		return errors.Wrap(err, "查询流程定义失败")
	}

	// 目标版本落库后再翻转 latest，保证任一时刻最多一个最新版本。
	if _, err := tx.ProcessDefinition.Update().
		Where(
			processdefinition.Key(tmpl.ID),
			processdefinition.TenantID(tenantID),
			processdefinition.VersionNEQ(version),
			processdefinition.IsLatest(true),
		).
		SetIsLatest(false).
		Save(ctx); err != nil {
		return errors.Wrap(err, "更新旧版本状态失败")
	}
	if _, err := tx.ProcessDefinition.Update().
		Where(
			processdefinition.Key(tmpl.ID),
			processdefinition.TenantID(tenantID),
			processdefinition.Version(version),
		).
		SetIsLatest(true).
		Save(ctx); err != nil {
		return errors.Wrap(err, "更新最新版本标记失败")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "提交模板部署事务失败")
	}
	return nil
}

// translateConstraintAsAlreadyDeployed 把部署事务内的约束冲突翻译为
// ErrTemplateAlreadyDeployed：内部事务此刻已回滚（defer Rollback），调用方
// 拿到该 sentinel 即可按幂等命中处理，不必也不能再做事务内善后。
func translateConstraintAsAlreadyDeployed(err error) error {
	if err != nil && ent.IsConstraintError(err) {
		return ErrTemplateAlreadyDeployed
	}
	return err
}

// ensureTemplateDeployment 幂等地确保部署记录存在（同 ID 复用），返回部署行。

func (s *BPMNTemplateService) ensureTemplateDeployment(ctx context.Context, tx *ent.Tx, tmpl *TemplateInfo, tenantID int, version string, now time.Time) (*ent.ProcessDeployment, error) {
	deploymentID := fmt.Sprintf("%s-v%s", tmpl.ID, version)
	deployment, err := tx.ProcessDeployment.Query().
		Where(
			processdeployment.DeploymentID(deploymentID),
			processdeployment.TenantID(tenantID),
		).
		First(ctx)
	if err == nil {
		return deployment, nil
	}
	if !ent.IsNotFound(err) {
		return nil, errors.Wrap(err, "查询部署记录失败")
	}
	return tx.ProcessDeployment.Create().
		SetDeploymentID(deploymentID).
		SetDeploymentName(fmt.Sprintf("%s v%s", tmpl.Name, version)).
		SetDeploymentTime(now).
		SetTenantID(tenantID).
		SetDeployedBy("system").
		Save(ctx)
}

// GetTemplateList 获取模板列表（不部署）
func (s *BPMNTemplateService) GetTemplateList() ([]*TemplateInfo, error) {
	return s.listTemplates()
}

// SuggestTemplates searches the immutable built-in catalog. tenantID is kept
// in the contract so future tenant-owned catalog entries cannot be returned
// without an authenticated scope.
func (s *BPMNTemplateService) SuggestTemplates(ctx context.Context, tenantID int, keyword, processType string) ([]*dto.BPMNTemplateSuggestion, error) {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" || tenantID <= 0 {
		return []*dto.BPMNTemplateSuggestion{}, nil
	}

	result := make([]*dto.BPMNTemplateSuggestion, 0)
	builtIns, err := s.listTemplates()
	if err != nil {
		return nil, errors.Wrap(err, "列出内置模板失败")
	}
	for _, tmpl := range builtIns {
		if processType != "" && tmpl.Category != processType {
			continue
		}
		text := strings.ToLower(tmpl.ID + " " + tmpl.Name + " " + tmpl.Description)
		if !strings.Contains(text, keyword) {
			continue
		}
		result = append(result, &dto.BPMNTemplateSuggestion{ID: tmpl.ID, Name: tmpl.Name, Description: tmpl.Description, ProcessType: tmpl.Category, Score: templateMatchScore(text, keyword)})
	}

	sort.SliceStable(result, func(i, j int) bool { return result[i].Score > result[j].Score })
	return result, nil
}

func templateMatchScore(text, keyword string) float64 {
	if text == keyword {
		return 1
	}
	if strings.HasPrefix(text, keyword) {
		return 0.95
	}
	return 0.75
}

// DeployTemplateByName 根据名称部署单个模板
func (s *BPMNTemplateService) DeployTemplateByName(ctx context.Context, name string, tenantID int) error {
	templates, err := s.listTemplates()
	if err != nil {
		return err
	}

	for _, tmpl := range templates {
		if tmpl.ID != name {
			continue
		}
		data, err := bpmnTemplates.ReadFile(filepath.Join("bpmn", tmpl.Filename))
		if err != nil {
			return errors.Wrap(err, "读取模板文件失败")
		}
		return s.deployTemplate(ctx, tmpl, tenantID, tmpl.Version, data)
	}

	return fmt.Errorf("模板 %s 不存在", name)
}

// GetTemplateContent 获取模板内容
func (s *BPMNTemplateService) GetTemplateContent(name string) ([]byte, error) {
	path := filepath.Join("bpmn", name+".bpmn")
	return bpmnTemplates.ReadFile(path)
}

// ExportTemplateToFile 将已部署的流程导出为BPMN文件
func (s *BPMNTemplateService) ExportTemplateToFile(ctx context.Context, key string, tenantID int, outputPath string) error {
	definition, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(key),
			processdefinition.TenantID(tenantID),
		).
		Only(ctx)
	if err != nil {
		return errors.Wrap(err, "查询流程定义失败")
	}

	// 写入文件
	return os.WriteFile(outputPath, definition.BpmnXML, 0o644)
}
