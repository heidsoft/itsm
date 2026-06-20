-- ITSM Marketplace 迁移脚本
-- Date: 2026-06-20
-- Description: 创建 marketplace_items / item_versions / tenant_installations 表
-- 依据: itsm-backend/ent/schema/marketplace_item.go, item_version.go, tenant_installation.go

BEGIN;

-- ==================== marketplace_items 主表 ====================
CREATE TABLE IF NOT EXISTS marketplace_items (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('connector', 'skill', 'plugin')),
    title VARCHAR(255) NOT NULL,
    provider VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    long_description TEXT NOT NULL DEFAULT '',
    icon_url VARCHAR(1024) NOT NULL DEFAULT '',
    screenshots JSONB NOT NULL DEFAULT '[]'::jsonb,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    rating DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    install_count INTEGER NOT NULL DEFAULT 0,
    latest_version VARCHAR(50) NOT NULL DEFAULT '',
    min_system_version VARCHAR(50) NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'reviewing', 'published', 'rejected', 'deprecated')),
    is_official BOOLEAN NOT NULL DEFAULT false,
    is_free BOOLEAN NOT NULL DEFAULT true,
    price DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    category VARCHAR(255) NOT NULL DEFAULT '',
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    required_permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
    config_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    author_id VARCHAR(255) NOT NULL DEFAULT '',
    author_name VARCHAR(255) NOT NULL DEFAULT '',
    homepage VARCHAR(1024) NOT NULL DEFAULT '',
    repository VARCHAR(1024) NOT NULL DEFAULT '',
    license VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_marketplace_items_type_category ON marketplace_items(type, category);
CREATE INDEX IF NOT EXISTS idx_marketplace_items_status_official ON marketplace_items(status, is_official);
CREATE INDEX IF NOT EXISTS idx_marketplace_items_rating_install ON marketplace_items(rating, install_count);

-- ==================== item_versions 版本表 ====================
CREATE TABLE IF NOT EXISTS item_versions (
    id BIGSERIAL PRIMARY KEY,
    item_id BIGINT NOT NULL REFERENCES marketplace_items(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    changelog TEXT NOT NULL DEFAULT '',
    download_url VARCHAR(1024) NOT NULL DEFAULT '',
    manifest_path VARCHAR(1024) NOT NULL DEFAULT '',
    min_system_version VARCHAR(50) NOT NULL DEFAULT '',
    dependencies JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(50) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'deprecated', 'withdrawn')),
    download_count INTEGER NOT NULL DEFAULT 0,
    released_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    config_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(item_id, version)
);

CREATE INDEX IF NOT EXISTS idx_item_versions_status_released ON item_versions(status, released_at);

-- ==================== tenant_installations 安装记录表 ====================
CREATE TABLE IF NOT EXISTS tenant_installations (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    item_id BIGINT NOT NULL REFERENCES marketplace_items(id) ON DELETE CASCADE,
    installed_version VARCHAR(50) NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'installing' CHECK (status IN ('installing', 'active', 'disabled', 'failed', 'uninstalled')),
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    auto_upgrade BOOLEAN NOT NULL DEFAULT true,
    installed_by VARCHAR(255) NOT NULL DEFAULT '',
    installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_updated_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, item_id)
);

CREATE INDEX IF NOT EXISTS idx_tenant_installations_status ON tenant_installations(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_tenant_installations_installed ON tenant_installations(status, installed_at);

-- ==================== 种子数据 ====================
INSERT INTO marketplace_items (name, type, title, provider, description, latest_version, status, is_official, is_free, category, capabilities, tags)
VALUES
  ('feishu-connector', 'connector', '飞书连接器', 'ITSM官方', '集成飞书消息发送/审批回调', 'v1.0.0', 'published', true, true, '办公协同', '["message","approval"]'::jsonb, '["即时通讯","审批"]'::jsonb),
  ('slack-connector', 'connector', 'Slack连接器', '社区', '集成Slack消息', 'v0.9.0', 'published', false, true, '办公协同', '["message"]'::jsonb, '["即时通讯"]'::jsonb),
  ('triage-skill', 'skill', 'AI智能分诊', 'ITSM官方', '基于LLM的工单智能分诊', 'v2.1.0', 'published', true, true, 'AI能力', '["classify","priority"]'::jsonb, '["AI","NLP"]'::jsonb),
  ('rag-skill', 'skill', 'RAG知识检索', 'ITSM官方', '向量数据库增强的知识库检索', 'v1.5.0', 'published', true, true, 'AI能力', '["search","embeddings"]'::jsonb, '["AI","向量检索"]'::jsonb),
  ('prometheus-plugin', 'plugin', 'Prometheus监控', '社区', '集成Prometheus告警', 'v1.2.3', 'published', false, true, '监控告警', '["metrics","alerts"]'::jsonb, '["监控"]'::jsonb)
ON CONFLICT (name) DO NOTHING;

-- 同步版本
INSERT INTO item_versions (item_id, version, changelog, status)
SELECT id, latest_version, '初版发布', 'published'
FROM marketplace_items
WHERE NOT EXISTS (SELECT 1 FROM item_versions WHERE item_versions.item_id = marketplace_items.id AND item_versions.version = marketplace_items.latest_version)
ON CONFLICT (item_id, version) DO NOTHING;

COMMIT;