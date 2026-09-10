-- ====================================================================
-- add_missing_indexes_batch2.sql - 索引残留批次（2026-09-11）
-- 范围：preflight 口径的 68 张仅 PK 表全量收口
--   - 47 张有 tenant_id：idx_<table>_tenant (tenant_id)
--   - 19 张纯关联/令牌表：按访问模式补查询列索引
-- 约定：全部 CONCURRENTLY + IF NOT EXISTS（幂等，可重复执行）
-- ====================================================================

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ai_analysis_results_tenant ON ai_analysis_results (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_applications_tenant ON applications (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_approval_records_tenant ON approval_records (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_approval_workflows_tenant ON approval_workflows (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_tenant ON audit_logs (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_bootstrap_tokens_tenant ON bootstrap_tokens (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cab_members_tenant ON cab_members (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_change_pi_rs_tenant ON change_pi_rs (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contracts_tenant ON contracts (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_conversations_tenant ON conversations (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_departments_tenant ON departments (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_endpoint_ac_ls_tenant ON endpoint_ac_ls (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_engineer_skills_tenant ON engineer_skills (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_groups_tenant ON groups (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_incident_alerts_tenant ON incident_alerts (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_incident_escalation_rules_tenant ON incident_escalation_rules (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_incident_events_tenant ON incident_events (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_incident_metrics_tenant ON incident_metrics (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_incident_rule_executions_tenant ON incident_rule_executions (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_incident_rules_tenant ON incident_rules (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_knowledge_article_likes_tenant ON knowledge_article_likes (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_known_errors_tenant ON known_errors (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_menus_tenant ON menus (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_microservices_tenant ON microservices (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notification_preferences_tenant ON notification_preferences (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_tenant ON notifications (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_permission_definitions_tenant ON permission_definitions (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_permissions_tenant ON permissions (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_tenant ON projects (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_role_permissions_tenant ON role_permissions (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_roles_tenant ON roles (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_root_cause_analyses_tenant ON root_cause_analyses (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_service_catalog_items_tenant ON service_catalog_items (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sla_alert_histories_tenant ON sla_alert_histories (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sla_alert_rules_tenant ON sla_alert_rules (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sla_definitions_tenant ON sla_definitions (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sla_metrics_tenant ON sla_metrics (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sla_policies_tenant ON sla_policies (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_standard_changes_tenant ON standard_changes (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tags_tenant ON tags (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_teams_tenant ON teams (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_approvals_tenant ON ticket_approvals (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_assignment_rules_tenant ON ticket_assignment_rules (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_attachments_tenant ON ticket_attachments (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_automation_rules_tenant ON ticket_automation_rules (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_categories_tenant ON ticket_categories (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_ccs_tenant ON ticket_ccs (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_comments_tenant ON ticket_comments (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_notifications_tenant ON ticket_notifications (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_tags_tenant ON ticket_tags (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_templates_tenant ON ticket_templates (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_views_tenant ON ticket_views (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_workflow_records_tenant ON ticket_workflow_records (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_tenant ON users (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_vendors_tenant ON vendors (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_tasks_tenant ON workflow_tasks (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_versions_tenant ON workflow_versions (tenant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflows_tenant ON workflows (tenant_id);

-- ---- 非租户仅 PK 表：按访问模式补查询列索引 ----
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_application_tags_tag_id ON application_tags (tag_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_configuration_item_incidents_incident_id ON configuration_item_incidents (incident_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_configuration_item_tags_ci_tag_id ON configuration_item_tags (ci_tag_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_department_tags_tag_id ON department_tags (tag_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_incident_related_incidents_incident_id ON incident_related_incidents (incident_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_knowledge_article_session_participants_knowledge_article_participant_id ON knowledge_article_session_participants (knowledge_article_participant_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_created_at ON messages (created_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_microservice_tags_tag_id ON microservice_tags (tag_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_msp_allocations_created_at ON msp_allocations (created_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_password_reset_tokens_token ON password_reset_tokens (token);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_problem_changes_problem_id ON problem_changes (problem_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_problem_changes_change_id ON problem_changes (change_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_problem_incidents_problem_id ON problem_incidents (problem_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_problem_incidents_incident_id ON problem_incidents (incident_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_project_tags_tag_id ON project_tags (tag_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_team_tags_team_id ON team_tags (team_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_team_tags_tag_id ON team_tags (tag_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_category_tickets_ticket_id ON ticket_category_tickets (ticket_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ticket_related_tickets_ticket_id ON ticket_related_tickets (ticket_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_article_participations_user_id ON user_article_participations (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_article_sessions_user_id ON user_article_sessions (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_roles_user_id ON user_roles (user_id);
