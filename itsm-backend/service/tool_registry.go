package service

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ReadOnly    bool   `json:"read_only"`
}

type ToolRegistry struct {
	rag      *RAGService
	incident *IncidentService
	cmdb     *CMDBService
	client   *ent.Client
}

func NewToolRegistry(rag *RAGService, incident *IncidentService, cmdb *CMDBService, client *ent.Client) *ToolRegistry {
	return &ToolRegistry{rag: rag, incident: incident, cmdb: cmdb, client: client}
}

func (t *ToolRegistry) ListTools() []ToolDefinition {
	return []ToolDefinition{
		{Name: "get_incident_stats", Description: "获取当前租户的事件统计", ReadOnly: true},
		{Name: "list_kb", Description: "按关键词检索知识库文章（RAG 简化）", ReadOnly: true},
		{Name: "list_cis", Description: "列出配置项（分页）", ReadOnly: true},
		{Name: "create_ticket", Description: "创建工单（需审批）", ReadOnly: false},
		{Name: "update_ticket", Description: "更新工单（需审批）", ReadOnly: false},
	}
}

func (t *ToolRegistry) Execute(ctx context.Context, tenantID int, name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	case "get_incident_stats":
		return t.incident.GetIncidentStats(ctx, tenantID)
	case "list_kb":
		q := ""
		if v, ok := args["q"].(string); ok {
			q = v
		}
		limit := 5
		if v, ok := args["limit"].(float64); ok {
			limit = int(v)
		}
		return t.rag.Ask(ctx, tenantID, q, limit)
	case "list_cis":
		limit := 10
		offset := 0
		if v, ok := args["limit"].(float64); ok {
			limit = int(v)
		}
		if v, ok := args["offset"].(float64); ok {
			offset = int(v)
		}
		req := &dto.ListCIsRequest{TenantID: tenantID, Limit: limit, Offset: offset}
		return t.cmdb.ListCIs(ctx, req)
	case "create_ticket":
		return map[string]any{"needs_approval": true}, nil
	case "update_ticket":
		return map[string]any{"needs_approval": true}, nil
	default:
		return nil, nil
	}
}
