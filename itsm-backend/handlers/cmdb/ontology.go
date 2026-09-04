package cmdb

import (
	"encoding/json"

	"itsm-backend/common"
	"itsm-backend/common/handlerctx"
	"itsm-backend/dto"
	"itsm-backend/ent/schema"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
)

// cmdbOntologyVersion 本体词表版本号。
// 词表/枚举/工具面发生破坏性变更时递增，Agent 侧可据此缓存失效。
const cmdbOntologyVersion = "2026-09-04.1"

// SetToolRegistry 注入 AI 工具注册表（bootstrap 装配；未注入时 ontology 响应不含 aiTools）。
func (c *ProductionService) SetToolRegistry(tr *service.ToolRegistry) { c.toolRegistry = tr }

// GetOntology CMDB 本体自描述端点（AI-Native introspect 入口）
// @Summary 获取 CMDB 本体自描述
// @Description 一次性返回 CI 类型（含属性定义与 schema）、受控关系词表、受控枚举值域与可用 AI 工具，供 LLM Agent 自省与闭环操作
// @Tags CMDB
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} common.Response{data=dto.CMDBOntologyResponse}
// @Router /api/v1/cmdb/ontology [get]
func (c *ProductionService) GetOntology(ctx *gin.Context) {
	tenantID, ok := handlerctx.ResolveTenantID(ctx)
	if !ok {
		return
	}

	resp := dto.CMDBOntologyResponse{
		Version:           cmdbOntologyVersion,
		CITypes:           make([]dto.CMDBOntologyCIType, 0, 32),
		RelationshipTypes: make([]dto.CMDBOntologyRelationshipType, 0, len(schema.CIRelationshipTypeVocabulary)),
		Enums:             cmdbOntologyEnums(),
	}

	// 1. CI 类型（含解析后的属性 schema + 类型级属性定义）
	typeRes, err := c.ciTypeService.ListCITypes(ctx.Request.Context(), tenantID, 1, 500, "")
	if err != nil {
		c.logger.Errorw("ontology: list ci types failed", "error", err, "tenant_id", tenantID)
		common.InternalError(ctx, "获取CMDB本体失败")
		return
	}
	for _, t := range typeRes.Items {
		entry := dto.CMDBOntologyCIType{
			ID:           t.ID,
			Name:         t.Name,
			Description:  t.Description,
			Icon:         t.Icon,
			Color:        t.Color,
			ParentTypeID: t.ParentTypeID,
		}
		if t.AttributeSchema != "" {
			entry.AttributeSchema = parseAttributeSchemaOrRaw(t.AttributeSchema)
		}
		defs, defErr := c.ciAttributeDefinitionService.ListCIAttributeDefinitionsByCITypeID(ctx.Request.Context(), t.ID, tenantID)
		if defErr != nil {
			// 单类型属性定义失败不阻断整体自省（降级为无属性定义）
			c.logger.Warnw("ontology: list attribute definitions failed", "error", defErr, "ci_type_id", t.ID, "tenant_id", tenantID)
		} else if len(defs) > 0 {
			entry.AttributeDefinitions = defs
		}
		resp.CITypes = append(resp.CITypes, entry)
	}

	// 2. 关系词表：从 ent/schema 单一受控源派生（含反向类型，供 Agent 理解有向边）
	for _, m := range schema.CIRelationshipTypeVocabulary {
		resp.RelationshipTypes = append(resp.RelationshipTypes, dto.CMDBOntologyRelationshipType{
			Type:        string(m.Type),
			Name:        m.Name,
			Description: m.Description,
			Direction:   m.Direction,
			Icon:        m.Icon,
			ReverseType: string(m.Reverse),
		})
	}

	// 3. AI 工具面（按租户动态生成 ci_type 枚举；未注入注册表时省略该节）
	if c.toolRegistry != nil {
		tools := c.toolRegistry.ListToolsForTenant(ctx.Request.Context(), tenantID)
		resp.AITools = make([]dto.CMDBOntologyTool, 0, len(tools))
		for _, td := range tools {
			resp.AITools = append(resp.AITools, dto.CMDBOntologyTool{
				Name:        td.Name,
				Description: td.Description,
				ReadOnly:    td.ReadOnly,
				Resource:    td.Resource,
				Action:      td.Action,
				ArgsSchema:  td.ArgsSchema,
			})
		}
	}

	common.Success(ctx, &resp)
}

// parseAttributeSchemaOrRaw 尝试把 CIType.attributeSchema 文本解析为 JSON 对象；
// 非法 JSON 时原样返回字符串（schema 由 CIType 管理端维护，历史上是非结构化 Text）。
func parseAttributeSchemaOrRaw(raw string) interface{} {
	var parsed interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return raw
	}
	return parsed
}

// cmdbOntologyEnums 受控枚举值域。
// 值域来源（禁止在此文件发明新值）：
//   - lifecycleStatus / ownershipMode：ent/schema/configurationitem.go 字段默认值与注释
//   - relationshipStrength / impactLevel：ent/schema/ci_relationship.go 枚举
//   - ciStatus / environment / criticality：ent schema 默认值（active/production/medium）
//     与既有 API 契约（dto.UpdateCILifecycleStateRequest、CIListRequest 过滤值）
func cmdbOntologyEnums() dto.CMDBOntologyEnums {
	return dto.CMDBOntologyEnums{
		CIStatus:        []string{"active", "inactive", "maintenance", "planned", "decommissioned"},
		Environment:     []string{"production", "staging", "testing", "development"},
		Criticality:     []string{"critical", "high", "medium", "low"},
		LifecycleStatus: []string{common.CILifecycleStatusDraft, common.CILifecycleStatusOnline, common.CILifecycleStatusMaintenance, common.CILifecycleStatusOffline, common.CILifecycleStatusScrapped},
		OwnershipMode:   []string{"managed", "customer", "sla"},
		Strength:        []string{"critical", "high", "medium", "low"},
		ImpactLevel:     []string{"critical", "high", "medium", "low"},
	}
}
