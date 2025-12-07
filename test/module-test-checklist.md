# 模块级功能测试检查清单（端到端）

- 日期: 2025-12-07
- 范围: itsm-backend（Go）与 itsm-prototype（Next.js）
- 目标: 为每个业务模块提供可执行的测试用例清单、接口与页面映射、验收标准与当前阻塞项

## 方法说明

- 覆盖维度：API 合同/权限与租户、UI 交互/布局/A11y、数据校验与错误处理、性能（分页/懒加载/虚拟滚动）
- 统一响应合同：`code/message/data`；错误码按项目规则（成功=0）
- 统一环境：`NEXT_PUBLIC_API_URL`；后端 `JWT+RBAC+TenantMiddleware`

## 认证与账户

- 后端端点：`itsm-backend/router/router.go:76-77` 登录/刷新；认证组 `itsm-backend/router/router.go:88-95`
- 控制器：`itsm-backend/controller/auth_controller.go:21-37`、`:39-55`、`:57-115`
- 前端页面：`itsm-prototype/src/app/(auth)/login/page.tsx`
- 用例
  - 登录成功/失败（错误提示与焦点回到首错输入）
  - 刷新令牌有效性与续期
  - 获取用户信息与租户列表、切换租户后页面与数据隔离生效
- 验收
  - 成功返回 `code=0`，失败信息准确；切换租户影响到列表查询头 `X-Tenant-ID`
- 阻塞
  - 前端测试选择器不精确导致失败；参考 `test/system-testing-report.md`

## 工单（Tickets）

- 后端端点：`itsm-backend/router/router.go:133-199,201-210,215-256`
- 控制器：`itsm-backend/controller/ticket_controller.go:30-65` 创建；更新 `:67-79`；其余同文件
- 前端页面：`itsm-prototype/src/app/(main)/tickets/*`；组件 `itsm-prototype/src/components/business/tickets/*`
- 用例
  - 列表筛选与分页（参数映射、URL 同步）；创建/更新/删除；详情查看
  - 子任务/评论/附件/通知/评分/视图（CRUD 与权限）；导入/导出；智能分配与规则测试
- 验收
  - 统一响应；角色权限校验；操作后列表刷新与提示（Toast）
- 阻塞
  - 组件测试路径错误（`__tests__` 相对路径不匹配）

## 事件（Incidents）

- 后端端点：`itsm-backend/router/router.go:258-268`
- 控制器：`itsm-backend/controller/incident_controller.go`（列表/创建/详情/更新/统计）
- 前端页面与组件：`itsm-prototype/src/app/(main)/incidents/*`、`IncidentList.tsx`、`IncidentFilters.tsx`、`IncidentStats.tsx`
- 用例
  - 列表筛选与分页；新建事件；详情与更新；统计图表与过滤
- 验收
  - 统一响应；筛选与分页参数正确；统计数据与列表一致
- 阻塞
  - 组件中直接 `fetch/localStorage`（`itsm-prototype/src/components/business/IncidentManagement.tsx:162-168`）违反统一 API 层规则

## 问题与调查（Problems）

- 后端端点：`itsm-backend/router/problem_investigation_router.go`
- 控制器/服务：问题与调查、步骤、根因分析、解决方案
- 前端：`itsm-prototype/src/app/(main)/problems/*`、`ProblemList.tsx`、`ProblemFilters.tsx`、`ProblemStats.tsx`
- 用例
  - 新建问题与调查；步骤 CRUD；根因报告生成与确认；解决方案挂接与摘要
- 验收
  - 审批/流程状态联动；操作审计可用

## 变更与审批/工作流（Changes/Workflow）

- 后端端点：工单下审批 `itsm-backend/router/router.go:221-230` 与 BPMN 注册 `itsm-backend/router/router.go:407-411`
- 控制器：`itsm-backend/controller/approval_controller.go`、`bpmn_workflow_controller.go`
- 前端：`itsm-prototype/src/app/(main)/changes/*`、`itsm-prototype/src/app/(main)/workflow/*`
- 用例
  - 新建与审批流程；BPMN 设计与部署；实例与任务操作；版本与自动化
- 验收
  - 审批状态机一致；BPMN 有效部署与执行；权限校验与审计
- 阻塞
  - BPMN 适配器 API 不匹配导致后端编译失败（见报告）

## CMDB

- 后端：`cmdb_controller.go` 与服务
- 前端：`itsm-prototype/src/app/(main)/cmdb/*`、`CIList.tsx`、`RelationGraph.tsx`
- 用例
  - CI 创建/更新/删除；关系图展示与影响评估；拓扑视图与筛选

## SLA

- 后端端点：`itsm-backend/router/router.go:270-291`
- 控制器/服务：定义、指标、违约、监控、预警规则
- 前端：`itsm-prototype/src/app/(main)/sla*`、`sla-dashboard`、`sla-monitor`
- 用例
  - SLA 定义 CRUD；合规检查；违规列表与状态更新；预警规则与历史

## 服务目录与请求

- 后端：`service_catalog*`、`service_request*`
- 前端：`itsm-prototype/src/app/(main)/service-catalog/*` 与 `forms/*`
- 用例
  - 服务项 CRUD；请求提交流程（审批/完成/评分/收藏）；统计与报表

## 知识库与集成

- 后端：`knowledge*`
- 前端：`itsm-prototype/src/app/(main)/knowledge-base/*`
- 用例
  - 文章 CRUD 与分类/标签；版本与协作；关联工单与推荐

## 分析/预测/报表

- 后端：`analytics*`、`prediction*`、`handlers/DashboardHandler`
- 前端：`itsm-prototype/src/app/(main)/reports/*`、`(main)/dashboard/*`
- 用例
  - 过滤与时间范围；导出；保存为仪表盘卡片；分享链接

## AI（搜索/分诊/摘要/聊天/工具）

- 后端端点：`itsm-backend/router/router.go:318-332`
- 控制器：`itsm-backend/controller/ai_controller.go`
- 前端：`itsm-prototype/src/app/lib/ai-service.ts` 与相关页面入口
- 用例
  - 搜索与分诊；长文本摘要；相似事件；工具执行与审批；Embedding 管线

## 用户/团队/部门/租户/项目/应用

- 后端端点：`itsm-backend/router/router.go:294-305,360-389`
- 前端：`itsm-prototype/src/app/(main)/enterprise/*`、`(main)/projects`、`(main)/applications`、`(main)/admin/users`
- 用例
  - CRUD 与成员管理；树形结构展示；租户切换流程；权限角色关联

## 标签与审计

- 后端端点：`itsm-backend/router/router.go:391-405`
- 前端：`itsm-prototype/src/app/(main)/admin/*` 对应管理页
- 用例
  - 标签 CRUD 与绑定；审计日志查询与权限校验

## 通用 UI 与设计系统

- 统一组件：`itsm-prototype/src/components/ui/LoadingEmptyError.tsx`、`Toast.tsx`、`UnifiedForm.tsx`
- 设计系统：Tailwind 与 Ant Design 并存（建议统一 Tailwind）；移除 CSS Modules
- A11y
  - 键盘导航、焦点陷阱、ARIA、`aria-live` 配置；错误播报

## 当前测试进展（自动执行）

- 前端
  - `npm ci` 成功；`npm run type-check` 失败（46 错误）；`npm run test:ci` 执行但 7/10 套件失败
- 后端
  - `go test ./...` 编译失败；`go vet ./...` 报告与编译错误一致

## 验收总标准

- API：响应合同一致，错误码与信息准确；权限与租户校验到位
- UI：表单与加载/空态/错误模式统一；交互一致、键盘与屏幕阅读器友好
- 测试：前端 CI 通过率 ≥ 80%，后端能编译并通过核心服务单元测试
- 规范：统一设计系统与 API 层封装；禁止组件内直接 `fetch/localStorage`
