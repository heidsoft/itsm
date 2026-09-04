package service

import (
	"context"
	"fmt"
	"strings"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/citype"
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
	rag            *RAGService
	incident       *IncidentService
	cmdb           *ConfigurationItemService
	ciRelationship *CIRelationshipService
	client         *ent.Client
	ticket         *TicketService
	ticketType     *TicketTypeService
	impactExplain  *ImpactExplanationService
}

func NewToolRegistry(rag *RAGService, incident *IncidentService, cmdb *ConfigurationItemService, client *ent.Client) *ToolRegistry {
	return &ToolRegistry{rag: rag, incident: incident, cmdb: cmdb, client: client}
}

// SetTicketService / SetTicketTypeService 注入写工具所需的领域服务。
// 采用 setter 而非构造函数参数，避免破坏既有 NewToolRegistry(nil,...) 测试调用，
// 也便于在 bootstrap 中按依赖就绪顺序分别装配（ticketType 在 ticket 之后构造）。
func (t *ToolRegistry) SetTicketService(s *TicketService)             { t.ticket = s }
func (t *ToolRegistry) SetTicketTypeService(s *TicketTypeService)     { t.ticketType = s }
func (t *ToolRegistry) SetCIRelationshipService(s *CIRelationshipService) {
	t.ciRelationship = s
}

// SetImpactExplainer P1-4：影响分析 AI 解释服务（可选注入，未注入时不报错）
func (t *ToolRegistry) SetImpactExplainer(s *ImpactExplanationService) { t.impactExplain = s }

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
	case "create_ci_relationship", "delete_ci_relationship":
		return t.cmdb != nil && t.ciRelationship != nil
	default:
		return true
	}
}

// ListToolsForTenant 返回按租户动态化的工具清单。
// P0-1（CMDB AI-Native）：list_cis 的 ci_type 枚举不再硬编码 5 值，
// 而是从该租户的 CIType 表实时读取——租户自定义类型（如 serverless）对 Agent 可见。
// 查询失败或无类型时静默回落到静态清单（fail-open 仅影响参数提示，不影响执行校验）。
func (t *ToolRegistry) ListToolsForTenant(ctx context.Context, tenantID int) []ToolDefinition {
	tools := t.ListTools()
	if t.client == nil {
		return tools
	}
	types, err := t.client.CIType.Query().
		Where(citype.TenantIDEQ(tenantID), citype.IsActiveEQ(true)).
		Order(ent.Asc(citype.FieldName)).
		Limit(50).
		All(ctx)
	if err != nil || len(types) == 0 {
		if err != nil {
			// 仅影响参数 schema 提示质量，降级为静态枚举
			// logger 未注入 ToolRegistry，保持静默（与 ListTools 同等约束）
			_ = err
		}
		return tools
	}
	enumVals := make([]any, 0, len(types))
	for _, ty := range types {
		enumVals = append(enumVals, ty.Name)
	}
	for i := range tools {
		if tools[i].Name != "list_cis" {
			continue
		}
		props, ok := tools[i].ArgsSchema["properties"].(map[string]interface{})
		if !ok {
			continue
		}
		if ct, ok := props["ci_type"].(map[string]interface{}); ok {
			ct["enum"] = enumVals
		}
	}
	return tools
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
			Description: "列出配置项（CMDB）。支持按 CI 编号精确定位、按名称/资产标签/序列号模糊搜索与按类型过滤，用于定位受影响 IT 资产。ci_type 枚举以 GET /cmdb/ontology 返回的租户实际类型为准",
			ReadOnly:    true,
			Resource:    "cmdb",
			Action:      "read",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"search":    map[string]interface{}{"type": "string", "description": "按名称/资产标签/序列号/厂商/云资源ID模糊搜索"},
					"ci_number": map[string]interface{}{"type": "string", "description": "CI唯一编号精确匹配（如 CI-202609-000001），优先于 search，多轮对话间用它稳定定位同一资产"},
					"ci_type":   map[string]interface{}{"type": "string", "description": "CI类型（租户自定义类型见 /cmdb/ontology）", "enum": []any{"server", "database", "application", "network", "storage", "cloud_resource"}},
					"limit":     map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 100},
					"offset":    map[string]interface{}{"type": "integer", "minimum": 0},
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
		// ============================================================
		// P1-3 CMDB AI 工具扩展（get_ci / get_ci_relationships /
		//   create_ci_relationship / delete_ci_relationship / get_ci_impact）
		// 全部围绕 CI 业务编号（自然键）或多轮稳定的 ci_id 提供，让 Agent
		// 能完整驱动 CMDB 拓扑编辑与影响面分析。
		// ============================================================
		{
			Name:        "get_ci",
			Description: "按 CI 编号或 ID 精确获取一个配置项（含完整字段与 1 跳关系摘要）。多轮对话中用 ci_number 稳定定位同一资产，先 list_cis 拿到 ci_number 再 get_ci 取详情最稳妥",
			ReadOnly:    true,
			Resource:    "cmdb",
			Action:      "read",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ci_number":      map[string]interface{}{"type": "string", "description": "CI 唯一业务编号（如 CI-202609-000001），与 ci_id 二选一"},
					"ci_id":          map[string]interface{}{"type": "integer", "description": "CI 数据库 ID，先用 list_cis 定位"},
					"with_relations": map[string]interface{}{"type": "boolean", "description": "是否返回 1 跳关系摘要（默认 true）"},
				},
			},
			ResultSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "get_ci_relationships",
			Description: "获取某 CI 的所有关系（出边 + 入边 + 类型汇总）。用于回答「这个资产依赖什么 / 被谁依赖」类问题",
			ReadOnly:    true,
			Resource:    "cmdb",
			Action:      "read",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ci_number": map[string]interface{}{"type": "string", "description": "CI 唯一业务编号"},
					"ci_id":     map[string]interface{}{"type": "integer", "description": "CI 数据库 ID"},
					"direction": map[string]interface{}{"type": "string", "enum": []any{"outgoing", "incoming", "both"}, "description": "出边/入边/双向，默认 both"},
				},
			},
			ResultSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "create_ci_relationship",
			Description: "在两个 CI 之间创建一条关系（需审批）。关系类型见 GET /cmdb/ontology#relationshipTypes 单一源；创建前必须先 list_cis 拿到 source/target 的 ci_id。系统会自动判环（wouldCreateCycle）",
			ReadOnly:    false,
			Resource:    "cmdb",
			Action:      "write",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"source_ci_id":     map[string]interface{}{"type": "integer", "description": "源 CI ID"},
					"target_ci_id":     map[string]interface{}{"type": "integer", "description": "目标 CI ID"},
					"relationship_type": map[string]interface{}{"type": "string", "description": "关系类型枚举（如 depends_on/hosts/connects_to/...）"},
					"strength":         map[string]interface{}{"type": "string", "enum": []any{"strong", "medium", "weak"}, "description": "关系强度"},
					"description":      map[string]interface{}{"type": "string", "description": "备注"},
				},
				"required": []string{"source_ci_id", "target_ci_id", "relationship_type"},
			},
			ResultSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "delete_ci_relationship",
			Description: "删除一条 CI 关系（需审批）。仅传 relationship_id（先用 get_ci_relationships 查）",
			ReadOnly:    false,
			Resource:    "cmdb",
			Action:      "write",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"relationship_id": map[string]interface{}{"type": "integer", "description": "关系 ID"},
				},
				"required": []string{"relationship_id"},
			},
			ResultSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "get_ci_impact",
			Description: "对指定 CI 做影响分析（多跳上下游遍历）：返回上下游节点/距离/关系类型/风险等级/受影响工单与事件，用于回答「这个 CI 故障会影响哪些资产与服务」。hops=1 表示直接依赖与被依赖，hops=3 推荐用于深度分析（max=10）",
			ReadOnly:    true,
			Resource:    "cmdb",
			Action:      "read",
			ArgsSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ci_number": map[string]interface{}{"type": "string", "description": "CI 唯一业务编号"},
					"ci_id":     map[string]interface{}{"type": "integer", "description": "CI 数据库 ID"},
					"hops":      map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 10, "description": "BFS 跳数（默认 3）"},
				},
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
		ciNumber, _ := args["ci_number"].(string)
		page := offset/limit + 1
		result, err := t.cmdb.ListCIs(ctx, tenantID, &dto.ListCIRequest{Page: page, Size: limit, Search: search, CIType: ciType, CINumber: ciNumber})
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

	// ============================================================
	// P1-3 CMDB AI 工具实现
	// ============================================================
	case "get_ci":
		if t.cmdb == nil {
			return nil, fmt.Errorf("cmdb service not initialized")
		}
		ciNumber, _ := args["ci_number"].(string)
		var ciID int
		if v, ok := args["ci_id"].(float64); ok {
			ciID = int(v)
		}
		if ciID == 0 && ciNumber == "" {
			return nil, fmt.Errorf("get_ci: ci_number or ci_id is required")
		}
		withRelations := true
		if v, ok := args["with_relations"].(bool); ok {
			withRelations = v
		}
		// ci_number 解析：用 list_cis 路径（避免新增 GetCIByNumber，list 已有 ci_number 精确过滤）
		if ciID == 0 {
			listResp, err := t.cmdb.ListCIs(ctx, tenantID, &dto.ListCIRequest{Page: 1, Size: 1, CINumber: ciNumber})
			if err != nil {
				return nil, fmt.Errorf("get_ci: lookup by ci_number failed: %w", err)
			}
			if len(listResp.Items) == 0 {
				return map[string]interface{}{"found": false, "ci_number": ciNumber}, nil
			}
			ciID = listResp.Items[0].ID
		}
		ci, err := t.cmdb.GetCIByID(ctx, ciID, tenantID, withRelations)
		if err != nil {
			return nil, err
		}
		if ci == nil {
			return map[string]interface{}{"found": false, "ci_id": ciID}, nil
		}
		return map[string]interface{}{"found": true, "ci": ci}, nil

	case "get_ci_relationships":
		if t.ciRelationship == nil {
			return nil, fmt.Errorf("ci relationship service not initialized")
		}
		ciNumber, _ := args["ci_number"].(string)
		var ciID int
		if v, ok := args["ci_id"].(float64); ok {
			ciID = int(v)
		}
		if ciID == 0 && ciNumber == "" {
			return nil, fmt.Errorf("get_ci_relationships: ci_number or ci_id is required")
		}
		if ciID == 0 {
			listResp, err := t.cmdb.ListCIs(ctx, tenantID, &dto.ListCIRequest{Page: 1, Size: 1, CINumber: ciNumber})
			if err != nil {
				return nil, fmt.Errorf("get_ci_relationships: lookup failed: %w", err)
			}
			if len(listResp.Items) == 0 {
				return map[string]interface{}{"found": false, "ci_number": ciNumber}, nil
			}
			ciID = listResp.Items[0].ID
		}
		direction, _ := args["direction"].(string)
		if direction == "" {
			direction = "both"
		}
		outgoing, err := t.ciRelationship.ListCIRelationshipsByCIID(ctx, ciID, tenantID, "outgoing")
		if err != nil {
			return nil, fmt.Errorf("get_ci_relationships: outgoing failed: %w", err)
		}
		incoming, err := t.ciRelationship.ListCIRelationshipsByCIID(ctx, ciID, tenantID, "incoming")
		if err != nil {
			return nil, fmt.Errorf("get_ci_relationships: incoming failed: %w", err)
		}
		out := outgoing
		in := incoming
		if direction == "outgoing" {
			in = nil
		}
		if direction == "incoming" {
			out = nil
		}
		return map[string]interface{}{
			"ci_id":    ciID,
			"outgoing": out,
			"incoming": in,
			"total":    len(out) + len(in),
		}, nil

	case "create_ci_relationship":
		if t.cmdb == nil || t.ciRelationship == nil {
			return nil, fmt.Errorf("cmdb relationship service not initialized")
		}
		var sourceID, targetID int
		if v, ok := args["source_ci_id"].(float64); ok {
			sourceID = int(v)
		}
		if v, ok := args["target_ci_id"].(float64); ok {
			targetID = int(v)
		}
		if sourceID <= 0 || targetID <= 0 {
			return nil, fmt.Errorf("create_ci_relationship: source_ci_id and target_ci_id are required")
		}
		relType, _ := args["relationship_type"].(string)
		if relType == "" {
			return nil, fmt.Errorf("create_ci_relationship: relationship_type is required (see /cmdb/ontology#relationshipTypes)")
		}
		strength, _ := args["strength"].(string)
		if strength == "" {
			strength = "medium"
		}
		description, _ := args["description"].(string)

		req := &dto.CreateCIRelationshipRequest{
			SourceCIID:       sourceID,
			TargetCIID:       targetID,
			RelationshipType: dto.CIRelationshipType(relType),
			Strength:         dto.RelationshipStrength(strength),
			Description:      description,
		}
		created, err := t.ciRelationship.CreateCIRelationship(ctx, req, tenantID)
		if err != nil {
			return nil, err
		}
		return created, nil

	case "delete_ci_relationship":
		if t.ciRelationship == nil {
			return nil, fmt.Errorf("ci relationship service not initialized")
		}
		var relID int
		if v, ok := args["relationship_id"].(float64); ok {
			relID = int(v)
		}
		if relID <= 0 {
			return nil, fmt.Errorf("delete_ci_relationship: relationship_id is required")
		}
		if err := t.ciRelationship.DeleteCIRelationship(ctx, relID, tenantID); err != nil {
			return nil, err
		}
		return map[string]interface{}{"deleted": true, "relationship_id": relID}, nil

	case "get_ci_impact":
		if t.cmdb == nil || t.ciRelationship == nil {
			return nil, fmt.Errorf("cmdb relationship service not initialized")
		}
		ciNumber, _ := args["ci_number"].(string)
		var ciID int
		if v, ok := args["ci_id"].(float64); ok {
			ciID = int(v)
		}
		if ciID == 0 && ciNumber == "" {
			return nil, fmt.Errorf("get_ci_impact: ci_number or ci_id is required")
		}
		if ciID == 0 {
			listResp, err := t.cmdb.ListCIs(ctx, tenantID, &dto.ListCIRequest{Page: 1, Size: 1, CINumber: ciNumber})
			if err != nil {
				return nil, fmt.Errorf("get_ci_impact: lookup failed: %w", err)
			}
			if len(listResp.Items) == 0 {
				return map[string]interface{}{"found": false, "ci_number": ciNumber}, nil
			}
			ciID = listResp.Items[0].ID
		}
		hops := 3
		if v, ok := args["hops"].(float64); ok {
			hops = int(v)
		}
		if hops < 1 {
			hops = 1
		}
		if hops > 10 {
			hops = 10
		}
		impact, err := t.ciRelationship.GetCIImpactAnalysis(ctx, ciID, tenantID, hops)
		if err != nil {
			return nil, err
		}
		// P1-4：影响分析 AI 解释层（fail-open，未注入 / LLM 失败时返回 nil）
		var explain map[string]interface{}
		if t.impactExplain != nil && impact != nil {
			if ie, _ := t.impactExplain.ExplainImpact(ctx, tenantID, ciID, hops, impact); ie != nil {
				explain = map[string]interface{}{
					"summary":     ie.Summary,
					"rootCauses":  ie.RootCauses,
					"slaRisks":    ie.SLARisks,
					"suggestions": ie.Suggestions,
					"generatedAt": ie.GeneratedAt,
					"model":       ie.Model,
				}
			}
		}
		return map[string]interface{}{
			"impact":     impact,
			"hops":       hops,
			"ci_id":      ciID,
			"explain_ai": explain,
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}
