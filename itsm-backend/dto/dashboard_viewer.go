package dto

// DashboardViewer 工作台 / 报表读取者身份。
//
// KPI 里的工单类指标按读取者的行级范围统计，与 /tickets 列表同一规则
// （datascope.ForTicketReader）；ScopeAll 用于后台任务等没有请求者上下文的调用，
// 显式声明全租户口径而不是靠空角色推断。
//
// 该值类型同时出现在 service 层方法签名与 handlers/<domain> 的窄契约里，
// 因此下沉到最底层的 dto 包；service 侧保留类型别名，公开 API 与类型身份不变。
// 依赖 datascope 的三个口径函数仍留在 service 包（未导出），不向上污染 dto。
type DashboardViewer struct {
	UserID   int
	Role     string
	ScopeAll bool
}
