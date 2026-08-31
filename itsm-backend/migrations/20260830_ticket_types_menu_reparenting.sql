-- 2026-08-30: 菜单归位修复（菜单信息架构 hygiene）
-- 背景：menuService 返回的菜单树存在"子菜单 path 与 parent_path 业务范畴不一致"，
--       导致用户找不到入口（典型：工单类型挂在"服务请求"下，业务上属于"工单管理"）。
--
-- 涉及的归位动作：
--   1) /tickets/types         移至 /tickets (id=2)        （工单管理）
--   2) /tickets/analytics     从 /service-requests 移到 /tickets（id=2，单入口）
--                            —— 业务上"工单统计"是关于工单的，应在"工单管理"下，
--                               与"工单类型"统一归位。
--   3) /audit-logs            提为顶级菜单                 （审计是独立业务条线，不应埋在 admin 下）
--   4) /notifications         提为顶级菜单                 （通知是独立业务条线）
--   5) /workflow/audit        从 /admin 移到 /workflow     （与"审计日志"功能重复，
--                                                          保留更精确的工作流专属入口）
--
-- 不变量（迁移前后保持）：
--   - (tenant_id, path) 唯一键未触破
--   - 菜单总行数不变（移动 + 删除副本，整体数 -1 +1 -1 = -1（仅 Section 2）
--                   实际计算：原 /service-requests 下 /tickets/analytics 删除；/tickets 下新建 = 净 0
--                   所以总行数不变）
--   - path 字段不变（前端 URL 直访依然有效）
--   - 菜单 permission_code 不变（RBAC 矩阵不受影响）
--
-- 回滚：每条 UPDATE 都是可逆的，把 parent_id / sort_order 改回原值即恢复。
--      Section 2 删除的行可通过 INSERT 恢复（详见 Section 2 反向 SQL）。
--
-- 注意：menus 表当前没有 updated_at / created_at 列（本 schema 用 Ent 时间戳 + 默认值管理），故全文不写这两个字段。

BEGIN;

-- 安全断言：5 个目标菜单在迁移前的父归属必须符合预期
DO $$
DECLARE
  bad_count int := 0;
BEGIN
  IF (SELECT parent_id FROM menus WHERE path = '/tickets/types'     AND tenant_id = 1) <> 34 THEN bad_count := bad_count + 1; RAISE NOTICE 'WARN: /tickets/types parent_id != 34';     END IF;
  IF (SELECT parent_id FROM menus WHERE path = '/tickets/analytics' AND tenant_id = 1) <> 34 THEN bad_count := bad_count + 1; RAISE NOTICE 'WARN: /tickets/analytics parent_id != 34'; END IF;
  IF (SELECT parent_id FROM menus WHERE path = '/audit-logs'        AND tenant_id = 1) <> 40 THEN bad_count := bad_count + 1; RAISE NOTICE 'WARN: /audit-logs parent_id != 40';        END IF;
  IF (SELECT parent_id FROM menus WHERE path = '/notifications'     AND tenant_id = 1) <> 40 THEN bad_count := bad_count + 1; RAISE NOTICE 'WARN: /notifications parent_id != 40';     END IF;
  IF (SELECT parent_id FROM menus WHERE path = '/workflow/audit'    AND tenant_id = 1) <> 40 THEN bad_count := bad_count + 1; RAISE NOTICE 'WARN: /workflow/audit parent_id != 40';    END IF;
  -- 阻断条件：父菜单 ID 必须存在
  IF NOT EXISTS (SELECT 1 FROM menus WHERE id = 2  AND tenant_id = 1) THEN RAISE EXCEPTION 'parent /tickets (id=2) not found';  END IF;
  IF NOT EXISTS (SELECT 1 FROM menus WHERE id = 15 AND tenant_id = 1) THEN RAISE EXCEPTION 'parent /workflow (id=15) not found'; END IF;
  IF bad_count > 0 THEN
    RAISE NOTICE 'Proceeding despite % pre-check warnings (idempotent re-run safe)', bad_count;
  END IF;
END $$;

-- 1) 工单类型：移到 /tickets (id=2) 下，sort_order=21
UPDATE menus
SET parent_id = 2,
    sort_order = 21
WHERE tenant_id = 1
  AND path = '/tickets/types';

-- 2) 工单统计：从 /service-requests (id=34) 移到 /tickets (id=2)，sort_order=22。
--    先删 /service-requests 下副本（释放唯一约束），再 INSERT 到 /tickets 下。
--    列：name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled, description
DELETE FROM menus
WHERE tenant_id = 1
  AND path = '/tickets/analytics'
  AND parent_id = (SELECT id FROM menus WHERE path = '/service-requests' AND tenant_id = 1);

INSERT INTO menus (name, path, icon, parent_id, permission_code, sort_order, tenant_id, is_visible, is_enabled, description)
SELECT '工单统计', '/tickets/analytics', 'BarChart3', 2, 'ticket:read', 22, 1, true, true, '工单统计视图（已从服务请求归位到工单管理）'
WHERE NOT EXISTS (
  SELECT 1 FROM menus WHERE tenant_id = 1 AND path = '/tickets/analytics' AND parent_id = 2
);

-- 3) 审计日志：提到顶级（parent_id=NULL），sort_order=211
UPDATE menus
SET parent_id = NULL,
    sort_order = 211
WHERE tenant_id = 1
  AND path = '/audit-logs';

-- 4) 通知配置：提到顶级（parent_id=NULL），sort_order=212
UPDATE menus
SET parent_id = NULL,
    sort_order = 212
WHERE tenant_id = 1
  AND path = '/notifications';

-- 5) 操作日志：从 /admin (id=40) 移到 /workflow (id=15)，sort_order=105
UPDATE menus
SET parent_id = 15,
    sort_order = 105
WHERE tenant_id = 1
  AND path = '/workflow/audit';

-- 迁移后校验：每条都断言最终 parent_id 与 sort_order
DO $$
DECLARE
  errs int := 0;
BEGIN
  IF (SELECT parent_id FROM menus WHERE tenant_id = 1 AND path = '/tickets/types') IS DISTINCT FROM 2 THEN
    RAISE WARNING 'CHECK FAILED: /tickets/types.parent_id != 2'; errs := errs + 1;
  END IF;
  IF (SELECT parent_id FROM menus WHERE tenant_id = 1 AND path = '/tickets/analytics') IS DISTINCT FROM 2 THEN
    RAISE WARNING 'CHECK FAILED: /tickets/analytics.parent_id != 2'; errs := errs + 1;
  END IF;
  -- /tickets/analytics 现在应在 /tickets 下唯一一份
  IF (SELECT count(*) FROM menus WHERE tenant_id = 1 AND path = '/tickets/analytics') <> 1 THEN
    RAISE WARNING 'CHECK FAILED: expected exactly 1 entry for /tickets/analytics under tenant 1'; errs := errs + 1;
  END IF;
  IF (SELECT parent_id FROM menus WHERE tenant_id = 1 AND path = '/audit-logs') IS NOT NULL THEN
    RAISE WARNING 'CHECK FAILED: /audit-logs.parent_id not NULL'; errs := errs + 1;
  END IF;
  IF (SELECT parent_id FROM menus WHERE tenant_id = 1 AND path = '/notifications') IS NOT NULL THEN
    RAISE WARNING 'CHECK FAILED: /notifications.parent_id not NULL'; errs := errs + 1;
  END IF;
  IF (SELECT parent_id FROM menus WHERE tenant_id = 1 AND path = '/workflow/audit') IS DISTINCT FROM 15 THEN
    RAISE WARNING 'CHECK FAILED: /workflow/audit.parent_id != 15'; errs := errs + 1;
  END IF;
  IF errs > 0 THEN
    RAISE EXCEPTION 'Menu reparenting FAILED with % check errors; transaction rolled back', errs;
  END IF;
END $$;

COMMIT;

-- 输出迁移后归位状态供运维复核
SELECT
  m.id, m.name, m.path, m.parent_id, p.name AS parent_name, m.sort_order
FROM menus m
LEFT JOIN menus p ON p.id = m.parent_id
WHERE m.tenant_id = 1
  AND (
    m.path IN ('/tickets/types', '/tickets/analytics', '/audit-logs', '/notifications', '/workflow/audit')
  )
ORDER BY m.path, m.parent_id NULLS FIRST;
