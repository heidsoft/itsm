package dto

// TimerListFilter 定时器列表查询条件。定义在公共值类型包，使 handlers/timer
// 的窄契约与 service.TimerStore 的实现共享同一类型身份（不是两个同名结构）。
type TimerListFilter struct {
	TenantID             int
	Status               string
	TimerType            string
	ProcessInstanceID    *int
	ProcessDefinitionKey string
	Page                 int
	PageSize             int
}

// TimerStats 定时器状态计数。
type TimerStats struct {
	Pending   int `json:"pending"`
	Fired     int `json:"fired"`
	Completed int `json:"completed"`
	Cancelled int `json:"cancelled"`
	Failed    int `json:"failed"`
	Paused    int `json:"paused"`
	Total     int `json:"total"`
}
