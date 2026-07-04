package dto

// TicketViewFilter 工单视图过滤器（强类型，取代 map[string]interface{}）
type TicketViewFilter struct {
	Status      []string          `json:"status,omitempty"`
	Priority    []string          `json:"priority,omitempty"`
	Type        []string          `json:"type,omitempty"`
	Category    []string          `json:"category,omitempty"`
	AssigneeID  []int             `json:"assignee_id,omitempty"`
	RequesterID []int             `json:"requester_id,omitempty"`
	Search      string            `json:"search,omitempty"`     // 全文搜索
	DateFrom    string            `json:"date_from,omitempty"`  // ISO 日期
	DateTo      string            `json:"date_to,omitempty"`    // ISO 日期
	SLAStatus   []string          `json:"sla_status,omitempty"` // SLA状态过滤
	TagIDs      []int             `json:"tag_ids,omitempty"`    // 标签ID过滤
	Custom      map[string]string `json:"custom,omitempty"`     // 自定义字段过滤
}

// TicketViewSortConfig 工单视图排序配置（强类型，取代 map[string]interface{}）
type TicketViewSortConfig struct {
	Field     string `json:"field"`     // 排序字段
	Direction string `json:"direction"` // asc / desc
}

// TicketViewGroupConfig 工单视图分组配置（强类型，取代 map[string]interface{}）
type TicketViewGroupConfig struct {
	Field     string `json:"field"`               // 分组字段
	Order     string `json:"order,omitempty"`     // 分组排序
	Collapsed bool   `json:"collapsed,omitempty"` // 是否折叠
}
