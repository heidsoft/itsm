package service

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"itsm-backend/ent"
	"itsm-backend/ent/processdefinition"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

//go:embed bpmn/*.bpmn
var bpmnTemplates embed.FS

// BPMNTemplateService BPMN模板服务
type BPMNTemplateService struct {
	client *ent.Client
}

// NewBPMNTemplateService 创建BPMN模板服务
func NewBPMNTemplateService(client *ent.Client) *BPMNTemplateService {
	return &BPMNTemplateService{client: client}
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

// LoadAndDeployTemplates 加载并部署所有内置模板
func (s *BPMNTemplateService) LoadAndDeployTemplates(ctx context.Context, tenantID int) ([]*TemplateInfo, error) {
	templates, err := s.listTemplates()
	if err != nil {
		return nil, errors.Wrap(err, "列出模板失败")
	}

	deployed := make([]*TemplateInfo, 0, len(templates))

	for _, tmpl := range templates {
		// 检查是否已部署
		exists, err := s.isTemplateDeployed(ctx, tmpl.ID, tenantID)
		if err != nil {
			return nil, errors.Wrap(err, "检查模板部署状态失败")
		}

		if !exists {
			// 部署模板
			err := s.deployTemplate(ctx, tmpl, tenantID)
			if err != nil {
				return nil, fmt.Errorf("部署模板 %s 失败: %w", tmpl.Name, err)
			}
		}

		deployed = append(deployed, tmpl)
	}

	return deployed, nil
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
		default:
			info.Name = key
			info.Category = "default"
		}

		info.Version = "1.0.0"
		templates = append(templates, info)
		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "遍历模板目录失败")
	}

	return templates, nil
}

// isTemplateDeployed 检查模板是否已部署
func (s *BPMNTemplateService) isTemplateDeployed(ctx context.Context, templateID string, tenantID int) (bool, error) {
	count, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(templateID),
			processdefinition.TenantID(tenantID),
		).
		Count(ctx)

	if err != nil {
		return false, errors.Wrap(err, "查询流程定义失败")
	}

	return count > 0, nil
}

// deployTemplate 部署单个模板
func (s *BPMNTemplateService) deployTemplate(ctx context.Context, tmpl *TemplateInfo, tenantID int) error {
	// 读取模板文件
	data, err := bpmnTemplates.ReadFile(filepath.Join("bpmn", tmpl.Filename))
	if err != nil {
		return errors.Wrap(err, "读取模板文件失败")
	}

	// 获取当前时间
	now := time.Now()

	// 先创建部署记录
	deployment, err := s.client.ProcessDeployment.Create().
		SetDeploymentID(fmt.Sprintf("%s-v1", tmpl.ID)).
		SetDeploymentName(fmt.Sprintf("%s v1", tmpl.Name)).
		SetDeploymentTime(now).
		SetTenantID(tenantID).
		SetDeployedBy("system").
		Save(ctx)
	if err != nil {
		return errors.Wrap(err, "创建部署记录失败")
	}

	// 创建流程定义（关联部署ID）
	_, err = s.client.ProcessDefinition.Create().
		SetKey(tmpl.ID).
		SetName(tmpl.Name).
		SetDescription(tmpl.Description).
		SetVersion(tmpl.Version).
		SetCategory(tmpl.Category).
		SetBpmnXML(data).
		SetIsActive(true).
		SetIsLatest(true).
		SetTenantID(tenantID).
		SetDeploymentID(deployment.ID).
		SetDeployedAt(now).
		Save(ctx)

	if err != nil {
		return errors.Wrap(err, "保存流程定义失败")
	}

	return nil
}

// GetTemplateList 获取模板列表（不部署）
func (s *BPMNTemplateService) GetTemplateList() ([]*TemplateInfo, error) {
	return s.listTemplates()
}

// DeployTemplateByName 根据名称部署单个模板
func (s *BPMNTemplateService) DeployTemplateByName(ctx context.Context, name string, tenantID int) error {
	templates, err := s.listTemplates()
	if err != nil {
		return err
	}

	for _, tmpl := range templates {
		if tmpl.ID == name {
			return s.deployTemplate(ctx, tmpl, tenantID)
		}
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
	return os.WriteFile(outputPath, definition.BpmnXML, 0644)
}
