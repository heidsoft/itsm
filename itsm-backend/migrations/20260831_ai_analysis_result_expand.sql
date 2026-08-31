-- P0-C: create ai_analysis_result table for AI analysis persistence (triage/summary/rca/...).
-- Idempotent / safe to rerun. Mirrors the RLS policy pattern from 20260828_create_alerts.sql
-- so the table is queryable under RLS_MODE=enforce (production default).
-- Column types follow Ent field mapping for field.Int (INTEGER) / field.Float (DOUBLE PRECISION)
-- / field.String (VARCHAR) / field.Time (TIMESTAMPTZ), matching ent auto migration output.

BEGIN;

CREATE TABLE IF NOT EXISTS ai_analysis_result (
    id BIGSERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    user_id INTEGER NULL,
    analysis_type VARCHAR(40) NOT NULL,
    ticket_id INTEGER NULL,
    incident_id INTEGER NULL,
    ticket_number VARCHAR(100) NULL,
    ticket_title VARCHAR(500) NULL,
    request_prompt TEXT NOT NULL,
    result_json TEXT NOT NULL,
    model VARCHAR(100) NULL,
    latency_ms INTEGER NULL,
    total_tokens INTEGER NULL,
    cost_usd DOUBLE PRECISION NULL,
    confidence_score DOUBLE PRECISION NULL,
    degraded BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS ai_analysis_result_tenant_type_idx
    ON ai_analysis_result (tenant_id, analysis_type);
CREATE INDEX IF NOT EXISTS ai_analysis_result_tenant_ticket_idx
    ON ai_analysis_result (tenant_id, ticket_id);
CREATE INDEX IF NOT EXISTS ai_analysis_result_tenant_incident_idx
    ON ai_analysis_result (tenant_id, incident_id);

ALTER TABLE ai_analysis_result ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_ai_analysis_result ON ai_analysis_result;
CREATE POLICY tenant_isolation_ai_analysis_result ON ai_analysis_result
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::BIGINT)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::BIGINT);

COMMIT;
