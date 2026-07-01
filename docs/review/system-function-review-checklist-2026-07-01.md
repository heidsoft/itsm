# ITSM 系统功能开发复盘与 Review 完善 Checklist

**日期**: 2026-07-01  
**范围**: 后端、前端、API 契约、角色闭环、测试与工程治理  
**目标**: 将当前“功能面已铺开”的系统，收敛为可验收、可回归、可发布的企业级 ITSM 版本。  
**状态说明**: `[ ]` 待处理，`[~]` 进行中，`[x]` 已完成，`[!]` 阻塞或需决策。

---

## 0. 执行原则

- [ ] 每个 review 项必须留下证据：代码路径、测试命令、测试报告、截图或 API 响应样例。
- [ ] 只把“真实跑通”的能力标为完成，不能仅凭存在页面、路由或 schema 判定完成。
- [ ] 前后端契约以实际后端路由和 DTO 为准，前端 API client 必须显式对齐。
- [ ] 涉及已有未提交变更时，先确认变更归属，不覆盖用户工作区修改。
- [ ] 所有新增或修改的业务逻辑必须配套最小回归测试。
- [ ] P0/P1 修复完成后，必须更新本 checklist 和对应测试报告。

---

## P0. 基线验收与历史问题复测

### P0-1 工作区与基线确认

- [ ] 记录当前分支、commit、未提交文件。
  - 命令: `git status --short && git rev-parse --abbrev-ref HEAD && git rev-parse --short HEAD`
  - 当前已知未提交文件:
    - `itsm-backend/router/cmdb_routes.go`
    - `itsm-backend/router/router.go`
    - `itsm-frontend/src/lib/api/incident-api.ts`
- [ ] 确认上述未提交变更是否属于本轮 review 范围。
- [ ] 生成本轮 review 工作记录文档或 issue 列表。

### P0-2 后端基础验证

- [ ] 后端编译通过。
  - 命令: `cd itsm-backend && go build ./...`
  - 验收: 0 编译错误。
- [ ] 后端测试通过。
  - 命令: `cd itsm-backend && go test ./...`
  - 验收: 0 failed package；如因本地缓存或外部依赖失败，记录失败包与原因。
- [ ] 后端格式和静态检查通过。
  - 命令: `cd itsm-backend && go vet ./...`
  - 可选: `gofumpt -l .`
- [ ] 生成后端覆盖率基线。
  - 命令: `cd itsm-backend && go test ./... -coverprofile=coverage.out`
  - 验收: 记录 total coverage，并标注低覆盖包。

### P0-3 前端基础验证

- [ ] 前端依赖安装状态确认。
  - 命令: `cd itsm-frontend && npm ci`
- [ ] TypeScript 类型检查通过。
  - 命令: `cd itsm-frontend && npm run type-check`
- [ ] 前端生产构建通过。
  - 命令: `cd itsm-frontend && npm run build`
- [ ] 前端单测通过。
  - 命令: `cd itsm-frontend && npm test -- --runInBand`
  - 如项目脚本不同，以 `package.json` 为准记录实际命令。

### P0-4 历史问题复测

历史报告: `output/itsm-system-test-report.md`

- [ ] 复测登录成功后前端是否正确保存 token 并跳转主界面。
  - 页面: `/login`
  - 账号: `admin / admin123` 或 seed 中有效管理员账号。
- [ ] 复测 CMDB API 是否仍有 404。
  - 重点路径: `/api/v1/cmdb/*`、`/api/v1/configuration-items`、前端 CMDB API client 使用路径。
- [ ] 复测服务目录 API 是否仍有 404。
  - 重点路径: `/api/v1/service-catalog`、`/api/v1/service-catalog/services`、`/api/v1/service-requests`。
- [ ] 复测 SLA API 是否仍有 404。
  - 重点路径: `/api/v1/sla/*`、`/api/v1/sla-templates/*`。
- [ ] 复测 BPMN API 是否仍有 404。
  - 重点路径: `/api/v1/bpmn/*`、流程定义、流程实例、任务完成接口。
- [ ] 复测工单 workflow 操作路径。
  - 重点路径: `/api/v1/tickets/workflow/*` 与 `/api/v1/tickets/:id/workflow/*` 是否存在历史兼容问题。
- [ ] 产出复测结果表。
  - 输出建议: `docs/review/system-function-review-result-2026-07-01.md`

---

## P1. API 契约一致性 Review

### P1-1 契约清单建设

- [ ] 从后端路由生成核心 API 清单。
  - 来源: `itsm-backend/router/router.go`、`itsm-backend/router/cmdb_routes.go`、controller 内注册函数。
- [ ] 从前端 API client 生成调用清单。
  - 来源: `itsm-frontend/src/lib/api/*.ts`
- [ ] 对比后端路由、前端 client、页面实际调用是否一致。
- [ ] 建立 API 契约矩阵。
  - 字段: 模块、前端页面、前端 client 方法、HTTP method、后端路由、DTO、权限、测试覆盖、状态。

### P1-2 分页与列表响应统一

- [ ] 工单列表统一返回 `page`、`pageSize`、`total`、`totalPages`。
- [ ] 事件列表统一返回 `page`、`pageSize`、`total`、`totalPages`。
- [ ] 问题列表统一返回 `page`、`pageSize`、`total`、`totalPages`。
- [ ] 变更列表统一返回 `page`、`pageSize`、`total`、`totalPages`。
- [ ] 资产列表统一返回 `page`、`pageSize`、`total`、`totalPages`。
- [ ] CMDB CI、CI Type、Relationship 列表统一分页契约。
- [ ] 前端列表组件不再依赖历史字段，如 `items`、`size`、`page_size` 响应字段。
- [ ] 为上述列表补契约测试。

### P1-3 DTO 与字段命名

- [ ] Controller 禁止直接返回 Ent 模型。
  - 验收: `common.Success(c, dto.ToXxxResponse(...))` 或 service 已返回 DTO。
- [ ] 响应字段统一 camelCase。
  - 例: `ticketNumber`、`assigneeId`、`createdAt`。
- [ ] 请求兼容策略明确。
  - 若支持 snake_case 兼容字段，必须在 DTO 中显式标注，避免前端继续扩散旧字段。
- [ ] 核查 ticket、incident、change、problem、cmdb、sla、workflow DTO。
- [ ] 对关键 Mapper 增加单测。

### P1-4 权限与错误响应

- [ ] API 成功响应统一 `{ code: 0, message, data }`。
- [ ] 参数错误统一 `1001+`。
- [ ] 认证失败统一 `2001` 或既定 auth 错误码。
- [ ] 内部错误统一 `5001` 或既定系统错误码。
- [ ] 401/403 场景前端能展示可理解错误，不误判为空数据。
- [ ] RBAC 权限缺失时后端不泄漏跨租户数据。

---

## P2. 角色闭环 Review

参考: `specs/001-role-based-testing/tasks.md`

### P2-1 end_user 提单闭环

- [ ] 用户登录。
- [ ] 创建服务请求或工单。
- [ ] 查看自己的请求列表，仅能看到本人数据。
- [ ] 查看详情、评论或补充信息。
- [ ] 处理完成后确认关闭。
- [ ] 提交满意度评分。
- [ ] 验证越权访问他人工单被拒绝。
- [ ] E2E 覆盖: `itsm-frontend/tests/e2e/roles/end-user.spec.ts`

### P2-2 engineer 处理闭环

- [ ] 查看分配给自己的工单、事件、问题。
- [ ] 工单状态推进: `open -> in_progress -> resolved`。
- [ ] 事件处理: 确认、分派、升级、解决、关闭。
- [ ] 事件转问题或关联问题。
- [ ] 问题关联变更。
- [ ] 变更执行后更新 CI 或影响分析。
- [ ] 审计日志记录关键动作。
- [ ] E2E 覆盖: `itsm-frontend/tests/e2e/roles/engineer.spec.ts`

### P2-3 approver 审批闭环

- [ ] 查看待审批任务。
- [ ] 变更审批通过后进入下一状态。
- [ ] 审批拒绝后业务单据进入拒绝或退回状态。
- [ ] 多级审批节点流转正确。
- [ ] 超时升级记录可查。
- [ ] 审批人不能修改非审批字段。
- [ ] E2E 覆盖: `itsm-frontend/tests/e2e/roles/approver.spec.ts`

### P2-4 sd_manager 运营闭环

- [ ] Dashboard 显示今日工单、事件、SLA 风险。
- [ ] 创建或启用自动分派规则。
- [ ] 新工单命中规则后自动分派。
- [ ] SLA 风险列表可见。
- [ ] 手动升级工单或事件后写入审计。
- [ ] 团队队列、未分配队列、超期队列可用。
- [ ] E2E 覆盖: `itsm-frontend/tests/e2e/roles/sd-manager.spec.ts`

### P2-5 super_admin 治理闭环

- [ ] 菜单列表可见且和前端路由一致。
- [ ] GA readiness 端点返回模块状态。
- [ ] 租户切换后用户、工单、CMDB 数据隔离。
- [ ] 角色权限修改后重新登录立即生效。
- [ ] 连接器 lifecycle: install、enable、health、disable、uninstall。
- [ ] 系统配置页面可读写。
- [ ] E2E 覆盖: `itsm-frontend/tests/e2e/roles/super-admin.spec.ts`

### P2-6 security 只读审计闭环

- [ ] security 角色可查看审计日志。
- [ ] security 角色不能创建或修改工单。
- [ ] security 角色不能修改用户、角色、SLA、CMDB。
- [ ] JWT 篡改返回认证失败。
- [ ] 多次失败登录能在审计日志检索。
- [ ] E2E 覆盖: `itsm-frontend/tests/e2e/roles/security.spec.ts`

### P2-7 tenant_admin 租户隔离闭环

- [ ] 仅能查看本租户用户、团队、工单、CMDB。
- [ ] 跨租户读返回空集或 403。
- [ ] 跨租户写被拒绝或服务端忽略租户篡改字段。
- [ ] 菜单和权限受租户范围约束。
- [ ] E2E 覆盖: `itsm-frontend/tests/e2e/roles/tenant-admin.spec.ts`

---

## P3. 模块深度 Review

### P3-1 工单管理

- [ ] 创建工单字段完整: type、priority、category、requester、assignee、tenant。
- [ ] 列表筛选、搜索、排序、分页可用。
- [ ] 详情页展示基础信息、评论、附件、审批、SLA、历史。
- [ ] 分派、升级、解决、关闭、重开动作可用。
- [ ] 自动化规则和智能分派规则可用。
- [ ] 统计数据和 dashboard 指标一致。

### P3-2 事件管理

- [ ] 事件创建、列表、详情、编辑、删除可用。
- [ ] 分派弹窗实际挂载并能提交。
- [ ] 事件升级、确认、解决、关闭动作可用。
- [ ] 根因、影响评估、分类字段与后端契约一致。
- [ ] 告警、事件、指标关联可用。

### P3-3 问题管理

- [ ] 问题创建、列表、详情、状态推进可用。
- [ ] 已知错误库可用。
- [ ] RCA 信息可记录。
- [ ] 问题和 incident/change/ticket 关联可用。
- [ ] 趋势分析页面数据来源明确。

### P3-4 变更与发布管理

- [ ] 变更创建、提交、审批、开始、完成、回滚、取消可用。
- [ ] 风险评估和 CMDB impact 可用。
- [ ] CAB 或审批链语义清晰。
- [ ] PIR 可创建和查看。
- [ ] 发布管理和变更关联策略明确。

### P3-5 服务目录与服务请求

- [ ] 服务目录列表、详情、搜索、统计可用。
- [ ] 服务请求创建、列表、详情、审批、删除可用。
- [ ] `compliance_ack` 等必填字段校验明确。
- [ ] 服务请求能绑定 SLA 和流程。
- [ ] 用户门户和管理端路径不冲突。

### P3-6 CMDB

- [ ] CI Type 管理可用。
- [ ] CI 实例创建、列表、详情、编辑、删除可用。
- [ ] CI 属性定义和继承策略可用。
- [ ] CI 关系创建、列表、拓扑展示可用。
- [ ] 影响分析结果可信，并受 tenant_id 过滤。
- [ ] 导入导出任务可用。
- [ ] 云账号、云资源、云服务 discovery scaffold 状态明确。

### P3-7 BPMN 与工作流

- [ ] 流程定义列表、创建、部署、版本可用。
- [ ] 流程实例启动、暂停、终止、查看可用。
- [ ] 用户任务认领、完成、转派、加签可用。
- [ ] 流程变量持久化和并发更新安全。
- [ ] CandidateGroups 审批语义与角色权限一致。
- [ ] ServiceTask callback registry 可审计。
- [ ] 流程失败时有可读错误和恢复路径。

### P3-8 SLA 与升级

- [ ] SLA 模板安装可用。
- [ ] SLA policy/definition CRUD 可用。
- [ ] 工单 SLA 绑定规则明确。
- [ ] SLA 监控、违约记录、告警历史可用。
- [ ] 升级矩阵可配置。
- [ ] 连接器通知链路可观测。

### P3-9 知识库与 RAG

- [ ] 知识文章创建、发布、评审、版本可用。
- [ ] 普通用户仅能看到 published 内容。
- [ ] 全文搜索可用。
- [ ] RAG 降级路径可用，pgvector 不可用时不影响核心系统。
- [ ] 知识推荐与工单上下文关联可用。
- [ ] AI 输出保留模型、prompt version、confidence、审计记录。

### P3-10 AI Native 与 Skill/Connector

- [ ] LLM Gateway provider 配置可用。
- [ ] triage skill 可给出分类、优先级、处理人建议。
- [ ] summarize skill 可生成摘要并可审计。
- [ ] skill registry manifest 可解析。
- [ ] AI audit console 或 API 可查看 accept/reject。
- [ ] Feishu、DingTalk、WeCom、Webhook connector health 可测。
- [ ] connector marketplace item 生命周期状态明确。

---

## P4. 工程治理与发布质量

### P4-1 大文件拆分

- [ ] 拆分 `itsm-backend/controller/ticket_controller.go`。
- [ ] 拆分 `itsm-backend/controller/incident_controller.go`。
- [ ] 拆分 `itsm-backend/controller/cmdb_controller.go`。
- [ ] 拆分 `itsm-backend/controller/bpmn_workflow_controller.go`。
- [ ] 拆分后路由注册保持兼容。
- [ ] 拆分后测试覆盖不下降。

### P4-2 测试覆盖提升

- [ ] service 层关键路径覆盖率达到 40% 阶段目标。
- [ ] controller 层关键路径覆盖率达到 40% 阶段目标。
- [ ] 新增或修改代码增量覆盖率达到 60%。
- [ ] 为 BPMN XML parser 增加 property-based 或边界测试。
- [ ] 为租户隔离增加跨模块回归测试。
- [ ] 为权限矩阵增加角色维度回归测试。

### P4-3 CI 与门禁

- [ ] `backend-ci` 能稳定运行，不依赖本地缓存偶然状态。
- [ ] `frontend-ci` 能稳定运行，不依赖临时 patch eslint 的长期方案。
- [ ] `ga-gate` 覆盖 G1-G4 并产出清晰 summary。
- [ ] `itsm-cli-ci` 包含最小命令 smoke。
- [ ] `itsm-agent-ci` 包含 build 与 `--version` smoke。
- [ ] `itsm-skill-ci` 校验 manifest schema。
- [ ] 安全扫描结果有 triage 流程。

### P4-4 仓库清理

- [ ] 清理 `.DS_Store`。
- [ ] 清理 `.bak`、`.orig`、`.rej` 文件，若需保留则移入 archive 并说明原因。
- [ ] 清理重复或废弃路由文件，如 `router.go.backup`。
- [ ] 检查 README、ROADMAP、docs 是否与当前功能状态一致。
- [ ] 检查 Docker Compose 生产部署必须显式 `--env-file .env.prod` 的文档和脚本一致。

### P4-5 开源项目治理

- [ ] Issue template 覆盖 bug、feature、security、question。
- [ ] PR template 要求测试证据、截图或 API 响应。
- [ ] CODEOWNERS 覆盖后端、前端、文档、CI。
- [ ] Roadmap 状态与 release note 同步。
- [ ] 建立 review 标签: `area/backend`、`area/frontend`、`area/api-contract`、`area/qa`、`priority/p0-p3`。

---

## 建议执行节奏

### 第 1 周: P0 + P1

- [ ] 完成基线验证和历史问题复测。
- [ ] 输出 API 契约矩阵初版。
- [ ] 修复 P0 阻塞问题。

### 第 2 周: P2

- [ ] 跑通 end_user、engineer、super_admin 三条 P1 角色闭环。
- [ ] 每条闭环至少补 1 条稳定 E2E。

### 第 3 周: P3

- [ ] 深入 CMDB、BPMN、SLA、服务目录、AI audit。
- [ ] 明确哪些能力是 GA、哪些是 beta、哪些只是 scaffold。

### 第 4 周: P4

- [ ] 开始 controller 拆分和覆盖率回填。
- [ ] 清理仓库历史产物。
- [ ] 调整 CI 门禁与贡献规范。

---

## 交付物清单

- [ ] `docs/review/system-function-review-result-2026-07-01.md`
- [ ] API 契约矩阵，可放在 `docs/review/api-contract-matrix-2026-07-01.md`
- [ ] 角色闭环测试报告，可放在 `docs/review/role-flow-review-2026-07-01.md`
- [ ] P0/P1 修复 PR 或 commit 列表。
- [ ] 更新后的 `ROADMAP.md` 和 GA readiness 文档。

---

## 完成定义

本轮 review 只有在以下条件满足时才算完成:

- [ ] P0 全部关闭。
- [ ] P1 API 契约矩阵覆盖核心模块，且无 P0/P1 契约不一致。
- [ ] 至少 3 条关键角色闭环 E2E 稳定通过。
- [ ] CMDB、BPMN、SLA、服务目录、AI audit 的状态被明确标注为 GA、beta 或 scaffold。
- [ ] 所有遗留问题都有 owner、优先级、计划完成时间。
- [ ] 文档、测试报告和 Roadmap 与实际代码状态一致。
