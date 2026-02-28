-- 自动化规则示例数据
-- 用于演示工单自动化规则功能
-- 共 12 条规则，覆盖常见自动化场景
-- 最后更新：2026-02-28

INSERT INTO ticket_automation_rules (name, description, priority, conditions, actions, is_active, execution_count, tenant_id, created_at, updated_at, created_by)
SELECT * FROM (VALUES
    -- P1: 紧急工单处理
    ('紧急工单自动分配', '紧急级别工单自动分配给值班工程师', 10,
     '[{"field": "priority", "operator": "equals", "value": "urgent"}]'::jsonb,
     '[{"type": "auto_assign"}, {"type": "send_notification", "content": "您有新的紧急工单，请立即处理"}]'::jsonb,
     true, 0, 1, NOW(), NOW(), 1),
    
    -- P1: 超时升级
    ('超时自动升级', '工单超过 24 小时未处理自动升级优先级', 9,
     '[{"field": "status", "operator": "in", "value": ["open", "assigned"]}]'::jsonb,
     '[{"type": "escalate"}, {"type": "send_notification", "content": "工单已超时，优先级已升级"}]'::jsonb,
     true, 0, 1, NOW(), NOW(), 1),
    
    -- P1: VIP 客户工单
    ('VIP 客户工单优先处理', 'VIP 客户的工单自动设置为高优先级并通知经理', 8,
     '[{"field": "priority", "operator": "equals", "value": "high"}]'::jsonb,
     '[{"type": "set_priority", "priority": "urgent"}, {"type": "send_notification", "content": "VIP 客户工单，请优先处理"}]'::jsonb,
     true, 0, 1, NOW(), NOW(), 1),
    
    -- P2: 工单分类自动设置
    ('邮件类工单自动分类', '来自邮件渠道的工单自动归类', 7,
     '[]'::jsonb,
     '[{"type": "set_category", "category_id": 1}]'::jsonb,
     true, 0, 1, NOW(), NOW(), 1),
    
    -- P2: 未分配工单提醒
    ('未分配工单提醒', '工单创建 30 分钟未分配自动通知主管', 6,
     '[{"field": "assignee_id", "operator": "equals", "value": 0}]'::jsonb,
     '[{"type": "send_notification", "content": "工单尚未分配，请及时处理"}]'::jsonb,
     true, 0, 1, NOW(), NOW(), 1),
    
    -- P2: 已解决工单自动关闭
    ('已解决工单自动关闭', '状态为已解决超过 7 天的工单自动关闭', 5,
     '[{"field": "status", "operator": "equals", "value": "resolved"}]'::jsonb,
     '[{"type": "set_status", "status": "closed"}]'::jsonb,
     true, 0, 1, NOW(), NOW(), 1),
    
    -- P3: 低优先级工单批量分配
    ('低优先级工单批量分配', '低优先级工单自动分配给空闲工程师', 4,
     '[{"field": "priority", "operator": "equals", "value": "low"}]'::jsonb,
     '[{"type": "auto_assign"}]'::jsonb,
     true, 0, 1, NOW(), NOW(), 1),
    
    -- P3: 特定关键词工单
    ('安全相关工单高优处理', '标题包含安全相关关键词的工单自动设为高优', 3,
     '[{"field": "status", "operator": "equals", "value": "open"}]'::jsonb,
     '[{"type": "set_priority", "priority": "high"}, {"type": "send_notification", "content": "安全相关工单，请优先处理"}]'::jsonb,
     true, 0, 1, NOW(), NOW(), 1),
    
    -- P3: 部门工单自动分配
    ('财务部工单自动分配', '财务部提交的工单自动分配给财务支持组', 2,
     '[{"field": "department_id", "operator": "equals", "value": 2}]'::jsonb,
     '[{"type": "assign", "user_id": 5}]'::jsonb,
     true, 0, 1, NOW(), NOW(), 1),
    
    -- P4: 工单满意度调查
    ('工单关闭后发送满意度调查', '工单关闭后自动发送满意度调查通知', 1,
     '[{"field": "status", "operator": "equals", "value": "closed"}]'::jsonb,
     '[{"type": "send_notification", "content": "请对本次服务进行评价"}]'::jsonb,
     false, 0, 1, NOW(), NOW(), 1),
    
    -- P4: 重复工单检测
    ('重复工单标记', '相似标题工单自动标记为可能重复', 0,
     '[]'::jsonb,
     '[{"type": "send_notification", "content": "发现相似工单，请确认是否重复"}]'::jsonb,
     false, 0, 1, NOW(), NOW(), 1),
    
    -- P4: 夜间工单自动响应
    ('夜间工单自动响应', '非工作时间创建的工单发送自动回复', -1,
     '[]'::jsonb,
     '[{"type": "send_notification", "content": "我们已收到您的工单，将在工作时间尽快处理"}]'::jsonb,
     true, 0, 1, NOW(), NOW(), 1)
) AS v(name, description, priority, conditions, actions, is_active, execution_count, tenant_id, created_at, updated_at, created_by)
WHERE NOT EXISTS (SELECT 1 FROM ticket_automation_rules WHERE name = v.name);

-- 验证自动化规则
SELECT id, name, priority, is_active FROM ticket_automation_rules ORDER BY priority DESC;
