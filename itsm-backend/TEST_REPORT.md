# ITSM 后端 API 测试报告

## 测试日期
2026-03-05

## 测试结果汇总

| 序号 | 功能模块 | 测试路径 | 状态 | 备注 |
|------|----------|----------|------|------|
| 1 | 登录认证 | POST /api/v1/auth/login | ✅ 正常 | |
| 2 | 用户管理 | GET /api/v1/users/1 | ✅ 正常 | |
| 3 | 工单列表 | GET /api/v1/tickets | ✅ 正常 | |
| 4 | 创建工单 | POST /api/v1/tickets | ✅ 正常 | |
| 5 | 事件管理 | GET /api/v1/incidents | ✅ 正常 | |
| 6 | 问题管理 | GET /api/v1/problems | ✅ 正常 | |
| 7 | 变更管理 | GET /api/v1/changes | ✅ 正常 | |
| 8 | 服务目录 | GET /api/v1/service-catalogs | ✅ 正常 | 原路径错误 |
| 9 | 知识库 | GET /api/v1/knowledge/articles | ✅ 正常 | |
| 10 | 仪表盘 | GET /api/v1/dashboard/stats | ✅ 正常 | |
| 11 | CMDB | GET /api/v1/cmdb/cis | ✅ 正常 | |
| 12 | 审批流程 | GET /api/v1/approval-workflows | ⚠️ 权限不足 | 需要权限 |
| 13 | 工作流 | GET /api/v1/tickets/workflow/* | ✅ 正常 | |
| 14 | SLA 监控 | GET /api/v1/sla/definitions | ✅ 正常 | |
| 15 | 团队管理 | GET /api/v1/org/teams | ✅ 正常 | 原路径错误 |
| 16 | 角色权限 | GET /api/v1/roles | ✅ 正常 | |
| 17 | 租户管理 | GET /api/v1/tenants | ⚠️ 权限不足 | 需要超级管理员 |
| 18 | 部门管理 | GET /api/v1/org/departments/tree | ✅ 正常 | 原路径错误 |
| 19 | 统计分析 | GET /api/v1/tickets/stats | ✅ 正常 | |
| 20 | 系统配置 | GET /api/v1/system-configs | ⚠️ 权限不足 | 需要管理员权限 |
| 21 | AI 功能 | POST /api/v1/ai/chat | ✅ 正常 | |
| 22 | 通知 | GET /api/v1/notifications | ⚠️ 参数错误 | |
| 23 | 工单评论 | GET /api/v1/tickets/1/comments | ✅ 正常 | |
| 24 | 工单附件 | GET /api/v1/tickets/1/attachments | ✅ 正常 | |
| 25 | Swagger | GET /swagger/index.html | ✅ 正常 | |
| 26 | 健康检查 | GET /api/v1/health | ✅ 正常 | 原路径错误 |

## 修复的问题

1. **godotenv 支持** - 添加了自动加载 .env 文件
2. **Team seed 错误** - 添加了 Code 字段自动生成
3. **ProcessBinding seed 错误** - 修复了必填边缘问题
4. **ticket_types 检查** - 添加了表存在性检查
5. **路由冲突** - 修复了 /tickets/analytics 被 /:id 匹配的问题
6. **向量数据库** - 启用了 pgvector 扩展

## 正确的 API 路径参考

### 常用路径
- 健康检查: `/api/v1/health`
- 版本信息: `/api/v1/version`
- 服务目录: `/api/v1/service-catalogs`
- 团队: `/api/v1/org/teams`
- 部门: `/api/v1/org/departments/tree`
- 系统配置: `/api/v1/system-configs`
- 审批工作流: `/api/v1/approval-workflows`

### 工单相关
- 工单列表: `GET /api/v1/tickets`
- 工单统计: `GET /api/v1/tickets/stats`
- 创建工单: `POST /api/v1/tickets`
- 工单详情: `GET /api/v1/tickets/:id`
- 工单评论: `GET /api/v1/tickets/:id/comments`

### 需要注意
- 大部分管理功能需要管理员权限
- 某些 API 需要额外的查询参数
