package dto

// TicketAnalyticsFilters 工单分析过滤条件（强类型，用于替代 DeepAnalyticsRequest 中的 map[string]interface{} Filters）
type TicketAnalyticsFilters struct {
	Status      []string          `json:"status,omitempty"`
	Priority    []string          `json:"priority,omitempty"`
	Category    []string          `json:"category,omitempty"`
	Type        []string          `json:"type,omitempty"`
	AssigneeID  *int              `json:"assigneeId,omitempty"`
	RequesterID *int              `json:"requesterId,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Custom      map[string]string `json:"custom,omitempty"` // 自定义字段过滤（key=fieldName）
}

// HasFilters 检查是否有任何过滤条件
func (f *TicketAnalyticsFilters) HasFilters() bool {
	if f == nil {
		return false
	}
	return len(f.Status) > 0 || len(f.Priority) > 0 || len(f.Category) > 0 ||
		len(f.Type) > 0 || f.AssigneeID != nil || f.RequesterID != nil ||
		len(f.Tags) > 0 || len(f.Custom) > 0
}

// AnalyticsTrend 趋势数据点（强类型，用于取代 []map[string]interface{} Trends）
type AnalyticsTrend struct {
	Period   string  `json:"period"`   // 时间段标签
	Created  int     `json:"created"`  // 新建数
	Resolved int     `json:"resolved"` // 解决数
	Backlog  int     `json:"backlog"`  // 积压数
	AvgTime  float64 `json:"avgTime"`  // 平均处理时间
}

// TicketNotificationData 工单通知数据（强类型，用于替代 map[string]interface{} Data）
// 当通知类型为 email/sms/webhook 时，使用此结构体组织通知内容
type TicketNotificationData struct {
	TicketNumber  string            `json:"ticketNumber,omitempty"`
	TicketTitle   string            `json:"ticketTitle,omitempty"`
	Status        string            `json:"status,omitempty"`
	Priority      string            `json:"priority,omitempty"`
	AssigneeName  string            `json:"assigneeName,omitempty"`
	RequesterName string            `json:"requesterName,omitempty"`
	Action        string            `json:"action,omitempty"`  // 触发通知的操作
	Comment       string            `json:"comment,omitempty"` // 审批/操作备注
	URL           string            `json:"url,omitempty"`     // 工单链接
	Extra         map[string]string `json:"extra,omitempty"`   // 扩展字段
}
