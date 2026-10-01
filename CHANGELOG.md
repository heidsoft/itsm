# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added

- **会话真相端点 `GET /api/v1/auth/session`** — 后端一次返回 `{user, tenants, expiresIn}`：身份与权限复用 `/auth/me` 的同一个组装入口，租户列表复用同一个查询，`expiresIn` 由服务端从已认证 access token 的签发时间算出剩余秒数。此前前端要靠 `/auth/me` + `/auth/tenants` 两个探活再加本地推断来决定「我是否已登录」，任一瞬时失败都会产生与后端不一致的会话判断。响应不含任何令牌值（契约测试断言 body 里搜不到 cookie 中的 JWT）。
- **登录/刷新响应新增 `expiresIn`** — 续签调度改用它，不再用浏览器时钟和固定 10 分钟间隔猜会话何时过期。
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

### Tooling

- **空卷首装门禁 `make fresh-install-gate` / `fresh-install-gate.yml`** — 把「私有化全新部署装得起来」变成可复用的独立证明：用一次性 compose project + 全新命名卷起 postgres+redis，跑 `itsm-init`（migrate+seed）并要求退出码 0，随后断言迁移入账数、磁盘迁移 checksum 非空（空 checksum 就是「登记过但从未执行」，正是只在新装才炸的来源；历史收养条目 001-006 除外）、BPMN 流程模板已部署、活跃流程绑定指向的流程定义全部存在。此前唯一会跑到这条链的 ga-gate「Start core stack」自 2026-09-22 起连续 17 次红、且不在 main 的必需检查里，等于守卫存在但没人消费。实测：正向 `迁移 52 条（空 checksum 0）、活跃流程定义 18、活跃流程绑定 10（悬空 0）` 退出 0；负向用 `scripts/fixtures/fresh-gate-negative.override.yml` 注入一条引用不存在表的磁盘迁移，门禁如实报 `failed to apply migration 20991231_gate_negative_fixture: ... (42P01)` 并退出 1。CI 侧 push/PR/每日 schedule 触发，失败上传 init 日志。
- **把「只在全新安装暴露」的迁移缺陷类锁死** — `migration/fresh_install_reference_test.go` 新增三条守卫：禁止任何磁盘迁移引用 Ent 从未生成的旧式单数 RBAC 对象（`permission_definition`/`role_permission`/`role`/`endpoint_acls`）、禁止给已退役死表建索引、禁止迁移直接写 `process_bindings`，并锁住退役迁移必须以原版本名保留且仍为 no-op；`pkg/seeder/process_bindings_manifest_test.go` 断言内置流程绑定的每个 key 都能在 `service/bpmn/*.bpmn` 找到可部署模板。四个守卫在修复前实测为红、修复后为绿。
- **四域浏览器旅程改走真实登录链路** — `tests/playwright-four-domain-journey.cjs` 原先读 `POST /auth/login` 响应体的 `accessToken`，令牌改 httpOnly cookie 后必然拿不到；现改为真实 SPA 表单登录并断言已离开 `/login`，写请求统一由 `authedFetch` 附带 Double-Submit CSRF 头。运行时 `API_URL` 必须与站点同源（cookie 按 host 下发，发给 `127.0.0.1` 不会带上 `localhost` 的令牌）。同时修掉 `scripts/rebuild_and_verify.sh` 里前端镜像的错误 tag（`itsm-itsm-frontend:latest` → `itsm-frontend:latest`，此前 compose 拿不到刚构建好的镜像）。
- **文档门禁 CI 转绿（C.1 + C.3）** — `docs-gate` 在 CI 连续红于 5 个 commit：C.1 检出 `.github/workflows/ga-gate.yml` 的 4 处口令夹具、C.3 检出 19 条指向 `output/*.md` 与 `itsm-rag/README.md` 的内链。根因是这两个门禁只扫 `git ls-files` 的跟踪文件，而 `output/`（`.gitignore:271`）与 `itsm-rag/`（`.gitignore:213`）都被忽略——本地文件在、链接解析得到，CI checkout 里不存在，所以 `make docs-gate` 本地绿、CI 红（CI 传 `--strict`，本地默认 advisory）。修法：19 条内链中把「决策对话」类的 2 条改指仓库内 `docs/product/decision-dialogue-2026-09-24.md`、`docs/product/ai-automation-status.md` 的 `./ROADMAP.md` 修正为 `../../ROADMAP.md`（原本就指错位置），其余 16 条降级为行内代码并标注「本地工作稿，未纳入版本控制」；`ga-gate.yml` 的口令夹具按门禁既有词汇补 `开发环境…不得用于生产` 注释（值与 YAML 语义不变，已用解析器校验）。验证：在 HEAD 的干净 worktree 里 `run-all.sh --strict` = 7 total / 0 failed（C.1 0 处、C.3 0 条）。
- **Gate C.7 产品需求收敛守卫**（[check-scope-creep.sh](./scripts/docs-gate/check-scope-creep.sh)）— 把"继续扩散 = 构建失败"作为机器守卫生效，对应 2026-09-28 盘点的三类扩散面：C.7.1 空壳前端模块零容忍（有 `page.tsx` 但穿透 `@/components` 引用后仍无后端调用的模块，**新增空壳无豁免通道**；⚠️ 修正：初版检出「10 个模块/15 页」经逐文件复核**全部为误判**——漏 `.tsx` 扩展名解析、漏 `lib/services/*` 封装层、未排除 `redirect()` 兼容路由页，真实空壳为 **0**，白名单已清空，三类教训写入脚本与白名单文件头）/ C.7.2 预览域棘轮（README「预览」能力域数只减不增，当前 9）/ C.7.3 规划能力面冻结（ROADMAP v2.0+v3.0 未完条目只减不增，当前 14）。已接线 `run-all.sh`，新增 `make scope-creep` 目标；注入回归 5 用例验证生效（基线收紧/白名单过期/注入空壳均 FAIL，advisory 不阻断）。收敛方案与待拍板决策见 [plans/scope-convergence-plan-2026-09-28.md](./plans/scope-convergence-plan-2026-09-28.md)
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

- **access token 一过期就登不出，7 天 refresh cookie 把会话「复活」** — `POST /api/v1/auth/logout` 挂在 `AuthMiddleware` 后面，而 access token 只有 15 分钟：用户放一会儿页面再点登出，请求在中间件里就 401，处理凭证的 handler 从未执行，两类 cookie 一个都没清。留下的 `refresh_token`（7 天、httpOnly）随后被自动续签，登出等于没发生。现在登出刻意不挂鉴权中间件：先无条件清 cookie，再尽力吊销请求里携带的 access/refresh token，吊销存储故障时返回 5003（5003 业务码）而不是伪装成功——cookie 已清、服务端未吊销是必须让运维看见的状态。无凭证重复登出幂等返回 200。回归：`router/auth_handler_routes_test.go` 用「过期 access token + 有效 refresh cookie」断言 200、两个 cookie 的 `Max-Age<0`、随后用同一枚 refresh cookie 续签必须 401。
- **登出从不吊销 refresh token** — 旧实现只调 `RevokeAccessToken`，浏览器 cookie 清掉之后服务端那枚 7 天凭证仍然有效；任何拿到它的人（或那份遗留 cookie）都能继续换新 access token。现在登出同时吊销两类凭证。
- **会话凭证的名称与生命周期只剩一处定义** — `900`/`604800`/`"access_token"` 这些字面量原先分散在 `handlers/common/handler.go`、`handlers/auth/handler.go`、`middleware/auth.go` 与 service 的签发调用里，改一处就让浏览器 cookie 窗口和服务端校验窗口悄悄分叉。现在统一由 `middleware.AccessTokenCookie`/`RefreshTokenCookie`/`AccessTokenTTL`/`RefreshTokenTTL` 派生，Set-Cookie 的 `Max-Age`、JWT 过期时间和契约测试断言都从同一处取。
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

- **⚠️ 列表响应收敛为唯一信封 `{items,total,page,pageSize,totalPages}`** — `common.ListResponse.MarshalJSON` 原来会用 reflect 按元素类型追加领域名别名（`items` 与 `tickets`/`incidents`/`changes`/`articles` 同时返回同一份数组），再把同一组分页事实嵌套成第二个 `pagination` 对象；同一接口因此有两套真相，前端 `response.items ?? response.tickets ?? []` 这类被 AGENTS.md 禁止的多字段 fallback 反而被"合法化"。现在键集合固定为五个平铺键，别名表（`domainAliases`/`singularAliases`/`inferDomainAlias`/`lowerFirst`）与嵌套 `pagination` 整段删除。写入侧跟着收敛：`ListChanges` 从 `gin.H{"changes":…}` 改走 `SuccessWithPagination`，知识文章改用标准生产者（补齐此前缺失的 `totalPages`），活跃告警补齐 `totalPages`，`TicketTypeListResponse.Types`、`IncidentListResponse` 的 `incidents`/`pagination` 与 `KnowledgeArticleListResponse` 等别名字段、重复 DTO 一并删除。请求侧同一批：事件列表与活跃告警改读 `pageSize`（原 `size`），前端 `IncidentAPI` 里 `pageSize→size` 的翻译器和 BPMN 两处 `pagination.total` 读取同步移除；消费侧 20 余处 `response.tickets ?? …` 改为单键读取。**升级影响**：走标准生产者的列表接口其领域名顶层键与 `data.pagination` 不再返回，外部集成方必须改读 `data.items`，见 [UPGRADE.md](./UPGRADE.md) §1.8。守卫：`tests/contract/list_envelope_ratchet_test.go` 用 go/ast 扫 `dto/*.go`（类型名含 List 且带 total 即判为信封）做三条双向棘轮——领域名集合键（基线 30）、标准分页键不完整（基线 11）、`size` 等分页别名（基线 9），新增违规与基线过期都失败；`handlers/incident/list_contract_test.go`、`handlers/knowledge/list_envelope_test.go` 断言真实路由的精确键集合，并锁死 `size` 不再是事件列表的契约参数。未收敛部分按基线登记为债务：`roles`/`menus`/`tenants`/`catalogs`/`allocations`/`notifications` 等 30 个手写 DTO 仍返回领域名集合键，CMDB/服务目录/通知 9 个信封仍用 `size`
- **`api-contract-check` 由红转绿** — 该 workflow 唯一失败项是 swagger 新鲜度（`git diff --exit-code` 三件套），根因是 `itsm-backend/docs/*` 落后于代码。重新生成后定义数从 344 降到 174——移除的 171 个（含 162 个 `ent.*` 模型和 9 个 Ent 枚举/schema 类型）全是 handler 注解直接序列化泄漏出来的持久化模型，新增 1 个 `dto.IncidentListResponse`；路径数保持 159，但集合换了 6 条：删掉从未注册的 `/api/v1/projects`、`/api/v1/projects/{id}` 与实际不存在别名 `/api/v1/departments/{id}`，补上 5 个真实注册但从未文档化的部门路由（`/api/v1/org/departments`、`/api/v1/org/departments/tree`、`/api/v1/org/departments/{id}`，加上既有的 `/api/v1/departments`、`/api/v1/departments/tree`）。悬空 `$ref` 0（仍有 6 个 `ent.*` 定义由市场/安装路由泄漏，属既有债务）。前端 `api-contract.test.ts` 的两处存量不匹配也修绿：菜单列表改为静态 base 常量再分支拼 query（字符串拼接路径无法被契约扫描解析）；`MspApi.getAllocationHistory` 指向的 `GET /api/v1/msp/allocations/history` 从未被 `router/msp_routes.go` 注册，属悬空接口而非命名漂移，因此按仓库既有的能力开关通道置 `PRODUCT_CAPABILITIES.mspAllocationHistory = false` 并在 `DISABLED_API_CONTRACTS` 登记原因、MSP 页面隐藏「分配历史」Tab，而不是补一个假接口
- `config/seed/default.json` 不再整段替换服务目录清单，内置 Go 清单成为唯一权威来源（原 8 条与内置子项引用冲突的目录已移除，"权限变更"并入账户申请子项）
- 初始化组件 checksum 现在覆盖代码内产品清单（权限目录、菜单规格、内置角色/审批组/角色授权、工单类型、BPMN 模板内容），修改这些定义会被版本账本察觉，无需手工升版
- 生产部署配置：`RLS_MODE` 默认值调整为 `off`（与后端安全默认对齐）。已配置 `.env.prod` 的部署不受影响

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
