-- AI-Native NOC 邮件智能报障 PoC
-- PostgreSQL production migration. Ent Schema.Create remains the development bootstrap path.

CREATE TABLE IF NOT EXISTS service_customers (
  id BIGSERIAL PRIMARY KEY, tenant_id BIGINT NOT NULL, name VARCHAR(255) NOT NULL,
  normalized_name VARCHAR(255) NOT NULL, short_name VARCHAR(120), aliases JSONB DEFAULT '[]',
  historical_names JSONB DEFAULT '[]', status VARCHAR(32) NOT NULL DEFAULT 'active',
  linked_customer_tenant_id BIGINT, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, normalized_name)
);

CREATE TABLE IF NOT EXISTS customer_branches (
  id BIGSERIAL PRIMARY KEY, tenant_id BIGINT NOT NULL, customer_id BIGINT NOT NULL REFERENCES service_customers(id),
  name VARCHAR(255) NOT NULL, normalized_name VARCHAR(255) NOT NULL, aliases JSONB DEFAULT '[]', status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, customer_id, normalized_name)
);

CREATE TABLE IF NOT EXISTS source_organizations (
  id BIGSERIAL PRIMARY KEY, tenant_id BIGINT NOT NULL, name VARCHAR(255) NOT NULL, normalized_name VARCHAR(255) NOT NULL,
  email_addresses JSONB DEFAULT '[]', email_domains JSONB DEFAULT '[]', status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE (tenant_id, normalized_name)
);

CREATE TABLE IF NOT EXISTS support_contracts (
  id BIGSERIAL PRIMARY KEY, tenant_id BIGINT NOT NULL, customer_id BIGINT NOT NULL REFERENCES service_customers(id),
  branch_id BIGINT REFERENCES customer_branches(id), contract_number VARCHAR(160) NOT NULL, normalized_contract_number VARCHAR(160) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active', start_at TIMESTAMPTZ, end_at TIMESTAMPTZ, source_document_id BIGINT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE (tenant_id, normalized_contract_number)
);

CREATE TABLE IF NOT EXISTS external_contract_references (
  id BIGSERIAL PRIMARY KEY, tenant_id BIGINT NOT NULL, source_organization_id BIGINT NOT NULL REFERENCES source_organizations(id),
  support_contract_id BIGINT NOT NULL REFERENCES support_contracts(id), customer_id BIGINT NOT NULL, branch_id BIGINT,
  external_contract_number VARCHAR(160) NOT NULL, normalized_external_contract_number VARCHAR(160) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, source_organization_id, normalized_external_contract_number)
);

CREATE TABLE IF NOT EXISTS email_conversations (
  id BIGSERIAL PRIMARY KEY, tenant_id BIGINT NOT NULL, external_thread_id VARCHAR(512), conversation_token VARCHAR(120) NOT NULL,
  source_organization_id BIGINT REFERENCES source_organizations(id), customer_id BIGINT REFERENCES service_customers(id),
  branch_id BIGINT REFERENCES customer_branches(id), support_contract_id BIGINT REFERENCES support_contracts(id),
  status VARCHAR(40) NOT NULL DEFAULT 'PROCESSING', canonical_data JSONB DEFAULT '{}', field_sources JSONB DEFAULT '{}',
  missing_fields JSONB DEFAULT '[]', confidence DOUBLE PRECISION NOT NULL DEFAULT 0, version BIGINT NOT NULL DEFAULT 1,
  last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, conversation_token)
);

CREATE TABLE IF NOT EXISTS inbound_email_messages (
  id BIGSERIAL PRIMARY KEY, tenant_id BIGINT NOT NULL, conversation_id BIGINT NOT NULL REFERENCES email_conversations(id),
  provider VARCHAR(40) NOT NULL DEFAULT 'imap', mailbox_instance_key VARCHAR(160) NOT NULL, uid_validity BIGINT NOT NULL, uid BIGINT NOT NULL,
  external_message_id VARCHAR(512), in_reply_to VARCHAR(512), "references" JSONB DEFAULT '[]', from_address VARCHAR(320) NOT NULL,
  to_addresses JSONB DEFAULT '[]', reply_to_address VARCHAR(320), subject VARCHAR(998), plain_text TEXT, sanitized_html TEXT,
  raw_mime BYTEA, raw_sha256 VARCHAR(64) NOT NULL, attachment_metadata JSONB DEFAULT '[]', processing_status VARCHAR(40) NOT NULL DEFAULT 'RECEIVED',
  last_error VARCHAR(2000), received_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, mailbox_instance_key, uid_validity, uid)
);

CREATE TABLE IF NOT EXISTS email_intake_analyses (
  id BIGSERIAL PRIMARY KEY, tenant_id BIGINT NOT NULL, conversation_id BIGINT NOT NULL REFERENCES email_conversations(id),
  message_id BIGINT NOT NULL REFERENCES inbound_email_messages(id), provider VARCHAR(80), model VARCHAR(160), prompt_version VARCHAR(80) NOT NULL,
  raw_result TEXT, result JSONB DEFAULT '{}', confidence DOUBLE PRECISION NOT NULL DEFAULT 0, latency_ms BIGINT NOT NULL DEFAULT 0,
  status VARCHAR(40) NOT NULL DEFAULT 'pending', validation_error VARCHAR(2000), reviewed_by BIGINT, corrections JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS email_outbound_messages (
  id BIGSERIAL PRIMARY KEY, tenant_id BIGINT NOT NULL, conversation_id BIGINT NOT NULL REFERENCES email_conversations(id),
  mailbox_instance_key VARCHAR(160) NOT NULL, reply_type VARCHAR(40) NOT NULL, revision BIGINT NOT NULL, to_address VARCHAR(320) NOT NULL,
  subject VARCHAR(998) NOT NULL, body_text TEXT NOT NULL, in_reply_to VARCHAR(512), "references" JSONB DEFAULT '[]',
  status VARCHAR(40) NOT NULL DEFAULT 'PENDING', attempts BIGINT NOT NULL DEFAULT 0, last_error VARCHAR(2000), sent_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, conversation_id, reply_type, revision)
);

CREATE TABLE IF NOT EXISTS on_call_schedules (
  id BIGSERIAL PRIMARY KEY, tenant_id BIGINT NOT NULL, group_id BIGINT NOT NULL REFERENCES groups(id), name VARCHAR(160) NOT NULL,
  timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai', status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE (tenant_id, group_id, name)
);

CREATE TABLE IF NOT EXISTS on_call_shifts (
  id BIGSERIAL PRIMARY KEY, tenant_id BIGINT NOT NULL, schedule_id BIGINT NOT NULL REFERENCES on_call_schedules(id), user_id BIGINT NOT NULL REFERENCES users(id),
  start_at TIMESTAMPTZ NOT NULL, end_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (end_at > start_at)
);

CREATE TABLE IF NOT EXISTS connector_configs (
  id BIGSERIAL PRIMARY KEY, tenant_id BIGINT NOT NULL, name VARCHAR(100) NOT NULL, provider VARCHAR(100) NOT NULL,
  connector_type VARCHAR(50), enabled BOOLEAN NOT NULL DEFAULT FALSE, encrypted_credentials TEXT, settings JSONB DEFAULT '{}', labels JSONB DEFAULT '{}',
  status VARCHAR(40) NOT NULL DEFAULT 'configured', last_error VARCHAR(2000), last_health_check_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE (tenant_id, name, provider)
);

ALTER TABLE incidents ADD COLUMN IF NOT EXISTS assignment_group_id BIGINT REFERENCES groups(id);
ALTER TABLE incidents ADD COLUMN IF NOT EXISTS email_conversation_id BIGINT REFERENCES email_conversations(id);
CREATE UNIQUE INDEX IF NOT EXISTS incidents_tenant_email_conversation_uq ON incidents (tenant_id, email_conversation_id) WHERE email_conversation_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS email_conversations_queue_idx ON email_conversations (tenant_id, status, last_message_at DESC);
CREATE INDEX IF NOT EXISTS inbound_email_messages_processing_idx ON inbound_email_messages (tenant_id, processing_status, received_at);
CREATE INDEX IF NOT EXISTS email_outbound_messages_delivery_idx ON email_outbound_messages (tenant_id, status, created_at);
CREATE INDEX IF NOT EXISTS on_call_shifts_lookup_idx ON on_call_shifts (tenant_id, schedule_id, start_at, end_at);

INSERT INTO permission_definitions (resource, action, description, display_name, category)
SELECT p.resource, p.action, p.description, p.display_name, 0
FROM (VALUES
 ('email_intake','read','查看邮件报障会话与分析','查看邮件报障'), ('email_intake','review','修正和确认邮件报障','复核邮件报障'),
 ('email_intake','retry','重试邮件处理','重试邮件报障'), ('email_intake','override','带原因强制开单','强制邮件开单'),
 ('customer_master','read','查看服务客户主数据','查看服务客户'), ('customer_master','write','维护服务客户主数据','管理服务客户'),
 ('support_contract','read','查看支持合同','查看支持合同'), ('support_contract','write','维护支持合同','管理支持合同'),
 ('on_call','read','查看值班排班','查看值班'), ('on_call','write','维护值班排班','管理值班')
) AS p(resource, action, description, display_name)
WHERE NOT EXISTS (SELECT 1 FROM permission_definitions d WHERE d.resource=p.resource AND d.action=p.action);

INSERT INTO menus (name, path, icon, permission_code, sort_order, tenant_id, is_visible, is_enabled)
SELECT '邮件智能报障', '/email-intake', 'MailWarning', 'email_intake:read', 125, id, TRUE, TRUE FROM tenants
ON CONFLICT (tenant_id, path) DO NOTHING;
