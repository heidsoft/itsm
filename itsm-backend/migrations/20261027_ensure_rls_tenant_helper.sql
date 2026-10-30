-- =============================================================================
-- 20261027_ensure_rls_tenant_helper.sql
--
-- 目的：补齐 RLS 谓词依赖的租户助手函数 public.get_current_tenant_id()。
--
-- 背景（为什么需要本文件）：
--   全部 RLS policy 的谓词都是 tenant_id = get_current_tenant_id()，
--   该函数的定义原本由历史迁移 009_enable_rls_tenant_isolation 声明、
--   019_align_rls_tenant_variable 对齐到 app.current_tenant 会话变量。
--   这两个历史迁移的源文件已从仓库删除，因此：
--     · 既有安装（prod）—— 函数在建 policy 时已落地，账本里有 009/019 记录，不受影响；
--     · 全新安装 —— 磁盘流与 Go 注册表都没有函数定义，建 policy 时必然报
--       "function get_current_tenant_id() does not exist"，整条初始化链中断。
--   20261028_rls_core_tables.sql 里的守卫（函数缺失即 RAISE EXCEPTION）
--   正是为暴露这个缺口而设，本文件让全新安装能够真正走通。
--
-- 定义来源：取自生产库的 pg_get_functiondef 权威定义（public schema），
-- 与 database/rls/driver.go 注入的会话变量名 app.current_tenant 严格一致。
--
-- 幂等性：CREATE OR REPLACE FUNCTION，同签名覆盖，既有环境重放是 no-op。
--   对已有该函数的库，替换后的定义与原定义一致，行为不变。
--
-- 顺序：本文件的版本号 20261027 早于依赖它的 20261028_rls_core_tables.sql，
--   迁移流按版本号字典序执行（migration/discovery.go: sort.Strings(order)），
--   因此 helper 一定先于 consumer 建立。
-- =============================================================================

CREATE OR REPLACE FUNCTION public.get_current_tenant_id()
RETURNS integer
LANGUAGE plpgsql
STABLE
AS $function$
BEGIN
    RETURN NULLIF(current_setting('app.current_tenant', true), '')::INTEGER;
EXCEPTION
    WHEN invalid_text_representation THEN
        RETURN NULL;
END;
$function$;

COMMENT ON FUNCTION public.get_current_tenant_id() IS
    'Tenant id read from the app.current_tenant session variable; NULL when unset/invalid (RLS fail-closed)';
