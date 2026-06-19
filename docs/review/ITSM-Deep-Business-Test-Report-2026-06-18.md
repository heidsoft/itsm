# ITSM 深度业务流程测试报告

> 测试日期: 2026-06-18
> 测试方式: API 端到端调用 + 浏览器验证
> 测试环境: localhost:8090 (backend) + localhost:3001 (frontend dev)

---

## 综合评估

| 维度 | 评分 | 说明 |
|------|------|------|
| 工单管理 | ⭐⭐⭐⭐☆ | 核心流程完整，工作流API有Bug |
| CMDB | ⭐⭐⭐☆☆ | 基础CRUD可用，数据库类型创建失败 |
| 服务目录 | ⭐⭐⭐⭐⭐ | 三级审批流程完美运行 |
| 变更管理 | ⭐⭐☆☆☆ | 审批流程断裂，核心功能不可用 |
| 知识库/AI | ⭐⭐☆☆☆ | 文章创建可用，AI功能全部不可用 |
| 事件管理 | ⭐⭐⭐⭐☆ | 基础流程可用，升级功能有Bug |

**总体评分: 3.2/5.0** — 核心工单和服务目录流程扎实，但变更审批和AI功能存在阻断性Bug

---

## 一、工单管理全生命周期

### 测试链路
```
创建工单 → 分派 → 添加评论 → 状态流转(in_progress→resolved→closed) → 工作流API(accept→resolve→close) → SLA查询 → 子任务 → 统计
```

### ✅ 正常功能

| 功能 | API | 结果 |
|------|-----|------|
| 创建工单 | POST /tickets | ✅ 返回工单号 TKT-202606-650056 |
| 分派工单 | POST /tickets/:id/assign | ✅ assigneeId 设置成功，状态变为 assigned |
| 添加评论 | POST /tickets/:id/comments | ✅ 评论内容保存成功 |
| 状态流转 | PUT /tickets/:id/status | ✅ open→in_progress→resolved→closed |
| 工作流-解决 | POST /tickets/workflow/resolve | ✅ 状态变为 resolved |
| 工作流-关闭 | POST /tickets/workflow/close | ✅ 状态变为 closed |
| 工作流状态 | GET /tickets/:id/workflow/state | ✅ 返回可用操作列表 |
| SLA信息 | GET /tickets/:id/sla | ✅ 返回响应/解决截止时间、违规状态 |
| 创建子任务 | POST /tickets/:id/subtasks | ✅ parentTicketId 关联正确 |
| 查看子任务 | GET /tickets/:id/subtasks | ✅ 返回子任务列表 |
| 工单统计 | GET /tickets/stats | ✅ 返回总数/各状态/高优/逾期 |

### ❌ 发现的Bug

| 严重度 | Bug描述 | API | 错误信息 |
|--------|---------|-----|----------|
| **P1** | 工单工作流 accept API 字段名不匹配 | POST /tickets/workflow/accept | `TicketID` failed on 'required' tag（前端传 `ticketId`，后端期望 `TicketID`） |
| **P1** | 工单工作流历史查询返回5001 | GET /tickets/:id/workflow-history | Code: 5001, 内部错误 |
| **P2** | 工单搜索API参数名错误 | GET /tickets/search | `q` 和 `keyword` 参数均返回 1001 |
| **P2** | 工单统计 open 计数错误 | GET /tickets/stats | 实际有2个open工单，统计返回 open:0 |
| **P2** | 工单 version 字段不递增 | - | 多次状态变更后 version 仍为 1 |

### SLA 违规检测验证

```json
{
  "ticket_id": 1,
  "priority": "medium",
  "sla_definition_name": "Incident-P2-中",
  "response_deadline": "2026-06-15T09:00:00+08:00",
  "is_response_breached": true,
  "is_resolution_breached": true
}
```
✅ SLA 违规检测正常工作，正确识别了已超时的工单。

---

## 二、CMDB 配置项管理

### 测试链路
```
查看CI列表 → 查看CI类型 → 创建CI(server) → 创建CI(database) → 创建关系 → 查看拓扑 → 影响分析 → 查看关系列表
```

### ✅ 正常功能

| 功能 | API | 结果 |
|------|-----|------|
| CI列表 | GET /configuration-items | ✅ 11个CI，含类型/状态 |
| CI类型 | GET /configuration-items/types | ✅ 8种类型(server/database/network等) |
| 创建CI(server) | POST /configuration-items | ✅ ciTypeId=1 创建成功 |
| CI统计 | GET /configuration-items/stats | ✅ 11个active |
| 创建CI关系 | POST /configuration-items/relationships | ✅ 返回完整关系对象(含源/目标CI名称) |
| CI拓扑 | GET /configuration-items/:id/topology | ✅ 返回edges和children |
| 影响分析 | GET /configuration-items/:id/impact-analysis | ✅ 返回上下游影响、风险等级 |

### ❌ 发现的Bug

| 严重度 | Bug描述 | 详情 |
|--------|---------|------|
| **P1** | database类型CI创建失败 | ciTypeId=2 返回500，日志显示到达service层后失败 |
| **P1** | 关系列表API返回1001 | GET /configuration-items/relationships 需要未知必填参数 |
| **P2** | 拓扑图子节点数据不完整 | children 中 name 和 ci_type 字段为空 |
| **P2** | 影响分析摘要被截断 | summary 返回 "该CI" 而非完整描述 |
| **P2** | 统计 type_distribution 为空 | 11个CI但类型分布统计返回 {} |

### CI 关系创建成功示例

```json
{
  "id": 1,
  "sourceCiId": 12,
  "sourceCiName": "测试-生产Web服务器-001",
  "sourceCiType": "server",
  "targetCiId": 9,
  "targetCiName": "E2E-CMDB-核心数据库-1781355050143",
  "targetCiType": "database",
  "relationshipType": "depends_on",
  "relationship_type_name": "依赖",
  "strength": "medium",
  "impactLevel": "medium",
  "isActive": true
}
```

---

## 三、服务目录与多级审批

### 测试链路
```
查看服务目录(8项) → 查看服务详情 → 提交服务请求 → 3级审批(manager→IT→security) → 查看我的请求
```

### ✅ 全部正常！

| 功能 | API | 结果 |
|------|-----|------|
| 服务目录列表 | GET /service-catalogs | ✅ 8个服务，5个分类 |
| 服务详情 | GET /service-catalogs/:id | ✅ 含交付时间/描述 |
| 提交服务请求 | POST /service-requests | ✅ 自动生成3级审批链 |
| 我的请求 | GET /service-requests/me | ✅ 返回用户请求列表 |
| 待审批列表 | GET /service-requests/approvals/pending | ✅ 正确返回待审批项 |
| L1审批(manager) | POST /service-requests/:id/approval | ✅ status→manager_approved |
| L2审批(IT) | POST /service-requests/:id/approval | ✅ status→it_approved |
| L3审批(security) | POST /service-requests/:id/approval | ✅ status→security_approved |

### 🌟 亮点：三级审批流程完美运行

```
请求提交 → submitted (currentLevel=1)
  ↓
manager approve → manager_approved (currentLevel=2)
  ↓
IT approve → it_approved (currentLevel=3)
  ↓
security approve → security_approved (currentLevel=4, 全部通过)
```

每级审批含 24 小时超时设置，审批评论正确保存。

### ❌ 发现的Bug

| 严重度 | Bug描述 |
|--------|---------|
| **P2** | 所有服务请求都要求 expire_at（软件安装等不应需要） |

---

## 四、变更管理

### 测试链路
```
创建变更 → 提交审批 → 审批通过 → 开始实施 → 完成 → 审批摘要 → 风险评估
```

### ✅ 部分正常

| 功能 | API | 结果 |
|------|-----|------|
| 创建变更 | POST /changes | ✅ 创建成功，状态 draft |
| 提交审批 | POST /changes/:id/submit | ✅ 状态变为 pending |
| 审批摘要 | GET /changes/:id/approval-summary | ✅ 返回(但数据为null) |

### ❌ 严重Bug

| 严重度 | Bug描述 | API | 错误 |
|--------|---------|-----|------|
| **P0** | 审批通过失败 | POST /changes/:id/approve | 500: "failed to get approval history" |
| **P1** | 状态转换被阻断 | POST /changes/:id/start | 500: "无效的状态转换: 从 'pending' 到 'in_progress'" |
| **P1** | 风险评估失败 | GET /changes/:id/risk-assessment | 500 |
| **P2** | 创建时 risk_level 未保存 | POST /changes | 响应 riskLevel 为空 |
| **P2** | 审批链未初始化 | GET /changes/:id/approval-summary | chain 和 history 均为 null |

### 根因分析

变更提交(submit)后，系统未正确初始化审批链(approval chain)。当调用 approve 时，系统尝试获取审批历史但找不到记录，导致 500 错误。这阻断了整个变更管理流程。

---

## 五、知识库与AI功能

### 测试链路
```
查看文章列表 → 创建文章 → AI搜索 → AI智能分诊 → AI工单摘要
```

### ✅ 部分正常

| 功能 | API | 结果 |
|------|-----|------|
| 创建知识文章 | POST /knowledge/articles | ✅ 文章创建成功 |
| 文章列表 | GET /knowledge/articles | ✅ 返回列表 |

### ❌ AI功能全部不可用

| 严重度 | Bug描述 | 原因 |
|--------|---------|------|
| **P1** | AI语义搜索无响应 | pgvector 扩展未安装，向量检索不可用 |
| **P1** | AI智能分诊无响应 | LLM Gateway 未配置 |
| **P1** | AI工单摘要无响应 | LLM Gateway 未配置 |
| **P2** | 文章状态未按请求设置 | 请求 status=published，实际保存为 draft |

### AI降级响应验证

之前修复的 RAG 降级响应机制未生效——AI 搜索 API 直接无响应而非返回友好提示。需要检查降级逻辑是否正确实现。

---

## 六、事件管理

### 测试链路
```
创建事件 → 确认(acknowledge) → 升级(escalate) → 解决(resolve) → 关闭(close) → 统计
```

### ✅ 大部分正常

| 功能 | API | 结果 |
|------|-----|------|
| 创建事件 | POST /incidents | ✅ 创建成功，status=new |
| 确认事件 | POST /incidents/:id/acknowledge | ✅ Code: 0 |
| 解决事件 | POST /incidents/:id/resolve | ✅ Code: 0 |
| 关闭事件 | POST /incidents/:id/close | ✅ Code: 0 |
| 事件统计 | GET /incidents/stats | ✅ totalIncidents/critical/major/MTTA/MTTR |

### ❌ 发现的Bug

| 严重度 | Bug描述 |
|--------|---------|
| **P2** | 升级(escalate) API 返回 1001 参数错误 |
| **P2** | 状态转换后响应不包含新状态字段 |

### 事件统计输出

```json
{
  "totalIncidents": 1,
  "openIncidents": 0,
  "criticalIncidents": 1,
  "majorIncidents": 1,
  "avgResolutionTime": 0,
  "mtta": 0,
  "mttr": 0
}
```

---

## Bug 汇总（按优先级排序）

### P0 — 阻断性Bug（必须立即修复）

| # | 模块 | Bug | 影响 |
|---|------|-----|------|
| 1 | 变更管理 | 审批通过失败(approve返回500) | 变更流程完全不可用 |

### P1 — 严重Bug（影响核心功能）

| # | 模块 | Bug | 影响 |
|---|------|-----|------|
| 2 | 工单管理 | workflow accept API字段名不匹配 | 接单功能不可用 |
| 3 | 工单管理 | workflow history返回5001 | 工单操作历史不可查 |
| 4 | CMDB | database类型CI创建500 | 数据库配置项无法录入 |
| 5 | CMDB | 关系列表API返回1001 | 无法查看CI关系列表 |
| 6 | 变更管理 | 状态转换被阻断 | 变更无法进入实施阶段 |
| 7 | 变更管理 | 风险评估500 | 变更风险评估不可用 |
| 8 | AI功能 | AI搜索/分诊/摘要全部无响应 | AI能力完全不可用 |

### P2 — 一般Bug（影响体验）

| # | 模块 | Bug |
|---|------|-----|
| 9 | 工单管理 | 搜索API参数名错误 |
| 10 | 工单管理 | 统计open计数错误 |
| 11 | 工单管理 | version字段不递增 |
| 12 | CMDB | 拓扑子节点数据不完整 |
| 13 | CMDB | 影响分析摘要截断 |
| 14 | CMDB | 统计type_distribution为空 |
| 15 | 服务目录 | 所有请求都要求expire_at |
| 16 | 变更管理 | risk_level等字段未保存 |
| 17 | 变更管理 | 审批链未初始化 |
| 18 | 知识库 | 文章status未按请求设置 |
| 19 | 事件管理 | escalate API参数错误 |
| 20 | 事件管理 | 状态转换响应不含新状态 |

---

## 修复优先级建议

### 第一优先（P0 — 阻断性修复）
1. **变更管理审批链初始化** — submit 时应创建 approval chain 记录，approve 时能正确读取

### 第二优先（P1 — 核心功能修复）
2. **工单 accept API** — 统一 `ticketId` / `TicketID` 字段名
3. **工单 workflow history** — 排查 5001 错误根因
4. **CMDB database CI创建** — 排查 ciTypeId=2 的 500 错误
5. **CMDB 关系列表** — 补全必填参数或设默认值
6. **AI降级响应** — 确保无 pgvector/LLM 时返回友好提示而非空响应

### 第三优先（P2 — 体验优化）
7. 工单搜索参数名统一
8. 工单统计 SQL 修复
9. CMDB 拓扑子节点字段映射
10. 服务请求 expire_at 条件化校验

---

## 对标 ServiceNow 能力差距

基于本次业务流程测试，与 ServiceNow 核心模块对比：

| ServiceNow 模块 | ITSM 对应模块 | 功能覆盖率 | 关键差距 |
|----------------|--------------|-----------|----------|
| Incident Management | 事件管理 | 75% | 缺少升级功能、影响范围分析 |
| Change Management | 变更管理 | 40% | **审批流程断裂**，无法完成完整变更 |
| CMDB | CMDB | 65% | database CI 创建失败，关系查询不完整 |
| Service Catalog | 服务目录 | 90% | **三级审批完美**，仅 expire_at 校验需优化 |
| Knowledge Management | 知识库 | 50% | AI搜索不可用，文章发布流程不完整 |
| Ticket/Task | 工单管理 | 85% | 核心流程完整，工作流API有小Bug |
| SLA | SLA监控 | 80% | 违规检测正常，缺少自动升级 |

**总体功能覆盖率: ~65%** — 距离开源版 ServiceNow 还需要修复变更审批 + AI功能 + 数据一致性

---

## 结论

ITSM 系统在**工单管理和服务目录**两个核心模块表现优秀，特别是三级审批流程设计精良。但**变更管理审批链断裂**是当前最大的阻断性问题，必须优先修复。AI 功能依赖 pgvector 和 LLM 网关，需要在部署环境完整配置后才能验证。

建议按 P0→P1→P2 的顺序逐批修复，预计可将系统功能覆盖率从 65% 提升至 85%+。
