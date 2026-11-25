# 产品架构「能力—场景—承诺」对应关系审计与修复方案（2026-11-25）

> 审计方式：codex（`gpt-6.1-sol`）只读全链审计 + 独立源码反向复核。
> 审计基线 HEAD：`c93be3903c3d6385e6142f21f093609c2a7b4dfd`（主树 53 处并行 WIP 未纳入证据；权限修复分支 `428b6a175` 不在范围内）。
> 本报告不触发代码改动；实施前需对「拍板清单」逐项确认。

## 一句话结论

能力矩阵（`handlers/capability/handler.go` 的 28 项 registry）是运行时成熟度事实源，README（15 域）与商业契约是承诺展示层，两者之间的 C.6.1 守卫**只比对两份文档、不读 registry**；10 个场景测试多数是 service/repository 层测试而非生产入口验收，其中 scenario8 的名称与注释承诺了代码里不存在的 RAG 命中断言。这不是「能力没做」，而是「承诺无法自动追溯到事实，测试名称超过了证据边界」。

## 发现总表

| # | 发现 | 严重度 | 关键证据 | 复核结论 |
|:--|:--|:--|:--|:--|
| 1 | 成熟度守卫未闭合运行时 registry | 高 | `scripts/docs-gate/check-product-drift.sh:166-262` | 成立：只解析 README + 商业契约，跳过 `registry`；`__SUBDOMAIN__` 明列跳过 TicketType/云发现 |
| 2 | README 独立承诺在 registry 无身份 | 高 | `README.md:105,117`；`handler.go:66-251` | 成立但需区分：TicketType/RBAC 无独立 key；反向 `project/application/tag/template/identity.oidc` 无 README 独立行；属映射缺口，非「未实现」证据 |
| 3 | scenario8 标题/注释承诺 RAG 基线但无 RAG 断言 | 高 | `tests/scenarios/scenario8_ai_triage_rag_baseline_test.go:12-23,111-117` | 成立：全仓 `tests/scenarios/` 中 RAG 仅出现在本文件注释与函数名，实际只构造 `NewTriageService(nil)` 测关键词分诊 |
| 4 | 场景测试多为服务层直连，不能冒充生产端到端 | 中 | `scenario1:27-42`、`scenario7:28-42,379`、`scenario9:75-86`；`scenario5:207,224` | 成立：10 个场景中 1–6、10 直连 service/repo + SQLite；scenario9 的成功 consumer 是测试回调；scenario5 有 `t.Skip` 可跳过隔离断言 |
| 5 | C.7.1 空壳扫描可被同模块无关 API 标记绕过 | 中 | `scripts/docs-gate/check-scope-creep.sh:133,160-192` | 成立：`has_api()` 扫整个模块目录（含兄弟页与注释），且跟随 `@/components` 导入；当前检出 0 个不等于将来不会假绿 |
| 6 | `product-scope-baseline.txt` 注释 14 域与实读 15 域不符 | 低 | `product-scope-baseline.txt:17`；`README.md:100-120` | 成立：仅注释漂移，`preview_domains=7` 与实测 7 个预览一致，棘轮值不改 |
| 7 | DBOnly 空授权集回退 / 8 个前端 flag | —（误报纠正） | `middleware/rbac.go:350-356,811-818`；`product-capabilities.ts:6-26` | 不成立：已配置空集拒绝；flag 实为 7 个（另 5 条 `DISABLED_API_CONTRACTS`），且描述子功能契约而非成熟度 |

## 逐条详述与反驳判据

### 1. 成熟度守卫没有闭合运行时事实源（高）

`check_maturity_parity()` 的输入只有 `README.md` 和 `itsm-commercial-capability-contract.md`。`contract_for()` 把「工单类型与动态表单」「CMDB 云发现」映射为 `__SUBDOMAIN__` 后直接 `continue`，`state_ok()` 用子串 `*GA*`/`*Pilot*` 判定。因此守卫绿只能证明**两份文档互相一致**，不能证明 registry 中 `workflow` 仍是 Pilot、`marketplace` 已是 GA。`handler_test.go:73-95` 只把 `acceptanceVersion` 与商业契约、CHANGELOG 日期挂钩，并未做成熟度全量对照。

- **反驳判据**：存在一个自动检查，遍历 registry 全部 28 个 key，与 README/契约的域归属双向比对，且单独改一个 key 的成熟度必然变红——目前不存在。
- **不是问题**：若 registry 只作为内部 UI 元数据、不承担对外承诺，则无需闭合；但当前端点 `GET /api/v1/capabilities` 已被 `/admin/capabilities` 页直接消费（`page.tsx:25,39`），且商业契约明文声明同口径，故此前提不成立。

### 2. 承诺缺身份与反向漏报（高，但需拍板粒度）

registry 28 项中不存在 `ticketType` / `rbac` / `auth` 独立 key，而 README 把它们列为独立能力域；反向 `project`、`application`、`tag`、`template`、`identity.oidc` 在 README 表无独立行（可能被聚合进其他域）。聚合口径不同本身合法，问题在于**没有可校验的归属规则**，改一处无法知道该同步几处。

- **反驳判据**：存在受测的显式规则把每个 README 域映射到 ≥1 个 registry key，且新增/删除 key 会触发失败。
- **不能据此声称**：TicketType/RBAC「没有实现」——已有 handler/路由/测试，只是没有独立能力身份。

### 3. scenario8 名实不符（高）

文件头注释列了 5 条覆盖，最后一条是「RAG 关键词搜索基本命中（标题匹配得分高于内容匹配）」；函数体到 117 行结束，只有 `service.NewTriageService(nil, logger)` 的关键词分诊、`SuggestForTenant` 等断言，没有任何知识库检索调用。`git grep -iE 'rag|retriev' c93be390 -- tests/scenarios/` 仅命中注释与函数名。用它向客户证明「RAG 命中基线」属于把测试名称当验收证据。

- **反驳判据**：在该文件内找到真实知识库检索调用与得分断言；或在别处有明确命名的 RAG 验收测试且被本场景引用为外部证据（当前没有）。
- **修复选择**：补真实 RAG 场景，或把标题/注释改名为 `AI 关键词分诊启发式基线`；两者都是单次提交可完成，但不能只改名字不改事实——若真实 RAG 验收本就不存在，要在 ROADMAP 如实标注缺失。

### 4. 场景层级混杂（中）

场景 1–6、10 用 `enttest.Open(sqlite3)` 直连 service/repository（如 `scenario1:27-42`），scenario7 直接测连接器算法与内部 Router（`scenario7:28-42,379`），scenario9 运行真实 command worker 但成功 consumer 是测试回调（`scenario9:75-86`）。这些测试对业务规则有真实价值；但它们绕开 HTTP，不能证明鉴权、CSRF、错误映射在生产链可用。`scenario5:207,224` 的条件 `t.Skip("无策略，跳过跨租户测试")` 会在 fixture 失效时把隔离断言变成假绿。

- **反驳判据**：存在真实 `router.SetupRoutes` + 普通角色（非 super_admin）+ CSRF 开启的端到端测试，覆盖同一断言。
- **注意**：`super_admin` 生产直通确实存在（`middleware/rbac.go:755-759`），scenario6 用它排除行级范围（`scenario6:218-219`），不能用来证明普通角色授权。

### 5. C.7.1 空壳扫描的判定面过宽（中）

`API = /api/v1|lib/api|lib/services|apiClient|[a-zA-Z]+Api\b|useMutation|useQuery|axios|fetch\(|request\(/` 对模块目录下所有 `.ts(x)` 做文本搜索，兄弟页里一处 API 调用或注释里的 `apiClient` 就会让整个模块通过；`components_of()` 还会跟随 `@/components/*` 导入继续搜。首版误判 10 模块后已修过一轮，但不能把「当前检出 0」当作守卫已闭合。

- **反驳判据**：构造一个只有静态文本的 `page.tsx`，同模块兄弟页含 API 调用或注释含 `apiClient`，守卫仍判 FAIL——当前不会。

### 6. 注释漂移（低）

`preview_domains=7` 的实测值正确；注释「共 14 域：7 可用 + 7 预览」与 README 实读 15 域（8 可用 + 7 预览）不符。只改注释，不动棘轮值；不要把域拆分/合并当成规避棘轮的手段。

## 修复方案（按批次，一个 commit 一个主题）

| 批次 | 目标与单一源设计 | 守卫与反向验证 | 文档同步 |
|:--|:--|:--|:--|
| **P0-A 承诺闭合** | registry 仍是成熟度唯一事实源；新增受测的「README 域 ↔ registry key」归属规则（建议放 `handlers/capability/` 旁，不把成熟度复制到第二份表）。域状态由映射到的 key 推导（建议：全部 GA 才「可用」，任一 Pilot 即「预览」），C.6.1 继续只管文档互相一致 | 新增 Go 测试：28 key 全部有归属、README 15 域全部有 key、悬空证据失败；反向验证：单独改 `workflow` 或 `marketplace` 的 maturity 必须红在断言 | README、商业契约、开源能力说明、CHANGELOG |
| **P0-B 场景正名** | 二选一：补真实 RAG 检索场景（接知识库生产路径，断言标题/内容得分差），或把 scenario8 改名为关键词分诊启发式基线并在 ROADMAP 标注 RAG 验收缺口 | 若补测试：断开检索装配必须红；若改名：旧名断言同步更新，防回退 | CHANGELOG、ROADMAP（若标注缺口） |
| **P1 生产入口验收** | 保留现有 service 层场景；为 1 个关键闭环（建议工单生命周期）补 `router.SetupRoutes` + 普通角色 + CSRF 开启 + 真实 DB 的入口测试；scenario5 的 `t.Skip` 改成 fixture 缺失即 fail | 撤权/跨租户/缺 CSRF/断开 worker consumer 各产生预期拒绝或失败；删除路由必红 | 验收清单、CHANGELOG |
| **P1 菜单闭合** | 菜单基线继续以 DB 种子为事实源；为菜单项建立显式能力归属校验，Router 仍是 URL 事实源 | 未知 key、菜单死链必须红；C.7.1 改为「页面自身或强关联组件」判定，去掉兄弟页与注释命中 | API 文档、CHANGELOG |
| **P2 口径治理** | `product-scope-baseline.txt:17` 注释 14→15；历史 plans 的「25 项」属日期快照不动；workflow 签字栏空保持诚实阻断 | 当前统计可复算；不借拆合域绕过棘轮 | 仅注释与 CHANGELOG 记录 |

## 拍板清单（实施前必须确认）

1. **TicketType 与 RBAC 是否成为独立能力 key**（影响 registry 数量与 C.6.4 棘轮；建议先不新增 key，用归属映射解决）。
2. **聚合域状态推导规则**：建议「全部 GA 才可用，任一 Pilot 即预览」，与通知 GA + 连接器 Pilot → README「预览」的现状一致。
3. **project / application / tag / template / identity.oidc 是否纳入对外承诺**（若不纳入，需在 README 或契约明示它们是内部支撑面）。
4. **scenario8 走「补 RAG 测试」还是「改名 + 标注缺口」**，以及真实租户验收负责人与环境（workflow 签字栏不得由 AI 代签）。

> 实施纪律：独立 worktree 起步，显式暂存、反向验证红在断言、既有红不代调、不改基线只为让门禁变绿、一个 commit 一个主题。
