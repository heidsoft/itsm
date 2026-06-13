# ITSM 基于角色视角的产品测试方案

> **版本**: v1.1（Spec Kit 化）
> **日期**: 2026-06-13
> **目标**: v1.0 GA 交付前的"可上线"质量门禁
> **方法论**: 角色驱动 + 业务流闭环 + 风险加权 + 自动化优先
>
> **Spec Kit 工件**：本方案的需求/验收/成功标准已抽取为 Spec Kit 规范，详见：
>
> - 规范主体：[`specs/001-role-based-testing/spec.md`](../specs/001-role-based-testing/spec.md)
> - GA 检查单：[`specs/001-role-based-testing/checklists/ga-readiness.md`](../specs/001-role-based-testing/checklists/ga-readiness.md)
> - 后续 `/speckit-plan`、`/speckit-tasks`、`/speckit-clarify` 将基于该 spec 生成
>
> 本文件仍为"操作手册"，覆盖角色矩阵、API 冒烟、安全/性能/排期等执行细节；spec 文件覆盖"可测试需求 + 可度量准入"。两者交叉引用、不重复维护：FR-XXX 在 spec 中，EU-/EN-/FLOW- 等场景 ID 在本文件。

---

## 一、测试目标与判定标准

### 1.1 总体目标
- 覆盖 ITIL v3 全生命周期：工单→事件→问题→变更→服务请求→知识库
- 覆盖 7 类典型角色 × 50+ 场景 × 3 类质量维度（功能 / 安全 / 性能）
- 形成自动化套件（CI 可重复跑），关键路径必须 100% 通过

### 1.2 GA 准入门禁（DOR / DOD）

| 门禁项 | 阈值 | 当前状态 |
|---|---|---|
| 后端单元测试 | 100% 通过 | ✅ 17 packages OK |
| 前端单元测试 | 100% 通过 | ✅ 526 / 526 |
| TypeScript type-check | 0 错误 | ✅ |
| ESLint | 0 errors，warnings ≤ 500 | ✅ 490 |
| 后端构建 | 0 编译错 | ✅ |
| 前端构建 | 0 错误 | ✅ |
| GA Readiness | 12/12 模块 ready | ✅ |
| 关键 API 冒烟 | 100% 通过（明细见 §6） | 🔶 部分待修 |
| P1 角色场景 E2E | 100% 通过 | 🔶 待执行 |
| P2 场景 E2E | ≥ 90% 通过 | 🔶 待执行 |
| 已知 P0 缺陷 | 0 | 待评审 |
| 安全基线 | OWASP Top 10 复核 | 待执行 |

---

## 二、被测产品域划分

| 域 | 模块 | 关键路由 |
|---|---|---|
| 入口/认证 | 登录、SSO、Token、菜单 | `/auth/*`, `/menus`, `/auth/menus` |
| 核心业务 | 工单、事件、问题、变更、服务请求 | `/tickets`, `/incidents`, `/problems`, `/changes` |
| 资源/资产 | CMDB、CI 类型、关系、云服务 | `/configuration-items/*` |
| 知识/服务 | 知识库、服务目录、标准变更 | `/knowledge-articles`, `/service-catalogs`, `/standard-changes` |
| SLA/告警 | SLA 定义、监控、告警规则 | `/sla/*` |
| 流程引擎 | BPMN、审批链、流程定义/实例 | `/process-bindings`, `/approval-workflows`, `/workflow/*` |
| AI/RAG | 智能分诊、AI 摘要、AI 审计 | `/ai/*` |
| 系统管理 | 用户、角色、权限、租户、菜单、审计 | `/users`, `/roles`, `/permissions`, `/tenants`, `/audit-logs` |
| 集成/连接器 | 飞书、钉钉、企业微信、Console、Webhook | `/connectors/*` |
| 报表/分析 | 仪表盘、工单分析 | `/dashboard/overview`, `/analytics/*` |

---

## 三、角色矩阵（7 类）

> 实际系统中的 RBAC 角色：`super_admin`、`security`、`end_user`，加上业务期望存在的 4 类典型角色（用于场景驱动）。

| 角色 | 代号 | 测试账号（默认 seed） | 主要职责 | 高风险动作 |
|---|---|---|---|---|
| 超级管理员 | super_admin | `admin / admin123` | 跨租户管理、系统配置、用户/权限/菜单 | 删除租户、改 RBAC、改菜单 |
| 租户管理员 | tenant_admin | （由 super_admin 创建） | 租户内一切配置、角色分配 | 租户内配置变更 |
| 服务台主管 | sd_manager | （角色 seeded，需手工建用户） | SLA、分派、升级、报表 | SLA 变更、强制关闭 |
| 工程师 / 处理人 | engineer | （需建账号，挂 `engineer` 角色） | 接单、解决、根因、变更执行 | 状态推进、CI 修改 |
| 审批人 / 经理 | approver | （需建账号，挂 `manager` 角色） | 变更/服务请求审批 | 批准/驳回 |
| 安全管理员 | security | `security1 / security123` | 审计日志、安全事件 | 看审计、封禁账户 |
| 终端用户 | end_user | `user1 / user123` | 提单、查询自有工单、知识库浏览 | 仅自有工单写权限 |

> **测试账号补全脚本**（建议）：在 `pkg/seeder` 增 `engineer1/eng123`、`manager1/mgr123`、`tenant1admin/ta123`，否则需在前置步骤 super_admin 建账号。

---

## 四、按角色的测试场景清单

### 4.1 super_admin（最高优先级 P0）

| ID | 场景 | 验证点 | 类型 |
|---|---|---|---|
| SA-01 | 登录并加载菜单 | `/auth/login` → `/auth/me` → `/auth/menus` 含 80+ 菜单项 | API+E2E |
| SA-02 | 创建租户 | `POST /tenants` → 列表可见 → 切换租户 | API |
| SA-03 | 用户/角色 CRUD | 建用户 → 分配角色 → 撤销 → 删除 | API+E2E |
| SA-04 | 权限矩阵编辑 | `/permissions` + `/roles/:id/permissions` 增删 | API |
| SA-05 | 菜单管理 | `/menus` 增删、排序、隐藏 | API+E2E |
| SA-06 | 审计日志全量 | `/audit-logs` 可查所有租户记录 | API |
| SA-07 | 连接器全生命周期 | install → enable → health → send → disable → uninstall | API |
| SA-08 | GA Readiness | `/readiness/ga` 返回 12/12 ready | API |
| SA-09 | 系统配置改 SLA 模板 | 改默认 SLA 时长 → 新建工单生效 | E2E |
| SA-10 | 删除租户级联 | 删租户后用户/工单都不可见 | API |

### 4.2 tenant_admin（P0）

| ID | 场景 | 验证点 |
|---|---|---|
| TA-01 | 仅看到自己租户用户 | `/users` 不返回其他租户 |
| TA-02 | 不能跨租户读工单 | `GET /tickets?tenant_id=other` 返回 401/403 或为空 |
| TA-03 | 配置审批流 | `/approval-workflows` 增节点 → 应用到变更 |
| TA-04 | 配置 SLA | `/sla/definitions` 增/改 → 工单使用新 SLA |
| TA-05 | 服务目录上下架 | `/service-catalogs` 状态切换 |
| TA-06 | CMDB 类型扩展 | `/configuration-items/types` 增类型 + 属性 |

### 4.3 sd_manager 服务台主管（P1）

| ID | 场景 | 验证点 |
|---|---|---|
| SD-01 | 工作台总览 | `/dashboard/overview` 返回今日工单/事件/SLA 风险 |
| SD-02 | 分派工单 | `PUT /tickets/:id` 改 `assignee_id` + 评论 |
| SD-03 | 强制升级 | 工单升级 P0 + 触发 SLA 重计 |
| SD-04 | SLA 监控 | `POST /sla/monitoring` 返回风险列表 |
| SD-05 | 自动分派规则 | `/ticket-assignment-rules` 增规则 → 新工单自动分配 |
| SD-06 | 报表导出 | 工单分析 `/analytics/tickets` |
| SD-07 | 关闭/重开工单 | `/tickets/:id/workflow/close` + reopen |

### 4.4 engineer 处理人（P0）

| ID | 场景 | 验证点 |
|---|---|---|
| EN-01 | 我的待办 | `/tickets?assignee_id=me` 仅返回分配给我的 |
| EN-02 | 接受 → 进行中 → 解决 | 工单状态机：open → in_progress → resolved |
| EN-03 | 添加评论 | `POST /tickets/:id/comments` |
| EN-04 | 关联 CI | `/tickets/:id` PUT `affected_ci_ids` |
| EN-05 | 事件根因分析 | `POST /incidents/:id/root-cause` |
| EN-06 | 转问题 | 事件 → 创建 problem，引用关系建立 |
| EN-07 | 变更实施 | 变更状态 approved → in_progress → completed |
| EN-08 | 知识库引用 | 工单解决时附 KB ID |
| EN-09 | 工时记录 | 评论或专用 time-log（如有） |
| EN-10 | AI Triage 接受建议 | 触发 AI → `accepted=true` 写入 `/ai/audit` |

### 4.5 approver 审批人（P1）

| ID | 场景 | 验证点 |
|---|---|---|
| AP-01 | 待审批列表 | `/approvals` 或 `/approval-workflows/instances?status=pending` |
| AP-02 | 同意变更 | 多级审批中第一级通过 → 进入下一级 |
| AP-03 | 驳回 | 整个流程结束并标记 rejected |
| AP-04 | 超时（manager_timeout） | 配置 timeout 短 → 等待 → 自动升级或挂起 |
| AP-05 | 委托审批 | 转交他人 |
| AP-06 | 审批历史可查 | 操作有审计日志 |

### 4.6 security 安全管理员（P1）

| ID | 场景 | 验证点 |
|---|---|---|
| SE-01 | 仅可查审计日志 | `/audit-logs` 200，写工单 403 |
| SE-02 | 不能改用户/租户 | 试 PUT 用户应 403 |
| SE-03 | 不能改 SLA 配置 | `PUT /sla/definitions` 403 |
| SE-04 | 异常登录追踪 | 多次失败登录 → 审计有失败记录 |
| SE-05 | 数据导出限频 | 高频审计查询触发限流 |
| SE-06 | 安全连接器健康 | `/connectors/lifecycle` 可读但配置写 403 |

### 4.7 end_user 终端用户（P0）

| ID | 场景 | 验证点 |
|---|---|---|
| EU-01 | 仅看自己工单 | `/tickets` 只返回 `requester_id=self` |
| EU-02 | 提单（含附件） | `POST /tickets` + 上传 |
| EU-03 | 跟踪工单进度 | 状态变更可见 |
| EU-04 | 申请服务 | `POST /service-requests` |
| EU-05 | 浏览知识库 | `/knowledge-articles` 仅 `published` |
| EU-06 | 知识库点赞/搜索 | `/knowledge-articles/search?q=邮件` |
| EU-07 | 不能改他人工单 | `PUT /tickets/:other` 403 |
| EU-08 | 不能进入 CMDB | `/configuration-items` 403 |
| EU-09 | 不能进入审计 | `/audit-logs` 403 |
| EU-10 | 服务满意度评价 | `POST /tickets/:id/rating` |

---

## 五、跨角色端到端业务流（必须通过）

| 流程 ID | 名称 | 角色链 | 步骤 | 通过判据 |
|---|---|---|---|---|
| FLOW-1 | 标准事件闭环 | end_user → engineer → sd_manager | 提单 → 接单 → 评论 → 解决 → 关闭 → 评分 | 各步状态码 200，最终状态 closed，SLA 计算正确 |
| FLOW-2 | P0 紧急升级 | end_user → engineer → manager | 提 P0 → 自动升级 → 主管审批 → 解决 | 触发自动审批节点，审计有升级记录 |
| FLOW-3 | 事件→问题→变更 | engineer → engineer → approver → engineer | 事件根因 → 提问题 → 提变更 → 审批 → 实施 → CIs 更新 | 三类工单建立 reference 关系，CMDB 字段变更被审计 |
| FLOW-4 | 服务请求带审批 | end_user → approver → engineer | 选服务 → 审批 → 交付 → 完成 | `requires_approval=true` 流转到审批人，期限 `expire_at` 生效 |
| FLOW-5 | 标准变更 | engineer → 自动 | 选标准变更模板 → 自动批 → 实施 | 跳过人工审批 |
| FLOW-6 | AI 分诊 + 知识引用 | end_user → AI → engineer | 提单 → AI Triage → 推荐 KB → engineer 采纳 | `/ai/audit` 记录 `accepted=true`，AI 建议字段全 |
| FLOW-7 | SLA 违约告警 | system → sd_manager | 预设短 SLA → 等待 → 告警触发 → 邮件/IM 推送 | `slaalerthistory` 有记录，连接器发送成功 |
| FLOW-8 | CMDB 影响分析 | engineer | 改某 CI → `/configuration-items/:id/impact-analysis` | 返回受影响下游 CI 列表 |
| FLOW-9 | 多租户隔离 | t1.admin & t2.admin | 同名工单各自创建 → 互查 0 命中 | 任一接口 cross-tenant 返回均为 0 |
| FLOW-10 | 知识库 RAG | end_user → AI | 自然语言提问 → `/ai/rag/ask` | 返回 KB 引用列表，无 KB 时降级关键字 |

---

## 六、API 冒烟矩阵（自动化）

> 全部以 `admin / admin123` token 调用；通过则进入下一阶段角色测试。

```bash
# 见 docs/scripts/smoke-api.sh（建议落地为 CI 步骤）
```

| 分组 | 端点 | 方法 | 期望 | 当前 |
|---|---|---|---|---|
| 健康 | `/api/v1/health` | GET | 200 | ✅ |
| 就绪 | `/api/v1/readiness/ga` | GET | 200，12/12 ready | ✅ |
| 认证 | `/api/v1/auth/login` | POST | 0 + token | ✅ |
| 菜单 | `/api/v1/auth/menus` | GET | 0 | ✅ |
| 工单 | `/api/v1/tickets` | GET/POST | 0 | ✅ |
| 事件 | `/api/v1/incidents` | GET/POST | 0 | ✅ |
| 问题 | `/api/v1/problems` | GET | 0 | ✅ |
| 变更 | `/api/v1/changes` | GET | 0 | ✅ |
| CI | `/api/v1/configuration-items` | GET/POST(`ciTypeId`) | 0 | ✅ |
| CI Type | `/api/v1/configuration-items/types` | GET | 200 | ✅ |
| KB | `/api/v1/knowledge-articles` | GET/POST | 0 | ✅ |
| 服务目录 | `/api/v1/service-catalogs` | GET | 0 | ✅ |
| SLA | `/api/v1/sla/definitions` | GET | 0 | ✅ |
| SLA 监控 | `/api/v1/sla/monitoring` | **POST** | 0 | ✅ |
| 连接器 | `/api/v1/connectors/lifecycle` | GET | 0，5 内置 | ✅ |
| 流程绑定 | `/api/v1/process-bindings` | GET | 0 | ✅ |
| 审批流 | `/api/v1/approval-workflows` | GET | 0 | ✅ |
| 工作流实例 | `/api/v1/workflow/instances` | GET | 0 | ✅ |
| AI Triage | `/api/v1/ai/triage` | POST | 0 | ✅ |
| AI 审计 | `/api/v1/ai/audit` | **POST** | 0 | ✅ |
| 仪表盘 | `/api/v1/dashboard/overview` | GET | 0 | ✅ |
| 工单分析 | `/api/v1/analytics/tickets` | GET | 0 | ✅ |
| 审计 | `/api/v1/audit-logs` | GET | 0 | ✅ |
| 用户/租户/角色 | `/users` `/tenants` `/roles` | GET | 0 | ✅ |

**已知不可达**（需开发确认）：

- `/api/v1/reports/cmdb-quality` 等 reports/* — 菜单引用但路由未注册
- `/api/v1/templates/tickets` — 推测真实路径不同
- `/api/v1/incident-rules` — 路径错或未启用
- `/api/v1/process-instances` — 路径错（实际 `/api/v1/workflow/instances`）

---

## 七、安全测试要点（OWASP Top 10）

| 类别 | 用例 | 期望 |
|---|---|---|
| A01 失效访问控制 | end_user 直接访问 `/audit-logs`、`/tenants` | 403 |
| A01 横向越权 | 用户 A 改用户 B 工单 | 403 |
| A02 加密失效 | JWT 必须签名且不可伪造 | 改 token 任意位 → 401 |
| A03 注入 | 工单 title 含 `' OR 1=1 --`、CI 名 `<script>` | 持久化但渲染时转义；DB 不被破坏 |
| A05 安全错配 | 生产 mode、CORS 白名单、Redis 鉴权 | `SERVER_MODE=release`，`ITSM_CORS_ALLOWED_ORIGINS` 显式，`REDIS_PASSWORD` 必填 |
| A07 鉴权失效 | 暴力登录 5 次 | 限流并审计 |
| A08 数据完整性 | 篡改请求 `tenant_id` 字段 | 服务端忽略客户端 tenant_id，强制使用 token 中值 |
| A09 日志缺失 | 关键操作有 `audit-logs` 记录 | 登录/工单状态变化/审批/角色变更全留痕 |
| A10 SSRF | 连接器 webhook 配置 URL `127.0.0.1:22` | 拒绝或限制内网地址 |

---

## 八、性能基线（建议）

| 场景 | 工具 | 阈值 |
|---|---|---|
| 工单列表（page=1,size=20） | k6 / wrk | P95 < 300ms @ 50 RPS |
| 工单创建 | k6 | P95 < 500ms @ 20 RPS |
| 仪表盘 overview | k6 | P95 < 600ms |
| AI Triage | k6 | P95 < 3s（受 LLM 影响，单独基线） |
| 长跑稳定性 | 24h soak | 内存无泄漏，错误率 < 0.1% |

---

## 九、自动化分层与执行命令

### 9.1 后端
```bash
cd itsm-backend
go test ./... -count=1                  # 单元
go test -cover ./service/... -run "..." # 覆盖率
go test ./integration/... -v            # 集成
```

### 9.2 前端
```bash
cd itsm-frontend
npm run type-check
npm run lint:check
npm run test:unit
npm run build
npx playwright test tests/e2e/comprehensive-e2e.spec.ts   # E2E
```

### 9.3 API 冒烟（推荐落 CI）
```bash
# 1) 启后端
DB_PASSWORD=itsm_password_2026 JWT_SECRET=dev /tmp/itsm-backend-bin &
# 2) 跑冒烟
bash docs/scripts/smoke-api.sh
```

### 9.4 角色 E2E（建议补充）
```bash
# 已有 comprehensive-e2e.spec.ts 13 项 → 扩到角色驱动 50+ 项
itsm-frontend/tests/e2e/roles/
  ├── super-admin.spec.ts
  ├── tenant-admin.spec.ts
  ├── sd-manager.spec.ts
  ├── engineer.spec.ts
  ├── approver.spec.ts
  ├── security.spec.ts
  └── end-user.spec.ts
```

---

## 十、缺陷分级与处置 SLA

| 级别 | 定义 | 处置 SLA | 上线判定 |
|---|---|---|---|
| P0 阻塞 | 核心流程不可用、数据丢失、安全事故 | 立即修 | 不允许遗留 |
| P1 严重 | 主流程降级、非核心模块崩溃 | 当天修 | ≤2 项 with workaround |
| P2 一般 | 非主流程小问题、UI 错位、文档漂移 | 1 周内修 | 允许进入 GA |
| P3 提示 | 优化建议、可读性 | 排期处理 | 允许进入 GA |

---

## 十一、当前已知风险（需在交付前评审）

| ID | 严重性 | 项 | 建议 |
|---|---|---|---|
| R-01 | P2 | GA Readiness 列出的部分 endpoint 路径与实际不一致 | 修 `router/ga_readiness.go` 中的 `Endpoint` 字段为 GET 可达路径 |
| R-02 | P1 | 前端单元覆盖仅 8% | 至少把 `lib/api/*` `lib/services/*` 拉到 60%+ |
| R-03 | P1 | Redis 默认无密码（部分日志 `NOAUTH`） | 生产 compose 强制 `REDIS_PASSWORD`，启动 fail-fast |
| R-04 | P2 | `itsm-nginx-prod` 容器在 Restarting | 修 nginx 配置或在 GA 镜像中移除 |
| R-05 | P2 | 多个菜单引用的 `/reports/*` 后端无对应路由 | 要么补后端，要么菜单先隐藏 |
| R-06 | P2 | 490 条 `no-console` lint 警告 | 上线前用 logger 替换或加 lint-disable 范围 |
| R-07 | P3 | 默认 seed 缺 `engineer1`/`manager1` 测试账号 | 加 seed 或文档化建账号步骤，便于 E2E |
| R-08 | P1 | 前端 prod 容器构建产物日期为 6/9 | 重新打包并 push GA 镜像 |

---

## 十二、交付物清单

- [x] 本测试方案文档
- [ ] `docs/scripts/smoke-api.sh` （可独立运行的全量 API 冒烟）
- [ ] `tests/e2e/roles/*.spec.ts` 角色 E2E 套件
- [ ] 缺陷追踪表（按 §10 分级）
- [ ] CI 工作流：`go test` + `npm test` + 启后端跑冒烟
- [ ] 性能基线报告（k6/wrk）
- [ ] 安全扫描报告（OWASP）

---

## 十三、执行排期建议

| 周 | 内容 |
|---|---|
| W1 D1-2 | 修 R-01~R-08 中的 P1，跑通冒烟 |
| W1 D3-4 | 写 7 个角色 E2E spec，并入 CI |
| W1 D5 | 跑 FLOW-1~10 端到端 |
| W2 D1-2 | 性能基线 + 安全 OWASP 扫描 |
| W2 D3 | 缺陷收口 + 回归 |
| W2 D4 | GA 发布候选构建（RC） |
| W2 D5 | RC 验收 → GA |

---

## 附录 A — 角色 × 权限速查（核心 API）

| 角色 | tickets | incidents | changes | configuration-items | audit-logs | users | tenants | sla |
|---|---|---|---|---|---|---|---|---|
| super_admin | RWD | RWD | RWD | RWD | R | RWD | RWD | RWD |
| tenant_admin | RWD | RWD | RWD | RWD | R(本租户) | RWD | R(本租户) | RWD |
| sd_manager | RWD | RWD | R | R | R | R | - | RW |
| engineer | RW(分配) | RW(分配) | RW(分配) | RW | - | R | - | R |
| approver | R | R | RW(审批) | R | - | R | - | R |
| security | R | R | R | R | R | R | R | R |
| end_user | RW(自有) | R(自有) | - | - | - | R(自己) | - | - |

> R=read，W=write，D=delete；空=无权限。

---

## 附录 B — 关键测试数据准备

```sql
-- 1. 至少 2 个租户用于隔离测试
INSERT INTO tenants(name, code, domain, status) VALUES
  ('Tenant A', 'TENA', 'a.example.com', 'active'),
  ('Tenant B', 'TENB', 'b.example.com', 'active');

-- 2. 每个租户至少 1 个工程师 + 1 个审批人
-- 走 /api/v1/users + 分配 role_id 即可

-- 3. 标准变更模板：seed 已有 5 条，自动审批
-- 4. SLA: seed 已有 6 条 (P0~P3 + service_request + change)
-- 5. KB: seed 已有 23 条
-- 6. 服务目录: seed 已有 22+5 条
```

---

**结论**：本方案以"角色 × 流程 × 风险"三维矩阵覆盖 ITSM v1.0 GA 全部交付目标。当前自动化基线已经具备（go test / jest / 现有 13 项 E2E 全过），关键差距集中在 §11 列出的 8 个风险项与 §6 列出的 4 个不可达端点。优先级 P0/P1 全部落实后，即可发起 RC → GA 流程。
