-- B2（2026-09-28）：把 change.affected_cis 从 JSON 字段迁移到 M2M 关联表 change_affected_cis。
--
-- 背景
--   1. change.affected_cis 原为 field.JSON（字符串数组），无参照完整性、无法 JOIN，
--      影响分析只能 strconv.Atoi 后按 ID 反查（handlers/change/service.go GetCMDBImpactSummary）。
--   2. 该列存在两套互斥语义：handlers/change（生产唯一接线运行时）按「数字 CI ID」写入；
--      已停用的 service/change_service.go 按「CI 名称」写入并按 NameIn 校验。
--   3. 现已收敛为 Change ↔ ConfigurationItem 的多对多 edge，关联表 change_affected_cis
--      （列：change_id, configuration_item_id）。
--
-- 执行时机
--   必须在 client.Schema.Create（自动建表）之后运行；Schema.Create 不带 WithDropColumn，
--   故旧的 changes.affected_cis 列会保留，可作为本迁移的数据源与回滚依据。
--
-- 数据策略
--   - 仅回填可解析为数字且能在「同租户」CI 表命中的项（即现行运行时写入的合法数据）；
--   - 名称形态的存量无法构成关系，不回填，保留在原 JSON 列供人工核对，不静默丢弃；
--   - ON CONFLICT DO NOTHING，可重复执行。
--
-- 注意：关联表无 tenant_id，租户隔离由两端实体保证；已在
-- internal/schema/tenant_guard.go 登记为 derived 豁免（否则生产 policy=fatal 拒绝启动）。

INSERT INTO change_affected_cis (change_id, configuration_item_id)
SELECT c.id, ci.id
FROM changes c
CROSS JOIN LATERAL jsonb_array_elements_text(
    CASE
        WHEN c.affected_cis IS NULL OR c.affected_cis::text IN ('', 'null', '[]') THEN '[]'::jsonb
        ELSE c.affected_cis::jsonb
    END
) AS v(ci_raw)
JOIN configuration_items ci
  ON ci.tenant_id = c.tenant_id
 AND ci.id = v.ci_raw::int
WHERE v.ci_raw ~ '^[0-9]+$'
ON CONFLICT DO NOTHING;
