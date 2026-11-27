-- CMDB CI 身份键部分唯一索引
-- Date: 2026-11-26
-- Author: Codex（阻塞性预检，不是自动去重）
-- Description: 同租户资产标签及完整云身份由数据库防止并发重复写入。
-- 数据策略：报告全部重复键，人工确认保留行；其余修改身份字段或显式置 NULL 后重跑。
-- 不自动删除或合并 CI。大表部署需预留 SHARE 锁及普通索引构建窗口。
BEGIN;
LOCK TABLE configuration_items IN SHARE MODE;
DO $$
DECLARE
  asset_duplicates TEXT;
  cloud_duplicates TEXT;
BEGIN
  SELECT string_agg(format('tenant_id=%s/asset_tag=%L (count=%s)', tenant_id, asset_tag, n), ', ' ORDER BY tenant_id, asset_tag)
    INTO asset_duplicates
    FROM (SELECT tenant_id, asset_tag, COUNT(*) n FROM configuration_items
      WHERE asset_tag IS NOT NULL AND asset_tag <> ''
      GROUP BY tenant_id, asset_tag HAVING COUNT(*) > 1) repeated;
  SELECT string_agg(format('tenant_id=%s/provider=%L/account_id=%L/resource_id=%L (count=%s)', tenant_id, cloud_provider, cloud_account_id, cloud_resource_id, n), ', ' ORDER BY tenant_id, cloud_provider, cloud_account_id, cloud_resource_id)
    INTO cloud_duplicates
    FROM (SELECT tenant_id, cloud_provider, cloud_account_id, cloud_resource_id, COUNT(*) n FROM configuration_items
      WHERE cloud_provider IS NOT NULL AND cloud_provider <> ''
        AND cloud_resource_id IS NOT NULL AND cloud_resource_id <> ''
        AND cloud_account_id IS NOT NULL
      GROUP BY tenant_id, cloud_provider, cloud_account_id, cloud_resource_id HAVING COUNT(*) > 1) repeated;
  IF asset_duplicates IS NOT NULL OR cloud_duplicates IS NOT NULL THEN
    RAISE EXCEPTION 'Duplicate CI identities: asset_tag=[%]; cloud=[%]. 人工确认保留行，其余修改 asset_tag/cloud 字段或显式置 NULL 后重跑；禁止静默去重。', COALESCE(asset_duplicates, ''), COALESCE(cloud_duplicates, '');
  END IF;
END;
$$;
CREATE UNIQUE INDEX IF NOT EXISTS ci_asset_tag_tenant_unique
  ON configuration_items (tenant_id, asset_tag)
  WHERE asset_tag IS NOT NULL AND asset_tag <> '';
-- PostgreSQL 默认 NULLS DISTINCT：account_id=NULL 的相同 provider/resource 可有多行。
-- findExistingCI 仅在提供 account_id 时过滤账户；省略账户的查询仍可能逐行报身份歧义。
-- account_id='' 是非 NULL 值，同 tenant/provider/空账户/resource 仍受唯一约束；
-- provider/resource 的 NULL 与空串不占唯一槽。region 不属于唯一身份。
CREATE UNIQUE INDEX IF NOT EXISTS ci_cloud_identity_unique
  ON configuration_items (tenant_id, cloud_provider, cloud_account_id, cloud_resource_id)
  WHERE cloud_provider IS NOT NULL AND cloud_provider <> ''
    AND cloud_resource_id IS NOT NULL AND cloud_resource_id <> '';
COMMIT;
