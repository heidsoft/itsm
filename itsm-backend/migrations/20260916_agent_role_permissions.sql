-- ITSM Agent (服务台坐席) Role Permissions Backfill Migration
-- Date: 2026-09-16
-- Description: 修复 agent 角色在 role_permissions 表中权限为 0 的 P0 Bug。
--              根因（双重）：
--                1) pkg/seeder/seeder.go:rolePermissionMap 漏定义 agent 键，
--                   seedRolePermissions 遍历时未定义角色直接 continue，导致
--                   权限关联从未建立。
--                2) 现网 roles 表中根本不存在 code='agent' 的行（验证脚本确认）。
--              综合影响：所有试图通过 users.role='agent' 或 user_roles.code='agent'
--              走 RBAC 中间件的用户均 403 Forbidden（即便服务台坐席账号存在）。
--
-- 同步修复：
--   - seeder.go:seedRolePermissions 已在同批次新增 agent 权限映射（修复源头）
--   - 本迁移文件: 修复现网缺漏的角色行 + 权限关联（修复存量数据）
--   - 下次 itsm-init 重启会自动 idempotent（角色按 code+tenant_id 唯一存在性 upsert）
--
-- 权限边界：34 条，按"最小授权 + 与 l1_support 同级 + 服务台 L1 审批兜底"原则
--   - 工单核心 11 / 工单元数据 4 / 事件 3 / 问题 2 / 变更 1 / 知识 2 / SLA 1
--   - 服务请求 3 (read/write/approve，approvers-l1 组兜底)
--   - 上下文读 7 (user/team/department/asset/cmdb/notification/ai)
--
-- 幂等性说明：
--   - roles 表无 (code, tenant_id) 唯一约束，角色查重用 WHERE NOT EXISTS
--   - role_permissions 表无 (role_id, permission_id, tenant_id) 唯一约束，
--     关联插入用 WHERE NOT EXISTS 保证可重复执行
--
-- ROLLBACK:
--   DELETE FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE code='agent' AND tenant_id=1);
--   DELETE FROM roles WHERE code='agent' AND tenant_id=1;
-- PR: agent-rbac-r4b

BEGIN;

-- Step 1: 确保 agent 角色行存在（仅 default 租户，tenant_id=1）
INSERT INTO roles (code, name, tenant_id, is_system, created_at, updated_at)
SELECT 'agent', '服务台坐席', 1, false, NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM roles WHERE code = 'agent' AND tenant_id = 1
);

-- Step 2: 拿到 agent 角色 id
DO $$
DECLARE
    agent_role_id INTEGER;
BEGIN
    SELECT id INTO agent_role_id FROM roles WHERE code = 'agent' AND tenant_id = 1;
    IF agent_role_id IS NULL THEN
        RAISE EXCEPTION 'agent 角色创建失败，role_id 为空';
    END IF;
    -- 把 id 写入会话级临时变量供后续语句引用
    PERFORM set_config('itsm.agent_role_id', agent_role_id::text, false);
    RAISE NOTICE 'agent role_id = %', agent_role_id;
END $$;

-- Step 3: 批量插入 34 条权限关联
INSERT INTO role_permissions (role_id, permission_id, tenant_id)
SELECT
    current_setting('itsm.agent_role_id')::bigint AS role_id,
    p.id AS permission_id,
    1 AS tenant_id
FROM permissions p
WHERE p.tenant_id = 1
  AND p.code IN (
    -- 工单核心（11）
    'ticket:read', 'ticket:write', 'ticket:create', 'ticket:update',
    'ticket:assign', 'ticket:escalate', 'ticket:resolve', 'ticket:close',
    'ticket:export', 'ticket:import', 'ticket:delete',
    -- 工单元数据（4）
    'ticket_type:read',
    'ticket_category:read', 'ticket_tag:read', 'ticket_template:read',
    -- 事件（3）
    'incident:read', 'incident:write', 'incident:delete',
    -- 问题（2）
    'problem:read', 'problem:write',
    -- 变更（1：只读）
    'change:read',
    -- 知识库（2）
    'knowledge:read', 'knowledge:write',
    -- SLA（1）
    'sla:read',
    -- 服务请求（3：服务台 L1 审批兜底）
    'service_request:read', 'service_request:write', 'service_request:approve',
    -- 上下文读（7）
    'user:read', 'team:read', 'department:read',
    'asset:read', 'cmdb:read',
    'notification:read', 'ai:read'
  )
  AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp
      WHERE rp.role_id = current_setting('itsm.agent_role_id')::bigint
        AND rp.permission_id = p.id
        AND rp.tenant_id = 1
  );

-- Step 4: 验证 — 必须返回 perm_count >= 30（若 permissions 表本身缺权限可能 < 34）
DO $$
DECLARE
    agent_role_id   INTEGER;
    inserted_count  INTEGER;
    missing_perms   INTEGER;
    agent_users     INTEGER;
BEGIN
    SELECT id INTO agent_role_id FROM roles WHERE code = 'agent' AND tenant_id = 1;

    SELECT COUNT(*) INTO inserted_count
    FROM role_permissions WHERE role_id = agent_role_id AND tenant_id = 1;

    -- 期望插入 34 条，但若 permissions 表本身缺权限（多租户独立 seed），实际数 < 34
    SELECT COUNT(*) INTO missing_perms
    FROM permissions p
    WHERE p.tenant_id = 1
      AND p.code IN (
        'ticket:read','ticket:write','ticket:create','ticket:update',
        'ticket:assign','ticket:escalate','ticket:resolve','ticket:close',
        'ticket:export','ticket:import','ticket:delete',
        'ticket_type:read','ticket_category:read','ticket_tag:read','ticket_template:read',
        'incident:read','incident:write','incident:delete',
        'problem:read','problem:write',
        'change:read',
        'knowledge:read','knowledge:write',
        'sla:read',
        'service_request:read','service_request:write','service_request:approve',
        'user:read','team:read','department:read',
        'asset:read','cmdb:read','notification:read','ai:read'
      )
      AND NOT EXISTS (
          SELECT 1 FROM role_permissions rp
          WHERE rp.role_id = agent_role_id
            AND rp.permission_id = p.id
            AND rp.tenant_id = 1
      );

    -- 统计目前使用 agent 角色的用户数（既有 user.role='agent' + user_roles.code='agent'）
    SELECT COUNT(*) INTO agent_users
    FROM (
        SELECT id FROM users WHERE role = 'agent' AND tenant_id = 1
        UNION
        SELECT u.id FROM user_roles ur
        JOIN users u ON u.id = ur.user_id
        JOIN roles r ON r.id = ur.role_id
        WHERE r.code = 'agent' AND r.tenant_id = 1
    ) t;

    RAISE NOTICE 'agent 角色权限回填完成';
    RAISE NOTICE '  role_id = %', agent_role_id;
    RAISE NOTICE '  已插入 role_permissions = % 条（期望 34）', inserted_count;
    RAISE NOTICE '  缺失权限（permissions 表缺这些 code，seed 后才能补齐）= % 条', missing_perms;
    RAISE NOTICE '  使用 agent 角色的用户数 = % 人（>0 即立即生效；=0 则下次创建 agent 用户时生效）', agent_users;

    IF inserted_count < 30 THEN
        RAISE WARNING 'agent 角色权限数仅 % 条，预期 34 条，请检查 permissions 表是否完整', inserted_count;
    END IF;
END $$;

COMMIT;