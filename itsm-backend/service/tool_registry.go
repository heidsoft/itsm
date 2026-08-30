package service

import (
	"context"
	"fmt"
	"strings"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
)

type ToolDefinition struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	ReadOnly     bool                   `json:"readOnly"`
	Resource     string                 `json:"resource"`
	Action       string                 `json:"action"`
	ArgsSchema   map[string]interface{} `json:"argsSchema"`
	ResultSchema map[string]interface{} `json:"resultSchema"`
}

type ToolRegistry struct {
	rag        *RAGService
	incident   *IncidentService
	cmdb       *ConfigurationItemService
	client     *ent.Client
	ticket     *TicketService
	ticketType *TicketTypeService
}

func NewToolRegistry(rag *RAGService, incident *IncidentService, cmdb *ConfigurationItemService, client *ent.Client) *ToolRegistry {
	return &ToolRegistry{rag: rag, incident: incident, cmdb: cmdb, client: client}
}

// SetTicketService / SetTicketTypeService 注入写工具所需的领域服务。
// 采用 setter 而非构造函数参数，避免破坏既有 NewToolRegistry(nil,...) 测试调用，
// 也便于在 bootstrap 中按依赖就绪顺序分别装配（ticketType 在 ticket 之后构造）。
func (t *ToolRegistry) SetTicketService(s *TicketService)         { t.ticket = s }
func (t *ToolRegistry) SetTicketTypeService(s *TicketTypeService) { t.ticketType = s }

// GetTool 按名称查找工具定义，找不到返回 nil
// P2-6: AI 工具 RBAC 校验入口需要查询 ToolDefinition.Resource/Action
func (t *ToolRegistry) GetTool(name string) *ToolDefinition {
	for _, td := range t.ListTools() {
		if td.Name == name {
			tdCopy := td
			return &tdCopy
		}
	}
	return nil
}

// canExecuteWriteTool 判断某工具能否交由 ToolRegistry.Execute 统一执行。
// ToolQueue 审批通过后会调用它决定是否委派：写工具要求对应领域服务已注入，
// 未注入时返回 false，让 ToolQueue 回落到内联实现。
func (t *ToolRegistry) canExecuteWriteTool(name string) bool {
	td := t.GetTool(name)
	if td == nil || td.ReadOnly {
		return false
	}
	switch name {
	case "create_ticket", "update_ticket":
		return t.ticket != nil
	case "create_ticket_type":
		return t.ticketType != nil
	case "link_ticket_ci":
		return t.cmdb != nil
	default:
		return true
	}
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
			Name:        "list_tickets",
			Description: "列出当前租户的工单（分页，按创建时间倒序）",
			ReadOnly:    true,
			Resource:    "ticket",
			Action:      "read",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"page":     map[string]interface{}{"type": "integer", "minimum": 1},
					"pageSize": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 100},
				},
			},
			ResultSchema: map[string]interface{}{
				"type": "array",
			},
		},
		{
			Name:        "list_cis",
			Description: "列出配置项（CMDB）。支持按名称/资产标签/序列号模糊搜索与按类型过滤，用于定位受影响 IT 资产",
			ReadOnly:    true,
			Resource:    "cmdb",
			Action:      "read",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"search":  map[string]interface{}{"type": "string", "description": "按名称/资产标签/序列号/厂商/云资源ID模糊搜索"},
					"ci_type": map[string]interface{}{"type": "string", "description": "CI类型，如 server/database/application/network", "enum": []any{"server", "database", "application", "network", "storage", "cloud_resource"}},
					"limit":   map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 100},
					"offset":  map[string]interface{}{"type": "integer", "minimum": 0},
				},
			},
			ResultSchema: map[string]interface{}{
				"type": "array",
			},
		},
		{
			Name:        "get_ci_tickets",
			Description: "查询某个配置项（CI）上已关联的工单列表，用于影响面分析与重复报障判断：先用 list_cis 定位 ci_id，再查该资产上的历史/在办工单",
			ReadOnly:    true,
			Resource:    "cmdb",
			Action:      "read",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ci_id": map[string]interface{}{"type": "integer", "description": "配置项ID，先用 list_cis 定位"},
					"limit": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 100},
				},
				"required": []string{"ci_id"},
			},
			ResultSchema: map[string]interface{}{
				"type": "array",
			},
		},
		{
			Name:        "link_ticket_ci",
			Description: "将已存在的工单关联到 CMDB 配置项（需审批）：补建「工单 ↔ 受影响资产」本体关系，用于工单创建后才定位到具体设备的场景",
			ReadOnly:    false,
			Resource:    "cmdb",
			Action:      "write",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ticket_id": map[string]interface{}{"type": "integer", "description": "工单ID"},
					"ci_id":     map[string]interface{}{"type": "integer", "description": "配置项ID，先用 list_cis 定位"},
				},
				"required": []string{"ticket_id", "ci_id"},
			},
			ResultSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "create_ticket",
			Description: "创建工单（需审批）。可传入 ci_id 将工单关联到 CMDB 中受影响的配置项；ci_id 应先通过 list_cis 定位",
			ReadOnly:    false,
			Resource:    "ticket",
			Action:      "write",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ci_id": map[string]interface{}{"type": "integer", "description": "关联的配置项ID（CMDB 本体链路），先用 list_cis 定位"},
				},
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
		{
			Name:        "create_ticket_type",
			Description: "创建工单类型（需审批）：新增 ITSM 工单分类（如通用事件、数据库变更、账号权限申请），需提供 code 与 name",
			ReadOnly:    false,
			Resource:    "ticket_type",
			Action:      "write",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"code":             map[string]interface{}{"type": "string", "description": "工单类型编码，小写字母/数字/下划线，如 general_incident"},
					"name":             map[string]interface{}{"type": "string", "description": "工单类型名称，如 通用事件"},
					"description":      map[string]interface{}{"type": "string", "description": "类型说明"},
					"default_priority": map[string]interface{}{"type": "string", "enum": []any{"low", "medium", "high", "critical", "urgent"}},
					"icon":             map[string]interface{}{"type": "string"},
					"color":            map[string]interface{}{"type": "string"},
				},
				"required": []string{"code", "name"},
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
	case "list_tickets":
		page := 1
		if v, ok := args["page"].(float64); ok {
			page = int(v)
		}
		if page < 1 {
			page = 1
		}
		pageSize := 20
		if v, ok := args["pageSize"].(float64); ok {
			pageSize = int(v)
		}
		// pageSize clamp [1,100]，防止模型传入超大分页拖垮查询
		if pageSize < 1 {
			pageSize = 1
		}
		if pageSize > 100 {
			pageSize = 100
		}
		// 显式 TenantID 过滤 + DeletedAtIsNil 软删除过滤，绝不跨租户/含软删工单
		tickets, err := t.client.Ticket.Query().
			Where(ticket.TenantID(tenantID), ticket.DeletedAtIsNil()).
			Order(ent.Desc(ticket.FieldCreatedAt)).
			Offset((page - 1) * pageSize).
			Limit(pageSize).
			All(ctx)
		if err != nil {
			return nil, err
		}
		// 返回 DTO，不直接暴露 ent 模型
		return dto.ToTicketResponseList(tickets), nil
	case "list_cis":
		limit := 10
		offset := 0
		if v, ok := args["limit"].(float64); ok {
			limit = int(v)
		}
		if v, ok := args["offset"].(float64); ok {
			offset = int(v)
		}
		search, _ := args["search"].(string)
		ciType, _ := args["ci_type"].(string)
		page := offset/limit + 1
		result, err := t.cmdb.ListCIs(ctx, tenantID, &dto.ListCIRequest{Page: page, Size: limit, Search: search, CIType: ciType})
		if err != nil {
			return nil, err
		}
		return result.Items, nil
	case "get_ci_tickets":
		if t.cmdb == nil {
			return nil, fmt.Errorf("cmdb service not initialized")
		}
		ciID := 0
		if v, ok := args["ci_id"].(float64); ok {
			ciID = int(v)
		}
		if ciID <= 0 {
			return nil, fmt.Errorf("get_ci_tickets: ci_id is required")
		}
		limit := 20
		if v, ok := args["limit"].(float64); ok {
			limit = int(v)
		}
		return t.cmdb.ListCITickets(ctx, tenantID, ciID, limit)
	case "link_ticket_ci":
		if t.cmdb == nil {
			return nil, fmt.Errorf("cmdb service not initialized")
		}
		ciID := 0
		if v, ok := args["ci_id"].(float64); ok {
			ciID = int(v)
		}
		linkTicketID := 0
		if v, ok := args["ticket_id"].(float64); ok {
			linkTicketID = int(v)
		}
		if ciID <= 0 || linkTicketID <= 0 {
			return nil, fmt.Errorf("link_ticket_ci: ticket_id and ci_id are required")
		}
		if err := t.cmdb.LinkTicketToCI(ctx, tenantID, ciID, linkTicketID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"ticketId": linkTicketID, "ciId": ciID, "ciLinked": true}, nil
	case "create_ticket":
		if t.ticket == nil {
			return nil, fmt.Errorf("ticket service not initialized")
		}
		title, _ := args["title"].(string)
		if strings.TrimSpace(title) == "" {
			return nil, fmt.Errorf("create_ticket: title is required")
		}
		priority, _ := args["priority"].(string)
		if priority == "" {
			priority = "medium"
		}
		desc, _ := args["description"].(string)
		typ, _ := args["type"].(string)
		category, _ := args["category"].(string)
		var requesterID int
		if v, ok := args["requester_id"].(float64); ok {
			requesterID = int(v)
		} else if v, ok := args["user_id"].(float64); ok {
			requesterID = int(v)
		}
		var assigneeID int
		if v, ok := args["assignee_id"].(float64); ok {
			assigneeID = int(v)
		}
		var ticketTypeID *int
		if v, ok := args["ticket_type_id"].(float64); ok {
			i := int(v)
			ticketTypeID = &i
		}
		// CMDB 本体链路：ci_id 走 ConfigurationItem.tickets 真实外键边绑定，
		// 不塞 form_fields —— form_fields 会被工单类型的「未定义字段」校验拒绝。
		var ciID int
		if v, ok := args["ci_id"].(float64); ok && int(v) > 0 {
			ciID = int(v)
		}
		req := &dto.CreateTicketRequest{
			Title:        title,
			Description:  desc,
			Priority:     priority,
			Type:         typ,
			Category:     category,
			RequesterID:  requesterID,
			AssigneeID:   assigneeID,
			TicketTypeID: ticketTypeID,
		}
		created, err := t.ticket.CreateTicket(ctx, req, tenantID)
		if err != nil {
			return nil, err
		}
		resp := dto.ToTicketResponse(t.ticket.toEntTicket(created))
		if ciID > 0 && t.cmdb != nil {
			// 绑定失败不回滚工单：工单已是有效交付物，仅降级提示 AI 补充关联。
			if linkErr := t.cmdb.LinkTicketToCI(ctx, tenantID, ciID, created.ID); linkErr != nil {
				return map[string]interface{}{
					"ticket":     resp,
					"ciId":       ciID,
					"ciLinked":   false,
					"ciLinkWarn": linkErr.Error(),
				}, nil
			}
			return map[string]interface{}{
				"ticket":   resp,
				"ciId":     ciID,
				"ciLinked": true,
			}, nil
		}
		return resp, nil
	case "update_ticket":
		if t.ticket == nil {
			return nil, fmt.Errorf("ticket service not initialized")
		}
		ticketID := 0
		if v, ok := args["ticket_id"].(float64); ok {
			ticketID = int(v)
		}
		if ticketID == 0 {
			return nil, fmt.Errorf("update_ticket: ticket_id is required")
		}
		status, _ := args["status"].(string)
		priority, _ := args["priority"].(string)
		resolution, _ := args["resolution"].(string)
		var assigneeID int
		if v, ok := args["assignee_id"].(float64); ok {
			assigneeID = int(v)
		}
		req := &dto.UpdateTicketRequest{
			Status:     status,
			Priority:   priority,
			Resolution: resolution,
			AssigneeID: assigneeID,
		}
		updated, err := t.ticket.UpdateTicket(ctx, ticketID, req, tenantID, 0, "") // 0=系统操作，跳过 DataScope
		if err != nil {
			return nil, err
		}
		return dto.ToTicketResponse(t.ticket.toEntTicket(updated)), nil
	case "create_ticket_type":
		if t.ticketType == nil {
			return nil, fmt.Errorf("ticket type service not initialized")
		}
		code, _ := args["code"].(string)
		name, _ := args["name"].(string)
		if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("create_ticket_type: code and name are required")
		}
		desc, _ := args["description"].(string)
		defaultPriority, _ := args["default_priority"].(string)
		if defaultPriority == "" {
			defaultPriority = "medium"
		}
		icon, _ := args["icon"].(string)
		color, _ := args["color"].(string)
		var userID int
		if v, ok := args["user_id"].(float64); ok {
			userID = int(v)
		}
		req := &dto.CreateTicketTypeRequest{
			Code:            code,
			Name:            name,
			Description:     desc,
			DefaultPriority: defaultPriority,
			Icon:            icon,
			Color:           color,
		}
		created, err := t.ticketType.CreateTicketType(ctx, req, tenantID, userID)
		if err != nil {
			return nil, err
		}
		return created, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}
