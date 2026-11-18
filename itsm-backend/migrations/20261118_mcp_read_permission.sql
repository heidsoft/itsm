-- ITSM MCP 通道权限码播种迁移
-- Date: 2026-11-18
-- Description: 补齐 MCP 通道准入码 `mcp:read` 在既有部署中的权限码与角色关联。
--
-- 背景（2026-11-18 MCP 客户端真实联调时暴露）：
--   `mcp:read` 是 2026-11-18 新增的权限码（internal/authz/catalog.go 单一源）。
--   既有部署的 permissions 表由 seeder 播种，而生产 compose 的
--   ITSM_AUTO_SEED=false —— seeder 不在每次启动跑，**新码不会自动进入存量库**。
--   后果：MCP 端点已注册（路由返回 401「缺少认证token」证明在），但任何持
--   角色的请求都被 RequirePermission 判 403「权限不足」——因为 DB 里根本没有
--   `mcp:read` 这行，AuthorizeResource 查不到 → 一律拒绝。
--   这就是「新增权限码必须补存量播种」的部署缺口，代码侧的 seeder 修不了
--   已经跑过的实例。
--
-- 本迁移做两件事：
--   1. 往 permissions 插入 `mcp:read`（带 v1 策略信封）；
--   2. 给持有该码的角色建立 role_permissions 关联。
--
-- 权限边界（最小授权，与 internal/authz/roles.go 保持一致）：
--   - agent      —— MCP 的典型客户端（自动化代理通过协议发现工具面）
--   - audit_admin —— service 主体的载体角色（cmd/service-token-gen 签发的
--     service JWT 固定用它），必须能发现工具面
--   - sysadmin   —— 经 allPermissionCodes() 全量派生
--   刻意**不授予** end_user：自助旅程身份不进入对外协议编排面。
--
-- 策略（v1 信封）：
--   allowedPrincipals 恒为 ["human"] —— 这是 AI/服务主体零权限的硬不变量。
--   service 主体能否**调用**具体工具，由执行链（ExecuteTool 的 Gate 2.5）
--   按工具自身的 resource:action 逐个判定，须管理员在运行期策略显式放行；
--   本码只管「能否经 MCP 通道发现工具面」。
--
-- 幂等性说明：
--   - permissions 表无 (code, tenant_id) 唯一约束，插入用 WHERE NOT EXISTS；
--   - role_permissions 表无唯一约束，关联插入同样用 WHERE NOT EXISTS；
--   - 可重复执行。
--
-- ROLLBACK:
--   DELETE FROM role_permissions WHERE permission_id = (
--     SELECT id FROM permissions WHERE code = 'mcp:read' AND tenant_id = 1);
--   DELETE FROM permissions WHERE code = 'mcp:read' AND tenant_id = 1;

BEGIN;

-- Step 1: 插入 mcp:read 权限码（所有既有租户，不只 default）
--
-- 按租户插入而非只插 tenant_id=1：SaaS/MSP 部署下每个租户各有独立权限表，
-- 只补 default 租户会让其它租户的 MCP 通道同样 403。
-- description 内嵌 v1 策略信封；allowedPrincipals 恒为 ["human"]。
--
-- 🔴 category 必须取合法闭集值（authz 的 ValidatePolicy 会校验）：
--   serviceOps / identity / platform / assets / analytics / aiExtensions /
--   collaboration / other
-- 写错的后果不是「这一个码显示异常」，而是**整个权限目录解码失败**
-- （permission definition corrupt → 全部权限判定 fail-closed）——
-- 本迁移初版写了 "aiCapability"（不在闭集内），导致该租户所有权限判定失效。
-- mcp 是 AI 能力通道，与 ai:* 同类，故取 aiExtensions。
INSERT INTO permissions (code, name, description, resource, action, tenant_id, created_at, updated_at)
SELECT 'mcp:read',
       '查看MCP工具面',
       '{"v":1,"description":"列出 MCP 通道暴露的工具与能力","policy":{"riskLevel":"low","allowedPrincipals":["human"],"requiresApproval":false,"minApprovals":0,"category":"aiExtensions","isActive":true,"isSystem":true}}',
       'mcp',
       'read',
       t.id,
       NOW(),
       NOW()
FROM tenants t
WHERE NOT EXISTS (
    SELECT 1 FROM permissions p
    WHERE p.code = 'mcp:read' AND p.tenant_id = t.id
);

-- Step 2: 建立角色关联（agent / audit_admin）
--
-- 用码字取角色 id，不硬编码 id —— 角色 id 在不同部署里不同。
DO $$
DECLARE
    perm_id INTEGER;
    role_rec RECORD;
BEGIN
    SELECT id INTO perm_id FROM permissions WHERE code = 'mcp:read' AND tenant_id = 1 LIMIT 1;
    IF perm_id IS NULL THEN
        RAISE EXCEPTION 'mcp:read 权限码插入失败（tenant_id=1）';
    END IF;

    FOR role_rec IN
        SELECT r.id, r.tenant_id
        FROM roles r
        WHERE r.code IN ('agent', 'audit_admin')
    LOOP
        -- 每个角色按其自身租户找对应租户的权限码
        INSERT INTO role_permissions (role_id, permission_id, tenant_id)
        SELECT role_rec.id, p.id, role_rec.tenant_id
        FROM permissions p
        WHERE p.code = 'mcp:read'
          AND p.tenant_id = role_rec.tenant_id
          AND NOT EXISTS (
              SELECT 1 FROM role_permissions rp
              WHERE rp.role_id = role_rec.id
                AND rp.permission_id = p.id
          );
    END LOOP;
END $$;

COMMIT;
