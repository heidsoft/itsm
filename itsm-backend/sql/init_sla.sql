-- SLA 服务等级协议初始化数据
-- 执行方式: psql -h localhost -p 5432 -U dev -d itsm -f init_sla.sql
-- 密码: StrongPassword123!

-- SLA 定义初始化数据
INSERT INTO sla_definitions (name, description, service_type, priority, response_time, resolution_time, is_active, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('SLA-P0-紧急', 'P0紧急级别SLA，15分钟响应，2小时解决', 'incident', 'urgent', 15, 120, true, 1, NOW(), NOW()),
    ('SLA-P1-高', 'P1高级别SLA，30分钟响应，4小时解决', 'incident', 'high', 30, 240, true, 1, NOW(), NOW()),
    ('SLA-P2-中', 'P2中级别SLA，2小时响应，8小时解决', 'incident', 'medium', 120, 480, true, 1, NOW(), NOW()),
    ('SLA-P3-低', 'P3低级别SLA，4小时响应，24小时解决', 'incident', 'low', 240, 1440, true, 1, NOW(), NOW()),
    ('SLA-服务请求', '服务请求标准SLA，8小时响应，3天解决', 'service_request', 'medium', 480, 4320, true, 1, NOW(), NOW()),
    ('SLA-变更', '变更请求SLA，1小时响应，24小时解决', 'change', 'high', 60, 1440, true, 1, NOW(), NOW())
) AS v(name, description, service_type, priority, response_time, resolution_time, is_active, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM sla_definitions WHERE name = 'SLA-P0-紧急');

-- SLA 告警规则初始化数据
INSERT INTO sla_alert_rules (name, alert_level, threshold_percentage, notification_channels, escalation_enabled, escalation_levels, is_active, tenant_id, sla_definition_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('SLA-P0-响应告警', 'warning', 50, '["email", "sms"]'::jsonb, true, '[{"level":1,"percentage":50,"after_minutes":8},{"level":2,"percentage":80,"after_minutes":12}]'::jsonb, true, 1, 1, NOW(), NOW()),
    ('SLA-P0-解决告警', 'critical', 80, '["email", "sms", "phone"]'::jsonb, true, '[{"level":1,"percentage":80,"after_minutes":96},{"level":2,"percentage":95,"after_minutes":114}]'::jsonb, true, 1, 1, NOW(), NOW()),
    ('SLA-P1-响应告警', 'warning', 50, '["email"]'::jsonb, true, '[{"level":1,"percentage":50,"after_minutes":15}]'::jsonb, true, 1, 2, NOW(), NOW()),
    ('SLA-P1-解决告警', 'warning', 80, '["email"]'::jsonb, true, '[{"level":1,"percentage":80,"after_minutes":192}]'::jsonb, true, 1, 2, NOW(), NOW()),
    ('SLA-P2-响应告警', 'info', 50, '["email"]'::jsonb, false, NULL, true, 1, 3, NOW(), NOW()),
    ('SLA-P2-解决告警', 'warning', 80, '["email"]'::jsonb, true, '[{"level":1,"percentage":80,"after_minutes":384}]'::jsonb, true, 1, 3, NOW(), NOW()),
    ('SLA-服务请求-响应告警', 'info', 50, '["email"]'::jsonb, false, NULL, true, 1, 5, NOW(), NOW()),
    ('SLA-变更-响应告警', 'warning', 50, '["email"]'::jsonb, true, '[{"level":1,"percentage":50,"after_minutes":30}]'::jsonb, true, 1, 6, NOW(), NOW())
) AS v(name, alert_level, threshold_percentage, notification_channels, escalation_enabled, escalation_levels, is_active, tenant_id, sla_definition_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM sla_alert_rules WHERE name = 'SLA-P0-响应告警');

-- 验证数据
SELECT 'sla_definitions:' as info, COUNT(*) as count FROM sla_definitions
UNION ALL
SELECT 'sla_alert_rules:', COUNT(*) FROM sla_alert_rules;
