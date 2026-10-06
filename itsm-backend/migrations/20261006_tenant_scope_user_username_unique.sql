-- 20261006_tenant_scope_user_username_unique.sql
--
-- users.username 原本是字段级全局唯一，但管理员用户名按产品基线在每个租户都叫 admin：
-- 第二个租户创建 admin 必然撞全局唯一键，租户开通后该租户永远没有管理员。
-- 这里放宽为「按租户组合唯一」，与 AGENTS.md「业务唯一键默认按 tenant 组合唯一」一致。
--
-- users.email 保持全局唯一：找回密码按 email 单独定位账号（handlers/auth/service.go），
-- 若 email 也按租户唯一，重置流程会出现跨租户歧义。
--
-- 执行顺序说明：bootstrap 先跑 ent 的 client.Schema.Create（它会按新 schema 建好
-- user_tenant_id_username），再跑本文件。所以本文件的职责是删掉遗留的全局唯一约束，
-- CREATE 语句只对「跳过 auto-migrate、只跑版本化迁移」的库生效，两者都写成幂等。

DO $$
DECLARE
    dup_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO dup_count
    FROM (
        SELECT tenant_id, username
        FROM users
        GROUP BY tenant_id, username
        HAVING COUNT(*) > 1
    ) duplicates;
    IF dup_count > 0 THEN
        RAISE EXCEPTION
            'users has % duplicated (tenant_id, username) group(s); deduplicate before adding the unique index',
            dup_count
            USING ERRCODE = 'unique_violation';
    END IF;
END $$;

-- 旧的全局唯一键：ent 用字段级 Unique() 生成，Postgres 命名为 users_username_key。
-- 同时兼容手工建过的同名索引，二者都只在这条约束存在时才会被删除。
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_key;
DROP INDEX IF EXISTS users_username_key;

CREATE UNIQUE INDEX IF NOT EXISTS user_tenant_id_username
    ON users (tenant_id, username);
