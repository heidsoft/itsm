-- 运营测试数据初始化
-- 执行方式: psql -h localhost -p 5432 -U dev -d itsm -f seed_test_data.sql
-- 密码: StrongPassword123!

-- =============================================
-- 1. 添加更多测试用户
-- =============================================
INSERT INTO users (username, email, name, department, phone, password_hash, active, role, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('engineer1', 'engineer1@example.com', '工程师小李', 'IT部门', '13800001111', '$2a$10$1234567890abcdefghijklmnopqrstuvwxyz', true, 'engineer', 1, NOW(), NOW()),
    ('engineer2', 'engineer2@example.com', '工程师小王', '运维部门', '13800002222', '$2a$10$1234567890abcdefghijklmnopqrstuvwxyz', true, 'engineer', 1, NOW(), NOW()),
    ('support1', 'support1@example.com', '客服小张', '客服部门', '13800003333', '$2a$10$1234567890abcdefghijklmnopqrstuvwxyz', true, 'end_user', 1, NOW(), NOW())
) AS v(username, email, name, department, phone, password_hash, active, role, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM users WHERE username = 'engineer1');

-- =============================================
-- 2. 添加更多工单测试数据
-- =============================================
INSERT INTO tickets (title, description, status, priority, ticket_number, tenant_id, requester_id, assignee_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('数据库连接超时问题', '生产环境数据库频繁出现连接超时，需要紧急处理', 'open', 'high', 'TKT-202602-000008', 1, 1, 2, NOW(), NOW()),
    ('用户无法登录系统', '部分用户反馈无法登录系统，提示密码错误', 'in_progress', 'urgent', 'TKT-202602-000009', 1, 2, 3, NOW(), NOW()),
    ('API接口响应慢', '调用订单接口响应时间超过5秒', 'assigned', 'medium', 'TKT-202602-000010', 1, 3, 2, NOW(), NOW()),
    ('文件上传功能异常', '上传大于10MB的文件时提示失败', 'pending', 'low', 'TKT-202602-000011', 1, 1, NULL, NOW(), NOW()),
    ('需要开通VPN账号', '新员工张三需要开通VPN访问权限', 'open', 'medium', 'TKT-202602-000012', 1, 4, NULL, NOW(), NOW()),
    ('打印机无法使用', '3楼打印机无法连接', 'resolved', 'low', 'TKT-202602-000013', 1, 5, 2, NOW(), NOW()),
    ('申请服务器资源', '需要申请2台4核8G服务器用于新项目', 'submitted', 'high', 'TKT-202602-000014', 1, 2, NULL, NOW(), NOW())
) AS v(title, description, status, priority, ticket_number, tenant_id, requester_id, assignee_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM tickets WHERE ticket_number = 'TKT-202602-000008');

-- =============================================
-- 3. 添加更多事件测试数据
-- =============================================
INSERT INTO incidents (title, description, status, priority, incident_number, source, type, is_major_incident, reporter_id, tenant_id, detected_at, created_at, updated_at)
SELECT * FROM (VALUES
    ('服务器CPU 100%', 'Web服务器CPU使用率持续100%，服务响应缓慢', 'new', 'critical', 'INC-202602-000003', 'monitoring', 'incident', false, 1, 1, NOW(), NOW(), NOW()),
    ('数据库主从延迟', '数据库主从同步延迟超过5分钟', 'investigating', 'high', 'INC-202602-000004', 'monitoring', 'incident', false, 1, 1, NOW(), NOW(), NOW()),
    ('网站首页无法访问', '用户报告网站首页无法打开', 'confirmed', 'critical', 'INC-202602-000005', 'user', 'incident', true, 2, 1, NOW(), NOW(), NOW()),
    ('支付接口报错', '调用支付接口返回500错误', 'in_progress', 'high', 'INC-202602-000006', 'user', 'incident', false, 3, 1, NOW(), NOW(), NOW())
) AS v(title, description, status, priority, incident_number, source, type, is_major_incident, reporter_id, tenant_id, detected_at, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM incidents WHERE incident_number = 'INC-202602-000003');

-- =============================================
-- 4. 添加更多问题测试数据
-- =============================================
INSERT INTO problems (title, description, status, priority, category, created_by, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('数据库连接池耗尽', '多次出现数据库连接池耗尽导致服务不可用', 'open', 'high', 'performance', 1, 1, NOW(), NOW()),
    ('内存泄漏问题', '应用服务存在内存泄漏，每隔24小时需要重启', 'analyzing', 'critical', 'performance', 1, 1, NOW(), NOW()),
    ('网络丢包严重', '跨机房网络经常丢包，影响服务质量', 'identified', 'medium', 'network', 1, 1, NOW(), NOW())
) AS v(title, description, status, priority, category, created_by, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM problems WHERE title = '数据库连接池耗尽');

-- =============================================
-- 5. 添加更多变更测试数据
-- =============================================
INSERT INTO changes (title, description, justification, type, status, priority, risk_level, created_by, tenant_id, planned_start_date, planned_end_date, created_at, updated_at)
SELECT * FROM (VALUES
    ('数据库版本升级', '将MySQL 5.7升级到8.0', '提升性能和安全性', 'standard', 'approved', 'medium', 'medium', 1, 1, NOW() + INTERVAL '1 day', NOW() + INTERVAL '2 days', NOW(), NOW()),
    ('服务器扩容', '增加2台应用服务器应对流量高峰', '业务增长需要', 'standard', 'pending', 'high', 'low', 1, 1, NOW() + INTERVAL '3 days', NOW() + INTERVAL '4 days', NOW(), NOW()),
    ('核心路由切换', '切换到新的核心路由器', '设备老化需要更换', 'emergency', 'in_progress', 'urgent', 'high', 1, 1, NOW(), NOW() + INTERVAL '6 hours', NOW(), NOW())
) AS v(title, description, justification, type, status, priority, risk_level, created_by, tenant_id, planned_start_date, planned_end_date, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM changes WHERE title = '数据库版本升级');

-- =============================================
-- 6. 添加更多服务请求测试数据
-- =============================================
INSERT INTO service_requests (title, status, requester_id, catalog_id, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('开通云服务器', 'pending_approval', 1, 2, 1, NOW(), NOW()),
    ('申请域名', 'in_progress', 2, 2, 1, NOW(), NOW()),
    ('开通CDN服务', 'approved', 3, 2, 1, NOW(), NOW()),
    ('SSL证书申请', 'completed', 4, 2, 1, NOW(), NOW())
) AS v(title, status, requester_id, catalog_id, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM service_requests WHERE title = '开通云服务器');

-- =============================================
-- 7. 添加更多知识库文章
-- =============================================
INSERT INTO knowledge_articles (title, content, category, status, author, author_id, views, tags, is_published, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('VPN连接配置指南', '详细介绍如何配置VPN连接，包括Windows和Mac系统...', '操作指南', 'published', 'admin', 1, 156, '["vpn","配置"]', true, 1, NOW(), NOW()),
    ('数据库备份恢复手册', '讲解MySQL数据库的备份和恢复操作步骤...', '运维手册', 'published', 'admin', 1, 89, '["数据库","备份"]', true, 1, NOW(), NOW()),
    ('服务器申请流程', '新服务器申请的完整流程说明...', '流程文档', 'published', 'admin', 1, 234, '["服务器","申请"]', true, 1, NOW(), NOW()),
    ('故障排查手册', '常见系统故障的排查和处理方法...', '运维手册', 'draft', 'admin', 1, 45, '["故障","排查"]', false, 1, NOW(), NOW())
) AS v(title, content, category, status, author, author_id, views, tags, is_published, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM knowledge_articles WHERE title = 'VPN连接配置指南');

-- =============================================
-- 8. 添加更多配置项
-- =============================================
INSERT INTO configuration_items (name, ci_type, ci_type_id, status, owner, environment, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('Web服务器-生产', 'server', 1, 'active', '运维部门', 'production', 1, NOW(), NOW()),
    ('数据库服务器-生产', 'server', 1, 'active', '运维部门', 'production', 1, NOW(), NOW()),
    ('Redis缓存服务器', 'server', 1, 'active', '运维部门', 'production', 1, NOW(), NOW()),
    ('负载均衡器', 'network', 4, 'active', '运维部门', 'production', 1, NOW(), NOW()),
    ('防火墙设备', 'security', 4, 'active', '安全部门', 'production', 1, NOW(), NOW())
) AS v(name, ci_type, ci_type_id, status, owner, environment, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM configuration_items WHERE name = 'Web服务器-生产');

-- =============================================
-- 9. 添加更多服务目录项
-- =============================================
INSERT INTO service_catalogs (name, description, status, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('云资源申请', '包括云服务器、存储、网络等资源申请', 'enabled', 1, NOW(), NOW()),
    ('安全服务', '包括SSL证书、防火墙规则等申请', 'enabled', 1, NOW(), NOW()),
    ('运维支持', '包括监控、备份、日志等服务', 'enabled', 1, NOW(), NOW())
) AS v(name, description, status, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM service_catalogs WHERE name = '云资源申请');

-- =============================================
-- 验证数据
-- =============================================
SELECT 'users:' as info, COUNT(*) as count FROM users
UNION ALL SELECT 'tickets:', COUNT(*) FROM tickets
UNION ALL SELECT 'incidents:', COUNT(*) FROM incidents
UNION ALL SELECT 'problems:', COUNT(*) FROM problems
UNION ALL SELECT 'changes:', COUNT(*) FROM changes
UNION ALL SELECT 'service_requests:', COUNT(*) FROM service_requests
UNION ALL SELECT 'knowledge_articles:', COUNT(*) FROM knowledge_articles
UNION ALL SELECT 'configuration_items:', COUNT(*) FROM configuration_items
UNION ALL SELECT 'service_catalogs:', COUNT(*) FROM service_catalogs;
