package cmdb

// CMDB 生产服务的域内窄契约。方法集 = handlers/cmdb 全部生产文件对 7 个
// CI 服务与 ToolRegistry 的调用并集（48 个方法，用 Go AST 从 service 侧抽取）。
// 签名闭包里的 LifecycleChangeParams / ToolDefinition 已下沉 dto，共享层保留
// 同名别名，因此具体指针自动满足这些接口，app.go 与 router.go 零改动。

import (
	"context"

	"itsm-backend/dto"
)

// CITypeCatalog 是 CITypeService 在 CMDB 域内的消费面。
type CITypeCatalog interface {
	CreateCIType(ctx context.Context, req *dto.CreateCITypeRequest, tenantID int) (*dto.CITypeResponse, error)
	DeleteCIType(ctx context.Context, id, tenantID int) error
	GetCITypeByID(ctx context.Context, id, tenantID int) (*dto.CITypeResponse, error)
	ListCITypes(ctx context.Context, tenantID int, page, pageSize int, search string) (*dto.CITypeListResponse, error)
	UpdateCIType(ctx context.Context, id, tenantID int, req *dto.UpdateCITypeRequest) (*dto.CITypeResponse, error)
}

// CIAttributeDefinitionCatalog 是 CIAttributeDefinitionService 在 CMDB 域内的消费面。
type CIAttributeDefinitionCatalog interface {
	CreateCIAttributeDefinition(ctx context.Context, req *dto.CreateCIAttributeDefinitionRequest, tenantID int) (*dto.CIAttributeDefinitionResponse, error)
	DeleteCIAttributeDefinition(ctx context.Context, id, tenantID int) error
	GetCIAttributeDefinitionByID(ctx context.Context, id, tenantID int) (*dto.CIAttributeDefinitionResponse, error)
	ListCIAttributeDefinitionsByCITypeID(ctx context.Context, ciTypeID, tenantID int) ([]*dto.CIAttributeDefinitionResponse, error)
	UpdateCIAttributeDefinition(ctx context.Context, id, tenantID int, req *dto.UpdateCIAttributeDefinitionRequest) (*dto.CIAttributeDefinitionResponse, error)
}

// ConfigurationItemCatalog 是 ConfigurationItemService 在 CMDB 域内的消费面。
type ConfigurationItemCatalog interface {
	AddTagsToCI(ctx context.Context, ciID, tenantID int, tagIDs []int) (*dto.CIResponse, error)
	BatchCreateCI(ctx context.Context, req *dto.BatchCreateCIRequest, tenantID int) (*dto.BatchOperationResponse, error)
	BatchDeleteCI(ctx context.Context, req *dto.BatchDeleteCIRequest, tenantID int) (*dto.BatchOperationResponse, error)
	BatchUpdateCI(ctx context.Context, req *dto.BatchUpdateCIRequest, tenantID int) (*dto.BatchOperationResponse, error)
	BatchUpdateLifecycleStatus(ctx context.Context, ids []int, tenantID int, params dto.LifecycleChangeParams, operatorID int, operatorName string) (*dto.BatchOperationResponse, error)
	CreateCI(ctx context.Context, req *dto.CreateCIRequest, tenantID int) (*dto.CIResponse, error)
	DeleteCI(ctx context.Context, id, tenantID int) error
	GetCIByID(ctx context.Context, id, tenantID int, withRelations bool) (*dto.CIResponse, error)
	GetCIStats(ctx context.Context, tenantID int) (*dto.CIStatsResponse, error)
	GetLifecycleHistory(ctx context.Context, id int, tenantID int) ([]map[string]interface{}, error)
	ListCIs(ctx context.Context, tenantID int, req *dto.ListCIRequest) (*dto.CIListResponse, error)
	RemoveTagsFromCI(ctx context.Context, ciID, tenantID int, tagIDs []int) (*dto.CIResponse, error)
	SearchCI(ctx context.Context, tenantID int, req *dto.CISearchRequest) (*dto.ListResponse[dto.CIResponse], error)
	UpdateCI(ctx context.Context, id, tenantID int, req *dto.UpdateCIRequest) (*dto.CIResponse, error)
	UpdateLifecycleStatus(ctx context.Context, id int, tenantID int, params dto.LifecycleChangeParams, operatorID int, operatorName string) (*dto.CIResponse, error)
}

// CIRelationshipCatalog 是 CIRelationshipService 在 CMDB 域内的消费面。
type CIRelationshipCatalog interface {
	CreateCIRelationship(ctx context.Context, req *dto.CreateCIRelationshipRequest, tenantID int) (*dto.CIRelationshipResponse, error)
	DeleteCIRelationship(ctx context.Context, id, tenantID int) error
	GetCIImpactAnalysis(ctx context.Context, ciID, tenantID int, maxDepth int) (*dto.CIImpactAnalysisResponse, error)
	GetCIRelationshipByID(ctx context.Context, id, tenantID int) (*dto.CIRelationshipResponse, error)
	GetCITopology(ctx context.Context, ciID, tenantID, maxDepth int) (*dto.TopologyGraph, error)
	ListAllCIRelationships(ctx context.Context, tenantID int, page, pageSize int, relationType string) (*dto.CIRelationshipListResponse, error)
	ListCIRelationshipsByCIID(ctx context.Context, ciID, tenantID int, direction string) ([]*dto.CIRelationshipResponse, error)
	UpdateCIRelationship(ctx context.Context, id, tenantID int, req *dto.UpdateCIRelationshipRequest) (*dto.CIRelationshipResponse, error)
}

// CIHistoryCatalog 是 CIHistoryService 在 CMDB 域内的消费面。
type CIHistoryCatalog interface {
	GetCIHistory(ctx context.Context, ciID, tenantID int, page, pageSize int) (*dto.CIHistoryListResponse, error)
	RevertCIVersion(ctx context.Context, ciID, tenantID, operatorID int, operatorName string, req *dto.RevertCIVersionRequest) (*dto.CIResponse, error)
}

// CITagCatalog 是 CITagService 在 CMDB 域内的消费面。
type CITagCatalog interface {
	CreateCITag(ctx context.Context, req *dto.CreateCITagRequest, tenantID int) (*dto.CITagResponse, error)
	DeleteCITag(ctx context.Context, id, tenantID int) error
	GetCITagByID(ctx context.Context, id, tenantID int) (*dto.CITagResponse, error)
	ListCITags(ctx context.Context, tenantID int, page, pageSize int, search string) (*dto.CITagListResponse, error)
	UpdateCITag(ctx context.Context, id, tenantID int, req *dto.UpdateCITagRequest) (*dto.CITagResponse, error)
}

// CMDBImportExportCatalog 是 CMDBImportExportService 在 CMDB 域内的消费面。
type CMDBImportExportCatalog interface {
	CreateExportTask(ctx context.Context, req *dto.ExportCIRequest, tenantID int, operatorID int, operatorName string) (*dto.ExportCIResult, error)
	CreateImportTask(ctx context.Context, req *dto.ImportCIRequest, tenantID int, operatorID int, operatorName string) (*dto.ImportCIResult, error)
	GetExportTaskStatus(ctx context.Context, taskID string, tenantID int) (*dto.ExportCIResult, error)
	GetImportTaskStatus(ctx context.Context, taskID string, tenantID int) (*dto.ImportCIResult, error)
	ListExportTasks(ctx context.Context, tenantID int, page, pageSize int) (*dto.ListResponse[dto.ExportCIResult], error)
	ListImportTasks(ctx context.Context, tenantID int, page, pageSize int) (*dto.ListResponse[dto.ImportCIResult], error)
	ResolveExportDownload(ctx context.Context, taskID string, tenantID, operatorID int) (string, string, error)
}

// ToolRegistryCatalog 是 ToolRegistry 在 CMDB 域内的消费面。
type ToolRegistryCatalog interface {
	ListToolsForTenant(ctx context.Context, tenantID int) []dto.ToolDefinition
}
