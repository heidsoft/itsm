-- ITSM 数据库初始化脚本

-- 启用扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 创建部署表
CREATE TABLE IF NOT EXISTS deployments (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    plan VARCHAR(20) NOT NULL,
    instance_id VARCHAR(50) UNIQUE NOT NULL,
    instance_name VARCHAR(100) NOT NULL,
    domain VARCHAR(200),
    status VARCHAR(20) NOT NULL DEFAULT 'deploying',
    region VARCHAR(50),
    ip_address VARCHAR(50),
    username VARCHAR(100),
    password VARCHAR(200),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX idx_deployments_user_id ON deployments(user_id);
CREATE INDEX idx_deployments_status ON deployments(status);
CREATE INDEX idx_deployments_created_at ON deployments(created_at);

-- 插入示例数据
INSERT INTO deployments (id, user_id, plan, instance_id, instance_name, domain, status, region, username)
VALUES 
    ('1', 'u001', 'pro', 'i-001', 'openclaw-u001-pro', 'u001.openclaw.cn', 'running', 'cn-shanghai', 'user1'),
    ('2', 'u002', 'pro', 'i-002', 'openclaw-u002-pro', 'u002.openclaw.cn', 'running', 'cn-shanghai', 'user2'),
    ('3', 'u003', 'community', 'i-003', 'openclaw-u003-comm', 'u003.openclaw.cn', 'deploying', 'cn-shanghai', 'user3')
ON CONFLICT (id) DO NOTHING;

-- 创建监控数据表
CREATE TABLE IF NOT EXISTS deployment_metrics (
    id BIGSERIAL PRIMARY KEY,
    deployment_id VARCHAR(36) NOT NULL,
    cpu_usage FLOAT,
    memory_usage FLOAT,
    disk_usage FLOAT,
    network_in BIGINT,
    network_out BIGINT,
    qps BIGINT,
    response_time INT,
    timestamp BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
);

CREATE INDEX idx_metrics_deployment_id ON deployment_metrics(deployment_id);
CREATE INDEX idx_metrics_timestamp ON deployment_metrics(timestamp);

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(200) UNIQUE NOT NULL,
    password_hash VARCHAR(200) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 插入默认管理员
INSERT INTO users (id, username, email, password_hash, role)
VALUES ('admin-001', 'admin', 'admin@openclaw.cn', '$2a$10$hashed_password', 'admin')
ON CONFLICT (id) DO NOTHING;

-- 创建告警表
CREATE TABLE IF NOT EXISTS alerts (
    id VARCHAR(36) PRIMARY KEY,
    deployment_id VARCHAR(36) NOT NULL,
    type VARCHAR(50) NOT NULL,
    level VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    acknowledged_at TIMESTAMP,
    acknowledged_by VARCHAR(36),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
);

CREATE INDEX idx_alerts_deployment_id ON alerts(deployment_id);
CREATE INDEX idx_alerts_status ON alerts(status);

-- 创建日志表
CREATE TABLE IF NOT EXISTS deployment_logs (
    id BIGSERIAL PRIMARY KEY,
    deployment_id VARCHAR(36) NOT NULL,
    level VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE CASCADE
);

CREATE INDEX idx_logs_deployment_id ON deployment_logs(deployment_id);
CREATE INDEX idx_logs_created_at ON deployment_logs(created_at);

-- 更新更新时间触发器
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_deployments_updated_at BEFORE UPDATE ON deployments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 显示表信息
\dt
\echo '数据库初始化完成！'
