-- 20261014_create_sso_tables.sql
--
-- SSO（OIDC）后端落地：提供方配置 + 外部身份绑定。
--   - sso_providers：租户级 IdP 配置，client_secret 只存 AES-GCM 密文
--     （middleware.EncryptionService，与 connector_configs.encrypted_credentials 同口径）；
--     端点字段可空，为空时按 OIDC Discovery 从 issuer 派生。
--   - sso_identities：(provider_id, subject) → user_id 绑定，租户隔离由 tenant_id 承担。
--
-- 执行顺序说明：bootstrap 先跑 ent 的 client.Schema.Create（按 ent/schema/sso_provider.go、
-- sso_identity.go 建好两张表），再跑本文件。本文件面向「跳过 auto-migrate、只跑版本化迁移」
-- 的库，全部幂等可重复执行。行级隔离策略与 20260828_create_alerts.sql 一致。

CREATE TABLE IF NOT EXISTS sso_providers (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    protocol VARCHAR(20) NOT NULL DEFAULT 'oidc',
    client_id VARCHAR(200) NOT NULL,
    client_secret_encrypted TEXT NULL,
    issuer VARCHAR(500) NOT NULL,
    authorization_endpoint VARCHAR(1000) NULL,
    token_endpoint VARCHAR(1000) NULL,
    userinfo_endpoint VARCHAR(1000) NULL,
    jwks_uri VARCHAR(1000) NULL,
    scopes VARCHAR(500) NOT NULL DEFAULT 'openid profile email',
    redirect_uri VARCHAR(1000) NOT NULL,
    auto_provision BOOLEAN NOT NULL DEFAULT true,
    default_role VARCHAR(50) NOT NULL DEFAULT 'end_user',
    allowed_email_domains VARCHAR(1000) NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(40) NOT NULL DEFAULT 'configured',
    last_error VARCHAR(2000) NULL,
    last_health_check_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 租户内提供方编码唯一；登录发起无租户上下文，按 (code, enabled) 解析。
CREATE UNIQUE INDEX IF NOT EXISTS ssoprovider_tenant_id_code
    ON sso_providers (tenant_id, code);
CREATE INDEX IF NOT EXISTS ssoprovider_code_enabled
    ON sso_providers (code, enabled);

CREATE TABLE IF NOT EXISTS sso_identities (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    provider_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    subject VARCHAR(255) NOT NULL,
    external_username VARCHAR(255) NULL,
    external_email VARCHAR(255) NULL,
    last_login_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 同一提供方内 subject 唯一（回调定位用户）；
-- 同一用户在同一提供方只保留一条绑定（解绑/重绑不累积历史行）。
CREATE UNIQUE INDEX IF NOT EXISTS ssoidentity_tenant_id_provider_id_subject
    ON sso_identities (tenant_id, provider_id, subject);
CREATE UNIQUE INDEX IF NOT EXISTS ssoidentity_tenant_id_user_id_provider_id
    ON sso_identities (tenant_id, user_id, provider_id);
CREATE INDEX IF NOT EXISTS ssoidentity_provider_id
    ON sso_identities (provider_id);

ALTER TABLE sso_providers ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_sso_providers ON sso_providers;
CREATE POLICY tenant_isolation_sso_providers ON sso_providers
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::BIGINT);

ALTER TABLE sso_identities ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_sso_identities ON sso_identities;
CREATE POLICY tenant_isolation_sso_identities ON sso_identities
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::BIGINT);
