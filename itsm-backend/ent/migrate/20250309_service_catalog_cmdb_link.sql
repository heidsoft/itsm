-- Manual migration: link service catalogs and service requests to CMDB.

ALTER TABLE service_catalogs
  ADD COLUMN IF NOT EXISTS ci_type_id BIGINT,
  ADD COLUMN IF NOT EXISTS cloud_service_id BIGINT;

CREATE INDEX IF NOT EXISTS service_catalogs_ci_type_id
  ON service_catalogs (ci_type_id);

CREATE INDEX IF NOT EXISTS service_catalogs_cloud_service_id
  ON service_catalogs (cloud_service_id);

ALTER TABLE service_requests
  ADD COLUMN IF NOT EXISTS ci_id BIGINT;

CREATE INDEX IF NOT EXISTS service_requests_ci_id
  ON service_requests (ci_id);
