# ITSM 分阶段完善总控计划

**状态**: Draft  
**日期**: 2026-06-28  
**适用范围**: 产品功能、架构治理、业务流程、开源交付、AI Native 演进  
**协作背景**: qoder 正在并行完善后端缺失功能、浏览器测试问题和审批/工作流关系调研。本计划作为总控路线图，避免重复施工，并定义阶段门禁。

## CAPABILITY

项目团队可以按阶段把当前 ITSM 从“功能覆盖广”推进到“开源用户可信部署、企业客户可信评估、核心流程可信扩展、AI Native 能力可信落地”。每个阶段都必须有清晰的用户可见结果、工程门禁和 qoder/Codex 协作边界。

## CONSTRAINTS

- 开源用户可见能力必须真实闭环：写操作要落库，读操作要返回真实数据，未接入后端的能力必须显式标注实验性或规划中。
- v1.0 GA 优先级高于新增功能：先修复 stub、mock、契约不一致和部署问题，再扩大功能面。
- Controller 响应必须走 DTO，不直接泄露 Ent 模型或内部结构。
- 多租户、RBAC、审批、服务请求、SLA、CMDB 关系属于发布门禁，不能只按演示功能处理。
- qoder 正在处理的具体缺失功能和测试问题不重复实现；本计划只定义阶段目标、验收标准和协作接口。
- AI 能力默认实验性，除非具备模型配置、降级策略、审计记录、失败提示和回归测试。

## IMPLEMENTATION CONTRACT

### Actors

- 开源部署者：需要一条可复现的 Docker Compose / 本地开发 / 健康检查路径。
- IT 运维工程师：需要工单、事件、问题、变更、服务请求、知识库、CMDB、SLA 能真实使用。
- 系统管理员：需要 RBAC、多租户、菜单、审批、连接器配置可控。
- 流程管理员：需要审批模板、工单流转和 BPMN 工作流之间的边界清晰，并能逐步绑定。
- 贡献者：需要明确 Roadmap、Issue 模板、测试门禁和贡献入口。
- AI/自动化使用者：需要 AI 输出可追溯、可降级、可审计。

### Surfaces

- 产品文档：`docs/product/*`、`docs/v1-ga/*`、`docs/review/*`
- 后端：`itsm-backend/router/router.go`、`controller/`、`handlers/`、`service/`、`dto/`、`ent/schema/`
- 前端：`itsm-frontend/src/app/`、`src/lib/api/`、`src/lib/auth/`、`src/components/`
- 交付：`.github/workflows/*`、`docker-compose*.yml`、`scripts/smoke-test.sh`
- qoder 计划：`.qoder/plans/*.md`

## 阶段路线

### Phase 0: 协作基线与事实冻结

**目标**: 建立当前状态清单，明确 qoder 与 Codex 的工作边界。

**当前证据**:

- qoder 已有后端缺失功能计划：知识库版本、协作 Session、变更日历。
- qoder 已有浏览器功能测试计划：工作流、工单、事件、问题、变更、Dashboard、CMDB、知识库、Marketplace。
- qoder 已完成审批管理与 BPMN 工作流关系调研，结论是审批模板、工单流转、BPMN 三套系统并行。

**交付物**:

- 本总控计划。
- 每次阶段推进前更新 qoder/Codex 分工。

**门禁**:

- 不重复修 qoder 已声明负责的点。
- 所有新增阶段任务必须能映射到产品能力或 GA 风险。

### Phase 1: v1.0 GA 可信度修复

**目标**: 让开源用户从干净检出开始，能部署、登录、执行核心流程，并相信页面展示的不是假成功。

**必须修复**:

- 已开始：`router.go` 中 legacy fallback 路由不再返回假成功；旧路径会明确提示使用正式接口。
- 已开始：`handlers/dashboard/service.go` 中的旧 SLA dashboard mock 已移除，未接入真实数据时返回明确错误。
- 已开始：`ga-gate.yml` 聚合步骤恢复为单一通过/失败判断，避免 CI 门禁脚本自身语法损坏。
- 待完成：将 legacy 路由逐步替换为正式 controller/redirect/弃用策略，并补 API 兼容测试。
- 待完成：如果仍需要旧 SLA dashboard 包，接入真实查询；否则从路由和构造链路中彻底下线。
- 已开始：A2UI 与首页不再从 localStorage/sessionStorage 读取 token；A2UI 请求改为依赖 cookie 会话。
- 待完成：统一认证 token 策略，收敛后端 `access_token` 与前端 `auth-token` 兼容分支。
- DTO 收敛：优先治理 Asset、Release、Role、Service、Attachment、Comment 等仍可能返回 Ent/内部模型的 controller。
- 已开始：根目录 `.next/trace*` 构建痕迹移出版本控制，并补充根目录 `.next/` ignore。
- 待完成：清理开源仓库构建产物；`node_modules`、`.next`、coverage、standalone deploy 等不得进入版本控制。
- 待评估：`itsm-cli/dist` 当前被 `bin/itsm.js` 引用，不能直接删除；需要先补 CLI 开发/发布入口策略。

**验收**:

- `go test ./...`
- `npm run type-check && npm run build`
- `docker compose -f docker-compose.dev.yml up -d --build`
- `curl /api/v1/health`、登录、核心 API smoke test 通过
- `rg "Stub handler|realistic mock data"` 无 GA 阻断项

**qoder 协作边界**:

- qoder 可继续补 Marketplace migration、浏览器测试报告和具体 bug。
- Codex 负责总控门禁、stub/mock 清除策略、DTO/API 契约治理。

### Phase 2: ITIL 主流程闭环增强

**目标**: 把“模块存在”推进到“业务流程可跑、状态可追、异常可恢复”。

**重点能力**:

- 工单：创建、分派、接受、处理、解决、关闭、重开、评论、附件、评分、SLA、自动化。
- 事件：告警接入、升级、确认、解决、关闭、转问题。
- 问题：调查、根因、已知错误、workaround、关联知识。
- 变更：风险评估、CAB 审批、实施、回滚、PIR、日历视图。
- 服务请求：服务目录申请、审批、交付任务、状态追踪。

**验收**:

- 每条主流程至少有一条端到端 API 测试和一条前端 smoke 路径。
- 状态机转换失败必须返回明确错误，禁止静默成功。
- 所有列表接口分页、筛选、排序语义一致。

**qoder 协作边界**:

- qoder 当前后端缺失功能计划可归入本阶段：知识库版本、协作 Session、变更日历。
- Codex 负责跨模块状态机和端到端验收矩阵。

### Phase 3: 工作流与审批体系统一

**目标**: 解决审批模板、工单流转、BPMN 工作流三套系统割裂的问题。

**阶段设计**:

- 短期：文档和 UI 上明确三套系统的定位，避免用户误解。
- 中期：为 `approval_workflows` 增加可选 `bpmn_process_key` 或绑定表。
- 中期：审批记录可关联 `process_instance_id`。
- 长期：服务目录、变更、工单审批统一通过 BPMN 编排，轻量审批模板作为 BPMN 节点模板来源。

**验收**:

- `/admin/approvals` 能显示是否绑定 BPMN。
- `/workflow` 能查看哪些业务对象触发了流程实例。
- 旧审批模板未绑定 BPMN 时保持兼容。
- BPMN 节点配置可复用审批组、候选人、候选组。

**qoder 协作边界**:

- qoder 已完成关系调研，可继续做字段/API 设计草案。
- Codex 负责兼容策略、迁移顺序、产品语义收敛。

### Phase 4: 开源产品化与贡献者体验

**目标**: 从“代码开源”升级到“项目可被外部用户理解、部署、贡献、二次开发”。

**重点能力**:

- README 与能力矩阵保持一致，避免已完成/实验性/规划中混淆。
- OpenAPI 3.0 / Swagger UI 生成并纳入 CI。
- Issue/PR 模板连接到模块标签和复现模板。
- Demo seed 数据支持一键初始化，且明确只用于开发/演示。
- Release workflow 输出版本说明、镜像、二进制、迁移说明。

**验收**:

- 干净机器按 README 能在 30 分钟内完成部署和登录。
- 每个公开模块有最小截图/接口/数据初始化说明。
- `SECURITY.md`、`CONTRIBUTING.md`、`CHANGELOG.md` 与 release 节奏一致。

### Phase 5: 企业级集成与连接器市场

**目标**: 打通飞书、企业微信、钉钉、Webhook、Jira、监控系统，并为连接器/插件/Skill 市场奠定运行时规范。

**重点能力**:

- Connector manifest、配置 schema、安装、启用、健康检查、测试发送、卸载闭环。
- 凭据加密存储和租户隔离。
- 入站回调验签、幂等、审计。
- 连接器能力注册为流程 ServiceTask 或 AI Tool。

**验收**:

- 飞书、企微、钉钉、Webhook 至少各有一条真实配置和发送测试。
- Marketplace item、installation、version 表具备迁移和 seed。
- 连接器失败能在通知中心或运维日志看到。

### Phase 6: AI Native ITSM

**目标**: 形成区别于传统 ITSM 的 AI Native 能力，而不是简单把 LLM 接到按钮上。

**重点能力**:

- AI 分诊：分类、优先级、处理组推荐、置信度和人工确认。
- AI 摘要：工单/事件/问题/变更全生命周期摘要。
- RAG 知识推荐：向量召回、全文搜索降级、引用来源。
- 自然语言建流程：文本 → BPMN 草稿 → 人工编辑 → 发布。
- AI 变更风险评估：CMDB 影响、历史故障、SLA 风险。
- 智能 RCA：告警、日志、CMDB、历史工单关联。

**验收**:

- 每个 AI 输出包含 prompt version、model、confidence、输入摘要、审计记录。
- LLM 不可用时有规则降级或明确失败提示。
- AI 不直接执行高风险动作，必须有人审或策略授权。

## NON-GOALS

- 本计划不直接替代 qoder 的具体修复计划。
- 本计划不承诺 v1.0 完成所有 ServiceNow 对标能力。
- 本计划不定义商业版/开源版功能切分，除非后续产品明确要求。
- 本计划不把 AI 能力作为 GA 阻断项；AI 在 v1.0 仍可保持实验性。

## OPEN QUESTIONS

- 是否将审批模板与 BPMN 的绑定作为 v1.1 必做，还是延后到 v1.2 低代码阶段？
- 服务目录表单设计器是否采用 JSON Schema/FormIO 思路，还是自研轻量字段模型？
- 开源版是否需要在线 Demo，还是只提供 Docker Compose 本地体验？
- 连接器市场是否允许远程市场源，还是 v1.x 仅支持本地内置和私有上传？
- AI 流程生成是否优先服务 ITIL 模板，还是先做通用 BPMN 草稿生成？

## HANDOFF

- 立即进入 Phase 1：GA 可信度修复。
- qoder 继续执行具体功能补齐和浏览器/API 测试。
- Codex 下一步建议优先处理：
  1. stub/mock 清除与真实接口替换；
  2. DTO/API 契约治理；
  3. 仓库构建产物清理；
  4. 工作流/审批统一设计 RFC。
