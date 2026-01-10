-- Manual migration: add CMDB CI fields aligned with ent schema updates.
-- Safe to run multiple times due to IF NOT EXISTS.

ALTER TABLE configuration_items
  ADD COLUMN IF NOT EXISTS environment text NOT NULL DEFAULT 'production',
  ADD COLUMN IF NOT EXISTS criticality text NOT NULL DEFAULT 'medium',
  ADD COLUMN IF NOT EXISTS asset_tag text,
  ADD COLUMN IF NOT EXISTS model text,
  ADD COLUMN IF NOT EXISTS vendor text,
  ADD COLUMN IF NOT EXISTS assigned_to text,
  ADD COLUMN IF NOT EXISTS owned_by text,
  ADD COLUMN IF NOT EXISTS discovery_source text;
