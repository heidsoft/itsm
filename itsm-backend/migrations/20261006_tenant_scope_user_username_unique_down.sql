-- 20261006_tenant_scope_user_username_unique_down.sql
--
-- 回滚 20261006_tenant_scope_user_username_unique.sql：恢复 username 全局唯一。
--
-- 这条回滚是「有前提的」：本次放宽之后，多租户库里合法存在多个同名 admin，
-- 直接重建全局唯一键必然失败。因此回滚前必须先人工消解跨租户重名（改名或合并账号），
-- 本文件只做检测并拒绝，不自动改数据。前提不成立时正确处置是前滚修复，不是回滚。

DO $$
DECLARE
    dup_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT username
        FROM users
        GROUP BY username
        HAVING COUNT(*) > 1
    ) duplicates;
    IF dup_count > 0 THEN
        RAISE EXCEPTION
            'users has % globally duplicated username(s) across tenants; resolve them before restoring the global unique constraint',
            dup_count
            USING ERRCODE = 'unique_violation';
    END IF;
END $$;

DROP INDEX IF EXISTS user_tenant_id_username;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_key;
ALTER TABLE users ADD CONSTRAINT users_username_key UNIQUE (username);
