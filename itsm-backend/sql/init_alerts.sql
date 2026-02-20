-- 告警规则初始化数据
-- 执行方式: psql -h localhost -p 5432 -U dev -d itsm -f init_alerts.sql
-- 密码: StrongPassword123!

-- 告警规则初始化数据（只插入必需的字段）
INSERT INTO alert_rules (name, type, status, severity, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('CPU使用率过高', 'system', 'active', 'warning', 1, NOW(), NOW()),
    ('内存使用率过高', 'system', 'active', 'warning', 1, NOW(), NOW()),
    ('磁盘空间不足', 'system', 'active', 'critical', 1, NOW(), NOW()),
    ('网络延迟过高', 'network', 'active', 'warning', 1, NOW(), NOW()),
    ('数据库连接数过多', 'database', 'active', 'critical', 1, NOW(), NOW()),
    ('服务不可用', 'service', 'active', 'critical', 1, NOW(), NOW()),
    ('API响应时间过长', 'service', 'active', 'warning', 1, NOW(), NOW()),
    ('证书即将过期', 'security', 'active', 'warning', 1, NOW(), NOW())
) AS v(name, type, status, severity, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM alert_rules WHERE name = 'CPU使用率过高');

-- 验证数据
SELECT 'alert_rules:' as info, COUNT(*) as count FROM alert_rules
UNION ALL
SELECT 'alerts:', COUNT(*) FROM alerts;
