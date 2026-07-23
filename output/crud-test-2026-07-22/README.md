# ITSM 按钮级 CRUD 测试报告

- **测试时间**：2026-07-22 21:36–22:00 (GMT+8)
- **测试目标**：`http://localhost`（生产模式 Docker Compose）
- **测试范围**：13 模块全 CRUD（API 层 + 浏览器按钮级双通道）
- **产物**：本目录 `api-crud-report.json`、`api_crud_test.py`、`browser-*.png`

---

## 一句话结论

**发现 P0 生产阻断级 bug：前端所有"新建/编辑/删除"按钮 100% 提交失败**，根因是 CSRF 接口字段命名 snake_case / camelCase 不一致。API 层直连是好的（62/66 通过），但用户界面走不通。

---

## 1. API 层 CRUD（62/66 通过 = 93.9%）

| 模块 | LIST | CREATE | READ | UPDATE | DELETE | 备注 |
|---|:-:|:-:|:-:|:-:|:-:|---|
| Tickets | ✅ | ✅ | ✅ | ✅ | ✅ | |
| Incidents | ✅ | ✅ | ✅ | ✅ | ✅ | |
| Problems | ✅ | ✅ | ✅ | ✅ | ✅ | description 必填 |
| Changes | ✅ | ✅ | ✅ | ✅ | ✅ | |
| CMDB-CIs | ✅ | ✅ | ✅ | ✅ | ✅ | |
| Knowledge | ✅ | ✅ | ✅ | ✅ | ✅ | |
| ServiceCatalog | ✅ | ✅ | ✅ | ✅ | ✅ | status 枚举 `enabled/disabled` |
| Users | ✅ | ✅ | ✅ | ✅ | ✅ | |
| Roles | ✅ | ✅ | ✅ | ✅ | ✅ | |
| Groups | ✅ | ✅ | ✅ | ✅ | ✅ | |
| SLA-Definitions | ✅ | ✅ | ✅ | ✅ | ✅ | 路径 `/sla/definitions` |
| Departments | ✅ | ✅ | ❌ | ❌ | ✅ | 无 GET 单条；PUT 未传 name 触发 Ent 长度校验 |
| Teams | ✅ | ✅ | ❌ | ❌ | ✅ | 同上 |
| Workflows-BPMN | ✅ | — | — | — | — | 只测 LIST |

---

## 2. 浏览器按钮级测试（❌ 全部失败）

**场景**：登录 admin → 打开 `/tickets` → 点"新建工单" → 填表 → 点"创建工单"

| 步骤 | 期望 | 实际 |
|---|---|---|
| 登录 | 200 到 dashboard | ✅ 200 |
| 打开新建表单 | 页面渲染 | ✅ 正常 |
| 填标题+描述+优先级 | 表单接收 | ✅ 已填 |
| 点"创建工单"按钮 | 后端 200，跳工单列表 | ❌ **后端 403 Forbidden**，页面卡在 `/tickets/create` |

**后端日志实证**：
```
[GIN] POST "/api/v1/tickets"   403   4.46ms
[GIN] POST "/api/v1/tickets"   403   4.57ms
```

---

## 3. 🔥 P0 根因：CSRF 前后端字段命名不一致

### 现场证据

- **后端返回**（`GET /api/v1/csrf-token`）：
  ```json
  {"code":0,"data":{"csrf_token":"9nG7F_jfSjaOz9T1lywHhQ=="},"message":"success"}
  ```

- **前端读取**（`itsm-frontend/src/lib/security.ts`）：
  ```ts
  if (data.code === 0 && data.data?.csrfToken) {   // ← 读 camelCase
      csrfProtection.privateToken = data.data.csrfToken;
  }
  ```

- **结果**：`data.data.csrfToken === undefined` → `getToken` 返回 null → `X-CSRF-Token` header **永不发送** → CSRF 中间件拒绝 → 403。

### 反向验证（手动补 header）

在浏览器 console 里补上 `X-CSRF-Token: csrf_token` 后重发同一请求：
```json
{"code":0,"message":"success","data":{"id":13,"title":"CSRF-bypass-test",...}}
```
—— **同样的 payload、同样的 cookie，只多一个 header，立刻 200**。根因锁定，无异议。

### 影响面

- ✅ **GET 列表/详情/搜索** 不受影响（CSRF 只拦 mutating）
- ❌ **所有 POST/PUT/DELETE/PATCH** 通过前端 UI 走不通
- ❌ 用户完全无法：新建/编辑/删除工单、事件、问题、变更、CI、知识、目录、用户、角色、SLA……
- ❌ 触发审批、状态推进、评论、上传附件——全部阻断

### 修复方案（二选一）

**方案 A（推荐）：修前端**——一行改动
```ts
// itsm-frontend/src/lib/security.ts:第 ~30 行
if (data.code === 0 && data.data?.csrf_token) {           // 改成蛇形
    csrfProtection.privateToken = data.data.csrf_token;   // 同上
}
```

**方案 B：修后端**——保持前端 camelCase 契约
```go
// itsm-backend/router/router.go 或 csrf handler：
c.JSON(200, gin.H{"code": 0, "data": gin.H{"csrfToken": token}})
```

**建议 A**：改动最小，且与本项目"后端 snake_case + DTO Mapper 转 camelCase"的通用约定一致（说明 CSRF 端点漏了 Mapper）。

---

## 4. 已发现的其它 bug（复现）

### P1 CMDB CI 版本查询 NULL 崩溃
- 打开 CMDB 详情或点编辑触发：
  ```
  service/configuration_item_service.go:142  Failed to get max CI version
    error: converting NULL to int is unsupported, ci_id: 3
  ```
- 修复：`SELECT COALESCE(MAX(version), 0)` 或用 `sql.NullInt64`

### P2 Departments/Teams 无 GET 单条 & PUT 未保留 name
- `GET /departments/{id}` 返回 404（路由缺失）
- `PUT /org/departments/{id}` 只传 `description` 时，Ent 校验 `Department.name` 长度不足 → 500
- 修复：① 补 GET 路由；② handler 里对未传字段跳过 Set，或做局部更新

### P3 backend 容器 healthcheck 474 连击（沿用上轮）
- 容器无 wget，healthcheck 用 wget → 一直 unhealthy

### P4 `/api/v1/healthz` 404 vs `/api/v1/health` 200

### P5 前端 auth-storage.token 为 null
- Zustand 持久化里 token 是 null（依赖 HttpOnly cookie），刷新页面后如果 cookie 也过期，会掉登录 & 无法诊断

---

## 5. 修复优先级建议

| # | Bug | 级别 | 用户影响 | 修复工作量 |
|---|---|---|---|---|
| 1 | CSRF 字段命名不一致 | **P0** | **UI 100% 无法写数据** | 5 分钟（改一行） |
| 2 | CMDB CI 版本 NULL | P1 | CMDB 编辑主链路挂 | 15 分钟 |
| 3 | Dept/Team GET 单条 & 部分更新 | P2 | 编辑功能不可用 | 30 分钟 |
| 4 | backend 容器 healthcheck | P3 | orchestration 只是脏，业务不受影响 | 10 分钟 |
| 5 | /healthz 别名 | P4 | 监控告警噪音 | 5 分钟 |

---

## 6. 截图索引

| 场景 | 文件 |
|---|---|
| 登录成功 | `browser-01-login-ok.png` |
| 工单列表 | `browser-02-tickets-list.png` |
| 新建工单表单 | `browser-03-tickets-new-form.png` |
| 提交后仍停在表单页（403） | `browser-04-ticket-created.png` |
| CSRF 403 现场 | `browser-05-csrf-bug-403.png` |
| CMDB CI 详情（触发 NULL bug） | `browser-06-cmdb-ci-detail.png` |

---

## 7. 复现脚本

- **API 层 CRUD**：`python3 output/crud-test-2026-07-22/api_crud_test.py`（需先跑一次登录写 `/tmp/itsm_token`）
- **浏览器复现 CSRF**：
  ```bash
  agent-browser open http://localhost/login
  agent-browser fill e9 admin
  agent-browser fill e10 'AdminProd2026!'
  agent-browser click e8         # 登录
  agent-browser open http://localhost/tickets/create
  agent-browser fill e50 "复现测试"
  agent-browser fill e51 "描述"
  agent-browser click e48        # 创建 → 观察 backend 日志会出现 403
  ```
