-- Manual migration: hybrid-cloud CMDB tables and columns.

CREATE TABLE IF NOT EXISTS cloud_services (
  id BIGSERIAL PRIMARY KEY,
  provider VARCHAR NOT NULL,
  service_code VARCHAR NOT NULL,
  service_name VARCHAR NOT NULL,
  resource_type_code VARCHAR NOT NULL,
  resource_type_name VARCHAR NOT NULL,
  attribute_schema JSONB,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  tenant_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS cloud_services_tenant_provider_code_type
  ON cloud_services (tenant_id, provider, service_code, resource_type_code);

CREATE INDEX IF NOT EXISTS cloud_services_tenant_id
  ON cloud_services (tenant_id);

CREATE TABLE IF NOT EXISTS cloud_accounts (
  id BIGSERIAL PRIMARY KEY,
  provider VARCHAR NOT NULL,
  account_id VARCHAR NOT NULL,
  account_name VARCHAR NOT NULL,
  credential_ref VARCHAR,
  region_whitelist JSONB,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  tenant_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS cloud_accounts_tenant_provider_account
  ON cloud_accounts (tenant_id, provider, account_id);

CREATE INDEX IF NOT EXISTS cloud_accounts_tenant_id
  ON cloud_accounts (tenant_id);

CREATE TABLE IF NOT EXISTS cloud_resources (
  id BIGSERIAL PRIMARY KEY,
  cloud_account_id BIGINT NOT NULL,
  service_id BIGINT NOT NULL,
  resource_id VARCHAR NOT NULL,
  resource_name VARCHAR,
  region VARCHAR,
  zone VARCHAR,
  status VARCHAR,
  tags JSONB,
  metadata JSONB,
  first_seen_at TIMESTAMPTZ,
  last_seen_at TIMESTAMPTZ,
  lifecycle_state VARCHAR,
  tenant_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE cloud_resources
  ADD CONSTRAINT cloud_resources_cloud_account_id_fkey
  FOREIGN KEY (cloud_account_id) REFERENCES cloud_accounts(id) ON DELETE CASCADE;

ALTER TABLE cloud_resources
  ADD CONSTRAINT cloud_resources_service_id_fkey
  FOREIGN KEY (service_id) REFERENCES cloud_services(id) ON DELETE CASCADE;

CREATE UNIQUE INDEX IF NOT EXISTS cloud_resources_account_resource_id
  ON cloud_resources (cloud_account_id, resource_id);

CREATE INDEX IF NOT EXISTS cloud_resources_tenant_id
  ON cloud_resources (tenant_id);

CREATE INDEX IF NOT EXISTS cloud_resources_service_id
  ON cloud_resources (service_id);

CREATE INDEX IF NOT EXISTS cloud_resources_region
  ON cloud_resources (region);

CREATE TABLE IF NOT EXISTS relationship_types (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR NOT NULL,
  directional BOOLEAN NOT NULL DEFAULT TRUE,
  reverse_name VARCHAR,
  description VARCHAR,
  tenant_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS relationship_types_tenant_name
  ON relationship_types (tenant_id, name);

CREATE TABLE IF NOT EXISTS discovery_sources (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR NOT NULL,
  source_type VARCHAR NOT NULL,
  provider VARCHAR,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  description VARCHAR,
  tenant_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS discovery_sources_tenant_name
  ON discovery_sources (tenant_id, name);

CREATE TABLE IF NOT EXISTS discovery_jobs (
  id BIGSERIAL PRIMARY KEY,
  source_id BIGINT NOT NULL,
  status VARCHAR NOT NULL DEFAULT 'pending',
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  summary JSONB,
  tenant_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE discovery_jobs
  ADD CONSTRAINT discovery_jobs_source_id_fkey
  FOREIGN KEY (source_id) REFERENCES discovery_sources(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS discovery_jobs_tenant_id
  ON discovery_jobs (tenant_id);

CREATE INDEX IF NOT EXISTS discovery_jobs_source_id
  ON discovery_jobs (source_id);

CREATE INDEX IF NOT EXISTS discovery_jobs_status
  ON discovery_jobs (status);

CREATE TABLE IF NOT EXISTS discovery_results (
  id BIGSERIAL PRIMARY KEY,
  job_id BIGINT NOT NULL,
  ci_id BIGINT,
  action VARCHAR NOT NULL,
  resource_type VARCHAR,
  resource_id VARCHAR,
  diff JSONB,
  status VARCHAR NOT NULL DEFAULT 'pending',
  tenant_id BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE discovery_results
  ADD CONSTRAINT discovery_results_job_id_fkey
  FOREIGN KEY (job_id) REFERENCES discovery_jobs(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS discovery_results_tenant_id
  ON discovery_results (tenant_id);

CREATE INDEX IF NOT EXISTS discovery_results_job_id
  ON discovery_results (job_id);

CREATE INDEX IF NOT EXISTS discovery_results_status
  ON discovery_results (status);

ALTER TABLE configuration_items
  ADD COLUMN IF NOT EXISTS source VARCHAR,
  ADD COLUMN IF NOT EXISTS cloud_provider VARCHAR,
  ADD COLUMN IF NOT EXISTS cloud_account_id VARCHAR,
  ADD COLUMN IF NOT EXISTS cloud_region VARCHAR,
  ADD COLUMN IF NOT EXISTS cloud_zone VARCHAR,
  ADD COLUMN IF NOT EXISTS cloud_resource_id VARCHAR,
  ADD COLUMN IF NOT EXISTS cloud_resource_type VARCHAR,
  ADD COLUMN IF NOT EXISTS cloud_metadata JSONB,
  ADD COLUMN IF NOT EXISTS cloud_tags JSONB,
  ADD COLUMN IF NOT EXISTS cloud_metrics JSONB,
  ADD COLUMN IF NOT EXISTS cloud_sync_time TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS cloud_sync_status VARCHAR DEFAULT 'unknown';
