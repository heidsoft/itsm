-- Manual migration: add hierarchy fields to cloud_services and CI cloud resource reference.

ALTER TABLE cloud_services
  ADD COLUMN IF NOT EXISTS parent_id BIGINT,
  ADD COLUMN IF NOT EXISTS category VARCHAR,
  ADD COLUMN IF NOT EXISTS api_version VARCHAR,
  ADD COLUMN IF NOT EXISTS is_system BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS cloud_services_parent_id
  ON cloud_services (parent_id);

CREATE INDEX IF NOT EXISTS cloud_services_category
  ON cloud_services (category);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'cloud_services_parent_id_fkey'
  ) THEN
    ALTER TABLE cloud_services
      ADD CONSTRAINT cloud_services_parent_id_fkey
      FOREIGN KEY (parent_id) REFERENCES cloud_services(id) ON DELETE SET NULL;
  END IF;
END $$;

ALTER TABLE configuration_items
  ADD COLUMN IF NOT EXISTS cloud_resource_ref_id BIGINT;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'configuration_items_cloud_resource_ref_id_fkey'
  ) THEN
    ALTER TABLE configuration_items
      ADD CONSTRAINT configuration_items_cloud_resource_ref_id_fkey
      FOREIGN KEY (cloud_resource_ref_id) REFERENCES cloud_resources(id) ON DELETE SET NULL;
  END IF;
END $$;
