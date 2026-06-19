-- ========================================================
-- 清理旧的重复流程绑定，只保留最新的
-- ========================================================
-- 1. 查看当前绑定情况
SELECT business_type,
    COUNT(*) as count
FROM process_bindings
WHERE tenant_id = 1
GROUP BY business_type;
-- 2. 删除change类型的重复绑定，只保留第一条(按is_default=true排序)
DELETE FROM process_bindings
WHERE id IN (
        SELECT id
        FROM (
                SELECT id,
                    ROW_NUMBER() OVER (
                        PARTITION BY business_type
                        ORDER BY is_default DESC,
                            created_at DESC
                    ) as rn
                FROM process_bindings
                WHERE tenant_id = 1
            ) sub
        WHERE rn > 1
    );
-- 3. 查看清理后的绑定情况
SELECT id,
    business_type,
    process_definition_key,
    is_default,
    priority
FROM process_bindings
WHERE tenant_id = 1
ORDER BY business_type;