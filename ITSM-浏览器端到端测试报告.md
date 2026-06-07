# ITSM 浏览器端到端测试报告（终版）

测试时间：2026-06-07
测试工具：Playwright + Chrome（headless）
测试账号：admin/admin123、user1/user123、security1/security123
测试范围：27 菜单项 + 40+ 业务页面 + 30+ API 端点
截图证据：80+ 张（/tmp/itsm-test-*.png）

## 总体结论：发现 1 个 P0 + 11 个 P1/P2 Bug

**admin 视角**：27 模块全可访问、CRUD 流真实持久化
**非 admin 视角**：登录后 100% 页面崩溃（端用户/安全审批人等所有角色）
**AI 助手**：链路通但 RAG 检索无召回
**多角色协作**：核心"普通用户提交工单"场景直接断裂

---

## ⚠️ P0 致命 Bug：非管理员用户登录后 100% 页面崩溃

**严重程度**：P0 — 阻塞所有非管理员用户使用系统

**复现**：
1. 用 `user1/user123` 或 `security1/security123` 登录
2. 任意点击菜单（含 dashboard）
3. **所有页面**显示"页面出现错误" + 重试按钮

**证据**：
- 截图：`/tmp/itsm-test-99-u1-crash.png`
- 8 个路由 100% 崩溃：`/dashboard /tickets /tickets/create /incidents /knowledge /service-catalog /my-requests /profile`
- API 日志：`GET /api/v1/auth/tenants -> 403 code 2003 "权限不足"`
- 前端报错：`TypeError: Cannot read properties of null (reading 'map')` in `app/(main)/layout`

**根因**：
- `itsm-backend/router/router.go:833` 重复注册 `GET /api/v1/auth/tenants`（在 tenant 分组内 → 触发 `middleware/rbac.go:881`）
- `itsm-backend/router/auth_routes.go:29` 也注册同路径（无 RBAC）
- Gin 命中带 RBAC 的那条 → 端用户被拦
- 前端拿到 null 后 `.map()` 抛错 → ErrorBoundary 接管

**修复**：删除 `router.go:833` 的 `authGrp.GET("/tenants", ...)` 重复注册

**业务影响**：除 super_admin 外的所有角色完全无法使用系统

---

## 11 个 P1/P2 Bug

### Bug 2：服务目录"申请"按钮死链
- 位置：`itsm-frontend/src/app/(main)/service-catalog/components/ServiceItemCard.tsx:175`
- 现象：22 张服务卡"申请"全部 404
- 修复：新建 request 页面或改跳到 detail

### Bug 3：CSRF 误伤 CMDB/知识库 POST
- 现象：`POST /configuration-items` 与 `POST /knowledge/articles` 返回 403 CSRF missing
- 同样走代理的 `/tickets`、`/incidents` 都正常
- 修复：CSRF skip 列表加白名单

### Bug 4：AI 助手 RAG 完全无召回
- 知识库 8 篇文章，但 AI 对任何中文问题都返回"没有找到答案"
- 怀疑：向量库为空 / 检索阈值过高 / embedding 未加载
- 验证：`SELECT COUNT(*) FROM knowledge_vectors;`

### Bug 5：API 路径命名不统一
- 前端 `/cmdb/types`，后端 `/configuration-items/types`
- 前端 `/admin/tenants`，后端无此路径
- 建议：统一别名

### Bug 6：知识库详情页 404
- 位置：`itsm-frontend/src/app/(main)/knowledge/page.tsx:178`
- 现象：点击文章标题跳到 `/knowledge/articles/${id}` 但该路由不存在
- 影响：8 篇文章全部无法查看

### Bug 7：知识库新建页 404
- 位置：`itsm-frontend/src/app/(main)/knowledge/page.tsx:290`
- 现象："新建文章"按钮跳到 `/knowledge/articles/new` 但该路由不存在
- 影响：知识库只能看不能写

### Bug 8：服务请求创建需要"合规确认"字段
- 现象：`POST /api/v1/service-requests` 返回 500 "Compliance acknowledgement required"
- 但 UI 申请表单没渲染该字段
- 影响：服务申请 100% 失败

### Bug 9：工单分类页是 dev 占位文案
- 位置：`/tickets/types` 页面
- 现象：页面显示"当前版本为了清零 TS 阻塞，移除了对 legacy business 组件链路的依赖"
- 影响：明显是开发遗留的占位文案

### Bug 10：多租户管理 API 404
- 现象：`GET /api/v1/admin/tenants` 404
- 实际路径在 `/tenants` 下，前端调用路径不对
- 影响：租户管理功能不完整

### Bug 11：工单详情"采纳建议"按钮无副作用
- 现象：点击"采纳建议"只显示提示，无 API 调用
- 应调用 PATCH 真正写入分类/优先级

### Bug 12：CSRF 也阻挡评论接口
- 现象：`POST /tickets/7/comments` 返回 403 "CSRF token missing"
- 但 `POST /tickets` 正常工作
- 修复：CSRF skip 列表需补全

---

## admin 视角真实业务验证

### 登录 + 导航
- ✅ POST /api/v1/auth/login → 200，27 个菜单项可访问

### 工单管理
- 列表 200，6 条历史 + 表格
- 创建 #7 成功，POST 200
- 详情：状态/优先级/描述/SLA/AI 建议(55%)/8 个操作按钮
- 状态流转 open→in_progress PUT 200 真实持久化
- 分配模态加载 4 个候选人，POST 403（"您是此工单的申请人"）
- 评论、历史被 CSRF 拦截

### 事件管理
- 列表 200，11 条
- 详情 #14 完整（编辑/升级/解决/返回）
- 创建事件 POST 200，#14 入库

### 问题/变更/CMDB
- 问题 9 条，详情 #10
- 变更 9 条，详情 #12 完整（审批流 200）
- 变更新建 14 字段（title/description/justification/type/priority/计划时间/影响范围/风险等级/受影响 CI/实施计划/回滚计划）
- 已知错误 5 条（SSL 证书过期/内存泄漏/网络延迟/磁盘空间不足）
- CMDB 概览 2 配置项
- CMDB 创建表单 25 字段
- POST /configuration-items → 403 CSRF

### 服务目录
- 22 个服务卡片，5 大分类
- **Bug 2**：每张卡"申请"按钮 404
- **Bug 8**：申请 API 需要 Compliance 字段

### 知识库
- 8 篇文章，7 分类
- **Bug 6/7**：详情/新建页都 404
- 模板：2 模板、平均 4.3/5 评分

### SLA
- 7 条定义，0 违规，100% 合规率
- 监控大屏 30 秒刷新
- 仪表盘 KPI 实时（总工单 7、待处理 2、处理中 2、已完成 1、平均响应 2.5h）

### 工作流 + 审批
- 16 个工作流
- BPMN 设计器加载（Process_1 100%）
- 工单审批流设计器加载
- 待审批：当前无待办（已处理完成）

### 报表（6/6 正常）
- 总览：总工单 7、已解决 1(14.3%)
- CMDB 质量：平均完整度 82.5%
- SLA 性能：平均合规率 94.4%
- 事件趋势、问题效率、变更成功率 全部 200

### AI 助手 / RAG
- 链路打通（POST 200）
- 但 RAG 检索无召回（任何问题都"抱歉，没有找到相关的答案"）

### 管理后台（10/10 正常）
用户 4 / 角色 20 / 组 20 / 部门 14 / 团队 18 / 审批 3 / 租户 2 / SLA 配置 7 / 工单分类 9 / CI 类型 / 工作流管理

---

## 修复优先级

1. **P0（立刻）**：删除 `router.go:833` 重复的 `/auth/tenants` 注册
2. **P1**：知识库详情 + 新建页（`/knowledge/articles/[id]` + `/knowledge/articles/new`）
3. **P1**：服务目录"申请"按钮跳到 detail 或新建 request 页面
4. **P1**：CSRF skip 列表加 `POST /configuration-items`、`POST /knowledge/articles`、`POST /tickets/:id/comments`
5. **P1**：服务请求创建 UI 加 Compliance acknowledgement 勾选项
6. **P2**：AI 助手向量库初始化
7. **P2**：工单分类页去掉 dev 占位文案
8. **P2**：工单详情"采纳建议"按钮接入 PATCH
9. **P2**：API 路径统一（`/admin/tenants` vs `/tenants`）
