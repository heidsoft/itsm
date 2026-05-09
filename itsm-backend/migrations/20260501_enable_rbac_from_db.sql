-- Role Permissions Database Seeding
-- Purpose: Enable database-driven RBAC with multi-tenant support
-- Run after: ent generates and migrations complete
BEGIN;
-- =============================================================================
-- STEP 1: Ensure RBAC roles exist (match rbac.go RolePermissions codes)
-- =============================================================================
INSERT INTO roles (
        id,
        code,
        name,
        tenant_id,
        is_system,
        created_at,
        updated_at
    )
VALUES (
        100,
        'super_admin',
        '超级管理员',
        1,
        true,
        NOW(),
        NOW()
    ),
    (101, 'sysadmin', '系统管理员', 1, true, NOW(), NOW()),
    (102, 'admin', '管理员', 1, false, NOW(), NOW()),
    (103, 'manager', '经理', 1, false, NOW(), NOW()),
    (104, 'agent', '服务台坐席', 1, false, NOW(), NOW()),
    (105, 'technician', '技术员', 1, false, NOW(), NOW()),
    (106, 'security', '安全管理员', 1, false, NOW(), NOW()),
    (107, 'end_user', '普通用户', 1, false, NOW(), NOW()),
    (
        108,
        'msp_viewer',
        'MSP查看者',
        1,
        false,
        NOW(),
        NOW()
    ),
    (
        109,
        'msp_tech',
        'MSP技术员',
        1,
        false,
        NOW(),
        NOW()
    ),
    (
        110,
        'msp_specialist',
        'MSP专家',
        1,
        false,
        NOW(),
        NOW()
    ),
    (
        111,
        'msp_manager',
        'MSP经理',
        1,
        false,
        NOW(),
        NOW()
    ),
    (
        112,
        'msp_admin',
        'MSP管理员',
        1,
        false,
        NOW(),
        NOW()
    ) ON CONFLICT (id) DO
UPDATE
SET name = EXCLUDED.name,
    updated_at = NOW();
-- =============================================================================
-- STEP 2: Clear existing role_permissions for clean slate
-- =============================================================================
DELETE FROM role_permissions;
-- =============================================================================
-- STEP 3: Populate role_permissions based on rbac.go hardcoded RolePermissions
-- =============================================================================
-- SUPER_ADMIN (id=100): All permissions (*:*)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 100,
    p.id
FROM permissions p
WHERE p.tenant_id = 1;
-- SYSADMIN (id=101): All permissions (*:*)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 101,
    p.id
FROM permissions p
WHERE p.tenant_id = 1;
-- ADMIN (id=102): Comprehensive admin permissions (from rbac.go lines 60-135)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 102,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND (
        -- Ticket management
        (
            p.resource = 'ticket'
            AND p.action IN ('read', 'write', 'delete', 'admin')
        )
        OR -- Notification
        (
            p.resource = 'notification'
            AND p.action IN ('read', 'write')
        )
        OR -- Ticket category
        (
            p.resource = 'ticket_category'
            AND p.action IN ('read', 'write', 'delete')
        )
        OR -- Ticket tag
        (
            p.resource = 'ticket_tag'
            AND p.action IN ('read', 'write', 'delete')
        )
        OR -- Ticket template
        (
            p.resource = 'ticket_template'
            AND p.action IN ('read', 'write', 'delete')
        )
        OR -- User management
        (
            p.resource = 'user'
            AND p.action IN ('read', 'write', 'delete')
        )
        OR -- Dashboard
        (
            p.resource = 'dashboard'
            AND p.action IN ('read', 'admin')
        )
        OR -- Knowledge
        (
            p.resource = 'knowledge'
            AND p.action IN ('read', 'write', 'admin')
        )
        OR -- CMDB
        (
            p.resource = 'cmdb'
            AND p.action IN ('read', 'write', 'delete')
        )
        OR -- Incident
        (
            p.resource = 'incident'
            AND p.action IN ('read', 'write', 'admin')
        )
        OR -- Service catalog
        (
            p.resource = 'service'
            AND p.action IN ('read', 'write', 'delete')
        )
        OR -- Service request
        (
            p.resource = 'service_request'
            AND p.action IN ('read', 'write')
        )
        OR -- Change management
        (
            p.resource = 'change'
            AND p.action IN ('read', 'write', 'delete')
        )
        OR -- Problem management
        (
            p.resource = 'problem'
            AND p.action IN ('read', 'write', 'delete')
        )
        OR -- SLA
        (
            p.resource = 'sla'
            AND p.action IN ('read', 'write', 'delete')
        )
        OR -- Audit logs
        (
            p.resource = 'audit_logs'
            AND p.action = 'read'
        )
        OR -- AI
        (
            p.resource = 'ai'
            AND p.action IN ('read', 'write')
        )
        OR -- Role management
        (
            p.resource = 'role'
            AND p.action IN ('read', 'write', 'delete')
        )
        OR -- Permission
        (
            p.resource = 'permission'
            AND p.action = 'read'
        )
        OR -- System config
        (
            p.resource = 'system'
            AND p.action IN ('read', 'write')
        )
        OR -- Organization
        (
            p.resource = 'org'
            AND p.action IN ('read', 'write')
        )
        OR -- Groups
        (
            p.resource = 'groups'
            AND p.action IN ('read', 'write')
        )
        OR -- BPMN
        (
            p.resource = 'workflow'
            AND p.action IN ('read', 'write', 'delete')
        )
        OR -- Release
        (
            p.resource = 'release'
            AND p.action IN ('read', 'write', 'delete')
        )
        OR -- Asset
        (
            p.resource = 'asset'
            AND p.action IN ('read', 'write', 'delete')
        )
        OR -- License
        (
            p.resource = 'license'
            AND p.action IN ('read', 'write', 'delete')
        )
    );
-- MANAGER (id=103): Manager permissions (from rbac.go lines 136-167)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 103,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND (
        -- Ticket
        (
            p.resource = 'ticket'
            AND p.action IN ('read', 'write')
        )
        OR -- Notification
        (
            p.resource = 'notification'
            AND p.action IN ('read', 'write')
        )
        OR -- Incident
        (
            p.resource = 'incident'
            AND p.action IN ('read', 'write')
        )
        OR -- Dashboard
        (
            p.resource = 'dashboard'
            AND p.action = 'read'
        )
        OR -- Knowledge
        (
            p.resource = 'knowledge'
            AND p.action = 'read'
        )
        OR -- CMDB
        (
            p.resource = 'cmdb'
            AND p.action = 'read'
        )
        OR -- User (read only)
        (
            p.resource = 'user'
            AND p.action = 'read'
        )
        OR -- Service catalog
        (
            p.resource = 'service'
            AND p.action = 'read'
        )
        OR -- Service request
        (
            p.resource = 'service_request'
            AND p.action IN ('read', 'write')
        )
        OR -- Change (read only)
        (
            p.resource = 'change'
            AND p.action = 'read'
        )
        OR -- Problem (read only)
        (
            p.resource = 'problem'
            AND p.action = 'read'
        )
        OR -- BPMN
        (
            p.resource = 'workflow'
            AND p.action IN ('read', 'write')
        )
        OR -- Release
        (
            p.resource = 'release'
            AND p.action IN ('read', 'write')
        )
        OR -- Asset
        (
            p.resource = 'asset'
            AND p.action IN ('read', 'write')
        )
        OR -- License
        (
            p.resource = 'license'
            AND p.action IN ('read', 'write')
        )
        OR -- Groups
        (
            p.resource = 'groups'
            AND p.action IN ('read', 'write')
        )
    );
-- AGENT (id=104): Service desk agent (from rbac.go lines 168-191)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 104,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND (
        -- Ticket
        (
            p.resource = 'ticket'
            AND p.action IN ('read', 'write')
        )
        OR -- Notification
        (
            p.resource = 'notification'
            AND p.action IN ('read', 'write')
        )
        OR -- Dashboard
        (
            p.resource = 'dashboard'
            AND p.action = 'read'
        )
        OR -- Knowledge
        (
            p.resource = 'knowledge'
            AND p.action IN ('read', 'write')
        )
        OR -- CMDB
        (
            p.resource = 'cmdb'
            AND p.action = 'read'
        )
        OR -- Incident
        (
            p.resource = 'incident'
            AND p.action IN ('read', 'write')
        )
        OR -- Service catalog
        (
            p.resource = 'service'
            AND p.action = 'read'
        )
        OR -- Service request
        (
            p.resource = 'service_request'
            AND p.action IN ('read', 'write')
        )
        OR -- Change
        (
            p.resource = 'change'
            AND p.action IN ('read', 'write')
        )
        OR -- Problem
        (
            p.resource = 'problem'
            AND p.action IN ('read', 'write')
        )
        OR -- Groups (read only)
        (
            p.resource = 'groups'
            AND p.action = 'read'
        )
        OR -- BPMN
        (
            p.resource = 'workflow'
            AND p.action IN ('read', 'write')
        )
    );
-- TECHNICIAN (id=105): Technical support (from rbac.go lines 192-208)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 105,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND (
        -- Ticket
        (
            p.resource = 'ticket'
            AND p.action IN ('read', 'write')
        )
        OR -- Notification
        (
            p.resource = 'notification'
            AND p.action = 'read'
        )
        OR -- Knowledge (read only)
        (
            p.resource = 'knowledge'
            AND p.action = 'read'
        )
        OR -- CMDB (read only)
        (
            p.resource = 'cmdb'
            AND p.action = 'read'
        )
        OR -- Incident
        (
            p.resource = 'incident'
            AND p.action IN ('read', 'write')
        )
        OR -- Service catalog (read only)
        (
            p.resource = 'service'
            AND p.action = 'read'
        )
        OR -- Service request
        (
            p.resource = 'service_request'
            AND p.action IN ('read', 'write')
        )
        OR -- Groups (read only)
        (
            p.resource = 'groups'
            AND p.action = 'read'
        )
        OR -- BPMN (read only)
        (
            p.resource = 'workflow'
            AND p.action = 'read'
        )
    );
-- SECURITY (id=106): Security admin (from rbac.go lines 209-225)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 106,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND (
        -- User (read only)
        (
            p.resource = 'user'
            AND p.action = 'read'
        )
        OR -- Service catalog (read only)
        (
            p.resource = 'service'
            AND p.action = 'read'
        )
        OR -- Service request
        (
            p.resource = 'service_request'
            AND p.action IN ('read', 'write')
        )
        OR -- BPMN
        (
            p.resource = 'workflow'
            AND p.action IN ('read', 'write')
        )
        OR -- Release (read only)
        (
            p.resource = 'release'
            AND p.action = 'read'
        )
        OR -- Asset (read only)
        (
            p.resource = 'asset'
            AND p.action = 'read'
        )
        OR -- License (read only)
        (
            p.resource = 'license'
            AND p.action = 'read'
        )
    );
-- END_USER (id=107): End user / customer (from rbac.go lines 226-253)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 107,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND (
        -- Ticket (own data only)
        (
            p.resource = 'ticket'
            AND p.action IN ('read', 'write')
        )
        OR -- Notification
        (
            p.resource = 'notification'
            AND p.action IN ('read', 'write')
        )
        OR -- Knowledge (read only)
        (
            p.resource = 'knowledge'
            AND p.action = 'read'
        )
        OR -- Dashboard (read only)
        (
            p.resource = 'dashboard'
            AND p.action = 'read'
        )
        OR -- AI
        (
            p.resource = 'ai'
            AND p.action IN ('read', 'write')
        )
        OR -- Service catalog (read only)
        (
            p.resource = 'service'
            AND p.action = 'read'
        )
        OR -- Service request
        (
            p.resource = 'service_request'
            AND p.action IN ('read', 'write')
        )
        OR -- User (read own info)
        (
            p.resource = 'user'
            AND p.action = 'read'
        )
        OR -- SLA (read only)
        (
            p.resource = 'sla'
            AND p.action = 'read'
        )
        OR -- System config (read only)
        (
            p.resource = 'system'
            AND p.action = 'read'
        )
        OR -- Org (read only)
        (
            p.resource = 'org'
            AND p.action = 'read'
        )
        OR -- CMDB (read only)
        (
            p.resource = 'cmdb'
            AND p.action = 'read'
        )
        OR -- Incident (read only)
        (
            p.resource = 'incident'
            AND p.action = 'read'
        )
        OR -- Change (read only)
        (
            p.resource = 'change'
            AND p.action = 'read'
        )
        OR -- Problem (read only)
        (
            p.resource = 'problem'
            AND p.action = 'read'
        )
        OR -- BPMN (read only)
        (
            p.resource = 'workflow'
            AND p.action = 'read'
        )
        OR -- Release (read only)
        (
            p.resource = 'release'
            AND p.action = 'read'
        )
        OR -- Asset (read only)
        (
            p.resource = 'asset'
            AND p.action = 'read'
        )
        OR -- License (read only)
        (
            p.resource = 'license'
            AND p.action = 'read'
        )
    );
-- =============================================================================
-- STEP 4: MSP Roles
-- =============================================================================
-- MSP_VIEWER (id=108)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 108,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource IN (
        'msp',
        'msp_customer',
        'msp_ticket',
        'msp_allocation',
        'msp_report'
    )
    AND p.action = 'read';
-- MSP_TECH (id=109)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 109,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND (
        (
            p.resource = 'msp'
            AND p.action = 'read'
        )
        OR (
            p.resource = 'msp_customer'
            AND p.action = 'read'
        )
        OR (
            p.resource = 'msp_ticket'
            AND p.action IN ('read', 'write')
        )
        OR (
            p.resource = 'msp_allocation'
            AND p.action = 'read'
        )
        OR (
            p.resource = 'msp_report'
            AND p.action = 'read'
        )
    );
-- MSP_SPECIALIST (id=110)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 110,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND (
        (
            p.resource = 'msp'
            AND p.action = 'read'
        )
        OR (
            p.resource = 'msp_customer'
            AND p.action IN ('read', 'write')
        )
        OR (
            p.resource = 'msp_ticket'
            AND p.action IN ('read', 'write')
        )
        OR (
            p.resource = 'msp_allocation'
            AND p.action = 'read'
        )
        OR (
            p.resource = 'msp_report'
            AND p.action = 'read'
        )
    );
-- MSP_MANAGER (id=111)
INSERT INTO role_permissions (role_id, permission_id)
SELECT 111,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND (
        (
            p.resource = 'msp'
            AND p.action IN ('read', 'write')
        )
        OR (
            p.resource = 'msp_customer'
            AND p.action IN ('read', 'write')
        )
        OR (
            p.resource = 'msp_ticket'
            AND p.action IN ('read', 'write')
        )
        OR (
            p.resource = 'msp_allocation'
            AND p.action IN ('read', 'write')
        )
        OR (
            p.resource = 'msp_report'
            AND p.action IN ('read', 'write')
        )
    );
-- MSP_ADMIN (id=112): All MSP permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT 112,
    p.id
FROM permissions p
WHERE p.tenant_id = 1
    AND p.resource IN (
        'msp',
        'msp_customer',
        'msp_ticket',
        'msp_allocation',
        'msp_report'
    );
COMMIT;
-- =============================================================================
-- Verification
-- =============================================================================
-- SELECT r.code, COUNT(rp.id) as permission_count
-- FROM roles r
-- LEFT JOIN role_permissions rp ON r.id = rp.role_id
-- WHERE r.id >= 100
-- GROUP BY r.code
-- ORDER BY r.code;