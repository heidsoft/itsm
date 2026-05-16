package service

import (
	"context"
	"fmt"

	"itsm-backend/ent"
)

type ToolDefinition struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	ReadOnly     bool                   `json:"read_only"`
	Resource     string                 `json:"resource"`
	Action       string                 `json:"action"`
	ArgsSchema   map[string]interface{} `json:"args_schema"`
	ResultSchema map[string]interface{} `json:"result_schema"`
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
		{
			Name:        "get_incident_stats",
			Description: "获取当前租户的事件统计",
			ReadOnly:    true,
			Resource:    "incident",
			Action:      "read",
			ArgsSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			ResultSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"totalIncidents":    map[string]interface{}{"type": "integer"},
					"openIncidents":     map[string]interface{}{"type": "integer"},
					"resolvedIncidents": map[string]interface{}{"type": "integer"},
				},
			},
		},
		{
			Name:        "list_kb",
			Description: "按关键词检索知识库文章（RAG 简化）",
			ReadOnly:    true,
			Resource:    "knowledge",
			Action:      "read",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"q":     map[string]interface{}{"type": "string"},
					"limit": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 20},
				},
				"required": []string{"q"},
			},
			ResultSchema: map[string]interface{}{
				"type": "array",
			},
		},
		{
			Name:        "list_cis",
			Description: "列出配置项（分页）",
			ReadOnly:    true,
			Resource:    "cmdb",
			Action:      "read",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"limit":  map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 100},
					"offset": map[string]interface{}{"type": "integer", "minimum": 0},
				},
			},
			ResultSchema: map[string]interface{}{
				"type": "array",
			},
		},
		{
			Name:        "create_ticket",
			Description: "创建工单（需审批）",
			ReadOnly:    false,
			Resource:    "ticket",
			Action:      "write",
			ArgsSchema: map[string]interface{}{
				"type": "object",
			},
			ResultSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "update_ticket",
			Description: "更新工单（需审批）",
			ReadOnly:    false,
			Resource:    "ticket",
			Action:      "write",
			ArgsSchema: map[string]interface{}{
				"type": "object",
			},
			ResultSchema: map[string]interface{}{
				"type": "object",
			},
		},
	}
}

func (t *ToolRegistry) Execute(ctx context.Context, tenantID int, name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	case "get_incident_stats":
		// 使用ListIncidents来获取统计信息
		incidents, _, err := t.incident.ListIncidents(ctx, tenantID, 1, 1000, map[string]interface{}{})
		if err != nil {
			return nil, err
		}

		stats := map[string]interface{}{
			"totalIncidents":    len(incidents),
			"openIncidents":     0,
			"resolvedIncidents": 0,
		}

		for _, incident := range incidents {
			switch incident.Status {
			case "new", "in_progress":
				stats["openIncidents"] = stats["openIncidents"].(int) + 1
			case "resolved", "closed":
				stats["resolvedIncidents"] = stats["resolvedIncidents"].(int) + 1
			}
		}

		return stats, nil
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
		req := &ListCIsRequest{Limit: limit, Offset: offset}
		items, _, err := t.cmdb.ListCIs(ctx, req)
		return items, err
	case "create_ticket":
		return nil, fmt.Errorf("tool %s requires approval and is not implemented", name)
	case "update_ticket":
		return nil, fmt.Errorf("tool %s requires approval and is not implemented", name)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}
