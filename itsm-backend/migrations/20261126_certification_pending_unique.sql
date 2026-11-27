-- CMDB 待认证任务部分唯一索引
-- Date: 2026-11-26
-- Author: Codex（阻塞性预检，不是自动去重）
-- Description: 同租户同 CI 只允许一条 pending；confirmed/cancelled 历史保留。
-- 数据策略：报告全部重复 tenant_id/ci_id 后失败；人工确认保留记录，
--   将其余置为 cancelled/confirmed（confirmed 必须符合真实认证事实）后重跑。
-- 事务内阻止并发写入，关闭预检与建索引之间的窗口；大表部署需预留锁窗口。

BEGIN;
LOCK TABLE ci_certification_tasks IN SHARE MODE;

DO $$
DECLARE
  duplicates TEXT;
BEGIN
  SELECT string_agg(
    format('tenant_id=%s/ci_id=%s (pending=%s)', tenant_id, ci_id, pending_count),
    ', ' ORDER BY tenant_id, ci_id
  )
  INTO duplicates
  FROM (
    SELECT tenant_id, ci_id, COUNT(*) AS pending_count
    FROM ci_certification_tasks
    WHERE status = 'pending'
    GROUP BY tenant_id, ci_id
    HAVING COUNT(*) > 1
  ) AS repeated;

  IF duplicates IS NOT NULL THEN
    RAISE EXCEPTION 'Duplicate pending certification tasks: %. 人工确认哪条保留，把其余置为 cancelled/confirmed 后重新运行迁移；不得伪造认证事实。', duplicates;
  END IF;
END;
$$;

CREATE UNIQUE INDEX IF NOT EXISTS ci_cert_pending_unique
  ON ci_certification_tasks (tenant_id, ci_id) WHERE status = 'pending';
COMMIT;
