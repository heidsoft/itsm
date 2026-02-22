-- ITSM 系统数据库初始化脚本
-- 用于 Docker PostgreSQL 容器首次启动时执行

-- 创建 ITSM 数据库
CREATE DATABASE itsm_cmdb;

-- 连接到 ITSM 数据库
\c itsm_cmdb;

-- 创建必要的扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- 设置时区
SET timezone = 'Asia/Shanghai';

-- 创建角色（可选）
-- DO $$ BEGIN
--     IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'itsm_user') THEN
--         CREATE ROLE itsm_user WITH LOGIN PASSWORD 'itsm_password';
--     END IF;
-- END $$;

-- 授予权限（如果使用了上面创建的角色）
-- GRANT ALL PRIVILEGES ON DATABASE itsm_cmdb TO itsm_user;
-- GRANT ALL ON SCHEMA public TO itsm_user;
-- ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO itsm_user;

-- 创建初始的租户表（如果需要的话，在 Ent 迁移之前）
-- 注意：Ent ORM 会自动创建表结构，这里主要是为了确保数据库正确设置

COMMENT ON DATABASE itsm_cmdb IS 'IT Service Management System Database';
