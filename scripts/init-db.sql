-- 创建CMDB数据库
CREATE DATABASE itsm_cmdb;

-- 创建用户和权限
CREATE USER cmdb_user WITH PASSWORD 'cmdb_password';
GRANT ALL PRIVILEGES ON DATABASE itsm_cmdb TO cmdb_user;

-- 连接到CMDB数据库
\c itsm_cmdb;

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 设置时区
SET timezone = 'Asia/Shanghai';
