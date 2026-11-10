package bpmn

import (
	"context"

	"itsm-backend/dto"
)

// WorkflowTemplateCatalog 是模板目录 handler 消费的窄契约（批次 3 第 1 步）。
//
// 方法集只由 handlers/bpmn/workflow_template.go 的真实调用面决定；值类型与
// 错误 sentinel 全部取自 dto，因此本文件不引用共享层包（Gate C.8 / L.3 全文
// 扫描会把注释里的包路径也算命中，这里刻意不写）。
// 后续各组按同样方式在本文件追加接口，直到整个 bpmn 域归零。
type WorkflowTemplateCatalog interface {
	List(ctx context.Context, tenantID int, keyword, domain, status string, page, pageSize int) (*dto.WorkflowTemplateListResponse, error)
	Get(ctx context.Context, tenantID int, key, version string) (*dto.WorkflowTemplate, error)
	CreateDraft(ctx context.Context, tenantID, userID int, req *dto.CreateWorkflowTemplateRequest) (*dto.WorkflowTemplate, error)
	UpdateDraft(ctx context.Context, tenantID int, key string, req *dto.UpdateWorkflowTemplateRequest) (*dto.WorkflowTemplate, error)
	Publish(ctx context.Context, tenantID int, key string) (*dto.WorkflowTemplate, error)
	Archive(ctx context.Context, tenantID int, key string) error
	Reload(ctx context.Context, tenantID int, key string) (*dto.ReloadWorkflowTemplateResponse, error)
	Versions(ctx context.Context, tenantID int, key string) ([]*dto.WorkflowTemplate, error)
}
