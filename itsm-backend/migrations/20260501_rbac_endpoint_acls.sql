-- RBAC Endpoint ACL Migration Script
-- 复用现有 Permission 表存储端点 ACL
-- Run after: ent generates and migrations complete
BEGIN;
-- =============================================================================
-- Create endpoint_acls table for URL → Permission mapping
-- This replaces the hardcoded ResourceActionMap
-- =============================================================================
CREATE TABLE IF NOT EXISTS endpoint_acls (
    id SERIAL PRIMARY KEY,
    tenant_id INT NOT NULL DEFAULT 1,
    path_pattern VARCHAR(255) NOT NULL,
    method VARCHAR(10),
    -- GET, POST, PUT, DELETE, null=ALL
    resource VARCHAR(50) NOT NULL,
    -- Maps to permission.resource
    action VARCHAR(20) NOT NULL,
    -- Maps to permission.action (read, write, delete)
    description VARCHAR(500),
    priority INT DEFAULT 100,
    -- Higher = checked first
    is_active BOOLEAN DEFAULT true,
    is_whitelist BOOLEAN DEFAULT false,
    -- If true, bypasses permission check
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(tenant_id, path_pattern, method)
);
-- Create indexes if not exists
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM pg_indexes
    WHERE indexname = 'endpoint_acls_tenant_path'
) THEN CREATE INDEX endpoint_acls_tenant_path ON endpoint_acls(tenant_id, path_pattern);
END IF;
IF NOT EXISTS (
    SELECT 1
    FROM pg_indexes
    WHERE indexname = 'endpoint_acls_priority'
) THEN CREATE INDEX endpoint_acls_priority ON endpoint_acls(tenant_id, priority DESC);
END IF;
END $$;
-- =============================================================================
-- STEP 1: Insert whitelist endpoints (no permission required)
-- =============================================================================
INSERT INTO endpoint_acls (
        tenant_id,
        path_pattern,
        method,
        resource,
        action,
        description,
        priority,
        is_active,
        is_whitelist
    )
VALUES -- Public endpoints
    (
        1,
        '/api/v1/auth/login',
        NULL,
        'auth',
        'login',
        'Login - no auth required',
        1,
        true,
        true
    ),
    (
        1,
        '/api/v1/auth/register',
        NULL,
        'auth',
        'register',
        'Register - no auth required',
        1,
        true,
        true
    ),
    (
        1,
        '/health',
        NULL,
        'system',
        'health',
        'Health check',
        1,
        true,
        true
    ),
    (
        1,
        '/api/v1/health',
        NULL,
        'system',
        'health',
        'Health check',
        1,
        true,
        true
    ) ON CONFLICT (tenant_id, path_pattern, method) DO
UPDATE
SET is_whitelist = EXCLUDED.is_whitelist,
    is_active = EXCLUDED.is_active,
    priority = EXCLUDED.priority,
    updated_at = NOW();
-- =============================================================================
-- STEP 2: Insert ResourceActionMap entries (auto-inference)
-- This migrates the hardcoded ResourceActionMap to database
-- =============================================================================
INSERT INTO endpoint_acls (
        tenant_id,
        path_pattern,
        method,
        resource,
        action,
        description,
        priority,
        is_active,
        is_whitelist
    )
VALUES -- Ticket endpoints
    (
        1,
        '/api/v1/tickets',
        'GET',
        'ticket',
        'read',
        'Ticket list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/tickets',
        'POST',
        'ticket',
        'write',
        'Create ticket',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/tickets/*',
        'GET',
        'ticket',
        'read',
        'Get ticket',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/tickets/*',
        'PUT',
        'ticket',
        'write',
        'Update ticket',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/tickets/*',
        'DELETE',
        'ticket',
        'delete',
        'Delete ticket',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/tickets/stats',
        'GET',
        'ticket',
        'read',
        'Ticket stats',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/tickets/bulk-update',
        'POST',
        'ticket',
        'write',
        'Bulk update tickets',
        100,
        true,
        false
    ),
    -- Notification endpoints
    (
        1,
        '/api/v1/notifications',
        'GET',
        'notification',
        'read',
        'Notification list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/notifications/*',
        'GET',
        'notification',
        'read',
        'Get notification',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/notifications/*',
        'PUT',
        'notification',
        'write',
        'Update notification',
        100,
        true,
        false
    ),
    -- Ticket category endpoints
    (
        1,
        '/api/v1/ticket-categories',
        'GET',
        'ticket_category',
        'read',
        'Ticket category list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/ticket-categories/*',
        'GET',
        'ticket_category',
        'read',
        'Get ticket category',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/ticket-categories',
        'POST',
        'ticket_category',
        'write',
        'Create ticket category',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/ticket-categories/*',
        'PUT',
        'ticket_category',
        'write',
        'Update ticket category',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/ticket-categories/*',
        'DELETE',
        'ticket_category',
        'delete',
        'Delete ticket category',
        100,
        true,
        false
    ),
    -- Ticket template endpoints
    (
        1,
        '/api/v1/ticket-templates',
        'GET',
        'ticket_template',
        'read',
        'Ticket template list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/ticket-templates/*',
        'GET',
        'ticket_template',
        'read',
        'Get ticket template',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/ticket-templates',
        'POST',
        'ticket_template',
        'write',
        'Create ticket template',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/ticket-templates/*',
        'PUT',
        'ticket_template',
        'write',
        'Update ticket template',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/ticket-templates/*',
        'DELETE',
        'ticket_template',
        'delete',
        'Delete ticket template',
        100,
        true,
        false
    ),
    -- Ticket tag endpoints
    (
        1,
        '/api/v1/ticket-tags',
        'GET',
        'ticket_tag',
        'read',
        'Ticket tag list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/ticket-tags/*',
        'GET',
        'ticket_tag',
        'read',
        'Get ticket tag',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/ticket-tags',
        'POST',
        'ticket_tag',
        'write',
        'Create ticket tag',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/ticket-tags/*',
        'PUT',
        'ticket_tag',
        'write',
        'Update ticket tag',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/ticket-tags/*',
        'DELETE',
        'ticket_tag',
        'delete',
        'Delete ticket tag',
        100,
        true,
        false
    ),
    -- User endpoints
    (
        1,
        '/api/v1/users',
        'GET',
        'user',
        'read',
        'User list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/users',
        'POST',
        'user',
        'write',
        'Create user',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/users/*',
        'GET',
        'user',
        'read',
        'Get user',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/users/*',
        'PUT',
        'user',
        'write',
        'Update user',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/users/*',
        'DELETE',
        'user',
        'delete',
        'Delete user',
        100,
        true,
        false
    ),
    -- Dashboard endpoints
    (
        1,
        '/api/v1/dashboard',
        'GET',
        'dashboard',
        'read',
        'Dashboard',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/dashboard/*',
        'GET',
        'dashboard',
        'read',
        'Dashboard endpoints',
        100,
        true,
        false
    ),
    -- Knowledge endpoints
    (
        1,
        '/api/v1/knowledge',
        'GET',
        'knowledge',
        'read',
        'Knowledge list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/knowledge',
        'POST',
        'knowledge',
        'write',
        'Create knowledge',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/knowledge/*',
        'GET',
        'knowledge',
        'read',
        'Get knowledge',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/knowledge/*',
        'PUT',
        'knowledge',
        'write',
        'Update knowledge',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/knowledge/*',
        'DELETE',
        'knowledge',
        'delete',
        'Delete knowledge',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/knowledge/search',
        'GET',
        'knowledge',
        'read',
        'Knowledge search',
        100,
        true,
        false
    ),
    -- Knowledge articles endpoints
    (
        1,
        '/api/v1/knowledge-articles',
        'GET',
        'knowledge',
        'read',
        'Knowledge articles',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/knowledge-articles/*',
        'GET',
        'knowledge',
        'read',
        'Get knowledge article',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/knowledge-articles',
        'POST',
        'knowledge',
        'write',
        'Create knowledge article',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/knowledge-articles/*',
        'PUT',
        'knowledge',
        'write',
        'Update knowledge article',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/knowledge/articles',
        'GET',
        'knowledge',
        'read',
        'Knowledge articles (alt)',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/knowledge/articles/*',
        'GET',
        'knowledge',
        'read',
        'Get knowledge article (alt)',
        100,
        true,
        false
    ),
    -- CMDB endpoints
    (
        1,
        '/api/v1/cmdb',
        'GET',
        'cmdb',
        'read',
        'CMDB list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/cmdb',
        'POST',
        'cmdb',
        'write',
        'Create CMDB item',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/cmdb/*',
        'GET',
        'cmdb',
        'read',
        'Get CMDB item',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/cmdb/*',
        'PUT',
        'cmdb',
        'write',
        'Update CMDB item',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/cmdb/*',
        'DELETE',
        'cmdb',
        'delete',
        'Delete CMDB item',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/configuration-items',
        'GET',
        'cmdb',
        'read',
        'Configuration items',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/configuration-items/*',
        'GET',
        'cmdb',
        'read',
        'Get configuration item',
        100,
        true,
        false
    ),
    -- Incident endpoints
    (
        1,
        '/api/v1/incidents',
        'GET',
        'incident',
        'read',
        'Incident list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/incidents',
        'POST',
        'incident',
        'write',
        'Create incident',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/incidents/*',
        'GET',
        'incident',
        'read',
        'Get incident',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/incidents/*',
        'PUT',
        'incident',
        'write',
        'Update incident',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/incidents/*',
        'DELETE',
        'incident',
        'delete',
        'Delete incident',
        100,
        true,
        false
    ),
    -- Audit logs endpoints
    (
        1,
        '/api/v1/audit-logs',
        'GET',
        'audit_logs',
        'read',
        'Audit log',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/audit-logs/*',
        'GET',
        'audit_logs',
        'read',
        'Get audit log',
        100,
        true,
        false
    ),
    -- System config endpoints
    (
        1,
        '/api/v1/system/*',
        'GET',
        'system_config',
        'read',
        'System config read',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/system/*',
        'PUT',
        'system_config',
        'write',
        'System config write',
        100,
        true,
        false
    ),
    -- AI endpoints
    (
        1,
        '/api/v1/ai/*',
        'GET',
        'ai',
        'read',
        'AI read',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/ai/*',
        'POST',
        'ai',
        'write',
        'AI write',
        100,
        true,
        false
    ),
    -- Service Catalog endpoints
    (
        1,
        '/api/v1/service-catalogs',
        'GET',
        'service_catalog',
        'read',
        'Service catalog list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/service-catalogs/*',
        'GET',
        'service_catalog',
        'read',
        'Get service catalog',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/service-catalogs',
        'POST',
        'service_catalog',
        'write',
        'Create service catalog',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/service-catalogs/*',
        'PUT',
        'service_catalog',
        'write',
        'Update service catalog',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/service-catalogs/*',
        'DELETE',
        'service_catalog',
        'delete',
        'Delete service catalog',
        100,
        true,
        false
    ),
    -- Service Request endpoints
    (
        1,
        '/api/v1/service-requests',
        'GET',
        'service_request',
        'read',
        'Service request list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/service-requests',
        'POST',
        'service_request',
        'write',
        'Create service request',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/service-requests/*',
        'GET',
        'service_request',
        'read',
        'Get service request',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/service-requests/*',
        'PUT',
        'service_request',
        'write',
        'Update service request',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/service-requests/*',
        'DELETE',
        'service_request',
        'delete',
        'Delete service request',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/provisioning-tasks/*',
        'GET',
        'service_request',
        'read',
        'Provisioning tasks',
        100,
        true,
        false
    ),
    -- Problem endpoints
    (
        1,
        '/api/v1/problems',
        'GET',
        'problem',
        'read',
        'Problem list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/problems',
        'POST',
        'problem',
        'write',
        'Create problem',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/problems/*',
        'GET',
        'problem',
        'read',
        'Get problem',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/problems/*',
        'PUT',
        'problem',
        'write',
        'Update problem',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/problems/*',
        'DELETE',
        'problem',
        'delete',
        'Delete problem',
        100,
        true,
        false
    ),
    -- Change endpoints
    (
        1,
        '/api/v1/changes',
        'GET',
        'change',
        'read',
        'Change list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/changes',
        'POST',
        'change',
        'write',
        'Create change',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/changes/*',
        'GET',
        'change',
        'read',
        'Get change',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/changes/*',
        'PUT',
        'change',
        'write',
        'Update change',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/changes/*',
        'DELETE',
        'change',
        'delete',
        'Delete change',
        100,
        true,
        false
    ),
    -- Role endpoints
    (
        1,
        '/api/v1/roles',
        'GET',
        'role',
        'read',
        'Role list',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/roles/*',
        'GET',
        'role',
        'read',
        'Get role',
        100,
        true,
        false
    ),
    -- Permission endpoints
    (
        1,
        '/api/v1/permissions',
        'GET',
        'permission',
        'read',
        'Permission list',
        100,
        true,
        false
    ),
    -- SLA endpoints
    (
        1,
        '/api/v1/sla/*',
        'GET',
        'sla',
        'read',
        'SLA read',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/sla/*',
        'POST',
        'sla',
        'write',
        'SLA write',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/sla/*',
        'PUT',
        'sla',
        'write',
        'SLA update',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/sla/*',
        'DELETE',
        'sla',
        'delete',
        'SLA delete',
        100,
        true,
        false
    ),
    -- Organization endpoints
    (
        1,
        '/api/v1/org/*',
        'GET',
        'org',
        'read',
        'Organization read',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/org/*',
        'POST',
        'org',
        'write',
        'Organization write',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/org/*',
        'PUT',
        'org',
        'write',
        'Organization update',
        100,
        true,
        false
    ),
    -- Auth endpoints (special - user info)
    (
        1,
        '/api/v1/auth/me',
        'GET',
        'user',
        'read',
        'Get current user',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/auth/profile',
        'GET',
        'user',
        'read',
        'Get profile',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/auth/menus',
        'GET',
        'user',
        'read',
        'Get user menus',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/auth/tenants',
        'GET',
        'tenant',
        'read',
        'Get tenants',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/auth/logout',
        'POST',
        'auth',
        'logout',
        'Logout',
        100,
        true,
        false
    ),
    -- BPMN Workflow endpoints
    (
        1,
        '/api/v1/bpmn/*',
        'GET',
        'bpmn',
        'read',
        'BPMN read',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/bpmn/*',
        'POST',
        'bpmn',
        'write',
        'BPMN write',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/bpmn/*',
        'PUT',
        'bpmn',
        'write',
        'BPMN update',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/process-trigger/*',
        'GET',
        'bpmn',
        'read',
        'Process trigger read',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/process-trigger',
        'POST',
        'bpmn',
        'write',
        'Process trigger execute',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/process-bindings',
        'GET',
        'bpmn',
        'read',
        'Process bindings read',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/process-bindings/*',
        'GET',
        'bpmn',
        'read',
        'Process binding read',
        100,
        true,
        false
    ),
    -- MSP endpoints
    (
        1,
        '/api/v1/msp/status',
        'GET',
        'msp',
        'read',
        'MSP status',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/msp/context',
        'GET',
        'msp',
        'read',
        'MSP context',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/msp/allocations',
        'GET',
        'msp_allocation',
        'read',
        'MSP allocations read',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/msp/customers',
        'GET',
        'msp_customer',
        'read',
        'MSP customers read',
        100,
        true,
        false
    ),
    (
        1,
        '/api/v1/msp/customers/*',
        'GET',
        'msp_customer',
        'read',
        'MSP customer read',
        100,
        true,
        false
    ) ON CONFLICT (tenant_id, path_pattern, method) DO
UPDATE
SET resource = EXCLUDED.resource,
    action = EXCLUDED.action,
    description = EXCLUDED.description,
    priority = EXCLUDED.priority,
    is_active = EXCLUDED.is_active,
    updated_at = NOW();
-- =============================================================================
-- STEP 3: Ensure required permissions exist in permission_definition
-- =============================================================================
INSERT INTO permission_definition (
        resource,
        action,
        description,
        display_name,
        category
    )
VALUES ('auth', 'login', 'Login permission', '登录', 1),
    (
        'auth',
        'register',
        'Register permission',
        '注册',
        1
    ),
    ('auth', 'logout', 'Logout permission', '登出', 1),
    ('system', 'health', 'Health check', '健康检查', 1),
    ('bpmn', 'read', 'BPMN read', '流程查看', 2),
    ('bpmn', 'write', 'BPMN write', '流程管理', 2),
    ('msp', 'read', 'MSP read', 'MSP查看', 3),
    (
        'msp_allocation',
        'read',
        'MSP Allocation read',
        'MSP分配查看',
        3
    ),
    (
        'msp_customer',
        'read',
        'MSP Customer read',
        'MSP客户查看',
        3
    ),
    ('tenant', 'read', 'Tenant read', '租户查看', 4),
    (
        'audit_logs',
        'read',
        'Audit log read',
        '审计日志查看',
        4
    ),
    (
        'system_config',
        'read',
        'System config read',
        '系统配置查看',
        4
    ),
    (
        'system_config',
        'write',
        'System config write',
        '系统配置管理',
        4
    ),
    ('ai', 'read', 'AI read', 'AI查看', 5),
    ('ai', 'write', 'AI write', 'AI操作', 5),
    ('org', 'read', 'Organization read', '组织查看', 4),
    ('org', 'write', 'Organization write', '组织管理', 4) ON CONFLICT ON CONSTRAINT permission_definition_resource_action_key DO NOTHING;
-- =============================================================================
-- STEP 4: Ensure l1_engineer role has required permissions
-- =============================================================================
-- First, get l1_engineer role id
DO $$
DECLARE l1_role_id INT;
user_perm_id INT;
dashboard_perm_id INT;
ticket_perm_id INT;
notification_perm_id INT;
BEGIN -- Find l1_engineer role
SELECT id INTO l1_role_id
FROM role
WHERE code = 'l1_engineer'
LIMIT 1;
IF l1_role_id IS NOT NULL THEN -- Find permission definitions
SELECT id INTO user_perm_id
FROM permission_definition
WHERE resource = 'user'
    AND action = 'read'
LIMIT 1;
SELECT id INTO dashboard_perm_id
FROM permission_definition
WHERE resource = 'dashboard'
    AND action = 'read'
LIMIT 1;
SELECT id INTO ticket_perm_id
FROM permission_definition
WHERE resource = 'ticket'
    AND action = 'read'
LIMIT 1;
SELECT id INTO notification_perm_id
FROM permission_definition
WHERE resource = 'notification'
    AND action = 'read'
LIMIT 1;
-- Add user:read permission if not exists
IF user_perm_id IS NOT NULL THEN
INSERT INTO role_permission (role_id, permission_id)
VALUES (l1_role_id, user_perm_id) ON CONFLICT DO NOTHING;
END IF;
-- Add dashboard:read permission if not exists
IF dashboard_perm_id IS NOT NULL THEN
INSERT INTO role_permission (role_id, permission_id)
VALUES (l1_role_id, dashboard_perm_id) ON CONFLICT DO NOTHING;
END IF;
-- Add ticket:read permission if not exists
IF ticket_perm_id IS NOT NULL THEN
INSERT INTO role_permission (role_id, permission_id)
VALUES (l1_role_id, ticket_perm_id) ON CONFLICT DO NOTHING;
END IF;
-- Add notification:read permission if not exists
IF notification_perm_id IS NOT NULL THEN
INSERT INTO role_permission (role_id, permission_id)
VALUES (l1_role_id, notification_perm_id) ON CONFLICT DO NOTHING;
END IF;
RAISE NOTICE 'Updated l1_engineer permissions successfully';
ELSE RAISE NOTICE 'l1_engineer role not found, skipping permission update';
END IF;
END $$;
-- =============================================================================
-- STEP 5: Verification
-- =============================================================================
SELECT 'Total ACLs:' as label,
    COUNT(*) as count
FROM endpoint_acls
WHERE tenant_id = 1;
SELECT 'Active ACLs:' as label,
    COUNT(*) as count
FROM endpoint_acls
WHERE tenant_id = 1
    AND is_active = true;
SELECT 'Whitelist ACLs:' as label,
    COUNT(*) as count
FROM endpoint_acls
WHERE tenant_id = 1
    AND is_whitelist = true;
SELECT 'Permission definitions:' as label,
    COUNT(*) as count
FROM permission_definition;
COMMIT;
-- =============================================================================
-- Optional: Create trigger to auto-update updated_at
-- =============================================================================
CREATE OR REPLACE FUNCTION update_endpoint_acls_timestamp() RETURNS TRIGGER AS $$ BEGIN NEW.updated_at = NOW();
RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS update_endpoint_acls_timestamp ON endpoint_acls;
CREATE TRIGGER update_endpoint_acls_timestamp BEFORE
UPDATE ON endpoint_acls FOR EACH ROW EXECUTE FUNCTION update_endpoint_acls_timestamp();