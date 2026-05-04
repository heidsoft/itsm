-- =====================================================
-- ITSM 领域模型索引优化迁移脚本
-- 生成时间: 2026-05-02
-- =====================================================
-- =====================================================
-- 1. 工单表 (Tickets) 索引优化 - 高优先级
-- =====================================================
-- 按类型查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS ticket_tenant_type_idx ON tickets (tenant_id, type);
-- 按优先级查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS ticket_tenant_priority_idx ON tickets (tenant_id, priority);
-- 按分类查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS ticket_tenant_category_idx ON tickets (tenant_id, category_id);
-- 按 SLA 定义查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS ticket_tenant_sla_def_idx ON tickets (tenant_id, sla_definition_id);
-- 已解决工单时间统计 (部分索引)
CREATE INDEX CONCURRENTLY IF NOT EXISTS ticket_resolved_at_idx ON tickets (resolved_at)
WHERE resolved_at IS NOT NULL;
-- 按部门查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS ticket_tenant_dept_idx ON tickets (tenant_id, department_id);
-- =====================================================
-- 2. 事件表 (Incidents) 索引优化 - 高优先级
-- =====================================================
-- 多租户隔离
CREATE INDEX CONCURRENTLY IF NOT EXISTS incident_tenant_idx ON incidents (tenant_id);
-- 按状态查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS incident_tenant_status_idx ON incidents (tenant_id, status);
-- 按处理人查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS incident_assignee_idx ON incidents (assignee_id);
-- 按报告人查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS incident_reporter_idx ON incidents (reporter_id);
-- 按创建时间排序
CREATE INDEX CONCURRENTLY IF NOT EXISTS incident_created_at_idx ON incidents (created_at);
-- 按严重程度查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS incident_tenant_severity_idx ON incidents (tenant_id, severity);
-- 重大事件标识 (部分索引)
CREATE INDEX CONCURRENTLY IF NOT EXISTS incident_major_idx ON incidents (tenant_id, is_major_incident)
WHERE is_major_incident = true;
-- =====================================================
-- 3. 问题表 (Problems) 索引优化 - 中优先级
-- =====================================================
CREATE INDEX CONCURRENTLY IF NOT EXISTS problem_tenant_idx ON problems (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS problem_tenant_status_idx ON problems (tenant_id, status);
CREATE INDEX CONCURRENTLY IF NOT EXISTS problem_assignee_idx ON problems (assignee_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS problem_created_at_idx ON problems (created_at);
-- =====================================================
-- 4. 变更表 (Changes) 索引优化 - 中优先级
-- =====================================================
CREATE INDEX CONCURRENTLY IF NOT EXISTS change_tenant_idx ON changes (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS change_tenant_status_idx ON changes (tenant_id, status);
CREATE INDEX CONCURRENTLY IF NOT EXISTS change_type_idx ON changes (type);
CREATE INDEX CONCURRENTLY IF NOT EXISTS change_assignee_idx ON changes (assignee_id);
-- =====================================================
-- 5. 配置项表 (Configuration Items) 索引优化 - 高优先级
-- =====================================================
-- 多租户隔离
CREATE INDEX CONCURRENTLY IF NOT EXISTS ci_tenant_idx ON configuration_items (tenant_id);
-- 按类型查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS ci_tenant_type_idx ON configuration_items (tenant_id, ci_type);
-- 按状态查询
CREATE INDEX CONCURRENTLY IF NOT EXISTS ci_tenant_status_idx ON configuration_items (tenant_id, status);
-- 云资源关联
CREATE INDEX CONCURRENTLY IF NOT EXISTS ci_tenant_cloud_provider_idx ON configuration_items (tenant_id, cloud_provider);
-- =====================================================
-- 6. 知识库表 (Knowledge Articles) 索引优化 - 中优先级
-- =====================================================
CREATE INDEX CONCURRENTLY IF NOT EXISTS knowledge_tenant_idx ON knowledge_articles (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS knowledge_author_idx ON knowledge_articles (author_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS knowledge_category_idx ON knowledge_articles (category);
CREATE INDEX CONCURRENTLY IF NOT EXISTS knowledge_published_idx ON knowledge_articles (is_published)
WHERE is_published = true;
-- =====================================================
-- 7. 工作流实例表 (Workflow Instances) 索引优化 - 中优先级
-- =====================================================
CREATE INDEX CONCURRENTLY IF NOT EXISTS wf_instance_tenant_idx ON workflow_instances (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS wf_instance_ticket_idx ON workflow_instances (ticket_workflow_instances);
-- =====================================================
-- 8. 审批记录表 (Approval Records) 索引优化 - 中优先级
-- =====================================================
CREATE INDEX CONCURRENTLY IF NOT EXISTS approval_record_ticket_idx ON approval_records (ticket_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS approval_record_workflow_idx ON approval_records (workflow_id);
-- =====================================================
-- 9. 工单评论表 (Ticket Comments) 索引优化 - 中优先级
-- =====================================================
CREATE INDEX CONCURRENTLY IF NOT EXISTS ticket_comment_ticket_id_idx ON ticket_comments (ticket_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS ticket_comment_user_id_idx ON ticket_comments (user_id);
-- =====================================================
-- 10. JSONB 字段 GIN 索引 - 低优先级 (可选)
-- =====================================================
-- 配置项云元数据 (如需要全文搜索)
-- CREATE INDEX IF NOT EXISTS ci_cloud_metadata_idx ON configuration_items USING GIN (cloud_metadata);
-- 事件影响分析
-- CREATE INDEX IF NOT EXISTS incident_impact_idx ON incidents USING GIN (impact_analysis);
-- =====================================================
-- 11. SLA 违规表索引 - 中优先级
-- =====================================================
CREATE INDEX CONCURRENTLY IF NOT EXISTS sla_violation_tenant_idx ON sla_violations (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS sla_violation_ticket_idx ON sla_violations (ticket_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS sla_violation_def_idx ON sla_violations (sla_definition_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS sla_violation_created_idx ON sla_violations (created_at);
-- =====================================================
-- 验证索引创建
-- =====================================================
-- SELECT tablename, indexname, indexdef 
-- FROM pg_indexes 
-- WHERE schemaname = 'public' 
--   AND indexname LIKE 'ticket_%' 
--    OR indexname LIKE 'incident_%'
--    OR indexname LIKE 'ci_%'
-- ORDER BY tablename, indexname;