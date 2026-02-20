-- 扩展测试数据 - 用于生产环境数据填充
-- 执行方式: psql -h localhost -p 5432 -U dev -d itsm -f extend_test_data.sql

-- =============================================
-- 1. 添加更多工单（15条）
-- =============================================
INSERT INTO tickets (title, description, status, priority, ticket_number, tenant_id, requester_id, assignee_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('邮件服务器故障', '企业邮件服务器无法收发邮件，影响全员工作', 'open', 'critical', 'TKT-202602-000016', 1, 2, NULL, NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours'),
    ('OA系统登录异常', 'OA系统登录页面无法打开', 'in_progress', 'high', 'TKT-202602-000017', 1, 4, 5, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
    ('财务系统报表无法生成', '财务报表功能报错', 'assigned', 'medium', 'TKT-202602-000018', 1, 6, 4, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
    ('会议室预约系统故障', '无法预约会议室', 'pending', 'low', 'TKT-202602-000019', 1, 2, NULL, NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),
    ('视频会议系统卡顿', '视频会议经常卡顿断线', 'resolved', 'medium', 'TKT-202602-000020', 1, 4, 5, NOW() - INTERVAL '7 days', NOW() - INTERVAL '2 days'),
    ('密码忘记申请重置', '忘记系统密码需要重置', 'closed', 'low', 'TKT-202602-000021', 1, 6, 4, NOW() - INTERVAL '10 days', NOW() - INTERVAL '9 days'),
    ('增加邮箱配额', '邮箱存储空间不足申请扩容', 'open', 'medium', 'TKT-202602-000022', 1, 2, NULL, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
    ('网站内容更新', '官网产品介绍页面需要更新内容', 'assigned', 'low', 'TKT-202602-000023', 1, 4, 5, NOW() - INTERVAL '4 days', NOW() - INTERVAL '4 days'),
    ('新建域账号', '新员工入职需要创建域账号', 'resolved', 'medium', 'TKT-202602-000024', 1, 6, 4, NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 day'),
    ('打印机驱动安装', '新打印机需要安装驱动', 'closed', 'low', 'TKT-202602-000025', 1, 2, 5, NOW() - INTERVAL '6 days', NOW() - INTERVAL '5 days'),
    ('系统权限申请', '申请财务系统查看权限', 'submitted', 'medium', 'TKT-202602-000026', 1, 4, NULL, NOW() - INTERVAL '3 hours', NOW() - INTERVAL '3 hours'),
    ('数据导出需求', '需要导出上月业务数据报表', 'in_progress', 'high', 'TKT-202602-000027', 1, 6, 4, NOW() - INTERVAL '5 hours', NOW() - INTERVAL '5 hours'),
    ('应用服务器磁盘空间不足', '服务器磁盘即将满载需要清理', 'open', 'critical', 'TKT-202602-000028', 1, 2, NULL, NOW() - INTERVAL '1 hours', NOW() - INTERVAL '1 hours'),
    ('域名解析异常', '部分域名无法解析', 'investigating', 'high', 'TKT-202602-000029', 1, 4, 5, NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '30 minutes'),
    ('备份任务失败', '昨晚数据库备份任务执行失败', 'in_progress', 'critical', 'TKT-202602-000030', 1, 2, 4, NOW() - INTERVAL '4 hours', NOW() - INTERVAL '4 hours')
) AS v(title, description, status, priority, ticket_number, tenant_id, requester_id, assignee_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM tickets WHERE ticket_number = 'TKT-202602-000016');

-- =============================================
-- 2. 添加更多事件（6条）
-- =============================================
INSERT INTO incidents (title, description, status, priority, incident_number, source, type, is_major_incident, reporter_id, tenant_id, detected_at, created_at, updated_at)
SELECT * FROM (VALUES
    ('数据库CPU持续高负载', '数据库服务器CPU使用率超过90%持续超过30分钟', 'investigating', 'critical', 'INC-202602-000007', 'monitoring', 'incident', true, 1, 1, NOW() - INTERVAL '1 hour', NOW() - INTERVAL '1 hour', NOW() - INTERVAL '1 hour'),
    ('第三方API调用失败率上升', '调用外部支付API失败率从1%上升到15%', 'confirmed', 'high', 'INC-202602-000008', 'monitoring', 'incident', false, 1, 1, NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours'),
    ('CDN节点异常', '华南地区CDN节点响应缓慢', 'in_progress', 'medium', 'INC-202602-000009', 'monitoring', 'incident', false, 1, 1, NOW() - INTERVAL '3 hours', NOW() - INTERVAL '3 hours', NOW() - INTERVAL '3 hours'),
    ('Kubernetes集群节点故障', 'K8s集群一个工作节点宕机', 'resolved', 'high', 'INC-202602-000010', 'monitoring', 'incident', false, 1, 1, NOW() - INTERVAL '5 hours', NOW() - INTERVAL '5 hours', NOW() - INTERVAL '2 hours'),
    ('SSL证书即将过期', '多个域名SSL证书将在3天后过期', 'identified', 'medium', 'INC-202602-000011', 'monitoring', 'incident', false, 1, 1, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
    ('日志采集延迟', '日志采集延迟超过10分钟', 'new', 'low', 'INC-202602-000012', 'monitoring', 'incident', false, 1, 1, NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '30 minutes')
) AS v(title, description, status, priority, incident_number, source, type, is_major_incident, reporter_id, tenant_id, detected_at, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM incidents WHERE incident_number = 'INC-202602-000007');

-- =============================================
-- 3. 添加更多问题（4条）
-- =============================================
INSERT INTO problems (title, description, status, priority, category, created_by, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('API接口超时问题', '多个API接口存在超时问题，影响核心业务流程', 'investigating', 'critical', 'performance', 1, 1, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
    ('缓存一致性问题', '缓存与数据库数据不一致导致显示异常', 'identified', 'high', 'data', 1, 1, NOW() - INTERVAL '5 days', NOW() - INTERVAL '5 days'),
    ('定时任务执行慢', '多个定时任务执行时间过长影响系统性能', 'analyzing', 'medium', 'performance', 1, 1, NOW() - INTERVAL '7 days', NOW() - INTERVAL '7 days'),
    ('日志存储空间增长过快', '日志存储空间每周增长50%，需要优化', 'open', 'low', 'storage', 1, 1, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days')
) AS v(title, description, status, priority, category, created_by, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM problems WHERE title = 'API接口超时问题');

-- =============================================
-- 4. 添加更多变更（5条）
-- =============================================
INSERT INTO changes (title, description, justification, type, status, priority, risk_level, created_by, tenant_id, planned_start_date, planned_end_date, created_at, updated_at)
SELECT * FROM (VALUES
    ('应用服务版本升级', '将Spring Boot应用从2.5升级到3.0', '安全漏洞修复和性能提升', 'standard', 'draft', 'medium', 'medium', 1, 1, NOW() + INTERVAL '7 days', NOW() + INTERVAL '8 days', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
    ('负载均衡策略调整', '修改负载均衡算法从轮询改为加权最小连接', '提升资源利用率', 'standard', 'approved', 'medium', 'low', 1, 1, NOW() + INTERVAL '2 days', NOW() + INTERVAL '3 days', NOW() - INTERVAL '3 days', NOW() - INTERVAL '2 days'),
    ('Redis集群扩容', 'Redis集群从3节点扩容到5节点', '应对业务增长', 'standard', 'pending_review', 'high', 'medium', 1, 1, NOW() + INTERVAL '5 days', NOW() + INTERVAL '6 days', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days'),
    ('紧急安全补丁', '修复Apache Log4j高危漏洞', '安全漏洞修复', 'emergency', 'in_progress', 'critical', 'high', 1, 1, NOW(), NOW() + INTERVAL '2 hours', NOW() - INTERVAL '1 hour', NOW() - INTERVAL '1 hour'),
    ('数据库参数优化', '调整MySQL缓冲池大小和连接数参数', '性能优化', 'standard', 'scheduled', 'medium', 'low', 1, 1, NOW() + INTERVAL '3 days', NOW() + INTERVAL '4 days', NOW() - INTERVAL '5 days', NOW() - INTERVAL '4 days')
) AS v(title, description, justification, type, status, priority, risk_level, created_by, tenant_id, planned_start_date, planned_end_date, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM changes WHERE title = '应用服务版本升级');

-- =============================================
-- 5. 添加更多服务请求（6条）
-- =============================================
INSERT INTO service_requests (title, description, status, priority, request_number, requester_id, catalog_id, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('开通对象存储OSS', '申请开通阿里云OSS存储服务用于文件存储', 'pending_approval', 'medium', 'SR-202602-000006', 2, 2, 1, NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours'),
    ('申请负载均衡SLB', '申请开通负载均衡服务用于流量分发', 'approved', 'high', 'SR-202602-000007', 4, 2, 1, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day'),
    ('开通消息队列MQ', '申请开通RabbitMQ消息队列服务', 'in_progress', 'medium', 'SR-202602-000008', 6, 2, 1, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days'),
    ('申请NAT网关', '申请开通NAT网关用于内网访问外网', 'completed', 'medium', 'SR-202602-000009', 2, 2, 1, NOW() - INTERVAL '5 days', NOW() - INTERVAL '2 days'),
    ('开通日志服务SLS', '申请开通日志服务用于日志收集分析', 'pending_approval', 'low', 'SR-202602-000010', 4, 2, 1, NOW() - INTERVAL '4 hours', NOW() - INTERVAL '4 hours'),
    ('申请数据库读写分离', '申请开通数据库读写分离实例', 'draft', 'high', 'SR-202602-000011', 6, 2, 1, NOW() - INTERVAL '6 hours', NOW() - INTERVAL '6 hours')
) AS v(title, description, status, priority, request_number, requester_id, catalog_id, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM service_requests WHERE request_number = 'SR-202602-000006');

-- =============================================
-- 6. 添加更多知识库文章（6条）
-- =============================================
INSERT INTO knowledge_articles (title, content, category, status, author, author_id, views, tags, is_published, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('Kubernetes集群运维手册', '详细介绍K8s集群的日常运维操作，包括节点管理、Pod调度、故障排查等...', '运维手册', 'published', 'admin', 1, 345, '["kubernetes","运维","容器"]', true, 1, NOW() - INTERVAL '10 days', NOW() - INTERVAL '5 days'),
    ('MySQL慢查询优化指南', '讲解如何分析慢查询日志并进行SQL优化，提升数据库性能...', '性能优化', 'published', 'admin', 1, 567, '["mysql","性能","优化"]', true, 1, NOW() - INTERVAL '15 days', NOW() - INTERVAL '10 days'),
    ('Redis缓存使用最佳实践', '介绍Redis缓存的使用场景、过期策略、内存管理等问题...', '架构设计', 'published', 'admin', 1, 289, '["redis","缓存","架构"]', true, 1, NOW() - INTERVAL '20 days', NOW() - INTERVAL '15 days'),
    ('Nginx配置详解', '详细讲解Nginx的配置语法和常用场景，包括负载均衡、反向代理、SSL配置等...', '运维手册', 'published', 'admin', 1, 456, '["nginx","配置","运维"]', true, 1, NOW() - INTERVAL '25 days', NOW() - INTERVAL '20 days'),
    ('Gitlab CI/CD流水线配置', '介绍如何使用Gitlab CI/CD进行自动化构建和部署...', 'DevOps', 'draft', 'admin', 1, 123, '["gitlab","ci","cd"]', false, 1, NOW() - INTERVAL '5 days', NOW() - INTERVAL '3 days'),
    ('Docker容器化部署流程', '讲解应用容器化的完整流程和最佳实践...', 'DevOps', 'published', 'admin', 1, 234, '["docker","容器","部署"]', true, 1, NOW() - INTERVAL '30 days', NOW() - INTERVAL '25 days')
) AS v(title, content, category, status, author, author_id, views, tags, is_published, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM knowledge_articles WHERE title = 'Kubernetes集群运维手册');

-- =============================================
-- 7. 添加更多配置项（8条）
-- =============================================
INSERT INTO configuration_items (name, ci_type, ci_type_id, status, owner, environment, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('Kafka消息队列-生产', 'middleware', 3, 'active', '运维部门', 'production', 1, NOW() - INTERVAL '30 days', NOW() - INTERVAL '30 days'),
    ('Elasticsearch集群', 'middleware', 3, 'active', '运维部门', 'production', 1, NOW() - INTERVAL '60 days', NOW() - INTERVAL '60 days'),
    ('MinIO对象存储', 'storage', 2, 'active', '运维部门', 'production', 1, NOW() - INTERVAL '45 days', NOW() - INTERVAL '45 days'),
    ('GitLab代码仓库', 'application', 5, 'active', '开发部门', 'production', 1, NOW() - INTERVAL '90 days', NOW() - INTERVAL '90 days'),
    ('Jenkins持续集成', 'application', 5, 'active', '开发部门', 'production', 1, NOW() - INTERVAL '80 days', NOW() - INTERVAL '80 days'),
    ('VPN网关', 'network', 4, 'active', '安全部门', 'production', 1, NOW() - INTERVAL '100 days', NOW() - INTERVAL '100 days'),
    ('堡垒机', 'security', 4, 'active', '安全部门', 'production', 1, NOW() - INTERVAL '120 days', NOW() - INTERVAL '120 days'),
    ('DNS服务器', 'network', 4, 'active', '运维部门', 'production', 1, NOW() - INTERVAL '150 days', NOW() - INTERVAL '150 days')
) AS v(name, ci_type, ci_type_id, status, owner, environment, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM configuration_items WHERE name = 'Kafka消息队列-生产');

-- =============================================
-- 8. 添加更多告警规则
-- =============================================
INSERT INTO alert_rules (name, description, metric_type, condition, threshold, severity, enabled, tenant_id, created_at, updated_at)
SELECT * FROM (VALUES
    ('API错误率告警', 'API接口错误率超过阈值', 'api_error_rate', '>', 5, 'critical', true, 1, NOW(), NOW()),
    ('数据库连接数告警', '数据库活跃连接数超过阈值', 'db_connections', '>', 80, 'warning', true, 1, NOW(), NOW()),
    ('队列积压告警', '消息队列积压消息数过多', 'queue_backlog', '>', 10000, 'warning', true, 1, NOW(), NOW()),
    ('证书到期告警', 'SSL证书即将到期', 'cert_expiry', '<', 7, 'critical', true, 1, NOW(), NOW())
) AS v(name, description, metric_type, condition, threshold, severity, enabled, tenant_id, created_at, updated_at)
WHERE NOT EXISTS (SELECT 1 FROM alert_rules WHERE name = 'API错误率告警');

-- =============================================
-- 验证数据统计
-- =============================================
SELECT 'tickets:' as info, COUNT(*) as count FROM tickets
UNION ALL SELECT 'incidents:', COUNT(*) FROM incidents
UNION ALL SELECT 'problems:', COUNT(*) FROM problems
UNION ALL SELECT 'changes:', COUNT(*) FROM changes
UNION ALL SELECT 'service_requests:', COUNT(*) FROM service_requests
UNION ALL SELECT 'knowledge_articles:', COUNT(*) FROM knowledge_articles
UNION ALL SELECT 'configuration_items:', COUNT(*) FROM configuration_items
UNION ALL SELECT 'alert_rules:', COUNT(*) FROM alert_rules;
