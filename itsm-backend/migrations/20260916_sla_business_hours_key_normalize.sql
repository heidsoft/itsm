-- ITSM SLA Business Hours Key Normalization Migration
-- Date: 2026-09-16
-- Description: 修复 sla_definitions.business_hours JSONB 内的键名错配（P0 历史欠账）。
--              模板写入键名（workdays/work_hour_start/work_hour_end/is_24_7）与解析器
--              读取键名（work_days/start_time/end_time）不匹配，导致 P1 模板声明 24×7
--              但解析器 fallback 到默认 9-18 工时——全部 SLA 工时错算。
-- ROLLBACK: 无（已存在行按原值保留；修复在应用层兼容两种键名，无需回滚数据）
-- PR: R4-b

-- Step 1: 统一改写已存在的 business_hours JSONB 键名（兼容多种别名）
--         - workdays → work_days
--         - work_hour_start / start_hour → start_time
--         - work_hour_end / end_hour → end_time
--         - time_zone / timezone → time_zone (统一为 time_zone)
--         - is_24_7 → 标记解析器对 24×7 模板会跳过 start_time/end_time
UPDATE sla_definitions
SET business_hours = jsonb_strip_nulls(
    jsonb_build_object(
        'work_days',
        COALESCE(
            business_hours -> 'work_days',
            business_hours -> 'workdays'
        ),
        'start_time',
        COALESCE(
            business_hours ->> 'start_time',
            business_hours ->> 'work_hour_start',
            business_hours ->> 'start_hour'
        ),
        'end_time',
        COALESCE(
            business_hours ->> 'end_time',
            business_hours ->> 'work_hour_end',
            business_hours ->> 'end_hour'
        ),
        'time_zone',
        COALESCE(
            business_hours ->> 'time_zone',
            business_hours ->> 'timezone'
        ),
        'is_24_7',
        COALESCE(business_hours -> 'is_24_7', 'false'::jsonb),
        'holiday_list',
        COALESCE(business_hours -> 'holiday_list', '[]'::jsonb)
    )
)
WHERE business_hours IS NOT NULL
  AND business_hours <> '{}'::jsonb
  AND (
      business_hours ? 'workdays'
      OR business_hours ? 'work_hour_start'
      OR business_hours ? 'work_hour_end'
      OR business_hours ? 'start_hour'
      OR business_hours ? 'end_hour'
      OR business_hours ? 'is_24_7'
  );

-- Step 2: 验证：统计 24×7 模板数（用于跨会话对齐口径）
DO $$
DECLARE
    has_24_7_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO has_24_7_count FROM sla_definitions
    WHERE (business_hours ->> 'is_24_7')::boolean = true;
    RAISE NOTICE 'SLA 业务时间键名规范化完成';
    RAISE NOTICE '  is_24_7=true 的模板数: %', has_24_7_count;
    RAISE NOTICE '  所有 is_24_7 模板自 2026-09-16 起按租户时区计算（不再 fallback 9-18）';
END $$;