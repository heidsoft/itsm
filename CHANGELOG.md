# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Fixed

- **ga-gate 全链路 E2E 仍红的真实原因**：v1.6.14 的 §1.28 修复只解开了流程实例 404，工单类型全链路用例随即推进到下一步并暴露另一处契约漂移——`ticket-type-full-chain.spec.ts` 仍按 `logs` 读 `/api/v1/audit-logs`，而该端点自始由 `handlers/auditlog` 提供、返回平台五键 `{items,total,page,pageSize,totalPages}`（`total` 断言通过、`items` 被当 `logs` 读故为 `undefined`）。用例改为按 `items` 读取，`router/audit_routes.go` 里声称 `{logs,...}` 契约的注释同步纠正。该门自 2026-09-20 起持续红，非本版本引入。

**部署与运维**

- `scripts/deploy-prod.sh` 误回滚正常发布：前端探测用宿主机 `curl http://localhost:3000`，而 `docker-compose.prod.yml` 有意不把 3000 发布到宿主机（浏览器必须走 nginx，否则审计日志只记录容器网关 IP）。探测必然超时，部署在镜像完全健康的情况下被判失败并触发回滚，回滚还会把 `.deploy/current` 覆盖成旧版本。改为按容器 Docker healthcheck 判定，并补一条 `nginx -> frontend` 的宿主机侧连通性检查，`show_health` 与成功横幅同步修正。

## [1.6.14] - 2026-10-06

### Security

- **内置系统角色删除保护（B5.11 P0）**：`DeleteRole` 只看 `roles.is_system`，但 seeder 建的 9 个内置角色（super_admin/admin/manager/it_admin/security_admin/sysadmin/agent/technician/end_user）从未置为 `true`，任何有角色管理权限的用户都能删掉系统角色。现在种子统一标记 `IsSystem`、存量库下次启动自动纠正，并在 service 层按内置码词表二次拦截，命中即 409/4090，见 UPGRADE.md §1.30
- **登录不再降级为跨租户用户名匹配（P0）**：`tenantCode` 解析失败时此前静默走「未指定租户」分支，等于拿用户名全库匹配，现按凭证错误拒绝；用户名在多租户下重名时不再返回误导性的「用户名或密码错误」（旧实现查不出唯一行，用户被永久锁在门外），而是明确要求补租户，见 UPGRADE.md §1.32
- **错误响应不再外泄原始 driver / provider 串**：全仓错误泄漏棘轮从 2026-10-04 实测的 413 处 / 35 个文件清零（`err.Error()` 被当公共消息写进响应），handler 统一改走 `RespondError` / `ParamErrorWithErr` / `NotFoundWithErr` / `BadRequestWithErr`，原始错误只进 zap 日志，公共消息只剩稳定中文文案；`classifyError` 补 `*ent.NotFoundError → 404` 分支（此前所有「资源不存在」被兜底成 InternalError + 500）；`AuthFailedWithErr`（强制 2001+401）与 `ForbiddenWithErr`（强制 2003+403）两个语义强制 helper 补上认证/越权拒绝被 `classifyError` 误分类成 404/409 的缺口
- **流程审计读取加权限门**：重复入口 `/bpmn/monitoring/audit-logs` 删除，唯一入口 `/bpmn/dashboard/audit-logs` 补 `bpmn:read`（此前任何已认证用户可读全量流程审计），响应不再把 `tenant_id` 发给前端，见 UPGRADE.md §1.29

### Fixed

**租户与账号**

- **第二个租户开不出管理员（P0）**：`users.username` 原为字段级全局唯一，而产品基线要求每个租户的管理员都叫 `admin`，第二个租户创建 admin 必然撞唯一键；实测线上库印证 `beta-test` 租户 0 个管理员。唯一键改为 `(tenant_id, username)` 组合唯一，迁移 `20261006_tenant_scope_user_username_unique.sql` 在建索引前先检测存量重名并拒绝；`email` 按决策保持全局唯一（找回密码只按 email 定位账号）。用户创建与改名的查重同步改为按租户，见 UPGRADE.md §1.32

**契约与响应**

- CMDB ontology 自描述端点 `/api/v1/cmdb/ontology` 加 13 项锁契约（五键精确集合、version 稳定、关系词表与枚举值域与 `ent/schema` / `common` 常量同源、缺租户 fail-closed、跨租户拒绝）；`AITools` 改为指针 + `omitempty`，toolRegistry 未注入时整段省略，避免 LLM Agent 把 `aiTools: null` 误读成能力已就绪；前端 `relationship-vocabulary.ts` 三处词表与后端对齐，13 条受控词表一一对应
- 流程实例详情家族 404：`/api/v1/bpmn/process-instances/:id` 此前只有 GET 详情按数字主键寻址，暂停/恢复/终止/变量/审批历史按 `PI-*` 业务键寻址，拿列表返回的实例键回打详情必然 404（ga-gate 全链路 E2E 因此连续红）。现全家族统一以 `PI-*` 业务键寻址，见 UPGRADE.md §1.28
- 启动流程实例的响应从直接序列化数据库模型改为标准 DTO，响应键修正为 camelCase（此前返回 `process_instance_id` 等 snake_case）；审批中心待办的流程实例深链改传业务键
- `/workflow/audit` 页面「流程实例/活动/操作人/受理人」四列此前恒空，随审计表面收敛恢复正常；删除死路由 `/bpmn/dashboard/audit-logs/timeline`（参数名与处理器读取的键从不匹配，任何调用恒返回参数错误且无使用方）
- 审计日志页「total 正常增长、表格永远为空」：后端返回 `items`，前端 `auditlog-api.ts` 按 `logs` 读，现归一化并补 2 条回归测试
- 文本附件上传失败：`net/http.DetectContentType` 返回 `text/plain; charset=utf-8` 这类带参数媒体类型，与白名单里的裸 `text/plain` 精确匹配不上，现两侧都归一媒体类型后再比对
- connector settings 键名风格容错：管理员自由录入的键值对此前被各 builtin connector 硬编码单一风格（`base_url` / `callbackInstanceId`），写成另一种就静默报 "not configured"；新增 `SettingString` / `SettingInt` 按原样 → snake_case → camelCase 依次尝试，5 处消费方切换，前端必填键提示同步更新
- `handlers/ai/repository_impl.go#DeleteConversation` 返回类型与接口声明不匹配（Ent 1.7+ 的 `Exec` 返回 `(int, error)`），此前任何 `go test ./tests/contract/...` 都会在 `handlers/ai` 建包阶段失败

**可靠性**

- PostgreSQL outbox 撞唯一索引连坐回滚：`operational_commands` 的 `(tenant_id, command_type, idempotency_key)` 唯一索引此前让事务内直接 `INSERT` 重复键把整条事务标记为 aborted(25P02)，SLA 违规记录与预警历史被连坐回滚。新增 `internal/commandbus.EnqueueTxIdempotent`（`INSERT ... SELECT WHERE NOT EXISTS`，仍在事务内、不触发 abort path），`escalation_service` 与 `ticket_notification_service` 全量切换，幂等键计算前先对收件人去重

**分页单一所有者（E4-47）**

- 14 处手抄页长规则收敛为 `common.GetPaginationFromQuery(c)` 单点：release/standard_change/knowledge/known_error/bpmn/workflow_template/approval/timer/operations 与 sla/skill/workbench，硬编码缺省页长不再生效，统一跟随平台缺省 20 与上限 100。行为变化：`/sla/performance` 非法 page/pageSize 由 400 改回落缺省、`/skills` 越界页长由夹 100 改回落 20、sla 三列表信封补 `totalPages` 且空集合返回 `items: []`，详见 UPGRADE.md §1.30

### Changed

- **RBAC 单一真源（N4/N5 拍板落地）**：内置角色权限改为「扩权」收敛——`manager` 补 26 码（全域读 + `dashboard:read` 等）、`agent` 补 7 码、`end_user` 补 16 码（ITIL 读面与仪表盘/通知/AI 基线），存量库下次启动由播种「只增不减」自动补齐。同时删除 `middleware.RolePermissions` 约 380 行硬编码兜底表：DBOnly 未配置兜底、菜单与 `/auth/me` 权限展示统一改由 `authz.RolePermissionDefaults()` 派生，与 DB 播种码集结构性同源，双权威漂移不可能再发生；`sysadmin` 兜底由 `*:*` 通配收敛为与 DB 一致的枚举口径。详见 UPGRADE.md §1.31
- **删除 `/agent-ops-demo` 演示页（N6 拍板）**：216 行纯静态假数据页面（硬编码 evidence/guardrails 文案，流程全为 `useState`，不在菜单），访客易误判为真实 AI 运维能力；v1.7 如需演示资产将基于真实 AI 审计/评测链路重做
## [1.6.13] - 2026-10-04

### Security

- CMDB 云资源接口（`/api/v1/cmdb/cloud-*`）跨租户 GET/PUT/DELETE 从返回 500 或假成功改为统一 404 fail-closed，DELETE 命中 0 行不再报 200
- 工单关联三个读取端点（`/relations`、`/relations/stats`、`/configuration-items`）此前不带租户谓词，任何 ticket:read 用户可跨租户读取对方工单标题、状态与承办人；现已贯穿租户过滤
- 删除含非法权限码的 `seed_data.sql`，生产镜像只带 `default.json`；`init_admin.sh` 改为幂等、口令经 stdin 传递

### Fixed

**分页与列表信封收敛（~20 个端点）**

- BPMN 审计、AI 审计日志、事件列表/告警、问题列表、服务请求、CMDB 七个列表、服务目录四个列表、资产/许可证、工单规则/附件/抄送/分配推荐等约 20 个端点，此前写错 `pageSize`（abc、负数、超 100）会让 SQL 不带 `LIMIT` 子句、一次读出整表，响应信封又声明不同页长导致翻页丢数据。现统一走 `common.GetPaginationFromQuery`（缺省 20、上限 100），详见 UPGRADE.md §1.12–§1.26
- 12 个 `gin.H` 内联列表端点的集合键统一为平台标准的 `items`（此前用 `tools`/`conversations`/`rules`/`records` 等领域名），前端 13 处读取同步收敛

**契约与响应修复**

- 6 个 handler 共 24 处 `handlerctx.Resolve*` 拒绝后补写第二份 JSON 的缺陷，改为单一 `if !ok { return }`
- `rbac_precheck_gen.go` 此前静默丢弃 21 个端点的权限声明，DBOnly 模式下非 super_admin 恒定 403；扫描器现已按参数类型判定根前缀，解析失败直接报错
- 工单分析响应键从 snake_case 改为 camelCase，分析仪表板三张图从假空恢复
- 角色列表前端从 `roles` 键改为标准 `items`，五个管理页重新显示角色数据
- 错误分类收敛：`FailWithErr` 此前无条件写 500，现按业务码自动映射 404/403/409/422/503

**报表数据修复**

- 事件趋势报表此前读工单数据、均值单位差 60 倍、窗口选择器从未生效；现改读事件域自身聚合端点 `GET /incidents/stats/report`
- 问题效率报表此前取首页 100 条自算分布、失败伪装为空；现改读后端 `byStatus`/`byPriority`
- 变更统计此前缺 `draft`/`closed` 分桶与类型分布，成功率报表用 `Math.floor` 伪造数据；现补齐 11 状态 + `byType` 真实聚合
- 工单报表此前用 200 条首页冒充全量、超时永远显示 0；现改读 `GET /tickets/stats` 权威分桶

**前端修复**

- antd v6 弃用属性全量收敛（62 个文件 98 处 `Alert message` → `title`、`valueStyle` → `styles.content` 等），`lint:antd` 守卫扩展到 13 类
- 图标操作按钮补齐 `aria-label`（150 处）；服务请求中文标签、SLA 编辑跳转、空 Tooltip 空气泡、工作流设计器加载失败等走查缺陷一并修复
- CI 详情页「变更历史」Tab 此前读不存在的 `logs` 键恒显示空，现按真实信封渲染并支持翻页
- CMDB 前端列表候选集改用翻页读取（`getAllCIs`/`getAllCloudResources`），不再用有界整表冒充全量

**其他修复**

- SLA 后台扫描此前自建私有副本但未注入依赖，每轮 503 被日志吞掉、预警恒为 0；现复用已装配实例
- 统一工作台 `GET /api/v1/workbench` 上下文键名拼错导致恒 401；修复后同批补上计数 SQL 错误
- BPMN 监控时间参数非法值此前静默当成没传，现返回 400；SLA 合规报表日期格式修正为 RFC3339
- 变更审批原子性（P0-1）：审批记录与变更状态合入同一事务
- 工单关联写入路由（P0-2）：`PATCH /tickets/:id/relations` 补齐 HTTP 入口与父子环校验
- 事件优先级词表修正为 `low/medium/high/critical`（此前最高档误用 `urgent`）
- 权限播种改为只增不减，运维手工热修不再被静默撤销
- 租户引导 `roles_pkey` 序列死信修复（`setval` 幂等对齐）

### Changed

- 云资源 provider 只认六个规范值，别名退到适配器；`/api/v1/cloud/*` 旧路径删除，统一走 `/api/v1/cmdb/cloud-*`（UPGRADE.md §1.22–§1.23）
- 生产初始化装配收敛为 `seeder.NewProductionAssembly` 单一所有者
- 抄送可见性的拒绝理由从 500 改为真实权限判定（403/404）

### Tooling

- 前端孤儿文件检测脚本 + 首批 114 个死文件清理
- `lint:antd` 从 3 类扩展到 13 类，加组件归属判定与基线棘轮
- 发布脚本 `scripts/prepare-release.sh`，支持 `--dry-run`/`--tag-only`；`release.yml` 加 `workflow_dispatch` 手动重跑

## [1.6.11] - 2026-10-03

### Added

- **会话真相端点 `GET /api/v1/auth/session`** — 后端一次返回 `{user, tenants, expiresIn}`：身份与权限复用 `/auth/me` 的同一个组装入口，租户列表复用同一个查询，`expiresIn` 由服务端从已认证 access token 的签发时间算出剩余秒数。此前前端要靠 `/auth/me` + `/auth/tenants` 两个探活再加本地推断来决定「我是否已登录」，任一瞬时失败都会产生与后端不一致的会话判断。响应不含任何令牌值（契约测试断言 body 里搜不到 cookie 中的 JWT）。
- **登录/刷新响应新增 `expiresIn`** — 续签调度改用它，不再用浏览器时钟和固定 10 分钟间隔猜会话何时过期。
- **前端会话真相收敛到唯一的后端结论** — 新增 `itsm-frontend/src/lib/api/session-api.ts`，`loadSession / refreshSession / logoutSession` 只给出 `authenticated | unauthenticated | unavailable` 三态：401/403 才是后端判定的未登录，5xx、业务码非 0、响应缺 user 与网络故障一律是 unavailable，既不踢已登录用户也不停在假登录态。续签按后端 `expiresIn` 提前 60 秒调度并共享同一个在途请求（refresh token 单次可用，并行续签会把有效会话判成过期）；登出带 `keepalive`，调用方随后的整页跳转不会取消吊销请求，吊销失败返回可观察的 `revoked=false` 并由上层告警。`/login`、`/refresh` 响应只含 `{user, expiresIn}`，前端从不在本地持有凭证。
- **Change/Release 状态枚举类型安全化** — Change 与 Release 的 status 字段从 `field.String` 迁移到 `field.Enum`，Ent 生成类型安全的枚举（`change.Status`、`release.Status`），编译期即可捕获非法状态值；新增 Release→Change 关联 edge，支持通过 Ent 关系查询关联变更
- **变更域受影响配置项收敛为实体关系（B2）** — `change.affected_cis` 从 `field.JSON`（字符串数组）收敛为 Change ↔ ConfigurationItem 的多对多 edge，关联表 `change_affected_cis`，获得参照完整性与 JOIN 能力（影响分析不再靠 `strconv.Atoi` 反查）。API 契约 `affectedCis: string[]` 保持不变，前端无需改动。存量回填写入 [20260928_change_affected_cis_backfill.sql](./itsm-backend/ent/migrate/20260928_change_affected_cis_backfill.sql)（仅回填可解析为数字 CI ID 的项，名称形态保留在原列供人工核对）。关联表无 `tenant_id`，已在 `internal/schema/tenant_guard.go` 登记 derived 豁免——**未登记则生产 `policy=fatal` 会拒绝启动**。同时删除零引用的 `dto.ToChangeResponse/ToChangeResponseList`（死代码）。⚠️ 实证发现：同一列存在两套互斥语义——现行 `handlers/change` 按数字 CI ID 写入，已停用的 `service/change_service.go` 按 CI 名称写入并按 `NameIn` 校验；后者仅在测试中被引用，移植为「名称→ID 解析」以保持行为不变

### Security

- **Token 吊销收敛为一个存储，refresh token 改为单次使用** — 会话失效此前有四套互不接线的机制：`middleware` 的 access token 吊销存储、`handlers/common` 里 Redis-only 的 refresh 黑名单、从未被生产装配构造的 `service.TokenBlacklistService`（264 行死代码，已删）、以及只约束 access token 的 `MinIssuedAt`。收敛后只剩 `middleware.tokenRevocationStore` 一份，同时覆盖两类凭证：`RevokeRefreshToken` 用 Redis `SET NX` 做原子认领（「检查是否已用」与「标记已用」之间没有竞态，并发续签只有第一个能完成），键名取 JWT 的 SHA-256 摘要而不是明文（吊销列表里存的是仍未过期的有效凭证，明文进 keyspace 等于把凭据泄漏给任何能读 Redis 键的工具）；`MinIssuedAt` 的 TTL 从 1 小时提到 8 天以覆盖 7 天 refresh 生命周期，且刷新链路现在真的读它。语义变化：**所有部署形态下 refresh token 都是单次使用**，未配 Redis 时退化为进程内存储但不再静默跳过检查——旧行为是 `s.redis == nil` 就直接当作没吊销，单副本/开发环境的旧 refresh token 可以无限重放。装配层把「未配 Redis」从沉默变成可观察：启动日志明确写出吊销只在副本内生效。
- **改密/停用/降权后旧 refresh token 不再能换新 access token** — `ResetPassword` 走的批量吊销调用点仍调用从未构造的 `TokenBlacklistService`，等于改密后 7 天凭证照旧可用；现在统一走 `middleware.InvalidateUserTokens`，并由刷新链路的 `MinIssuedAt` 检查消费。吊销存储故障时不再伪装成功：记录安全延迟告警（密码已写入，回滚更糟），响应如实说明旧会话仍可续签。
- **登录/刷新响应令牌收敛** — access token 和 refresh token 不再通过 JSON 响应返回，改为仅通过 HttpOnly cookie 下发，防止 XSS 窃取
- **MSP 跨租户访问加固** — `handlers/msp` 的 `GetCustomerTickets` 与 `AssignMSPTechnician` 现在校验 `AllowedCustomers`，MSP operator 只能访问已授权客户租户，越权访问 fail-closed
- **Webhook 出站 Header 注入防护** — BPMN Webhook Connector 不再透传用户配置的 `Host`、`X-Forwarded-Host`、`X-Forwarded-For`、`X-Real-IP` 等 12 个敏感 header，防止 SSRF 虚拟主机绕过与 IP 伪造
- **加密密钥派生升级** — `EncryptionService` 从确定性 `SHA256(secret)` 升级为 HKDF-SHA256 + 随机盐 + 版本字节；新密文格式 `version(1) || salt_len(2) || salt || nonce(12) || ciphertext`，旧密文自动 fallback 解密（迁移期兼容），为未来密钥轮换奠定基础
- **审计日志敏感字段掩码扩充** — `MaskSensitiveFields` 新增 `signing_secret`、`corp_secret`、`agent_secret`、`encrypt_key`、`app_key`、`bot_token` 等 connector 密钥字段的正则规则，防止审计日志泄露连接器凭据
- **注册接口移除 role/tenantCode（P0-1）** — 自助注册此前接受请求体中的 `role` 和 `tenantCode`，攻击者可提权为任意租户的 super_admin。现在角色固定为 `end_user`，租户取系统唯一活跃租户；多活跃租户时 fail-closed 拒绝注册，需管理员开通

### Tooling

- **分页信封形状门禁（5.5）改为委托棘轮并升为 HARD，同时删掉与之矛盾的第二套扫描器** — `scripts/static-gates/check-pagination-shape.sh` 此前自带一个 `grep 'type [A-Za-z]*ListResponse struct'` + 60 行文本窗口的扫描器，打印违规却恒 `exit 0`（advisory，且没有任何 workflow 调用它），而同一件债务在 `itsm-backend/tests/contract/list_envelope_ratchet_test.go` 里已有 AST 棘轮随 `go test ./...` 在 backend-ci **硬失败**：两个判定互相矛盾。实测覆盖关系是棘轮为严格超集——`dto/` 里名称含 List 的 57 个结构体中 39 个是信封（18 个是 `List*Request`/过滤器，本就不该按信封计数），旧正则只命中 28 个 `*ListResponse` 且这 28 个全部含 `total`，棘轮另覆盖 11 个旧扫描器完全失明的 `ListTicketsResponse`/`ListCIsResponse`/`ListProblemsResponse`/`ListAuditLogsResponse` 等。**刻意未按「把旧规则升硬」的字面执行**：旧规则要求所有列表信封都含五元组，与 `docs/api-reference.md` 已评审承认的「不分页的列表 `{items,total}`」直接冲突，照字面加硬等于锁死一条永远无法满足的规则、后续只会被迫放宽阈值；正确方向是删掉重复扫描器、把判定交给棘轮。现在脚本只做三件事：跑 `common` 分页序列化单测（`totalPages` 必须由 `NewListResponse` 真实算出）→ 跑 `TestListEnvelope*` → 打印基线条目数，任一失败 **exit 1**；`totalPage`（少一个 s）的旧拼写检查从脚本 grep 迁进棘轮的 `forbiddenPagingKeys`，判定面不缩小。`docs/testing/static-analysis-gates.md` §5.5 同步改写，并新增表格写明真实阻断位置（结构体层=棘轮随 backend-ci 硬失败；handler 层=5.10；本地 `run-all.sh`=无 workflow 调用、不阻断），避免「脚本变硬 = CI 变硬」的误读。实测：脚本 PASS 并打印「存量基线条目 40」（同批 E4-6 把两条诚实的不分页信封改判为合法形状、PIR 按补全五元组收口后，实测为 37，见下一条）；负证明——临时注入 `dto/zz_gate_probe.go`（`foos` + `size` 双违规）后脚本 exit 1 且同时命中 `TestListEnvelopeRatchet` 与 `TestListEnvelopePagingAliases` 两条，删除探针后回 exit 0。
- **分页键棘轮改为只判「部分分页」，与文档承认的第二种合法形状对齐** — `TestListEnvelopeKeys` 此前对每个信封都要求凑满五元组，纯 `{items,total}` 也被登记成债务，这与 `docs/api-reference.md`「不分页的列表」条目直接矛盾（该契约同时**禁止**给不分页端点伪造 `page/pageSize/totalPages`）。现在判定是：信封里出现了 `page`/`pageSize`/`totalPages` 任一键才要求五键齐备。逐条读服务层后按实测分型，而不是批量放宽：`service/ticket_view_service.go:28` 与 `service/ticket_comment_service.go:104` 都是无 `Limit` 的 `All(ctx)`、handler 里 `total = len(items)` 诚实，从基线移除（属合法形状，补键反而是伪造契约）；`dto/change_pir_dto.go|ChangePIRListResponse` 相反——它真的做 `Offset/Limit`，因此按补全五元组收口（见上方 Fixed）。`envelopeKeyBaseline` 从 8 条降为 5 条（全部在 CMDB 一侧：有 `page` 却用 `size` 顶替 `pageSize`、也没有 `totalPages`，服务层实测真分页），存量债务合计从 40 条 / 33 个结构体降为 **37 条 / 30 个结构体**。**不是放水**：注入「只有 `page`」的探针实测仍 `--- FAIL: TestListEnvelopeKeys`（exit 1），注入纯 `{items,total}` 的探针为绿；门禁脚本 `check-pagination-shape.sh` 的打印条目数同步为 37，`docs/testing/static-analysis-gates.md` §5.5 记录两种形状的边界。


- **e2e 一次性隔离栈与「跑前自动证明」硬守卫** — 写业务数据的夹具此前只能对着常驻栈跑，而本机实测的污染路径是 `next.config.ts` 与 `src/app/api/[...path]/route.ts` 的代理 upstream 默认值 `http://localhost:8090`（`127.0.0.1:8090` 属于生产容器 `DB_NAME=itsm_prod`，`[::1]:8090` 属于另一个会话起的宿主二进制）：浏览器只打环回 `:3000` 也挡不住写进生产库，所以光靠 URL 白名单不构成隔离。新增 `scripts/e2e-isolated-stack.sh` + `docker-compose.e2e.yml`：新 compose project、唯一容器名、独占端口（`127.0.0.1:18090`/`127.0.0.1:3001`）、postgres/redis 不发布宿主端口、空卷首装，然后经 `:3001` 代理登录并 `POST /api/v1/users` 建一个随机名 canary，再用 `docker exec` 进本栈 postgres 数出恰好 1 行——代理若指向别的后端，这里立刻红。证明结果写成 `$TMPDIR/itsm-e2e-proofs/<project>.json`（0600，含本栈随机 admin 口令，仓库内不出现可用口令），`tests/e2e/harness.ts` 的 `isolatedBaseURL()` 只认这份 proof：要求声明 `ITSM_E2E_ISOLATED_STACK=1`、显式 `PLAYWRIGHT_BASE_URL` 在环回白名单内、proof 未过期、地址与 proof 逐字符一致（`localhost` 与 `127.0.0.1` 不做别名换算）、且 proof 记录的 upstream 不是环回地址。`playwright.config.ts` 在隔离模式下不再由宿主起 `:3000` dev server。八条拒绝/放行分支有离线回归 `tests/e2e/isolated-stack-guard.spec.ts`（不需要真实栈，实测 8 条全绿；篡改 proof 的四种输入逐条被拒），正向也在真栈上跑通一次 proof-backed cookie 会话。用法与变量见 `docs/dev-commands-reference.md` §2.5。
- **拆栈不能吞掉 compose 的解析错误** — 上一项的 `down` 实测出现「打印已拆除、容器与卷一个没少」：`itsm-init` 的 `${E2E_ADMIN_PASSWORD:?}` 在 compose **解析阶段**就校验，拆栈时没带该变量整条 `down` 直接失败，而脚本把 stderr 与退出码一起吞了。现在拆栈/查状态先用 up 时留存的口令文件补一个只用于让解析通过的值（拆栈不创建容器），失败输出可见、退出即报错，并按 `label=com.docker.compose.project` 复核残留，有残留就拒绝删 proof；`up` 里的预清理同样复核残留，同时拒绝把拆栈占位值当真实口令起栈。实测：修复后一次 `down` 清掉 5 个容器 + 4 个卷 + 网络，重复 `down` 幂等退出 0。
- **空卷首装门禁 `make fresh-install-gate` / `fresh-install-gate.yml`** — 把「私有化全新部署装得起来」变成可复用的独立证明：用一次性 compose project + 全新命名卷起 postgres+redis，跑 `itsm-init`（migrate+seed）并要求退出码 0，随后断言迁移入账数、磁盘迁移 checksum 非空（空 checksum 就是「登记过但从未执行」，正是只在新装才炸的来源；历史收养条目 001-006 除外）、BPMN 流程模板已部署、活跃流程绑定指向的流程定义全部存在。此前唯一会跑到这条链的 ga-gate「Start core stack」自 2026-09-22 起连续 17 次红、且不在 main 的必需检查里，等于守卫存在但没人消费。实测：正向 `迁移 52 条（空 checksum 0）、活跃流程定义 18、活跃流程绑定 10（悬空 0）` 退出 0；负向用 `scripts/fixtures/fresh-gate-negative.override.yml` 注入一条引用不存在表的磁盘迁移，门禁如实报 `failed to apply migration 20991231_gate_negative_fixture: ... (42P01)` 并退出 1。CI 侧 push/PR/每日 schedule 触发，失败上传 init 日志。
- **把「只在全新安装暴露」的迁移缺陷类锁死** — `migration/fresh_install_reference_test.go` 新增三条守卫：禁止任何磁盘迁移引用 Ent 从未生成的旧式单数 RBAC 对象（`permission_definition`/`role_permission`/`role`/`endpoint_acls`）、禁止给已退役死表建索引、禁止迁移直接写 `process_bindings`，并锁住退役迁移必须以原版本名保留且仍为 no-op；`pkg/seeder/process_bindings_manifest_test.go` 断言内置流程绑定的每个 key 都能在 `service/bpmn/*.bpmn` 找到可部署模板。四个守卫在修复前实测为红、修复后为绿。
- **四域浏览器旅程改走真实登录链路** — `tests/playwright-four-domain-journey.cjs` 原先读 `POST /auth/login` 响应体的 `accessToken`，令牌改 httpOnly cookie 后必然拿不到；现改为真实 SPA 表单登录并断言已离开 `/login`，写请求统一由 `authedFetch` 附带 Double-Submit CSRF 头。运行时 `API_URL` 必须与站点同源（cookie 按 host 下发，发给 `127.0.0.1` 不会带上 `localhost` 的令牌）。同时修掉 `scripts/rebuild_and_verify.sh` 里前端镜像的错误 tag（`itsm-itsm-frontend:latest` → `itsm-frontend:latest`，此前 compose 拿不到刚构建好的镜像）。
- **文档门禁 CI 转绿（C.1 + C.3）** — `docs-gate` 在 CI 连续红于 5 个 commit：C.1 检出 `.github/workflows/ga-gate.yml` 的 4 处口令夹具、C.3 检出 19 条指向 `output/*.md` 与 `itsm-rag/README.md` 的内链。根因是这两个门禁只扫 `git ls-files` 的跟踪文件，而 `output/`（`.gitignore:271`）与 `itsm-rag/`（`.gitignore:213`）都被忽略——本地文件在、链接解析得到，CI checkout 里不存在，所以 `make docs-gate` 本地绿、CI 红（CI 传 `--strict`，本地默认 advisory）。修法：19 条内链中把「决策对话」类的 2 条改指仓库内 `docs/product/decision-dialogue-2026-09-24.md`、`docs/product/ai-automation-status.md` 的 `./ROADMAP.md` 修正为 `../../ROADMAP.md`（原本就指错位置），其余 16 条降级为行内代码并标注「本地工作稿，未纳入版本控制」；`ga-gate.yml` 的口令夹具按门禁既有词汇补 `开发环境…不得用于生产` 注释（值与 YAML 语义不变，已用解析器校验）。验证：在 HEAD 的干净 worktree 里 `run-all.sh --strict` = 7 total / 0 failed（C.1 0 处、C.3 0 条）。
- **Gate C.7 产品需求收敛守卫**（[check-scope-creep.sh](./scripts/docs-gate/check-scope-creep.sh)）— 把"继续扩散 = 构建失败"作为机器守卫生效，对应 2026-09-28 盘点的三类扩散面：C.7.1 空壳前端模块零容忍（有 `page.tsx` 但穿透 `@/components` 引用后仍无后端调用的模块，**新增空壳无豁免通道**；⚠️ 修正：初版检出「10 个模块/15 页」经逐文件复核**全部为误判**——漏 `.tsx` 扩展名解析、漏 `lib/services/*` 封装层、未排除 `redirect()` 兼容路由页，真实空壳为 **0**，白名单已清空，三类教训写入脚本与白名单文件头）/ C.7.2 预览域棘轮（README「预览」能力域数只减不增，当前 9）/ C.7.3 规划能力面冻结（ROADMAP v2.0+v3.0 未完条目只减不增，当前 14）。已接线 `run-all.sh`，新增 `make scope-creep` 目标；注入回归 5 用例验证生效（基线收紧/白名单过期/注入空壳均 FAIL，advisory 不阻断）。收敛方案与待拍板决策见 [plans/scope-convergence-plan-2026-09-28.md](./plans/scope-convergence-plan-2026-09-28.md)
- **产品表面棘轮基线上调 `service_go_files` 330→331（Gate C.6.4）** — 新增的是 `service/msp_allocation_service_test.go`（E3-1 回归锁：重新分配不得改写已归档分配的 `deassigned_at`），不是新的生产逻辑；该指标按 `service/*.go` 计数、包含测试文件。理由已按门禁流程登记在 `scripts/docs-gate/product-surface-baseline.txt`。
- **产品表面棘轮基线同步（Gate C.6.4）** — `bootstrap_app_lines` 1811→1815：会话真相与吊销收敛的装配接线（`internal/bootstrap/app.go` 改用 `middleware.ConfigureTokenRevocationRedis` 接管吊销存储，并把「未配 Redis」从沉默降级改为显式告警，删掉 `SetRedis` 与 `NewService(..., nil)` 死参数）；`service_go_files` 333→332：删除从未被生产装配构造的 `service/token_blacklist_service.go`，四套 token 吊销机制收敛为一套。两处理由已按门禁流程登记在 [scripts/docs-gate/product-surface-baseline.txt](./scripts/docs-gate/product-surface-baseline.txt)。
- 新增技能管理端只读列表与详情：`GET /api/v1/admin/skills`、`GET /api/v1/admin/skills/:code`（`ai:read`，与市场发现共享实现），列表接入统一分页契约 `{items, total, page, pageSize, totalPages}`（page 默认 1、pageSize 默认 20 上限 100，非法入参回退默认值）
- 租户管理页新增「初始化」列与状态抽屉：按需加载并缓存每租户的产品基线安装状态（基线就绪/未就绪组件、安装命令状态与尝试次数、模板版本、命令错误），`dead_letter` 可一键重放安装（复用运维命令重放入口、保留原幂等键）；状态接口新增 `commandId` 以支撑重放
- e2e 登录工具适配令牌 Cookie 化（不再读取已废弃的 `accessToken`、移除硬编码口令改走 `E2E_ADMIN_PASSWORD`/`ADMIN_PASSWORD`），并新增 `tests/e2e/tenant-provisioning.spec.ts` 固化「新建租户 → outbox 安装基线 → 状态就绪」全链路
- 产品表面棘轮基线上调 `service_go_files` 326→327、`bootstrap_app_lines` 1776→1805：分别对应组件 checksum 的内嵌 BPMN 摘要（`service/bpmn_template_digest.go`）与租户开通 outbox 接线（`POST /api/v1/tenants` 的 `tenant.bootstrap.install` 主链路），理由已登记在 `scripts/docs-gate/product-surface-baseline.txt`；部署与回归记录见 [docs/testing/deploy-regression-2026-09-24.md](./docs/testing/deploy-regression-2026-09-24.md)
- `initialize -action audit-tenants`：逐租户只读基线审计，输出每组件 verified/error 的 JSON 差异报告，作为存量前滚修复方案的输入（写入拦截 + total_changes 不变的测试锁死只读）
- **Gate C.6 产品口径漂移守卫**（[docs-gate/check-product-drift.sh](./scripts/docs-gate/check-product-drift.sh)）— 把"口径不一致 = 构建失败"作为机器守卫生效，对应 2026-09-22 审计的 5 项无守卫平面（C.6.1 成熟度双向闭合 / C.6.2 领域清单走目录 / C.6.3 零路由域包 / C.6.4 表面棘轮 / C.6.5 覆盖率口径披露）。CI 默认 hard；存量债务（变更管理口径冲突、department/root_cause/dashboard 三个孤儿包）已在 [product-drift-waivers.txt](./scripts/docs-gate/product-drift-waivers.txt) 登记 owner+到期日，**10-31 到期未解决自动反向 FAIL**。详见 `output/product-drift-overdesign-audit-2026-09-22.md`（本地工作稿，`output/` 未纳入版本控制）§5
- **AGENTS.md 不再硬编码领域清单** — `handlers/<domain>/` 域包清单以目录为准，文档不再漂移；Ghost domain `cab` 已移除
- **ROADMAP 披露前端覆盖率口径** — 显式声明 jest `collectCoverageFrom` 仅含 `src/lib/**`，避免 80% 门槛被误读为产品覆盖率
- **删除空目录 `itsm-backend/handlers/team/`** — 未跟踪、无生产代码，纯负担

### Fixed

- **`GET /api/v1/changes` 的分页夹紧收给单一所有者，风险等级只留一套参数名** — 这里原来是裸 `strconv.Atoi(c.DefaultQuery(...))` 且忽略错误，参数原样进 Ent 的 `Offset/Limit`，而**响应的分页元数据走另一套夹紧**（`SuccessWithPagination` → `NewPaginationResponse` → `ValidatePagination`）：查询侧和声明侧对同一个非法入参给两个答案。实测三处后果：`pageSize=0` 在 Ent 的 `if q.Limit != 0` 分支里等于不加 `LIMIT`，**25 条数据一次返回 25 条，而响应同时写着 `pageSize:10`**；`pageSize=5000` 让 SQL 真去取 5000 条（只被数据量兜住），响应却声明 `pageSize:100`；`page=0`/负数算出负 `OFFSET`（SQLite 测试库把它当 0 吞掉，Postgres 则按规范拒绝负 OFFSET：`ERROR: OFFSET must not be negative`，该条在生产库上是 500 而不是静默）。和上面两条通知/PIR 端点是同一个缺陷类，修法也同一套：HTTP 入口统一走 `common.GetPaginationFromQuery`（缺省 1/20，只有落在 `(0,100]` 的查询值被采纳，越界回落默认页长而不是夹到 100）。夹紧归属保持**单点**——`handlers/change/service.go` 的 `ListChanges` 实测只有一个调用方（本 handler），因此不再补第二道校验，避免同一规则两处真相。排序补 `ID` 并列键（`created_at` 非唯一，同一秒创建的变更在页边界归属不确定）。参数名删掉 `risk_level`：后端曾「先读 snake_case、空则读 camelCase」，属 AGENTS.md 禁止的多字段兼容，而实测前端从未发送过 `risk_level`（`change-api.ts` 发 `riskLevel`），留下双名只会让两套契约共存。同步纠正一处**文档与实际相反**：`docs/api-reference.md` 的创建/更新示例写的 `risk`、`plannedStartAt` 从未被 DTO 绑定过，按真实 JSON tag 改为 `riskLevel`、`plannedStartDate`、`plannedEndDate`、`affectedCis`，并明确登记 `type`、`priority` 两个查询参数后端不识别（见计划台账 E4-11）。回归 `router/change_list_route_test.go`（11 例）打在 `SetupRoutes` 生产注册（含 `RequirePermission("change","read")`）上：五键精确集合、默认页长 20/`totalPages` 2、三页（10/10/5）拼接不重不漏、`pageSize=0`→20 条、`page=-1`→第 1 页、`pageSize=5000`→回落 20、`riskLevel=high` 过滤生效、`risk_level=high` 不生效（证明双名已删）、空结果序列化 `[]`、agent 角色行级数据范围只见自己创建/被分配的 3 条、租户 B 只见自己的 25 条、未认证 401。**负证明**：把 `handler.go` 与 `repository_impl.go` 还原到修复前形态后路由测 11 例中 **4 例转红**（默认页长、`pageSize=0` 整表、越界页长、`risk_level` 仍生效），handler 契约测 7 例中 **6 例转红**；`page=-1` 一例修复前后皆绿，因为它恰好被响应侧那套 `ValidatePagination` 夹紧掩盖——正是「两套夹紧」的旁证。前端 `change-api.ts` 删掉 `risk: params.risk` 的字段名转换、`changes/page.tsx` 两个调用点改用 `riskLevel`、`ChangeList.tsx` 去掉 `|| []`/`|| 0` 兜底，契约测从 `stringContaining('page=2')` 加强为精确 pathname + 精确 query 集合。默认页长 10→20 与 `risk_level` 移除属**⚠️ 破坏性契约变更**，见 [UPGRADE.md](./UPGRADE.md) §1.11。遗留债务：`service/change_service.go` 的 `ChangeService.ListChanges` + `dto.ChangeListResponse`（集合键 `changes`、无 `page`）实测零生产构造，是第二套死契约（台账 E4-12）。
- **通知列表两个端点收敛为各自唯一的信封，`pageSize=0` 不再整表返回** — `GET /api/v1/notifications` 与 `GET /api/v1/tickets/{id}/notifications` 都把集合键写成 `notifications`（AGENTS.md 禁止的领域名第二键），前者还额外用 `size` 顶替 `pageSize`、缺 `totalPages`。两者欠的不一样，补法也不同：通知列表实测真的做 `Count` + `Offset/Limit`（`service/notification_service.go`），因此补齐五元组；工单通知是按 ticket+tenant 一次取全量、无 `Limit`，属「不分页的列表」合法形状，只改集合键，**不伪造** `page/pageSize/totalPages`。夹紧规则此前在 handler 和 service 各写一份（同一规则两处真相，HTTP 侧读的还是 `size`）：现在 HTTP 入口统一走 `common.GetPaginationFromQuery`，service 保留一道 `common.ValidatePagination` 给非 HTTP 调用方（worker/其他 service），`totalPages` 由 `common.NewPaginationResponse` 单点算出；排序补 `ID` 并列键（`created_at` 非唯一，同秒入库的通知在页边界归属不确定）。请求 DTO 同时把 `userId`/`tenantId` 标成 `form:"-"`，客户端自报身份不再参与绑定——只能来自认证上下文。**实测更正一条口头说法**：`GetPaginationFromQuery` 对越界的 `pageSize`（如 5000）是**回落到默认 20**，不是夹到 100；夹紧到 100 是 `ValidatePagination` 的行为，两者对「非法输入怎么办」的答案不同，已按实测写进注释、swagger 与测试。回归 `router/notification_envelope_route_test.go`（12 例）打在 `SetupRoutes` 生产注册上：五键精确集合、三页（10/10/5）拼接不重不漏、`pageSize=0`→20 条而非整表、`page=-1`→第 1 页、`pageSize=5000`→回落 20、`read` 过滤同步收敛 `total`/`totalPages`、空结果序列化 `[]`、同租户另一用户与跨租户都看不到、自报 `userId/tenantId` 无法放大可见范围、未认证 401、工单通知键集合精确 `{items,total}`。负证明：把 5 个实现文件还原到修复前形态后 **10 例转红**（另 3 例锁的是修复前即正确的隔离与鉴权行为），恢复后全绿。前端 `ticket-notification-api.ts` 两个响应类型改单键并补 `pageSize/totalPages`（顺带删掉「后端分页字段是 size（不是 pageSize）」这条与线上相反的注释），`getUserNotifications` 参数 `size`→`pageSize`，`Header.tsx`、`NotificationCenter.tsx`、`notifications/page.tsx`、`TicketNotificationSection.tsx` 四个消费点去掉 `|| []`/`?? []` 兜底。**⚠️ 破坏性契约变更**，集成方迁移见 [UPGRADE.md](./UPGRADE.md) §1.10。

- **变更 PIR 列表补齐分页键，`pageSize=0` 不再整表返回** — `GET /api/v1/changes/pirs` 是真分页的端点（`service/pir_service.go` 做 `Count` + `Offset/Limit`），却只返回 `{items,total}`：调用方拿到当前页无从知道第几页、共几页，只能把这一页当成整个结果集。同时 handler 用裸 `strconv.Atoi` 读查询参数、参数原样进 Ent：实测 **`pageSize=0` 在 Ent 的 `if q.Limit != 0` 分支里等于不加 `LIMIT`，25 条数据一次返回 25 条**；`page=0`/负数会算出负 `OFFSET`。现在 DTO 补齐 `page/pageSize/totalPages`（由 `common.NewPaginationResponse` 同一算法算出，避免这里再写一份除法），handler 改走 `common.GetPaginationFromQuery`（缺省 1/20，越界值回落默认页长），service 侧先用 `common.ValidatePagination` 归一再落 `Offset/Limit`；排序补 `ID` 并列键——`review_date` 非唯一列，同一天录入的 PIR 在页边界归属不确定。默认页长从 10 变为 20 属契约变化，已同步 swagger 注解与前端 `PIRListResponse` 类型，`changes/pirs/page.tsx` 去掉 `response.items || []`/`response.total || 0` 兜底并加请求序号防止慢的旧页响应覆盖新过滤条件。回归 `router/change_pir_route_test.go`（7 例）打在**生产注册**（`SetupRoutes` → `SetupChangeRoutes` → `RequirePermission`）上：五键精确集合、三页切分不重不漏、`pageSize=0` 夹紧为 20、`page=-1` 回落第 1 页、`result` 过滤同步收敛 `total`、空结果序列化 `[]`、租户 B 只见自己、未认证 401。负证明：把三个文件临时还原到修复前形态后四条转红，其中信封断言实测 `expected [items,page,pageSize,total,totalPages] actual [items,total]`、`pageSize=0` 断言实测 `should have 20 item(s), but has 25`；恢复后全绿。PIR 端点从未写进 `docs/api-reference.md`，因此本批无文档可同步（该缺口登记在审计计划 E4 台账）。
- **邮件处理队列不再把「第一页 20 条」当成整个队列** — `GET /api/v1/email-intake/conversations` 后端本来就按 `page/pageSize` 分页并返回全量 `total`，但前端 `emailIntakeService.conversations(status)` 从不发送分页参数，于是永远命中后端默认 `pageSize=20`；页面又丢弃 `total`、让 Antd `Table` 对已到手的 20 行做本地分页，翻页器只在 20 行里翻页。队列超过 20 条时，运维看到的是「共 20 条」而不是还有多少待处理。现在前端显式发送 `page`/`pageSize` 并按后端 `total`/`totalPages` 做服务端分页（状态过滤归零页、旧响应由请求序号丢弃）；响应改走标准 `common.SuccessWithList`，此前手拼 `gin.H` 缺 `totalPages`（实测修复前信封只有四个键）。列表排序补上 `ID` 作为并列键：`last_message_at` 在 schema 里只有 `(tenant_id,status,last_message_at)` 复合索引、不是唯一键，同一秒到达的多条会话在页边界上归属不确定。回归 `handlers/email_intake/conversations_list_test.go`（6 例）打生产 `h.RegisterRoutes`，覆盖三页切分不重不漏、`total` 为全量而非页长、status 过滤同步收敛 `total`、租户 B 看不到租户 A、缺租户上下文 401 fail-closed、信封键集合精确；前端 `email-intake-service.test.ts` 把 `conversations` 的请求体锁成 `{page,pageSize,status}`。同域另外 7 个 `{items,total}` 列表（客户/分部/来源组织/合同/外部映射/值班/班次）实测本就一次返回全量、`total` 诚实，不存在同类截断，其缺 `page/pageSize/totalPages` 的信封形状归 E4。验证：`go build ./...`、`gofumpt -l` 空、staticcheck 无命中、`go test ./handlers/email_intake/` 绿、`npm run type-check` 干净、全量 jest 181 suites / 2 871 passed + 13 skipped、statements 80.39%（阈值未调低）。

- **RBAC 路径预检映射补上 MSP 分配历史，守卫重新为绿** — `middleware/rbac_precheck_gen.go` 是生成物，新鲜度由 `TestPrecheckMapIsFresh` 守卫；E3-1 接线 `GET /api/v1/msp/allocations/history` 时只跑了 `./handlers/msp/` 与 `./router/` 窄测，没有重跑 `./middleware/`，于是这条路由在全量套件里一直是红的（HEAD 实测 `--- FAIL: TestPrecheckMapIsFresh`，与本次 E3-3 改动无关，属存量缺陷）。按守卫自己的指令 `go run ./cmd/authz-gen` 重新生成，产物只新增这一条 `{msp_allocation, read}` 映射，与 `router/msp_routes.go` 的行内权限一致；`go test ./middleware/` 全绿。教训写入审计计划 E3 收口进度行 6：改路由必须跑 `./middleware/`，窄测范围不能凭直觉。

- **删掉 survey 与 feishu 的两份路由注册副本，并把飞书回调的安全测试挪到真实入口** — `handlers/survey/handler.go` 和 `handlers/feishu/handler.go` 各有一份从未被调用的 `RegisterRoutes`，把 `router/` 里的 11 条注册（survey 7 + feishu 4）逐字重复了一遍；两处若同时接线 gin 会直接 panic，所以它们是纯死代码，但会持续误导读者以为 handler 才是路由所有者。feishu 那份的分组参数由调用方决定，而旧测试正是以 `RegisterRoutes(api, api)`（同一分组既当 auth 又当 public）调用它的，按那种形态接线会把 OAuth 回调和事件 webhook 挪进鉴权组——四个安全分支测的是生产永不走的那条路径。现在副本删除、原地注明唯一所有者，四个分支（合法签名放行、nonce 重放 403、过期时间戳 403、未知 instance 400）改打真实的 `router.SetupFeishuRoutes`（`router/feishu_route_test.go`），并额外锁住注册表面恰好 4 条与分组归属（管理端点在鉴权哨兵之后、两个外部回调必须在 public；哨兵用 418，才能和 handler 自身缺租户上下文的 fail-closed 401 区分开）。survey 域此前零测试，新增 `router/survey_route_test.go` 锁 7 条路由表面；2026-09-17「越权写收口」的授权口径注释随所有者迁到 `router/survey_routes.go`。删副本还暴露一处陈旧白名单：`writeRouteExemptions` 里指向 `handlers/feishu/handler.go` 的条目被守卫的防腐化检查实测判红（"未命中任何已注册路由"），同批删除——这也说明该守卫真在跑。刻意不新增"重复注册"静态门禁：这类失效是响的（gin panic），而静态判定"不可达 route owner"需要调用图，`handlers/bpmn/` 里 7 个由同包转发的合法嵌套注册会被文本启发式误报。验证：`go build ./...`、`gofumpt -l` 空输出、staticcheck 三包无命中、`go test ./router/ ./handlers/feishu/ ./handlers/survey/` 全绿；负证明——把两个 `Setup*Routes` 的分组前缀改成 `/__disabled__` 后 surface 与归属测试转红，把 webhook 路径改名后安全契约首条即 404。

- **删除 78 个不可达前端文件（22 503 行），并补上邮件接入这一真实在用的契约锁** — 「不可达」不靠目录名猜：从 `src/app/**` 的 `page/layout/template/error/loading/route` 入口做 import 图 BFS（`@/` 解析到 `src/`，含动态 `import()` 与 `require()`），闭包外的生产源码即死代码。实测删除的九组：`template-api`/`batch-operations-api`/`reports-api`/`collaboration-api`/`change-classification-api`/`priority-matrix-api` 六个 client 及其 hook、组件目录与类型（后端路由实测零注册），`ticket-service-v2.ts`（与 `ticket-service` 双实现）、`src/lib/templates/**`（9 文件 1 543 行，连自己的测试都不引用它）、`src/lib/reports/report-engine.ts`、`components/business/ticket-modal/**`（15 文件 1 630 行，全仓零引用）、`TicketDependencyManager.tsx`（748 行，工单依赖实际走 `RelationPanel`）、`useServiceCatalog.ts`（18 个 hook 外部引用数实测全为 0，服务端状态已由页面直连 `service-catalog-api` 承担）。顺带摘掉两个把死实现重新对外露出的 barrel（`lib/api/index.ts`、`lib/services/index.ts`），`DISABLED_API_CONTRACTS` 从 11 行减到 5 行——实现已删除却仍留豁免会让 `api-contract.test.ts` 因陈旧条目失败，`changeClassification` 因 `change-api.ts` 仍发 `/changes/templates/*` 而保留。同批删除 E2-2b 留下的 `getServiceAnalytics` 显式「不可用」桩与 `ServiceAnalytics` 类型：前端已无任何消费方，保留一个「能调但必定失败」的方法等于把同一件未实现的事写在两处。**覆盖率不降门槛**：jest 只统计 `src/lib/**`，被删的恰好是其中覆盖最好的一片——HEAD 基线实测 statements 80.45%，第一轮删除后 78.91%；定位到同样落在 `src/lib` 内、零导入者的 `lib/templates/**`（192 语句、146 未覆盖）删掉后回到 80.17%；再删 `useServiceCatalog.ts` 落到 79.93%（差 6 条语句），这 6 条用真实契约测试补齐而不是调低阈值——新增 `src/lib/services/__tests__/email-intake-service.test.ts`（21 用例）锁死 `emailIntakeService` 全部路径/动词/请求体，逐项对照 `handlers/email_intake/handler.go` 已注册路由与 `conversationVersionRequest`/`correctionRequest`/`overrideRequest`（含六个状态迁移动作必须带乐观锁 `version`、`override` 必须 `confirmed: true`、可选查询参数缺失时发送 `undefined` 而不是空对象）。验证：`npm run type-check` 干净、`lint:check` 0 error（12 条存量 warning，均在本批未触达的文件）、全量 jest 181 suites / 2 871 passed + 13 skipped、statements 80.39%、`make docs-gate` 7 total / 0 failed。**实测纠正审计口径**：`ticket-relations-api` 并非死实现（`TicketDetail.tsx:69` → `RelationPanel` → `useTicketRelations` 真实在用），已保留。发现未在本批修：`GET /email-intake/conversations` 走后端默认 `pageSize=20` 而队列页丢弃 `total`、翻页器只对已到手的 20 行分页，运维看到的「第一页」会被当成整个待处理队列——归入 E3-4。

- **MSP 分配历史从「前端单打一条没注册的路由」变成真实接线，并修掉两份会让历史失真的缺陷** — `msp-api.ts` 一直在请求 `GET /api/v1/msp/allocations/history`，而 `router/msp_routes.go` 从未注册它，页签只能靠 `PRODUCT_CAPABILITIES.mspAllocationHistory=false` 藏着；同时 `dto.MSPAllocationHistory` 声明了 `deallocationReason`/`createdBy`/`createdByName`/`customerName` 四个表里根本没有列可供给的字段。实测 `msp_allocations` 已带 `assigned_at`/`deassigned_at`/`role`/`msp_user_id`/`customer_tenant_id`，真实历史读不需要迁移，因此选择接线而非删除：`service.ListHistory` 的租户边界只来自认证上下文（该表没有 `tenant_id`，只能经 `HasMspUserWith(user.TenantIDEQ(...))` 收敛，Count/List 用 `Clone()` 复用同一条件），响应统一用 `MSPAllocationDTO` + 标准 `items` 信封，`endDate` 按「包含当天」处理。同批修掉两个直接污染这份历史的缺陷：① `Create` 第 4 步把同一「员工×客户」下所有已解除归档行的 `deassigned_at` 刷成当前时间，每次重新分配都在改写历史（回归断言归档行保持原时刻，修复前实测拿到当前时间即红）；② `Deactivate` 忽略影响行数，匹配不到活跃行照样回成功，现返回 404/4004。前端删掉「解除原因」列与虚构类型，并把分配列表/管理页的 `customerName` 改为后端真实产出的 `customerTenantName`（此前那一列恒为空），`msp-service.getAllocationHistory` 去掉 `res || []` 的假成功兜底；能力开关打开、`DISABLED_API_CONTRACTS` 豁免同步删除（陈旧豁免会让契约测试失败）。回归：`router/msp_route_registration_test.go` 断言真实注册表含该 GET（去掉注册即红），`handlers/msp/allocation_history_test.go` 覆盖跨租户排除、含已解除行、缺上下文 401、畸形参数 400、deallocate 404。**遗留**：解除原因与操作人要真正可查，需要给 `msp_allocations` 加落库列并写入审计，属新契约面，不在本批伪造；前端 jest 覆盖阈值只在跑全量时成立，本批按 3 suites / 44 tests 验证。

- **收回产品文档把「有测试」当成「能力可用」的申报** — `docs/product/frontend-menu-and-features.md` §7 用 ✅ 同时表示测试存在与功能可用，实测有 11 条不成立：`change-classification-api`、`priority-matrix-api`、`batch-operations-api`、`collaboration-api`、`template-api` 五个 client 的请求路径在后端**一条都没有注册**（`grep` router：零 `/api/v1/priority*`、零 `/api/v1/change-classifications`、零 `/api/v1/tickets/batch/*`、零 `/api/v1/mentions/*`；通用 `/api/v1/templates` 也不存在，只有 `/api/v1/tickets/templates*`），另外 6 条 ✅ 的 hook（`useChangeClassification`、`usePriorityMatrix`、`useReports`、`useTemplateQuery`、`useCollaboration`、`useBatchOperations`）除自身测试外零生产消费方——`src/components/{change-classification,priority-matrix,batch-operations,collaboration,templates,reports}` 六个目录对目录外的引用数实测均为 0。§7 改为写明 ✅ 只等于「存在同名测试」，并把判定依据指向两份代码内清单（`api-contract.test.ts` 的 `KNOWN_UNMATCHED_FRONTEND_PATHS` 当前为空，未注册路径全部集中在 `product-capabilities.ts` 的 `DISABLED_API_CONTRACTS`）。同时把 `itsm-commercial-capability-contract.md` 与 `business-snapshot-2026-09-24.md` 里「优先级矩阵」从**工单/事件 GA 候选的业务证据**列删除并写入缺口：后端确实有一份 `service/priority_matrix_service.go`，但它的注入方法 `SetPriorityMatrixService` 全仓**没有任何调用点**（实测只有定义自身），所以 `incident_service.go:242,627` 的 impact×urgency→priority 派生分支在生产链路上永不执行——接线还是退役归入 E5 孤立能力裁决。
- **capability 控制平面不再把标签与模板申报为 GA** — `handlers/capability/handler.go` 的 `tag`、`template` 此前写 `MaturityGA`，而实测两者的数据面都未收敛：标签仍有 `ticket_tags` 与 `tags.code` 双表（E1-3 只收口了写侧唯一所有者，退役边界待拍板），模板只有工单域那一套有路由。现降为 `MaturityPilot`，侧边栏按 `Sidebar.tsx:173` 显示 Pilot 徽标，`read/manage` 动作与鉴权不变（降级只改申报口径，不放宽也不收紧访问面）。回归 `TestUnprovenCapabilitiesAreNotDeclaredGA` 打真实 `Handler`，断言两个 key 为 pilot——临时改回 GA 实测即红。
- **同文档的 capability 治理条目按实测更新** — §4.3 与 §9 第 6/7 条仍写"`capabilityPathRules` 在 `menu-config.ts` 与 `Sidebar.tsx` 各声明一份，建议去重并补 `/projects`、`/applications`、`/admin/tags`、`/templates` 规则"，实测这两项都已完成（`Sidebar.tsx:15` 只 import `capabilityForPath`，规则表唯一且含这 16 条）；§4.2 补齐这四行映射，§8.3 改为记录仍未保护的三个真实入口 `/sla-dashboard`、`/workflows`、`/improvements`（页面目录存在，但 `capabilityForPath` 只做 `prefix` 与 `prefix + '/'` 匹配，不会被 `/sla`、`/workflow` 顺带覆盖）。
- **MSP 报表把「调用者的 user ID」当成租户 ID 去查工单** — `GET /api/v1/msp/reports/{customers,performance}` 从认证上下文取 `user_id` 后传给按 `tenant_id` 过滤的服务方法，于是任何调用者读到的都是「ID 恰好等于自己用户 ID 的那个租户」的工单数与解决率，绩效报表还能用 `?mspUserId=<任意用户ID>` 指定这个错位维度。现在聚合租户只来自 `tenant_id`（缺失即 401 fail closed），`mspUserId` 这个从未实现的员工维度过滤改为 400 并写明原因，绝不静默忽略后返回租户级汇总。回归 `handlers/msp/handler_test.go` 造了「调用者 user ID == 别人租户 ID」的 fixture：本租户 1 张、对方租户 2 张，断言两个报表都只计 1 张——修复前实测两条断言都拿到 2（跨租户读数）。同批把 `GetCustomerTicketsForMSP` 的 `total` 改为真实全量计数（此前 handler 只能用当前页长度冒充 total，前端分页永远显示不出总量），元素统一过 `TicketResponse` DTO。
- **列表信封收敛：集合只放在 `items` 下，不分页的接口不再伪造分页** — `handlers/**` 里还有 10 个响应把集合放在领域键（`reports`/`instances`/`logs`/`templates`/`categories`/`notifications`/`tickets`）下，前端只能按每种别名取值。标准五元组走 `common.SuccessWithList`，确实不分页的（去重分类、日期区间聚合、模板全量）只带诚实的 `{items, total}`。新增静态门禁 **5.10 `check-list-envelope.sh`**（硬，已接进 `backend-ci` 的 lint job）：带 `page`/`pageSize`/`totalPages` 的 `gin.H` 必须有 `items` 键，领域列表键与分页字段同时出现即失败。引入当轮即命中两处真实缺陷：① `GET /known-errors/stats` 伪造 `page`/`totalPages`，顺藤摸出统计服务把同一个 Ent 查询复用于 8 个计数——Ent 的 `Where` 会就地改接收者（`_q.predicates = append(...)`），于是后 7 个计数全被 AND 串联，`resolved`/`deprecated` 与四个 severity 恒为 0，改为 `Clone()` 后按维度独立计数；② `GET /known-errors/categories` 返回 `{items}` 而前端按 `categories` 取值再 `|| []` 兜底，分类筛选下拉恒为空。前端同步删除虚构契约：`types/msp.ts` 声明的 `customerName`/`slaComplianceRate` 等后端从不产出的字段、`msp-service` 的 `Array.isArray(res.tickets)` 与 `res.total || 0` 猜测、MSP 页面按这些字段渲染的列（改为按真实 DTO 出两套装表列），以及 `known-errors` 页的 `response.categories || []`。契约与口径限制见 `docs/api-reference.md`「MSP 报表的口径与限制」，门禁说明见 `docs/testing/static-analysis-gates.md` §5.10。**遗留（行 2d）**：MSP 报表仍未按被服务的客户租户分组，一次只产出一行汇总——属统计口径缺口，不在本批伪造字段补齐。
- **静态门禁文档与实况不符** — `docs/testing/static-analysis-gates.md` 声称 CI 在 backend/frontend workflow 里跑 `run-all.sh`，实测 `grep -rn "static-gates" .github/workflows/` 只有新增的 5.10 真在跑；而标记为 HARD 的 5.1（禁止裸 `c.JSON`）实测有 6 处命中（`handlers/dingtalk` 3、`handlers/wecom` 2、`handlers/approval/routes.go` 1），本地 `run-all.sh` 以退出码 1 结束、CI 却是绿的。文档改为按实测写明接入位置与 5.1 的失败状态，并说明钉钉/企业微信回调是外部协议规定的响应形状，接 CI 前需先显式登记豁免而不是继续裸用。
- **服务目录的分析/收藏/评分把「没实现」显示成「没数据」** — `ServiceCatalogApi` 有四个方法根本不发请求却返回成功形态：`getServiceAnalytics` 造一整套全零指标和编造的统计周期、`getServiceRatings` 回 `{ratings:[],total:0,avgRating:0}`、`getFavorites` 回 `[]`、`recordServiceView` 静默丢弃写入（后端 `router/service_catalog_routes.go` 只有 list/search/stats/get/create/update/delete，没有这些接口）。现在统一走该文件已有的 `unsupportedFeature` 显式失败，读路径不再伪装空成功——用户无法区分「这个服务没人用过」和「分析能力没接」。`getCatalogStats` 同时停止替后端补零：它此前把后端从不返回的 `totalRequests`/`pendingRequests`/`topServices`/`trends` 全填 0，现按 `handlers/service_catalog.ServiceStats` 的真实契约收敛，前端 `ServiceCatalogStats` 类型同步减为 `totalServices`/`publishedServices`/`categories`。回归 `src/lib/api/__tests__/service-catalog-api.test.ts` 把原来给假数据作证的用例（"should return default analytics"、"should not throw"）改成断言显式失败与响应只含真实字段，修复前实测 6 个用例红、修复后 31 个全绿；`npm run type-check` 与改动文件 eslint 通过。**遗留（E3 裁决）**：`ServiceAnalytics` 类型与 `useServiceAnalyticsQuery`/`useFavoritesQuery`/`usePortalConfigQuery`/`useCatalogStatsQuery` 四个无消费方的 hook 暂留。
- **删除两套与真实链路并行的死实现：假云发现与重复的智能分配** — `itsm-backend/service/cloud_discovery_service.go`（674 行）没有任何生产调用者（`NewCloudDiscoveryService` 只被自己和自己的测试引用），而它整套发现逻辑是假成功：AWS 的 `discoverEC2`/`discoverS3`/`discoverRDS` 与 Azure 两个发现器返回空切片并记一条日志，阿里云/腾讯云/华为云直接 `return nil`，调用方拿到的「发现到 0 个资源」和「发现从没实现」完全同形；`upsertCloudResource` 的查重查询还缺租户条件（`cloud_discovery_service.go:422`，与历史评审记录一致），一旦被接回就是跨租户覆盖。真实链路是 command-bus 的 `cloud.DiscoveryWorker`（`internal/bootstrap/app.go:855`）加 `/api/v1/cmdb/discovery/jobs`——后者已按能力口径 fail-closed 返回 503。同理删除 `itsm-frontend/src/lib/services/smart-assignment-service.ts`（455 行）：它与 `lib/api/ticket-assignment-api.ts` 对同一用例双实现，且用 `POST` 打后端注册的 `GET /api/v1/tickets/assign-recommendations/:id`（`router/ticket_routes.go:247`），除自己的单测外无人引用；其 `getUserSkills`/`getAllUserWorkloads` 返回 `[]`、`getUserWorkload`/`updateUserSkill` 抛错，同一未实现能力两种语义。两处删除后 `go build ./...`、`go test ./service/ ./service/cloud/... ./handlers/cmdb/` 与 staticcheck 全绿，未留下悬空引用。
- **问题 SLA 端点把「能力没接」卖成「SLA 正常」** — `GET /api/v1/problems/:id/sla` 恒返回 200 + `{slaStatus:"none", responseTimeUsed:0, resolutionBreached:false}`，而问题域既没有 SLA 截止时间字段、也没有任何为 `aggregate_type=problem` 写 `sla_states` 的代码路径（`sla_policy` 的枚举里声明支持 problem，实现是空的）。调用方读到的是「这个问题有 SLA 且未超时」，与「SLA 未接入」完全不可区分。现在与问题评论走同一口径：先按租户校验资源存在（不存在/跨租户仍 404/4004，非数字 ID 400/1001，未认证 401），再返回 503/5003「问题 SLA 尚未接入 SLA 引擎」，响应不再带伪造的零值倒计时。前端顺带清掉替它圆谎的死面：`components/problem/ProblemSLACard.tsx` 不可从任何页面到达（只被自己的文件引用），删除；`Problem` 类型里后端从不返回的 `slaStatus`/`responseDeadline`/`resolutionDeadline` 三个字段与 `ProblemApi.getProblemSLA` 一并移除，`problemSla` 中文文案与替假契约作证的单测同步删除。回归 `router/problem_sla_route_test.go` 打真实 `SetupRoutes`，修复前实测 503/5003/消息/无 `slaStatus` 四条断言全红（拿到的是 200 假数据）。契约见 `docs/api-reference.md`「问题 SLA 与评论（能力未就绪）」，Swagger 已 `make swagger-gen` 重生成（CI 的 `Swagger Docs Freshness` 是硬门禁）。
- **供应商接口把后端故障和跨租户删除都伪装成成功** — `/api/v1/vendors` 有注册路由、有 RBAC 资源、库里也有数据，却没有任何前端引用，而它四条链里三条在骗调用方：`ListVendors` 用 `total, _ :=` / `vendors, _ :=` 吞掉 Count/All 的错误，查询失败仍返回 200 + `{"list":[],"page":1,"total":0}`（实测把「没有供应商」和「数据库挂了」做成同一个响应）；`GetVendor` 把所有错误压成业务码 404，而 404 不在 `common.Fail` 的 status 映射表里，实际是 HTTP 200 带 404 码；`DeleteVendor` 忽略条件删除的影响行数，删别人的、删不存在的都回 `{"message":"deleted"}`。现在服务层给出 `ErrVendorNotFound` / `ErrVendorCodeExists` 哨兵并把其余故障如实上抛（编码冲突走数据库唯一约束，不再当 500），handler 按语义映射为 409/4090（编码重复）、404/4004（不存在与跨租户同样处理，不确认对方资源存在）、400/1001（ID 非数字或缺必填）、500/5001（真故障，原始错误只进日志）；列表信封从 `{list,total,page}` 改为全站唯一的 `{items,total,page,pageSize,totalPages}`。回归 `router/vendor_routes_test.go` 打真实 `SetupRoutes`：租户只来自认证上下文（请求体自报 `tenantId=999` 被忽略）、tenant B 列表为空且读写删 tenant A 记录一律 404 且记录未被改掉、重复删除是 404、响应为 camelCase 且不含 `vendor_type`；数据库故障用例用第二条连接删掉 `vendors` 表制造「只有供应商查询失败、认证链路仍正常」的条件（整库关闭会先 401 掩盖语义），断言列表与单条读都是 500/5001 且响应不含 SQL 细节。修复前实测两个测试函数的全部关键断言为红。**待拍板**：`vendors.code` 是全局唯一索引而非按租户组合，跨租户占用同编码会冲突（E5 vendors 归属裁决一并处理）。
- **工单依赖影响分析每次 GET 必 400** — `GET /api/v1/tickets/:id/dependencies` 是 router 真实注册的只读入口（`router/ticket_routes.go:286`，权限 `ticket:read`），handler 却用 `ShouldBindJSON` 绑一个 `action` 必填的结构体：GET 不带请求体，任何调用都固定得到 400「请求参数错误」，这个入口从未可用过。影响分析的输入是枚举过滤语义，按契约归属查询参数，现改为绑定 `dto.RelationImpactAnalysisRequest`（`?action=close|delete|change_status&newStatus=...`，`binding:"required,oneof=..."` 保留），并把那个声明了 `ticketId` 却零引用的死 DTO 收成本接口唯一契约——被分析工单只来自路径段，租户只来自认证上下文。错误语义同时收敛：服务层新增 `ErrDependencyTicketNotFound` 哨兵，「工单不存在」与「跨租户」统一 404/4004（不确认对方资源存在），缺 `action`/非法 `action` 是 400/1001 且消息写明允许取值，其余故障才是 500/5001，原始 ent 错误只进日志。回归 `router/ticket_dependency_route_test.go` 打真实 `SetupRoutes`，10 个子用例：三种 action 的分析分支与 camelCase DTO、`riskLevel` 分级、租户 B 把 `parent_ticket_id` 指向租户 A 工单时不得进入 A 的影响面、跨租户与自报 `tenantId` 都是 404 且响应不含对方工单号/标题、不存在是 404、非数字 ID 与缺参是 400、未认证是 401；修复前实测 7 个子用例为红。契约文档见 `docs/api-reference.md`「工单依赖影响分析接口」。
- **侧边栏「全局标签」是 97 条菜单里唯一一条死链** — 菜单基线 `pkg/menubaseline/baseline.go:147` 注册的是 `/admin/tags`，页面实际在 `/tags`，点下去是 Next.js 404；而这条链以前没有任何守卫能发现（文档和门禁都是绿的）。修法是把页面 `git mv` 到 `src/app/(main)/admin/tags/page.tsx` 而不是改菜单 path：种子的匹配键是 `(tenant, path)`，改 path 会在每个已种子租户里新建一行并把老行变成孤儿，改页面归属则存量零变更即生效；`capabilityPathRules` 的 `/tags` → `tag` 同步为 `/admin/tags`，面包屑补「系统管理」层。同类问题加硬守卫：docs-gate 新增 **C.6.6 菜单入口可达性**，把 `menubaseline` 全部 path 与 `src/app/**/page.tsx`（剥掉 route group 段）逐条比对，命不中就 FAIL（hard 模式，CI `--strict` 已在跑）；实测撤掉页面移动即 1 FAIL / 退出码 1，恢复后 97 条 path 全绿。同步收回 `docs/product/frontend-menu-and-features.md` §8.1/§9 里「新增顶级菜单 `/tags`」的建议（与真实基线相反的方向）。**待拍板**：该菜单行 `PermissionCode: "system:read"` 与页面真实所需的 `ticket_tag:*` 不一致（`security_admin`/`audit_admin` 看得到却 403，持有 `ticket_tag:*` 的角色要靠 elevated 短路才看得到），该字段会随 `upsertMenu` 刷新到存量行，属访问控制变更，不在本批动。
- **全局标签页写了就坏，标签写链收敛到唯一所有者** — 「全局标签」页打的是只读别名 `/api/v1/tags`（后端只注册了 `GET /tags` 与 `GET /system/tags`），列表能看、新建/编辑/删除/绑定全部 404，而真正可写的 `/api/v1/ticket-tags` 没有任何 UI 调用；页面上点「编辑」改完再保存走的还是 create。前端改接 `ticketTagService`（列表信封按后端真实契约读 `items`，此前类型声明的是不存在的 `tags`），并删除只剩一份单元测试在替不存在的端点作证的 `tag-service.ts`；页面补上真实 PUT 更新、服务端分页、启用开关，反馈改走 `App.useApp()`。写链自身四个缺陷一并修掉：`CreateTagRequest.TenantID` 带 `binding:"required"`，客户端不自报租户就 400（租户只能来自认证上下文，改为 `json:"-"`）；`isActive` 是非指针 bool，请求体不带该字段时新标签全被建成停用；空颜色写成空串（Ent 字段默认值对显式 `SetColor` 不生效），现在补 `#1890ff`；名称按工单绑定路径 `ResolveTagIDsByNames` 的口径 trim，否则带空格的标签会与自动创建的同名标签重复共存。失败语义从「一律 500/5001 操作失败」改为可区分：同租户重名 409/4090、在用标签删除 409/4090、跨租户读写删 404/4004（不确认对方资源存在）、空白名称 400/1001，原始错误只进日志。回归 `router/ticket_tag_routes_test.go` 打真实 `SetupRoutes`，覆盖 10 个子用例：默认值与租户来源、重名与撞名、列表 `items`/`total` 及 tenant B 看不到 tenant A、跨租户读写删与解绑、在用删除被拒后解绑可删、绑定不存在的标签是 404 而不是 500、按名称自动创建的标签归属调用者租户；修复前实测 8 个子用例为红。契约文档见 `docs/api-reference.md`「工单标签接口」。**待拍板**：`tags` 表（`code` 列、seed 内置清单按 `Code` 键控、5 张关联表外键指向它）与只读别名的退役方案，见 `plans/edge-feature-stability-audit-2026-10-02.md`。
- **工单附件下载与预览点了就 404** — `handlers/ticket_attachment` 的 `DownloadAttachment`/`PreviewAttachment` 从未在 `router/ticket_routes.go` 注册，而前端两条入口（详情页附件区的下载链接与预览新窗口）用的正是这两个地址，浏览器拿到的是 Gin 的 `404 page not found`；列表响应的 `fileUrl` 还透传库里历史的 `/api/v1/tickets/attachments/<文件名>/download`，同样没有对应路由。现在注册 `GET /api/v1/tickets/:id/attachments/:attachment_id`（`attachment`）与 `…/preview`（`inline`），权限沿用 `ticket:read`；`fileUrl` 改由 `(ticketId, id)` 在 DTO 层推导，与注册路由同源，上传也不再落库这个纯派生值（`file_url` 列本就是可选，存量行无需迁移即自愈）。错误语义一并收敛：此前所有失败都回 500/5001，「工单或附件不存在」「跨租户」「同租户内非相关方」「磁盘读失败」在调用方眼里完全一样；现在分类为 404/4004（跨租户按不存在处理，不确认对方资源存在）、403/2003、500/5001，原始错误只进日志。回归 `router/ticket_attachment_route_test.go` 打真实 `SetupRoutes`：断言回吐磁盘原始字节与 `Content-Length`、两种 `Content-Disposition`、列表返回的 `fileUrl` 点下去就是 200、租户 B 拿不到租户 A 的附件内容、同租户有 `ticket:read` 但非相关方的 end_user 是 403、附件不存在是 404、文件已从磁盘消失仍是 500 且不泄漏绝对路径。修复前实测 7 个子用例 13 条断言为红。
- **工单导出永远失败** — `/tickets` 列表的「导出」按钮点下去必定得到「导出失败」：`handlers/ticket` 只留了一句 `export not implemented in handlers layer` 的占位，而带租户谓词的可用实现一直在 `service/ticket_service.go` 里，两者只差一次委派没接上，响应恒为 HTTP 500 / 业务码 5001。现在 handler 走真实实现，并顺带补齐导出文件本身的正确性：CSV 列序与行序改为显式声明（此前按 Go map 迭代生成表头，同一份数据两次导出的列顺序不一样，无法做自动化比对）、CSV 加 UTF-8 BOM（此前 Excel/WPS 打开中文列是乱码）、`format=excel` 产出真正的 xlsx（此前返回 CSV 文本却按 excel 命名，下载得到打不开的 `tickets.excel`）、下载头按格式给出准确的 `Content-Type` 与文件名。契约同时收窄：`dto.TicketExportRequest` 的 `format` 不再声明从未实现的 `pdf`，前端类型同步，请求 `pdf` 现在是 400/1001 参数错误而不是 500。回归 `router/ticket_export_route_test.go` 打真实 `SetupRoutes` 入口，断言 BOM、表头顺序、`Content-Disposition`、xlsx 的 `PK` 魔数、两次导出字节一致、`status` 过滤真实下查到查询，以及租户 A 与租户 B 互相看不到对方工单；修复前实测 8 条断言为红。
- **普通角色读到全租户工单（行级数据权限从未生效）** — `handlers/ticket` 与 `repository/ticket` 各有一套 `DataScope` 枚举：前者三档（`All=0, Department=1, OwnedOrAssigned=2`），后者两档（`All=0, OwnedOrAssigned=1`）。适配层用 `ticket.DataScope(dataScope)` 直接做数值转换，`2` 原样传下去后仓储层的行级谓词永不命中，于是 technician/end_user 拿到整个租户的工单。2026-10-02 在一次性隔离栈实测：technician 名下 23 条却看到 45 条。现在在适配器边界按档位显式映射，未知档位一律按最窄档处理（新增枚举值不会静默放宽权限）。回归两层：`handlers/ticket/repository_impl_datascope_test.go` 用 enttest 打真实仓储，断言三档结果集、`Department` 失败关闭、`userID=0` 返回空；`handler_test.go` 的 mock 仓储真的执行行谓词，经真实路由断言 technician/end_user/admin 各见 2/1/3 条并覆盖缺 userID 的 fail-closed。修复前两个用例都为红（非管理员档拿到全量）。
- **工单列表的筛选参数全部静默失效** — `ListTicketsRequest` 声明了 `assigneeId`、`requesterId`、`categoryId`、`parentTicketId`、`templateId`、`type`，但 handler 一个都没下传；仓储层按 `filters["assignee_id"]` 等键位取条件，漏键不报错只返回租户全量。实测 `GET /api/v1/tickets?assigneeId=33` 返回 41 条、受理人全是别人，而 `src/app/(main)/tickets/page.tsx` 正在发该参数——筛选界面展示的是无关数据。现在六个参数按指针判空后显式进 `filters`（`assigneeId=0` 视为「未指定」不下传，避免把合法零值当筛选）。回归：`handler_test.go` 断言每个已声明的过滤键都真实生效、键位与仓储契约一致、未传的过滤条件不得出现在下传 map 里（防止只修一个键又让其余回到静默失效）；隔离栈实测 `assigneeId`/`requesterId+type`/`categoryId` 三组过滤经前端代理返回正确子集，且他人工单被指派给本人后可见范围从 3 条正确扩到 4 条。
- **列表接口省略分页参数时 `totalPages` 溢出成天文数字** — `GET /api/v1/tickets`（以及任何把 `ShouldBindQuery` 结果直接下传的分页接口）在客户端不带 `page`/`pageSize` 时拿到的是 Go 零值：`common.NewPaginationResponse` 用 `math.Ceil(float64(total)/float64(pageSize))` 算总页数，除零得到 `+Inf`，而 `int(+Inf)` 在 amd64 上是 int64 最小值。2026-10-02 真实 API 实测响应为 `{"page":0,"pageSize":0,"total":3,"totalPages":-9223372036854775808}`，前端分页器拿到的是完全虚构的总页数；同时零值也下传给仓储，`repository/base` 算出负偏移。修法两层：`common.NewPaginationResponse` 先用既有 `ValidatePagination` 归一化（并夹紧负 total），任何调用方都无法再产出溢出信封；`handlers/ticket` 在调用 service 前归一化，保证「信封声明的页大小」与「真实查询的 LIMIT」一致。非法输入侧行为不变：`?page=abc` 仍是绑定错误 → 400/1001，不会被静默当成默认页。回归：`common/pagination_test.go`（0/0、负 page、超上限 pageSize、负 total、信封键完整性）+ `handlers/ticket/handler_test.go`（不带参数 / `page=-3&pageSize=0` / 显式分页三条旅程，并断言下传给仓储的 page/pageSize 与响应一致）；注入回归实测修复前 `totalPages` 为 `-9223372036854775808`。
- **创建/重置用户密码不合规则返回 500「操作失败」** — `POST /api/v1/users` 与 `PUT /api/v1/users/:id/password` 把 service 的密码策略错误（缺大写、长度不足等）统一当成内部错误：HTTP 500 + 业务码 5001 + 不含原因的「操作失败」，真实原因只在服务端日志里。调用方无法区分是自己给的密码不合规还是后端故障，自动化夹具 provisioning 只能靠翻日志定位。现在策略错误带上 `common.ErrCodeValidation` 分类，handler 映射为 400 + 业务码 1002 + 具体原因；未分类错误仍按内部错误处理，不透出驱动层细节。回归：`handlers/user/handler_test.go` 两个用例断言 status/code/message 三元组，`service/user_service_test.go` 断言分类在源头生效。
- **access token 一过期就登不出，7 天 refresh cookie 把会话「复活」** — `POST /api/v1/auth/logout` 挂在 `AuthMiddleware` 后面，而 access token 只有 15 分钟：用户放一会儿页面再点登出，请求在中间件里就 401，处理凭证的 handler 从未执行，两类 cookie 一个都没清。留下的 `refresh_token`（7 天、httpOnly）随后被自动续签，登出等于没发生。现在登出刻意不挂鉴权中间件：先无条件清 cookie，再尽力吊销请求里携带的 access/refresh token，吊销存储故障时返回 5003（5003 业务码）而不是伪装成功——cookie 已清、服务端未吊销是必须让运维看见的状态。无凭证重复登出幂等返回 200。回归：`router/auth_handler_routes_test.go` 用「过期 access token + 有效 refresh cookie」断言 200、两个 cookie 的 `Max-Age<0`、随后用同一枚 refresh cookie 续签必须 401。
- **登出从不吊销 refresh token** — 旧实现只调 `RevokeAccessToken`，浏览器 cookie 清掉之后服务端那枚 7 天凭证仍然有效；任何拿到它的人（或那份遗留 cookie）都能继续换新 access token。现在登出同时吊销两类凭证。
- **会话凭证的名称与生命周期只剩一处定义** — `900`/`604800`/`"access_token"` 这些字面量原先分散在 `handlers/common/handler.go`、`handlers/auth/handler.go`、`middleware/auth.go` 与 service 的签发调用里，改一处就让浏览器 cookie 窗口和服务端校验窗口悄悄分叉。现在统一由 `middleware.AccessTokenCookie`/`RefreshTokenCookie`/`AccessTokenTTL`/`RefreshTokenTTL` 派生，Set-Cookie 的 `Max-Age`、JWT 过期时间和契约测试断言都从同一处取。
- **前端有六处各自推断「我是否已登录」，结论与后端经常相反** — 除会话端点外，登录态还由这些本地信号决定：JS 写入的 `auth-token=1` 标记 cookie（只检查它非空，等于把「曾经登录成功过」当身份）、落地页与服务请求客户端扫 `document.cookie`、`localStorage` 里的 Zustand 状态配 `_hasConfirmedSession` 回放、Next 边缘中间件按 JWT 字符串形状放行（实测把已过期的 `header.eyJ1IjoiMSJ9.sig` 当有效会话，并把已在登录页的用户反复弹走）、API 代理自造 `{code:2001}` 的 401、以及 `/auth/me` + `/auth/tenants` 双探活失败时「保留会话」。现在六处全部删除，一律读 `GET /api/v1/auth/session`：`middleware.ts` 只做遗留菜单 307 重定向，代理不再伪造认证结论，`(main)` 布局在无法确认会话时渲染可重试的「暂时无法确认登录状态」而不是静默跳转。同步删除 `lib/auth/jwt-decoder.ts`、`components/layout/RouteGuard.tsx`、`components/providers/Providers.tsx`（守卫挂在永不为真的 `AuthService.getToken()` 上）、废弃客户端 `authApi.refreshToken()/validateToken()` 与 legacy `/api/v1/refresh-token` 前端调用、按浏览器时钟的 10 分钟定时续签，以及登录页「记住我」（凭证在 httpOnly cookie 里，勾选框从未改变服务端窗口）。SSO 回调与第三方登录不再从响应体取令牌，改为回读会话端点，读不到就如实报错。细节与遗留见 `plans/product-feature-completeness-plan-2026-10-01.md` §10。
- **站内通知 WebSocket 从未连上，旧 localStorage 能复活伪登录态** — 通知页与 Header 用 `user?.id && token` 给连接设门禁，而 store 里那个 `token` 字段自凭证 cookie 化后就恒为 `undefined`（真凭证在 httpOnly cookie，JS 读不到），于是这一页的重连、未读数推送全部静默失效；现按 `user?.id` 设门禁，`notificationWS.connect(userId)` 也不再接收无用的 token 形参。同时删掉 store 的 `token` 字段，并把 zustand persist 的默认浅合并换成显式 `merge`：默认合并会让旧版本写进 `localStorage['auth-storage']` 的 `user`/`isAuthenticated`/`token` 原样复活，等于用一份过期的用户对象继续冒充会话；现在 hydration 只采纳 `currentTenant`，身份一律由会话端点重新给出。验证：`cd itsm-frontend && npm run test:unit`（200 套件 / 3418 通过 / 13 跳过）、`npm run type-check`、`npm run lint:antd`、`npm run lint:check`（0 error）；新增 `src/lib/api/__tests__/session-api.test.ts`（三态映射、并行续签只发一次请求、登出 keepalive）、`auth-store.test.ts` hydration 回归（旧 payload 注入后 `user=null`、状态里无 `token` 键、仅租户恢复），`auth-service.test.ts` 断言登录成功仍须会话端点确认、会话查询失败时不谎报成功。
- **部门默认流程绑定的 key 收敛为唯一目录，未提供的流程不再假成功** — 「部门类型 → 流程 key」此前有三份互不同步的清单：真实入口 `POST /api/v1/departments/:id/init-processes`（`handlers/bpmn/process_trigger.go:63`）走 `service/bpmn_process_binding_service.go` 里的私有 switch；`service/scenario/scenarios.go` 另有一份；`service/department_process_service.go`（零调用方的重复实现）还有第三份 scenario→业务类型映射。三份清单引用的 11 个 key 里只有 3 个在 `service/bpmn/*.bpmn` 有模板载体，其余 8 个（`change_emergency_flow`、`release_test_flow`、`change_requirement_flow`、`expense/budget/procurement/leave/recruitment_approval_flow`）从未存在，而绑定循环对缺失是 `continue`：财务/HR 部门初始化一条绑定也不建、仍返回成功并提示「部门流程模板已初始化」。现在清单只保留 `service/scenario` 一份（补齐 `businessType`/`businessSubType`/`category`），可用性由 `go:embed bpmn/*.bpmn` 实测判定（新增 `service.BuiltinProcessTemplateKeys()`，不手抄第二份可用列表），私有 switch 与重复实现已删除；一条都没建成且没有命中已有绑定时返回明确错误并列出缺哪个 key，重复初始化仍幂等。守卫：`service/bpmn_department_bindings_guard_test.go` 对 ready（3）/unready（8）两个集合做双向基线，补模板或删模板都必须改基线；同文件用 enttest 覆盖「只建有载体的绑定」「无载体显式失败」「有载体但未部署显式失败」「未知部门类型」。⚠️ 未拍板：财务/HR 这些审批场景是补独立 BPMN 模板，还是复用已部署的通用 `service_request_flow`，属产品决策，本批只做诚实失败不代作决定。
- **初始化校验的悬空绑定报错不可读** — `pkg/seeder/initialization_adapter.go` 的 `verifyWorkflowTemplates` 把查询错误与「定义不存在」合进一个分支，用 `err=%w` 格式化 nil 错误，运维看到的是 `verify process definition expense_approval_flow: exists=false err=%!w(<nil>)`，既没有租户也没有绑定归属。现拆成两条：查询失败带 tenant 与 `businessType/businessSubType` 上下文并保留原始错误；定义缺失直接说明「该 key 必须有 `service/bpmn/*.bpmn` 载体并由 BPMNTemplateService 部署，否则 workflow-core 组件回滚」。
- **全新部署被磁盘迁移中断** — 日期化迁移在既有安装走「收养不执行」（账本 checksum 为空、从不运行），只有清卷全新安装才真正执行，所以这批缺陷一直只在私有化首装暴露：迁移在 seed 之前运行、引用 Ent 从未生成的对象或已退役的死表、并硬编码 `tenant_id = 1`。2026-10-01 从空卷实测逐个收敛 9 个脚本，报错位置与归属证据写在各文件注释里：`20260501_rbac_endpoint_acls`、`20260616_security_role_permissions`、`20260620_create_marketplace`、`20260628_add_connector_menu`、`20260830_ticket_types_menu_reparenting` 退役为 `SELECT 1;`（保留原版本名以解析既有账本；RBAC 授权、市场目录、菜单形态的唯一所有者是 `internal/authz` + `pkg/seeder` + `pkg/menubaseline`，SQL 里的第二套定义从未生效）；`add_missing_indexes` 与 `add_missing_indexes_batch2` 去掉给已退役死表（`workflow_instances`/`workflow_tasks`/`workflow_versions`/`workflows`/`cab_members`）建索引的语句——版本号按字典序排在 `2026*` 之后，全新安装里先建后 DROP 必报 42P01 且逐语句首错即停；`20260825_email_intake_hardening` 改为在角色可承受时自行 `CREATE EXTENSION btree_gist`，仍装不上才报可操作错误；`20260620_process_routing_enhancement` 删掉写入 13 条流程绑定的 Step 7——其中 4 个 flow key 在 BPMN 模板与种子清单里都没有载体，会让 `workflow-core` 组件模板校验失败并回滚整段基线。验证：空卷全新部署 init 退出 0、52 个迁移入账、18 个流程定义、13 条 Go 侧流程绑定零悬空 key，七容器健康并完成 SPA 登录与四域浏览器旅程。
- **变更「待审批」读写词表分裂（C2/N1）** — 状态枚举化后写入值改成 `pending`，但列表与日历过滤仍按枚举外的 `submitted` 匹配，存量待审批变更与新写入的待审批变更被切成两个互不可见的集合；同时迁移表仍以 `submitted` 为词表，只归一源值会把 `draft -> pending`（提交审批）误判为非法迁移。现在按待审批过滤同时命中两种历史写法、迁移表源值与目标值统一归一、写入只落枚举内的 `pending`，并用「还原枚举校验生效前写入的存量行」回归测试锁死读取映射、过滤与统计三处行为。
- **未接流程的租户工单/变更审批被误判为冲突（C3/N2）** — BPMN 审批桥接原来只有「完成 / 没完成」两态，「工单从未绑定流程」与「已绑定流程却没有可操作待办」返回同一个错误：未配置流程的租户审批全线 4090 不可用，而简单放行又会让已交流程裁决的对象绕过流程。现按三态收敛——未绑定 → 变更/工单回退审批链继续可用；已绑定但无待办 → 4090 中止，提示能区分成因；桥接未接线 → 5003 依赖不可用（不再伪装成用户参数错误）。服务请求与发布维持严格口径（审批必须由流程待办裁决），`TestScenario10_ServiceCatalogFullLoop` 的三级审批闭环已改为按生产装配接真实 BPMN 串行待办，7 个存量失败子用例全部转绿。
- **仪表盘「待审批变更」计数恒为 0** — 统计查询按 `"submitted"` 过滤 Change 状态，而状态枚举化后该值不在合法词表内（`draft/pending/approved/...`），库里不可能存在这种行，待审批变更永远统计不到。查询条件与测试 fixture 一并改为 `pending`。
- **SLA 监控在依赖未注入时直接崩溃进程** — `SLAMonitorService` 的 `sla_states` store 由启动装配事后 `SetSLAStore()` 注入，漏注入时读路径会在 nil receiver 上解引用，panic 终止 SLA watcher goroutine。`CheckSLAViolations` / `GetDashboardMetrics` 现在按 fail-closed 返回 5003（依赖不可用），不再伪装成空检查结果，也不会带崩整个进程；含「去掉守卫即 panic」的回归测试。
- **RBAC 预检生成物与路由不同步** — `middleware/rbac_precheck_gen.go` 落后于实际路由：仍登记已删除的 `/api/v1/projects*`，缺少 `/api/v1/admin/skills`、`/api/v1/admin/skills/:code`、`/api/v1/menus/export`，`TestPrecheckMapIsFresh` 因此失败。已用 `go run ./cmd/authz-gen` 重新生成。
- **测试首次真实执行后的存量失败收敛（批次 0）** — 状态枚举化后 16 个测试函数的失败按根因分五类修复：① 9 处断言仍拿 `string` 比枚举类型（`release.Status("draft")` ≠ `"draft"`），改用生成的枚举常量，生产值本身一直是对的；② SLA 场景与单测 fixture 只写工单内嵌的 `sla_*_deadline`，而 Phase 3 读路径以 `sla_states` 为权威源，导致「查不到违规」假通过，fixture 已补 `sla_states` 并把断言收紧到具体条数与合规率；③ 场景测试用 `NewIncidentService` 默认装配（未开 outbox），落到 fire-and-forget goroutine 分支，后台写库与下一步事务争抢同一块共享内存 sqlite，偶发 `database table is locked: incidents`——改为与生产装配一致的 `EnableWorkflowOutbox()+EnableRulesOutbox()`，抖动消除且不再依赖重试；④ 测试构造 `SLAMonitorService` 时未注入 store，现与启动装配对齐。剩余 2 类失败（change `submitted` 词表、无 BPMN 绑定时的 4090）是待拍板的契约决策，不在本批次动。
- **恢复后端测试守卫的真实执行（F8）** — `service` 包测试自状态枚举化（`662294e0`）起整包编译失败，SLA／变更／知识的全部用例实际空转，门禁显示"绿"却没有任何保护。补齐 `createViolation` 缺失的 `slaDefinitionID` 参数、修正 B2 后 `standard_change` 仍读取已删除 `affected_cis` 字段的断言、补齐 rbac `mockMenuService` 缺失的接口方法、`pq.ErrorCode` 改用 `pqerror.CheckViolation`，并删除零引用的 `toTicketResponse`。同时修复 `backend-ci` Lint job：28 个文件不符合 gofumpt 会连带阻塞 Build 与 Test，远端 `main` 自 09-27 起即为红。⚠️ 此前 staticcheck 报的 10 处"未使用代码"有 9 处是编译失败导致测试文件未加载的假象，修好后真实未使用仅 1 处，避免误删审批链与 SLA 测试覆盖。测试首次真实执行暴露的 5 个存量失败与 1 个 panic（`dashboard_domain_metrics_test.go` 用枚举外的 `"submitted"` 建 Change）已记入收敛计划，另行处理。
- 发布审批的跨租户、已停用或不存在的审批人现在返回 403（业务码 2003），不再误报 5001；拒绝时不修改发布、流程任务或审计记录。
- **修复 BPMN 流程审计操作人恒为空** — 流程启动与任务完成审计此前从 `ctx.Value("user")` 取操作人，而生产链路从未写入该 key，审计记录 operator 恒为 0/空名；现统一改走 `bpmn.BPMNUserIDContextKey` 并在事务内解析用户名，与审批决策审计同一模式（含回归测试）
- **修复市场页恒为空** — 生产初始化新增 `marketplace-items` 组件：写入 5 条内置市场条目（连接器/技能/插件，状态 published、纳入版本账本 checksum 与逐项验证），存量环境可前滚补齐；此前生产 DAG 无该组件、列表按 published 过滤后恒为空。市场列表/详情同步改为返回 DTO（`MarketplaceItemResponse`，camelCase），不再直接序列化 Ent 模型
- 修复工单/事件 SLA 暂停与恢复的假成功：工单侧此前直接返回"SLA已暂停"却完全不落库，事件侧返回"尚未接入"，而底层 `SLAMonitorService.PauseSLA/ResumeSLA` 早已是真实现；现两域统一接线，暂停真实落库并记录原因、恢复顺延截止时间，重复暂停/未暂停恢复按 409 冲突、跨租户按 404 拒绝
- 知识/问题评论占位接口不再伪装成功：评论能力尚未落地（无存储模型），统一显式返回 5003 unready；此前知识侧返回假的评论对象、问题侧用空列表假装"暂无评论"
- 仪表盘事件/变更指标去掉模拟值：`AvgResolutionTime=240 分钟`、`SuccessRate=95.5%` 写死的假数据改为真实领域统计（事件解决时长均值、变更 completed/(completed+failed)），且指标改从事件/变更域取数而非工单表；无数据时如实为 0
- 修复 SLA 定义整段跳过导致存量环境前滚被永久卡死：改为按租户按条目 reconcile 补齐清单缺失条目、不覆盖已有条目（2026-09-25 演练库前滚实证：verify 要求 `Incident-P0-紧急`，而"已有任一行即跳过"一直拒绝创建，组件永远过不了自校验）
- 新增数据修复迁移停用两代旧命名的 SLA 定义（`SLA-P0-*`、`SLA-服务请求`、`SLA-变更`、`[Template] *`）：只按封闭名称清单置 `is_active=false`，不删行不改名，历史工单/SLA 违规引用继续可解析且保留原 SLA 条款，新工单不再可选；幂等可回滚，`NOTICE` 如实报告本次翻转行数
- 修复开通过产品基线的租户无法删除：`system-baseline-*` 基线归属账号只是模板行审计归属、不可登录，不再计入「租户下还有用户」删除守卫；真实用户仍阻止删除
- 修复新建租户接口 500：产品基线安装的 outbox 命令归属目标租户，而创建请求运行在操作者租户上下文，被租户写护栏按跨租户插入拦截（浏览器回归实测发现）；跨租户的 bootstrap/seed 类命令入队改走显式 system context 并留审计，测试 fixture 同步注册生产安全拦截器，杜绝只测裸客户端漏掉真实入口
- teams 与工单类型改为按租户按条目 reconcile：存量租户的部分集合可前滚补齐（此前"任一行存在即整段跳过"会让缺口永远补不上），不覆盖客户改名
- 租户创建不再留下"已建但无基线"的孤儿租户：`POST /api/v1/tenants` 在同一事务内投递 `tenant.bootstrap.install` outbox 命令，入队失败连同租户一起回滚；worker 消费后按命令租户重新加载校验并安装产品基线（复用平台初始化同一套组件，不复制第二套实现），新增 `GET /api/v1/tenants/:id/initialization` 只读接口如实报告逐组件验证结果与命令状态
- 修复生产环境数据库迁移失败问题：部分迁移脚本内嵌事务控制语句导致整批迁移中止
- 修复初始化失败被静默吞掉：迁移发现、账本调和与执行任一环节失败都会中止启动；组件写入、事务内校验、租约 fencing 与成功账本现在同一次提交，失败整体回滚，不再残留半成品基线
- 修复独立 `initialize -action verify` 依赖上一次 Apply 内存基线的问题：验证基线改为从代码内产品清单确定性派生，缺失菜单/权限/角色授权、跨租户授权和悬空授权均可被检出
- 修复审批组未随生产初始化创建：`identity-rbac` 现在写入审批组并纳入验证，JSON 种子配置中的 `groups` 段不再被合并逻辑丢弃
- 修复 SLA 告警规则按硬编码名称匹配：改为按租户实际 SLA 定义派生，引用缺失直接失败（此前静默跳过且日志谎报创建数量）
- 修复部门 `parent_code` 未落库导致组织树平铺：现在按 code 补齐父子关系，可前滚修复已安装环境且不覆盖客户改名
- 修复多租户基线种子撞全局唯一键：ticket_categories.code 与 tags.code 从全局 `Unique()` 改为 `index.Fields("tenant_id","code").Unique()`，与 AGENTS.md「业务唯一键默认按 tenant 组合唯一」一致；迁移脚本 `20260923_tenant_scope_baseline_unique_keys.sql` 先放旧约束 → 显式检测 `(tenant_id, code)` 重复 → 重建组合索引；迁移失败立刻停止，绝不留下半索引状态
- 修复租户基线种子运行时引用 `default` 租户硬编码：`Seeder.baselineTenant(ctx)` 抽象替出，平台初始化时仍落到 default 租户，按租户 provisioning 时落到目标租户；新增 `system-baseline-<id>` 不可登录审计账号（bcrypt 永不可通过的 `!` 哈希）让 template 表的 `user_id` 引用有主；避免「customer data 假扮 product template」的复制陷阱
- 修复 `SeedAll` 中新增种子函数签名升级后未对应 `attempt` 包装的 4 行旧调用：CI/types/standard-changes/ticket-tags 与原 ticket-types 一致用 `attempt("...", fn)` 包裹，失败仅 sugar.Errorw 不阻断，与 SeedAll 容错继续哲学一致
- 修复服务目录子项父目录缺失时被静默跳过（约 12/15 子项丢失）：现在初始化失败并给出具体条目，验证也逐条检查子项存在性
- 修复菜单初始化每次运行都重置运维设置的隐藏/停用状态
- 修复种子配置合并遗漏 `groups` 段：JSON 种子配置中的审批组定义此前被静默丢弃，现按整段替换内置默认并保留未覆盖段（含回归测试）
- 修复生产部署登录失败：docker-compose 默认 RLS 模式从 `enforce` 改为 `off`，避免未携带租户上下文的公共路由（登录/注册）返回 401
- **修复通知重复投递** — `notification_delivery_command_handler` 在 connector `Send` 成功后，若更新 `NotificationDelivery` 状态为 `sent` 失败，不再返回 error 触发 worker 重试（消息已送达），改为记录告警并返回 nil，避免同一通知被重复发送
- **修复 BPMN 任务完成审计丢失** — `CustomProcessEngine.CompleteTask` 现在将任务完成审计（`ProcessAuditLog`）写入与任务状态更新同一事务内，审计写入失败则整体回滚，杜绝"任务已完成但无审计记录"的不一致状态
- **修复单条连接器配置解密失败导致全部连接器不可用** — `PersistentConfigStore.LoadAll` 逐条容错：单条解密或反序列化失败跳过并记录 ID，不影响其余连接器正常加载；新增 `LoadAllWithFailures` 返回失败 ID 列表供运维排查
- **修复批量关闭/更新工单部分失败中断** — `BatchCloseTickets` 与 `BatchUpdatePriority` 改为逐条容错，单条状态机校验失败不阻塞其余工单，结果通过 `BatchResult` 返回成功数与失败 ID 列表
- **优化工单分析查询** — `GetTicketAnalytics` 使用 `Select` 仅加载 status/priority/created_at/updated_at 四列，合并二次查询为单次遍历，减少内存占用和 DB IO
- **工单前端契约收敛为单一来源** — `Ticket`/状态/优先级/类型及请求体现在只在 `itsm-frontend/src/lib/api/ticket-api.ts` 声明（逐字段对齐后端 `dto/ticket_dto.go`），删掉 `lib/api/types.ts`、`api-config.ts`、`types/ticket.ts`、`lib/services/ticket-service.ts` 四处并行定义与四套各自手写的状态/优先级配色表
- 修复工单列表、看板与表格的状态徽标：映射键写成 camelCase（`inProgress`）而后端返回 snake_case（`in_progress`），"处理中/已分配/已批准/已拒绝"长期显示原始枚举值；现统一取自 `constants/taxonomy`
- 修复看板泳道与筛选面板提供后端不可能的取值：`pending_approval` 不是工单状态（该泳道恒为空），状态/优先级/类型选项现由后端 10 值状态机与 `oneof` 白名单派生
- 修复工单详情 SLA 卡片读取后端从不返回的 `slaName`/`isBreached`/`responseTimeRemaining`，改为 `slaDefinitionName`、`isResponseBreached`、`responseTimeLeftMinutes`；无 SLA 定义时不再渲染空卡片
- 清理请求体里后端不接收的字段：分配不再发 `comment`、升级不再发 `level`、解决不再双写 `resolutionCode`；创建工单不再由客户端提交 `requesterId`（身份只来自认证上下文）；分页参数统一为 `pageSize`，去掉 `size`/`pageSize` 双轨与 `response.size ?? response.pageSize` 猜测
- 修复仪表盘"最近工单"伪造 `category`/`tags`/`source`/`dueTime` 并把 `tenantId` 兜底成 1；批量导出不再发送后端不支持的 `ticketIds`（后端导出只按状态/优先级过滤，此前会把"导出 N 条选中"实为"导出全量"）
- 修复 BPMN 流程指标的时间范围参数静默失效：前端发送 `time_range` 而后端 `handlers/bpmn/monitoring.go` 读取 `timeRange`，导致"近 7 天/近 30 天"查询全部退回默认 24h；参数名与契约测试同步更正
- 修复工单类型弹窗的流程/分类下拉恒为空：依赖列表按 `definitions`、`categories` 等别名取值，而对应 handler 实际返回标准外壳 `items`（`common.NewListResponse`、`ListCategories`）；改按 `items` 读取后审批节点预览与"打开流程设计器"入口恢复

### Changed

- **删除 8 个「声明的形状与线上响应不一致」的死列表信封 DTO** — `dto` 包里 `PermissionListResponse`、`KnowledgeArticleVersionListResponse`、`KEDBListResponse`、`ServiceRequestListResponse`、`StandardChangeListResponse`、`ProjectListResponse`、`ConfigurationItemListResponse`、`SurveyListResponse` 在 itsm-backend 内的引用实测只有定义自身与棘轮基线字符串（逐个 `git grep -w <Type> HEAD -- ':!itsm-backend/dto'`，无任何 handler/service/mapper/测试构造点），前端 `kedb-api.ts`/`service-request-api.ts`/`standard-change-api.ts` 的同名标识是独立声明的 TS 接口、不引用 Go 类型，因此删除不改变任何线上响应。它们同时是**假契约**：被删的 `dto.ServiceRequestListResponse` 声明 `{items,total,page,size}`，而生产 `handlers/service_request/handler.go:283,369` 实际返回 `{items,total,page,pageSize,totalPages}`——留着只会误导下一个改契约的人。信封棘轮的三条基线同批从 30/11/9 收敛到 **25/8/7**（合计 40 条 / 33 个结构体；守卫是双向的，删了结构体却忘记收基线同样会失败。E4-6 按文档改判分页键规则后为 **25/5/7 = 37 条 / 30 个结构体**）。实测：`go build ./...` 通过、`go test ./tests/contract/ ./dto/... -count=1` 全绿、`gofumpt -l dto/ tests/contract/` 空输出、staticcheck 无命中、全量 `go test ./...` 退出码 0 / 0 条 FAIL。**顺带实测登记一条不在本批修的契约漂移**：服务请求列表同时存在三套形状（生产五元组 / 已删死结构体的 `size` 版 / 前端 `ServiceRequestListResponse{requests,total,page,size}`），前端靠 `service-request-api.ts:194` 的 `raw.requests || (raw as any).items || []` 与 `raw.size || (raw as any).pageSize || requests.length` 猜测——属 AGENTS 明令禁止的多字段 fallback，且 `total` 缺失时静默退化为「当前页长度」；修复需同时改 `src/app/(main)/service-requests/page.tsx`，该文件正被另一会话修改，为避免覆盖在途工作而延后（台账见 `plans/edge-feature-stability-audit-2026-10-02.md`「E4 收口进度」行 E4-4）。
- **⚠️ 列表响应收敛为唯一信封 `{items,total,page,pageSize,totalPages}`** — `common.ListResponse.MarshalJSON` 原来会用 reflect 按元素类型追加领域名别名（`items` 与 `tickets`/`incidents`/`changes`/`articles` 同时返回同一份数组），再把同一组分页事实嵌套成第二个 `pagination` 对象；同一接口因此有两套真相，前端 `response.items ?? response.tickets ?? []` 这类被 AGENTS.md 禁止的多字段 fallback 反而被"合法化"。现在键集合固定为五个平铺键，别名表（`domainAliases`/`singularAliases`/`inferDomainAlias`/`lowerFirst`）与嵌套 `pagination` 整段删除。写入侧跟着收敛：`ListChanges` 从 `gin.H{"changes":…}` 改走 `SuccessWithPagination`，知识文章改用标准生产者（补齐此前缺失的 `totalPages`），活跃告警补齐 `totalPages`，`TicketTypeListResponse.Types`、`IncidentListResponse` 的 `incidents`/`pagination` 与 `KnowledgeArticleListResponse` 等别名字段、重复 DTO 一并删除。请求侧同一批：事件列表与活跃告警改读 `pageSize`（原 `size`），前端 `IncidentAPI` 里 `pageSize→size` 的翻译器和 BPMN 两处 `pagination.total` 读取同步移除；消费侧 20 余处 `response.tickets ?? …` 改为单键读取。**升级影响**：走标准生产者的列表接口其领域名顶层键与 `data.pagination` 不再返回，外部集成方必须改读 `data.items`，见 [UPGRADE.md](./UPGRADE.md) §1.8。守卫：`tests/contract/list_envelope_ratchet_test.go` 用 go/ast 扫 `dto/*.go`（类型名含 List 且带 total 即判为信封）做三条双向棘轮——领域名集合键（基线 30）、标准分页键不完整（基线 11）、`size` 等分页别名（基线 9），新增违规与基线过期都失败；`handlers/incident/list_contract_test.go`、`handlers/knowledge/list_envelope_test.go` 断言真实路由的精确键集合，并锁死 `size` 不再是事件列表的契约参数。未收敛部分按基线登记为债务：`roles`/`menus`/`tenants`/`catalogs`/`allocations`/`notifications` 等 30 个手写 DTO 仍返回领域名集合键，CMDB/服务目录/通知 9 个信封仍用 `size`（**2026-10-03 更新**：删除 8 个零引用死 DTO 后三条基线降为 **25 / 8 / 7**，即 40 条 / 33 个结构体；同批 E4-6 把分页键规则改判为「只算部分分页」后为 **25 / 5 / 7**，即 37 条 / 30 个结构体。见下方 E4 条目与 `docs/testing/static-analysis-gates.md` §5.5）
- **`api-contract-check` 由红转绿** — 该 workflow 唯一失败项是 swagger 新鲜度（`git diff --exit-code` 三件套），根因是 `itsm-backend/docs/*` 落后于代码。重新生成后定义数从 344 降到 174——移除的 171 个（含 162 个 `ent.*` 模型和 9 个 Ent 枚举/schema 类型）全是 handler 注解直接序列化泄漏出来的持久化模型，新增 1 个 `dto.IncidentListResponse`；路径数保持 159，但集合换了 6 条：删掉从未注册的 `/api/v1/projects`、`/api/v1/projects/{id}` 与实际不存在别名 `/api/v1/departments/{id}`，补上 5 个真实注册但从未文档化的部门路由（`/api/v1/org/departments`、`/api/v1/org/departments/tree`、`/api/v1/org/departments/{id}`，加上既有的 `/api/v1/departments`、`/api/v1/departments/tree`）。悬空 `$ref` 0（仍有 6 个 `ent.*` 定义由市场/安装路由泄漏，属既有债务）。前端 `api-contract.test.ts` 的两处存量不匹配也修绿：菜单列表改为静态 base 常量再分支拼 query（字符串拼接路径无法被契约扫描解析）；`MspApi.getAllocationHistory` 指向的 `GET /api/v1/msp/allocations/history` 从未被 `router/msp_routes.go` 注册，属悬空接口而非命名漂移，因此按仓库既有的能力开关通道置 `PRODUCT_CAPABILITIES.mspAllocationHistory = false` 并在 `DISABLED_API_CONTRACTS` 登记原因、MSP 页面隐藏「分配历史」Tab，而不是补一个假接口
- `config/seed/default.json` 不再整段替换服务目录清单，内置 Go 清单成为唯一权威来源（原 8 条与内置子项引用冲突的目录已移除，"权限变更"并入账户申请子项）
- 初始化组件 checksum 现在覆盖代码内产品清单（权限目录、菜单规格、内置角色/审批组/角色授权、工单类型、BPMN 模板内容），修改这些定义会被版本账本察觉，无需手工升版
- 生产部署配置：`RLS_MODE` 默认值调整为 `off`（与后端安全默认对齐）。已配置 `.env.prod` 的部署不受影响
- **SLA 信息字段统一 camelCase** — `GetTicketSLAInfo` 返回的 map key 从 `snake_case` 迁移到 `camelCase`（`ticket_id`→`ticketId`、`sla_response_deadline`→`responseDeadline` 等），`resolution_deadline_breached` 拆分为 `isResponseBreached` + `isResolutionBreached`；前端 `TicketDetail` 对空 deadline 做 null-safe 兜底
- **前端零值安全修复** — SLA 定义、分配规则、自动化规则页面的 `||` 改为 `??`，允许 `0` 作为合法值（如响应时间 0 分钟 = 立即、优先级 0、可用性 0%）
- **CIList 双信封违规收敛** — 移除 `Array.isArray` + `{items}` 双形式兜底，收敛到 API 契约声明的单一 `CIType[]` 形式
- **ChangeDetail 审批模态框状态重置** — 打开/关闭审批和驳回弹窗时重置 `approvalComment`，避免残留上次输入
- **服务请求优先级显示修复** — 移除硬编码 `priority: '中'`，改为读取后端返回值

---

## [1.6.10] - 2026-09-20

### Added

- **BPMN 定时事件** — 完整实现 Timer Event 基础设施：调度引擎、BPMN 引擎集成、设计器属性面板、任务到期计时（dueDate）和定时启动事件（Timer Start Event），附带 Prometheus 监控指标和任务超时扫描
- **审批链运行时能力** — 新增加签（allowAddApprover）、委派（allowDelegate）、拒绝策略配置和动态层级适配，均通过审批链配置驱动、运行时生效
- **运维命令批处理** — 支持 Outbox 命令的批量重放与取消
- **钉钉/企微入站回调** — 入站消息回调 API 及幂等去重
- **BPMN 模板重载 API** — `POST /api/v1/bpmn/workflow-templates/:key/reload`，从已发布版本创建新部署，版本号自动递增
- **AI 工作流模板治理** — 租户隔离的模板目录 API 和 `/admin/workflows` 管理 UI，支持草稿、版本递增、发布前 Lint、发布与停用
- **CMDB ontology 自描述端点** — `GET /api/v1/cmdb/ontology` 返回租户 CMDB 的机器可读描述（CI 类型、关系词汇、AI 工具定义），AI Agent 可运行时发现 CMDB 契约
- **CI 可读编号** — 每个新建配置项获得 `CI-YYYYMM-NNNNNN` 唯一编号，支持按编号精确查询
- **一键演示数据集** — `make dev-seed-demo` 生成 8 个事件、2 个问题、3 个变更和 5 篇知识文章的演示数据
- **10 个管理页面使用指南** — 新增 `UsageGuideCard` 组件，为管理页面提供基于实际代码逻辑的操作指引
- **Swagger 文档重建 + CI 新鲜度门禁** — 重新生成 158 个 OpenAPI 路径，CI 自动检测文档漂移

### Changed

- **BREAKING: RBAC 授权平面收敛** — 统一权限码、补全预检映射、实现路由→权限码自动生成，关闭未授权写入漏洞；管理员/技术员角色数据库权限空集修复
- **BREAKING: 审批架构迁移** — 统一消费 BPMN 任务，消除双审批实例，ApprovalRecord 标记废弃，清理冗余 DTO
- **BREAKING: API 响应字段统一 camelCase** — Ticket/Incident/SLA/BPMN 响应不再暴露 snake_case 字段，外部集成需同步迁移
- **BREAKING: SLA/BPMN 查询参数 camelCase** — `ticketType`/`customerTier`/`startDate`/`endDate`/`timeRange` 替代原有 snake_case 参数
- **BREAKING: SLA/BPMN 监控端点标准信封** — 统一返回 `{code, message, data}` 格式
- **BREAKING: CI 关系类型受控词汇表** — 关系类型统一由 ent schema 管理，未知类型返回 400 而非静默持久化
- **BREAKING: Ant Design `direction` → `orientation`** — `Space` 组件统一使用 v6 API
- **服务请求 BPMN 集成** — 服务请求审批走 Outbox 事务投递
- **工单/事件写入路径加固** — 乐观锁、NULL 时间戳处理、错误语义修正、升级操作行级安全校验
- **变更/发布/DataScope 写入路径** — 行级安全校验补全
- **流程设计器修复** — 属性写入丢失、版本解析、菜单 404、模板完整性校验、Lint 审批节点校验
- **CSRF 全链路修复** — Cookie 安全属性与 Token 传递对齐
- **通知偏好** — 接入按事件类型偏好 API，修复虚假 API 处理和接收者广播排除
- **查询参数契约收尾** — 统一 camelCase，关闭残留 snake_case 不一致
- **角色域修复** — 移除虚假控件，API 契约对齐 camelCase
- **生产部署加固** — 数据库迁移与索引对齐、开发/生产隔离、Schema 变更脱敏、迁移脚本检查、`VERSION` 不可变要求、诊断端口绑定 loopback
- **基础设施加固** — AI vector/telemetry schema 版本化迁移、HTTP 响应缓存规范化、租户级 Raw SQL 执行器覆盖、Incident/CI 编号使用结构化审计事务
- **前端 UX 审计** — 工单列表列宽、创建按钮尺寸、折叠侧边栏 flyout、看板卡片溢出、工单分析页空白、管理后台仪表盘虚假指标等 20+ 项修复

### Fixed

- **AI 助手知识库无命中时丢失产品自知上下文** — RAG 降级链路现在注入 AI-Native ITSM 的真实能力边界
- **AI Native 工作流生产闭环** — AI BPMN 生成/预览/模板推荐走认证租户与 workflow 权限
- **AI 生成 BPMN 导入失败** — 自动为缺失 DI 的 BPMN XML 注入布局信息
- **表达式引擎 `avg()` 除零 panic** — 空数组或全零数组返回 0
- **会签投票逻辑修复** — `counterSign` 模式下投票结果计数修正
- **流程设计器属性丢失** — 保存时 `extensionElements` 丢失，6 处属性写入不一致统一修复
- **服务请求审批链双缺陷** — 配置未生效和审批人重复
- **审批人自动分配层级丢失** — `autoAssign=true` 时未携带 `level` 字段
- **流程管理 5 项修复** — 版本 API 路径、semver 解析、菜单 404、列表分页、模板导入 DI 补全
- **工单分配搜索崩溃** — `description` 字段缺失时 crash
- **三个用户可见 bug** — 工单列表分页失效、事件详情页关联 tab crash、变更审批评论丢失
- **杂项修复** — SLA 预测窗口校验、WebSocket 重连指数退避、知识文章链接 404

### Security

- RBAC 未授权写入关闭（incident/change/problem/service-request/release/knowledge），含批量回归测试
- CSRF 全链路修复（Cookie 属性 + Token 传递）
- DataScope / 事件 / 发布 / 工单升级写入路径行级安全校验
- 生产发布门禁加固：高/危依赖和 gosec 发现阻断发布

---

## [1.6.9] - 2026-08-20

### Added

- **Problem Management expansion** — Added Problem trends, hotspots, SLA lookup, and per-problem comment read/write endpoints so operators can analyze recurring issues and discuss root cause without leaving the Problem record.
- **Incident comment deletion** — Added `DELETE /api/v1/incidents/:id/comments/:commentId` for tenant-scoped comment cleanup, mirroring the existing create/list semantics.
- **Knowledge recommendations** — Replaced the previous stub implementations of `/api/v1/knowledge/recommendations` and `/api/v1/knowledge/recent` with tenant-scoped queries that return real published articles with proper permission filtering.

### Fixed

- **Frontend Edge Runtime compatibility** — Replaced `Buffer.from(...)` calls in `src/middleware.ts` and `src/app/api/[...path]/route.ts` with an `atob()`-based JWT decoder so the Next.js middleware no longer raises `Code generation from strings disallowed for this context` in Edge Runtime (was breaking `itsm-frontend-prod` with HTTP 500).

### CI

- **Backend lint toolchain** — Pinned `gofumpt` to `v0.7.0` and cached `~/go/bin` between CI runs to make formatting results reproducible and shave install time.

### Tests

- **CMDB Service layer coverage** — Added 9 table-driven test functions covering CloudService / CloudAccount / CloudResource CRUD, Reconciliation (bound / unbound / orphan / unlinked / mixed), and Discovery operations via a `mockRepository` that isolates the service from the database.

---

## [1.6.8] - 2026-08-04

### Security

- **Go runtime baseline** — Raised the backend build and release toolchain to Go 1.25.12 to include the latest TLS, X.509, and MIME-header security fixes required by the production vulnerability gate.

---

## [1.6.7] - 2026-08-04

### Security

- **Go runtime baseline** — Raised the backend build and release toolchain to Go 1.25.10 as an intermediate production security baseline.

### Fixed

- **SLA calendar enforcement** — SLA deadlines now apply each definition's configured business calendar consistently across creation, lookup, and overdue detection; the prior unused implementation was removed.
- **Backend quality gate** — Removed obsolete change-status validation code so the release static-analysis gate has no dead-code findings.

---

## [1.6.6] - 2026-08-03

### Fixed

- **Cookie-only authentication test contract** — Updated API integration and HTTP client tests to assert credentialed browser requests without exposing HttpOnly session cookies through JavaScript `Authorization` headers.

---

## [1.6.5] - 2026-08-03

### Security

- **Production dependency remediation** — Updated frontend production dependencies and lockfile overrides for known Axios, DOMPurify, lodash, PostCSS, Sharp, and UUID advisories; `npm audit --omit=dev --audit-level=high` now reports zero vulnerabilities.
- **Package-manager integrity** — Removed the obsolete `pnpm-lock.yaml`; the frontend is governed solely by the committed npm lockfile, matching the release CI configuration.

---

## [1.6.4] - 2026-08-03

### Fixed

- **CI formatting gate** — Applied the repository's `gofumpt` formatting standard to backend source and tests, restoring the required backend release pipeline gate.

---

## [1.6.3] - 2026-08-03

### Fixed

- **Production authentication hardening** — Browser sessions now use HttpOnly, SameSite=Lax cookies exclusively. Access and refresh tokens are no longer exposed in JSON responses or read by frontend JavaScript; refresh token rotation is performed through the cookie transport.
- **Secure proxy cookie handling** — Authentication cookies now receive the `Secure` attribute when the request is HTTPS or terminated by a trusted HTTPS reverse proxy.
- **CSRF refresh coverage** — Both supported refresh routes are explicitly covered by the CSRF session-refresh exception.
- **Initialization readiness** — `/api/v1/readyz` now requires the latest registered post-schema migration rather than a stale, hard-coded migration version.
- **Workflow validation contract** — Removed the frontend call to an unimplemented validation endpoint; workflow designer preflight validation is deterministic and local, while create/update remain server-validated.

### Changed

- **Release metadata** — Frontend package, backend version endpoint, system configuration endpoint, and GA-readiness report now identify the release as `1.6.3`.

---

## [1.6.0] - 2026-08-01

### Fixed

- **Critical: Form Boundary Issue** - Fixed TicketTypeFormModal where approval/SLA tab fields were outside `<Form>` component, causing form validation failures.
- **Critical: Auth State Persistence** - Removed `isAuthenticated` from Zustand persist partialize to prevent false login state after browser refresh.
- **High: Timer Leak in Export/Import** - Fixed memory leaks in TicketCategoryExport and TicketCategoryImport with proper useRef cleanup on unmount and error paths.
- **High: ITIL Status Guard** - Added readonly protection on change/edit and release forms to prevent editing approved/completed records.
- **High: Suspense Boundary** - Added Suspense wrapper for useSearchParams in tickets page to fix CSR bailout.
- **High: Query Key Normalization** - Fixed useTicketsQuery to only include request params in queryKey, not response data.
- **High: NaN Route Guard** - Added Number.isFinite check for marketplace/[id] route parameter.
- **High: Error Handling** - Fixed workflow-api to throw errors instead of silently returning empty arrays.
- **Medium: useFormMemory** - Added debounce (500ms), enabled flag for edit mode, and dayjs serialization.
- **Medium: CIEditorForm** - Added min/max/precision props to InputNumber for numeric fields.
- **Medium: ChangeDetail Null Guard** - Added null check for createdAt before dayjs conversion.

### Changed

- **Ant Design destroyOnHidden** - Added to ProblemInvestigationTab modals.
- **SchemaField Type** - Added validation property with minValue/maxValue/precision/pattern.

---

## [1.5.2] - 2026-07-31

### Fixed

- **SLA Compliance Statistics** - Fixed negative compliance numbers caused by mismatched time scopes between total ticket count (30 days) and violated ticket count (all time). Backend now uses `HasTicketWith` edge predicate for consistent scope.
- **Timezone Inconsistency** - Fixed dashboard activity timestamps using `time.RFC3339` format instead of `Format("2006-01-02 15:04:05")` which browsers parse as UTC.
- **TicketDetail Duplicate Request** - Fixed double API call on ticket detail page by removing `fetchTicket`/`fetchSLAInfo` from useEffect dependency arrays.
- **Login Error Message** - Login page now shows actual backend error messages (e.g., "invalid credentials") instead of generic "登录失败".

### Changed

- **Ant Design TabPane Deprecation Migrated** - Converted all `Tabs.TabPane` / `<TabPane>` usage to `items` prop pattern across 8 files: analytics, applications, NotificationCenter, TicketTypeFormModal, IncidentManagement, FieldDesigner, profile, dashboard.
- **Space direction → orientation** - Migrated 6 instances of deprecated `Space direction="vertical"` to `orientation="vertical"`.
- **destroyOnClose → destroyOnHidden** - Migrated 1 instance in ApprovalTimeline component.
- **alert()/confirm() → antd message/modal** - Replaced native browser dialogs in marketplace and installations pages with antd `App.useApp()` message/modal.
- **console.log Cleanup** - Removed debug console.log from TicketDetail and BPMNDesigner.

---

## [1.5.0] - 2026-07-30

### Added

- **Connector/Skill Manifest Hardening** - All official connector manifests now declare `version`, `requiredPermissions`, and a deterministic SHA-256 `checksum`; registration is fail-closed (incomplete manifests are rejected at startup). Skill manifests share the same validation and checksum convention. Market API exposes `isOfficial` / `requiredPermissions` / `checksum`.
- **Post-Schema Migrations 008-010** - Initialization ledger (008), PostgreSQL RLS tenant isolation (009), ticket types in `itil-core` transaction (010).
- **Bootstrap Token** - One-time hashed bootstrap token with TTL, concurrent-consumption protection, replay defense, and break-glass flow for first-admin creation.
- **Endpoint ACL Manifest** - Versioned ACL manifest with 100% static coverage gate over protected routes (route-ACL-permission-menu).
- **Fencing Token Hardening** - Owner/token/lease re-verified inside the committing transaction; stale-writer prevention proven via PostgreSQL fault-injection tests.
- **Audit Routes** - New audit trail API endpoints with tenant isolation support
- **CI Attribute Validator** - Moved from handlers to service layer for better separation of concerns
- **Operator Context** - Enhanced audit trail with operator context tracking

### Fixed

- **Ant Design v6 Select Compatibility** - Replaced deprecated `<Select><Option>` child pattern with `options` prop across 100+ files. Fixes issue where clicking Select dropdowns had no response in antd v6. Affected modules: Ticket, Incident, Problem, Change, CMDB, Workflow, SLA, Service Catalog, Admin, and Reports pages.

### Changed

- **CMDB API Route Convergence** - `/api/v1/cmdb/*` is now the canonical prefix for all CMDB endpoints (CIs, CI types, relationships, relationship types, topology, impact analysis, change history, stats). Frontend API clients (`cmdb-api.ts`, `cmdb-relationship.ts`) have been switched to the canonical prefix. Added `GET /api/v1/cmdb/relationship-types` to the canonical tree. Note: change history is `GET /api/v1/cmdb/cis/:id/history` (the old `change-history` suffix only exists on the deprecated alias).
- **CMDB Multi-Tenant Isolation** - Added `tenant_id` field to CI relationships, configuration item history, and discovery sources. Added tenant-aware backfill migration.
- **CI Attribute Validation** - Migrated from `handlers/cmdb/attribute_validation.go` to `service/ci_attribute_validator.go`.

### Deprecated

- **`/api/v1/configuration-items/*` routes** - Kept as a compatibility alias for clients not yet upgraded. No new endpoints will be added under this prefix; removal will be evaluated after a regression period.

### Migration Notes

- **CMDB tenant backfill**: Run `itsm-backend/migrations/20260610_cmdb_tenant_id_backfill.sql` on existing databases to populate the new `tenant_id` fields.
- **Preset CI types**: Enable with `itsm-backend/migrations/20260611_enable_preset_ci_types.sql`.
- **Audit routes**: New endpoints are protected by existing JWT + RBAC + tenant middleware.

---

## [1.0.0] - 2026-03-07

### Added

#### Core ITIL Modules
- **Ticket Management** - Complete ticket lifecycle management with creation, assignment, tracking, and closure; built-in SLA management, priority handling, comments and attachments
- **Incident Management** - Incident discovery, logging, classification, escalation; real-time monitoring and alerts
- **Problem Management** - Root Cause Analysis (RCA), Known Error Database, problem resolution tracking
- **Change Management** - Change requests, risk assessment, multi-level approval workflows

#### Service & Knowledge Base
- **Service Catalog** - Service request templates, self-service portal, SLA management
- **Knowledge Base** - RAG intelligent search, knowledge categorization, FAQ management, vector retrieval

#### Workflow Engine
- **BPMN Workflow** - Visual process designer, approval workflow automation
- **Task Management** - Workflow task assignment and tracking

#### AI-Powered Features
- **Intelligent Classification** - Auto-identify ticket type, priority, impact scope
- **Auto-Summary** - AI-generated ticket/incident summaries
- **RAG Knowledge Base** - Vector search-based intelligent knowledge recommendation
- **Smart Suggestions** - Recommended solutions, similar tickets

#### User & Permissions
- **Multi-Tenant Architecture** - Complete tenant isolation and management
- **Role-Based Access Control** - RBAC permission system, fine-grained access control
- **User Management** - User CRUD, team and department management

#### SLA Monitoring
- **SLA Definition** - Service Level Agreement configuration
- **Real-Time Monitoring** - SLA compliance rate tracking
- **Alert Rules** - SLA violation alerts and notifications

### Technical Stack

| Category | Technology |
|----------|------------|
| Backend | Go 1.25+ / Gin / Ent ORM |
| Frontend | Next.js 15 / React 19 / TypeScript / Ant Design 6 |
| Database | PostgreSQL 17 / Redis 7 |
| Deployment | Docker / Docker Compose |

### Quick Start

```bash
# Docker Compose (Recommended)
git clone https://github.com/heidsoft/itsm.git
cd itsm
make dev-up

# Access
# Frontend: http://localhost:3000
# Backend: http://localhost:8090
# API Docs: http://localhost:8090/swagger

# Login
# Username: admin
# Password: admin123
```

### Known Limitations

- Mobile PWA features are under development
- Enterprise integration features (LDAP/SSO) planned for future releases

### Documentation

- [Development Guide](./docs/install.md)
- [Deployment Guide](./docs/deployment-optimization.md)
- [API Documentation](./docs/api-reference.md)

---

*Thank you to all contributors for your support!*
