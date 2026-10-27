package middleware

// precheckFallbackPolicies 显式族级回退预检策略。
//
// 与 rbac_precheck_gen.go（从路由声明生成）合并构成 ResourceActionMap：
//
//	ResourceActionMap = 生成条目（路由声明，精确路径） + 本文件（族级通配/别名策略）
//
// 本文件承载的是策略而非路由声明的复制：
//   - 族级通配（如 POST /api/v1/changes/* = change:write）：为未声明/未来新增的
//     子路径提供 fail-closed 缺省预检；
//   - 历史别名与仅 AuthMiddleware 的路由（如 /api/v1/auth/*、
//     /configuration-items/* 兼容别名）：保留既定的预检口径。
//
// 匹配优先级：getPermissionFromPath 先精确后通配、按 specificity 取最具体，
// 因此生成条目（精确路径）天然优先于本文件的族级通配。
// 修改口径 = 直接编辑本文件并跑 TestRoutePrecheckAlignment 验证。
//
// 条目必须命中真实路由：TestPrecheckFallbackPoliciesMatchRegisteredRoutes 会用
// 与运行时同一套 matchPath 语义对账，未命中任何已注册路由的条目直接判红。
// 2026-10-27 据此清掉 55 条死条目（整块 PATCH 族、/api/v1/ticket-templates*
// 旧路径族、以及 base 与 /* 错位的一批）。死条目的危害不是「不生效」，而是
// 会变成未来路由的埋雷：一旦有人补上 POST /reports/export，就会继承一条
// dashboard:admin 的过时口径，而错配只会在某个低权角色被 403 时才暴露。
var precheckFallbackPolicies = map[string]map[string]Permission{
	"DELETE": {
		"/api/v1/applications/*": {Resource: "application", Action: "write"},
		"/api/v1/bpmn/*":         {Resource: "bpmn", Action: "delete"},
		"/api/v1/cmdb/*":         {Resource: "cmdb", Action: "delete"},
		"/api/v1/connectors/*":   {Resource: "connector", Action: "write"},
		"/api/v1/dashboard/*":    {Resource: "dashboard", Action: "admin"},
		"/api/v1/knowledge/*":    {Resource: "knowledge", Action: "delete"},
	},
	"GET": {
		"/api/v1/agent/*":           {Resource: "ai", Action: "read"},
		"/api/v1/ai/*":              {Resource: "ai", Action: "read"},
		"/api/v1/applications/*":    {Resource: "application", Action: "read"},
		"/api/v1/audit-logs":        {Resource: "audit", Action: "read"},
		"/api/v1/auth/me":           {Resource: "user", Action: "read"},
		"/api/v1/auth/menus":        {Resource: "user", Action: "read"},
		"/api/v1/bpmn/*":            {Resource: "bpmn", Action: "read"},
		"/api/v1/cmdb/*":            {Resource: "cmdb", Action: "read"},
		"/api/v1/connectors/*":      {Resource: "connector", Action: "read"},
		"/api/v1/dashboard/*":       {Resource: "dashboard", Action: "read"},
		"/api/v1/departments":       {Resource: "department", Action: "read"},
		"/api/v1/departments/*":     {Resource: "department", Action: "read"},
		"/api/v1/knowledge/*":       {Resource: "knowledge", Action: "read"},
		"/api/v1/msp/customers/*":   {Resource: "msp_customer", Action: "read"},
		"/api/v1/msp/reports/*":     {Resource: "msp_report", Action: "read"},
		"/api/v1/msp/status":        {Resource: "msp", Action: "read"},
		"/api/v1/notifications/*":   {Resource: "notification", Action: "read"},
		"/api/v1/org/*":             {Resource: "org", Action: "read"},
		"/api/v1/process-trigger/*": {Resource: "bpmn", Action: "read"},
		"/api/v1/reports/*":         {Resource: "report", Action: "read"},
		"/api/v1/service-catalog":   {Resource: "service_catalog", Action: "read"},
		"/api/v1/sla/*":             {Resource: "sla", Action: "read"},
		"/api/v1/system/*":          {Resource: "system_config", Action: "read"},
		"/api/v1/workflow/tasks/*":  {Resource: "task", Action: "read"},
	},
	"PATCH": {
		// 全仓只有 8 条 PATCH 路由（工单关系/子任务、审批、AI 审计、市场目录），
		// 全部已由路由级声明覆盖；此前那 22 条「PATCH /<资源>/*」是 REST 教科书
		// 式的想当然，本仓库的更新一律走 PUT。
	},
	"POST": {
		"/api/v1/admin/operations/commands/*": {Resource: "system", Action: "write"},
		"/api/v1/agent/*":                     {Resource: "ai", Action: "write"},
		"/api/v1/ai/*":                        {Resource: "ai", Action: "write"},
		"/api/v1/applications/*":              {Resource: "application", Action: "write"},
		"/api/v1/bpmn/*":                      {Resource: "bpmn", Action: "write"},
		"/api/v1/bpmn/tasks/*":                {Resource: "task", Action: "update"},
		"/api/v1/changes/*":                   {Resource: "change", Action: "write"},
		"/api/v1/cmdb/*":                      {Resource: "cmdb", Action: "write"},
		"/api/v1/configuration-items/*":       {Resource: "cmdb", Action: "write"},
		"/api/v1/connectors/*":                {Resource: "connector", Action: "write"},
		"/api/v1/dashboard/*":                 {Resource: "dashboard", Action: "admin"},
		"/api/v1/incidents/*":                 {Resource: "incident", Action: "write"},
		"/api/v1/knowledge/articles/*":        {Resource: "knowledge", Action: "write"},
		"/api/v1/menus/*":                     {Resource: "system_config", Action: "write"},
		"/api/v1/org/*":                       {Resource: "org", Action: "write"},
		"/api/v1/problems/*":                  {Resource: "problem", Action: "write"},
		"/api/v1/process-trigger/*":           {Resource: "bpmn", Action: "write"},
		"/api/v1/provisioning-tasks/*":        {Resource: "service_request", Action: "write"},
		"/api/v1/releases/*":                  {Resource: "release", Action: "write"},
		"/api/v1/roles/*":                     {Resource: "role", Action: "write"},
		"/api/v1/service-requests/*":          {Resource: "service_request", Action: "write"},
		"/api/v1/sla/*":                       {Resource: "sla", Action: "write"},
		"/api/v1/system/*":                    {Resource: "system_config", Action: "write"},
		"/api/v1/ticket-categories/*":         {Resource: "ticket_category", Action: "write"},
		"/api/v1/tickets/*":                   {Resource: "ticket", Action: "write"},
		"/api/v1/workflow/tasks/*":            {Resource: "task", Action: "update"},
	},
	"PUT": {
		"/api/v1/applications/*":   {Resource: "application", Action: "write"},
		"/api/v1/bpmn/*":           {Resource: "bpmn", Action: "write"},
		"/api/v1/bpmn/tasks/*":     {Resource: "task", Action: "update"},
		"/api/v1/cmdb/*":           {Resource: "cmdb", Action: "write"},
		"/api/v1/knowledge/*":      {Resource: "knowledge", Action: "write"},
		"/api/v1/notifications/*":  {Resource: "notification", Action: "write"},
		"/api/v1/workflow/tasks/*": {Resource: "task", Action: "update"},
	},
}
