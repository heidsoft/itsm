# 后端功能深入测试与分析

- 日期: 2025-12-07
- 范围: 控制器、服务、路由组织、鉴权与租户中间件、错误响应规范

## 路由组织与中间件
- 入口：`itsm-backend/router/router.go:55` `SetupRoutes` 注册公共与认证路由
- 公共端点：
  - 登录与刷新：`itsm-backend/router/router.go:76-77` → `AuthController.Login` / `AuthController.RefreshToken`
  - 状态与版本：`itsm-backend/router/router.go:80-85`
- 认证组：JWT + RBAC + 租户中间件，`itsm-backend/router/router.go:88-95`，所有业务路由在租户组下挂载

## 控制器功能要点
- 认证控制器：`itsm-backend/controller/auth_controller.go`
  - 登录：`Login` → `itsm-backend/controller/auth_controller.go:21-37`
  - 刷新令牌：`RefreshToken` → `itsm-backend/controller/auth_controller.go:39-55`
  - 租户列表：`GetUserTenants` → `itsm-backend/controller/auth_controller.go:57-73`
  - 切换租户：`SwitchTenant` → `itsm-backend/controller/auth_controller.go:75-97`
  - 当前用户：`GetUserInfo` → `itsm-backend/controller/auth_controller.go:99-115`
- 工单控制器：`itsm-backend/controller/ticket_controller.go`
  - 创建：`CreateTicket` → `itsm-backend/controller/ticket_controller.go:30-65`
  - 更新：`UpdateTicket` → `itsm-backend/controller/ticket_controller.go:67-79`
  - 其余：列表、详情、删除、分配、升级、解决、关闭、搜索、统计、分析、模板、子任务、评论、附件、通知、评分、视图、智能分配等由同文件导出方法提供（见 `router/router.go:133-256` 路由绑定）
- 事件控制器：`itsm-backend/controller/incident_controller.go`（路由绑定参考 `router/router.go:258-268`）
  - 列表/创建/详情/更新/统计端点已注册，关闭事件尚带TODO标记
- SLA 控制器：参考 `router/router.go:270-291`，提供定义 CRUD、合规检查、指标与违规、监控、预警规则与历史
- 其他控制器：`user_controller.go`、`approval_controller.go`、`analytics_controller.go`、`prediction_controller.go`、`root_cause_controller.go`、`ticket_*_controller.go`、`department_controller.go`、`team_controller.go`、`project_controller.go`、`application_controller.go`、`tag_controller.go`

## 服务层映射（摘要）
- 认证服务：`service/auth_service.go`
  - `Login`, `RefreshToken`, `GetUserTenants`, `SwitchTenant`, `GetUserInfo`
- 工单服务：`service/ticket_service.go`
  - `CreateTicket`, `UpdateTicket`, `GetTicket`, `DeleteTicket`, `ListTickets`
  - 操作类：`AssignTicket`, `EscalateTicket`, `ResolveTicket`, `CloseTicket`, `AssignTickets`
  - 分析与导入导出：`GetTicketStats`, `GetTicketAnalytics`, `ExportTickets`, `ImportTickets`
  - 模板：`GetTicketTemplates`, `CreateTicketTemplate`, `UpdateTicketTemplate`, `DeleteTicketTemplate`
- 事件服务：`service/incident_service.go` 列表/详情/创建/更新/统计与升级、监控/告警/指标
- SLA 服务：`service/sla_service.go` 定义 CRUD、违规/指标、监控与合规

## 鉴权与租户
- JWT 中间件：`itsm-backend/middleware/auth.go`（引用于 `router/router.go:89`）
- RBAC 权限：`itsm-backend/middleware/rbac.go`（全局保护认证组，`router/router.go:91`）
- 租户中间件：`itsm-backend/middleware/tenant.go`（认证组内再分租户组，`router/router.go:94`）

## 错误与响应规范
- 统一成功：`itsm-backend/common/response.go:Success`
- 统一失败：`itsm-backend/common/response.go:Fail` 及封装的 `ParamError`, `AuthFailed`, `Forbidden`, `InternalError`
- 控制器使用规范示例：`itsm-backend/controller/ticket_controller.go:33-35,58-65` 的参数校验与错误日志输出

## 深入测试执行与结果（后端）
- 构建测试：`go test ./...` → 编译失败
  - `itsm-backend/pkg/bpmn/engine_adapter.go:16` 不匹配的 `bpmn_engine.New` 调用（需要 `string` 参数）
  - `itsm-backend/pkg/bpmn/engine_adapter.go:43,51,57,62` 访问未导出或不存在的方法：`Export`, `Load`, `NewTaskHandler`, `GetProcessCache`
  - `itsm-backend/controller/user_controller_test.go:40,627,685` 构造函数签名不匹配（缺少 `*zap.SugaredLogger`）
  - DTO 字段不一致：`SearchUsersRequest` 缺少 `Page`, `Department` 等
  - Ent 方法不存在：`ent.UserCreate.SetStatus` 不可用；`dto.RefreshTokenResponse.RefreshToken` 字段缺失；`dto.UserInfo.DisplayName` 与 `Phone` 字段缺失
- 静态检查：`go vet ./...` → 与编译错误相同的签名与接口不匹配

## 改进建议（后端）
- 对齐 BPMN 适配器版本与公开 API，调整 `New` 调用与序列化/处理器注册方法
- 统一控制器构造函数签名，与测试同步更新（注入 `logger` 参数）
- 对齐 DTO 字段定义与服务/测试引用，清理不存在字段或在 DTO 中补充
- 使用 `enttest.NewClient()` 构建测试环境，避免真实依赖；对 `service` 层补齐 Table Driven Tests

