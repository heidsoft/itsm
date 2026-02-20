-- 自动化规则示例数据
-- 用于演示工单自动化规则功能

INSERT INTO ticket_automation_rules (name, description, priority, conditions, actions, is_active, execution_count, tenant_id, created_at, updated_at, created_by)
SELECT * FROM (VALUES
    ('紧急工单自动分配', '紧急级别工单自动分配给值班工程师', 1,
     '{"priority": ["urgent", "critical"], "status": "open"}'::jsonb,
     '{"assign_to_group": "oncall", "notify_assignee": true}'::jsonb,
     true, 0, 1, NOW(), NOW(), 1),
    ('超时自动升级', '工单超过24小时未处理自动升级', 2,
     '{"status": ["open", "assigned"], "hours_elapsed": 24}'::jsonb,
     '{"escalate": true, "notify_manager": true, "add_tag": "超时"}'::jsonb,
     true, 0, 1, NOW(), NOW(), 1),
    ('高优先级提醒', '高优先级工单创建时通知值班经理', 3,
     '{"priority": ["high"]}'::jsonb,
     '{"notify_role": "manager", "add_tag": "需要关注"}'::jsonb,
     true, 0, 1, NOW(), NOW(), 1),
    ('重复工单标记', '相同标题工单自动标记关联', 4,
     '{"similar_title_exists": true}'::jsonb,
     '{"add_tag": "重复", "link_similar": true}'::jsonb,
     false, 0, 1, NOW(), NOW(), 1)
) AS v(name, description, priority, conditions, actions, is_active, execution_count, tenant_id, created_at, updated_at, created_by)
WHERE NOT EXISTS (SELECT 1 FROM ticket_automation_rules WHERE name = v.name);

-- 验证自动化规则
SELECT id, name, priority, is_active FROM ticket_automation_rules ORDER BY priority;
