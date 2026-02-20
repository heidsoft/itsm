-- 工单视图示例数据
-- 用于演示工单视图功能

INSERT INTO ticket_views (name, description, filters, columns, sort_config, is_shared, tenant_id, created_at, updated_at, created_by)
SELECT * FROM (VALUES
    ('我的待办', '分配给我的未完成工单',
     '{"status": ["open", "in_progress", "assigned"], "assignee_id": 1}'::jsonb,
     '["id", "title", "priority", "status", "updated_at"]'::jsonb,
     '{"field": "updated_at", "order": "desc"}'::jsonb,
     false, 1, NOW(), NOW(), 1),
    ('全部待处理', '所有未处理工单',
     '{"status": ["open", "pending"]}'::jsonb,
     '["id", "title", "priority", "requester", "created_at"]'::jsonb,
     '{"field": "priority", "order": "asc"}'::jsonb,
     true, 1, NOW(), NOW(), 1),
    ('我提交的', '我创建的工单',
     '{"requester_id": 1}'::jsonb,
     '["id", "title", "status", "assignee", "created_at"]'::jsonb,
     '{"field": "created_at", "order": "desc"}'::jsonb,
     false, 1, NOW(), NOW(), 1),
    ('紧急工单', '所有紧急和严重级别工单',
     '{"priority": ["urgent", "critical"]}'::jsonb,
     '["id", "title", "priority", "status", "assignee", "sla_deadline"]'::jsonb,
     '{"field": "created_at", "order": "desc"}'::jsonb,
     true, 1, NOW(), NOW(), 1),
    ('已超时工单', '已超过SLA截止时间的工单',
     '{"is_overdue": true}'::jsonb,
     '["id", "title", "priority", "status", "sla_deadline", "overdue_hours"]'::jsonb,
     '{"field": "sla_deadline", "order": "asc"}'::jsonb,
     true, 1, NOW(), NOW(), 1)
) AS v(name, description, filters, columns, sort_config, is_shared, tenant_id, created_at, updated_at, created_by)
WHERE NOT EXISTS (SELECT 1 FROM ticket_views WHERE name = v.name);

-- 验证工单视图
SELECT id, name, is_shared FROM ticket_views;
