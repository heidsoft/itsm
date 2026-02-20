# ITSM系统迭代计划

> 最后更新: 2026-02-16
> 当前迭代: 3 (已完成前端体验优化)
> 状态: 迭代一、二、三已完成

---

## 系统现状总结

### ✅ 已完成模块 (2026-02-16)
| 模块 | 状态 | 说明 |
|------|------|------|
| 核心工单 | 完整 | CRUD、状态流转、分配、审批 |
| 事件管理 | 完整 | 事件创建、处理、升级 |
| 问题管理 | 完整 | 问题根因分析、KB关联 |
| 变更管理 | 完整 | 变更审批、风险评估 |
| 用户体系 | 完整 | 6用户、3部门、2团队、3角色 |
| SLA定义 | 完整 | 6个SLA定义，已启动监控服务(5分钟间隔) |
| BPMN流程 | 完整 | 6个流程绑定 |
| 审批工作流 | 完整 | 4个审批流程配置 |
| 工单视图 | 完整 | 5个预置视图 |
| 告警规则 | 完整 | 8条告警规则数据 |
| 前端API | 完整 | 50+ API客户端、hooks完善 |
| 前端优化 | 完整 | 搜索防抖(300ms)、筛选hook整合 |

### ⚠️ 待完善模块
| 模块 | 状态 | 说明 |
|------|------|------|
| 自动化规则 | 表结构存在 | Ent未生成，缺初始化数据 |
| SLA违规记录 | 空数据 | 暂无SLA超时记录 |
| 统计分析 | 基础 | 缺少高级报表 |
| 实时通知 | 基础 | WebSocket未集成 |
| 移动端适配 | 基础 | 响应式布局待优化 |
| AI知识问答 | 基础 | RAG服务已实现，界面待完善 |

---

## 迭代一：数据补充与核心功能修复 ✅
**目标**: 补充缺失数据，修复核心流程断点

### 1.1 初始化BPMN流程绑定
```sql
-- sql/init_process_binding.sql
INSERT INTO process_bindings (business_type, business_sub_type, process_definition_key, process_version, is_default, priority, is_active, tenant_id, created_at, updated_at)
VALUES
    ('ticket', NULL, 'ticket_general_flow', '1.0.0', true, 1, true, 1, NOW(), NOW()),
    ('incident', NULL, 'incident_emergency_flow', '1.0.0', true, 1, true, 1, NOW(), NOW()),
    ('change', 'standard', 'change_normal_flow', '1.0.0', true, 1, true, 1, NOW(), NOW()),
    ('change', 'emergency', 'change_normal_flow', '1.0.0', false, 2, true, 1, NOW(), NOW()),
    ('problem', NULL, 'problem_management_flow', '1.0.0', true, 1, true, 1, NOW(), NOW()),
    ('service_request', NULL, 'service_request_flow', '1.0.0', true, 1, true, 1, NOW(), NOW());
```

### 1.2 添加自动化规则示例
```sql
-- sql/init_automation_rules.sql
INSERT INTO ticket_automation_rules (name, description, trigger_type, conditions, actions, priority, is_active, tenant_id, created_at, updated_at)
VALUES
    ('紧急工单自动分配', '紧急级别工单自动分配给值班组', 'ticket_created', '{"priority":["urgent","critical"]}', '{"assign_to_group":"oncall"}', 1, true, 1, NOW(), NOW()),
    ('超时自动升级', '工单超过24小时未处理自动升级', 'scheduled', '{"status":["open","assigned"],"hours_elapsed":24}', '{"escalate":true,"notify_manager":true}', 2, true, 1, NOW(), NOW()),
    ('重复工单合并', '相同标题工单自动标记关联', 'ticket_created', '{"similar_title":true}', '{"auto_link":true}', 3, true, 1, NOW(), NOW());
```

### 1.3 添加工单视图示例
```sql
-- sql/init_ticket_views.sql
INSERT INTO ticket_views (name, description, filters, columns, is_default, is_shared, user_id, tenant_id, created_at, updated_at)
VALUES
    ('我的待办', '分配给我的未完成工单', '{"assignee_id":$user_id,"status":["open","in_progress"]}', '["id","title","priority","status","updated_at"]', true, false, NULL, 1, NOW(), NOW()),
    ('全部待处理', '所有未处理工单', '{"status":["open","pending"]}', '["id","title","priority","requester","created_at"]', false, true, NULL, 1, NOW(), NOW()),
    ('我提交的', '我创建的工单', '{"requester_id":$user_id}', '["id","title","status","assignee","created_at"]', true, false, NULL, 1, NOW(), NOW());
```

### 1.4 修复审批流程断点
- 检查工单创建时是否正确触发审批
- 检查变更创建时是否正确触发审批
- 添加审批记录查看接口

---

## 迭代二：SLA功能完善
**目标**: 完善SLA展示与预警

### 2.1 工单详情页添加SLA信息展示
```typescript
// 前端: tickets/[ticketId]/page.tsx
// 添加SLA信息卡片
<div className="sla-info-card">
  <Progress percent={slaProgress} status={slaStatus} />
  <div>响应截止: {formatDate(ticket.slaResponseDeadline)}</div>
  <div>解决截止: {formatDate(ticket.slaResolutionDeadline)}</div>
</div>
```

### 2.2 添加SLA定时检查任务
```go
// service/sla_monitor_service.go
// 添加StartSLAWatcher方法，启动定时检查
func (s *SLAMonitorService) StartSLAWatcher(ctx context.Context, interval time.Duration) {
    ticker := time.NewTicker(interval)
    for {
        select {
        case <-ticker.C:
            // 检查所有租户的SLA
            s.CheckAllTenantsSLA(ctx)
        case <-ctx.Done():
            return
        }
    }
}
```

### 2.3 SLA预警通知
- 接近截止时间时发送站内通知
- 违规时发送邮件通知

---

## 迭代三：前端体验优化
**目标**: 统一API封装，优化加载体验

### 3.1 统一API封装
```
lib/api/
├── http-client.ts      # 主HTTP客户端（保留）
├── base-api-handler.ts # 请求处理器
└── api.ts             # 统一导出
```
- 移除重复的base-api.ts, base-api-v2.ts
- 统一使用http-client.ts

### 3.2 加载状态优化
- 统一Loading组件
- 添加骨架屏
- 添加错误边界

### 3.3 完善工单列表筛选
- 保存筛选条件到localStorage
- 添加快捷筛选标签

---

## 迭代四：自动化与通知增强 ⏳
**目标**: 完善自动化规则引擎和实时通知系统

### 4.1 运行Ent生成automation_rules表
```bash
cd itsm-backend
go generate ./ent/schema/ticket_automation_rule.go
go generate ./ent
go build
```

### 4.2 初始化自动化规则数据
```sql
-- 运行 sql/init_automation_rules.sql
-- 4条示例规则已准备好
```

### 4.3 完善自动化规则服务
- service/ticket_automation_rule_service.go
- 规则匹配和执行逻辑

### 4.4 WebSocket实时通知
- 添加WebSocket端点
- 通知推送服务

### 4.5 前端通知中心
- 通知列表页面
- 已读/未读状态

### 4.6 邮件通知集成
- SMTP配置
- 模板邮件发送

---

## 迭代五：统计分析报表 ⏳
**目标**: 完善数据分析和报表导出功能

### 5.1 工单趋势统计API
- 按时间/类型/状态统计
- MTTR/MTTF计算

### 5.2 SLA合规率报表
- 各SLA定义达成率
- 趋势图表

### 5.3 统计页面UI
- 仪表盘图表
- ECharts集成

### 5.4 Excel导出
- 工单数据导出
- 批量操作

### 5.5 PDF报表生成
- 报表模板
- PDF下载

---

## 迭代六：知识库增强 ⏳
**目标**: 完善知识库和AI功能

### 6.1 知识库文章编辑器
- Markdown/富文本
- 版本历史

### 6.2 RAG问答集成
- 向量检索
- LLM调用

### 6.3 AI智能推荐
- 相关知识推荐
- 智能摘要

---

## 迭代七：移动端与体验优化 ⏳
**目标**: 提升移动端体验

### 7.1 响应式布局审计
### 7.2 移动端导航优化
### 7.3 PWA支持

---

## 迭代八：生产准备 ⏳
**目标**: 完善监控与测试

### 4.1 日志持久化
```go
// config.yaml
logger:
  level: info
  output: file
  file:
    path: /var/log/itsm/app.log
    max_size: 100
    max_backups: 10
```

### 4.2 添加Prometheus监控
```go
// middleware/metrics.go
var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )
)
```

### 4.3 添加单元测试
```
service/
├── ticket_service_test.go
├── approval_service_test.go
└── sla_service_test.go
```

---

## 迭代五：高级功能
**目标**: 智能化与自动化

### 5.1 智能工单分类
- 基于历史数据训练分类模型
- 自动推荐分类/优先级

### 5.2 自动摘要
- 工单创建时自动生成摘要
- 事件分析自动生成报告

### 5.3 预测分析
- 工单量预测
- SLA合规率预测

---

## 任务清单

| 迭代 | 任务 | 预估工作量 | 优先级 |
|------|------|------------|--------|
| 1 | 初始化process_bindings | 0.5d | P0 |
| 1 | 添加自动化规则示例 | 0.5d | P0 |
| 1 | 添加工单视图示例 | 0.5d | P0 |
| 2 | 工单详情SLA展示 | 1d | P1 |
| 2 | SLA定时检查 | 1d | P1 |
| 2 | SLA预警通知 | 1d | P1 |
| 3 | API封装统一 | 1d | P2 |
| 3 | 加载状态优化 | 1d | P2 |
| 4 | 日志持久化 | 1d | P2 |
| 4 | 监控指标 | 2d | P2 |
| 4 | 单元测试 | 3d | P2 |
| 5 | 智能分类 | 3d | P3 |
| 5 | 自动摘要 | 2d | P3 |
| 5 | 预测分析 | 3d | P3 |

---

## 总结

本迭代计划分为5个阶段：
- **迭代一(1-2周)**: 数据补充 + 核心修复
- **迭代二(2-3周)**: SLA功能完善
- **迭代三(3-4周)**: 前端体验优化
- **迭代四(4-5周)**: 生产准备
- **迭代五(6-8周)**: 高级功能

建议优先执行迭代一，修复核心数据问题后再进行功能迭代。
