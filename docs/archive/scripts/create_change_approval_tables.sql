-- Create change approval tables (not managed by Ent ORM)
-- These tables store approval records and approval chains for change management
-- Run this script after initial migration
--
-- 2026-07-18: 补齐 tenant_id 字段（阶段 A #6 P2 治本），保证跨租户隔离在 SQL 层强制生效。

BEGIN;

-- Change approval records table
CREATE TABLE IF NOT EXISTS change_approvals (
    id SERIAL PRIMARY KEY,
    change_id INT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    tenant_id INT NOT NULL DEFAULT 0,
    approver_id INT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    comment TEXT,
    approved_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Change approval chain table
CREATE TABLE IF NOT EXISTS change_approval_chains (
    id SERIAL PRIMARY KEY,
    change_id INT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    tenant_id INT NOT NULL DEFAULT 0,
    level INT NOT NULL,
    approver_id INT NOT NULL,
    role TEXT DEFAULT 'approver',
    status TEXT DEFAULT 'pending',
    is_required BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Change risk assessment table
CREATE TABLE IF NOT EXISTS change_risk_assessments (
    id SERIAL PRIMARY KEY,
    change_id INT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
    tenant_id INT NOT NULL DEFAULT 0,
    risk_level TEXT NOT NULL DEFAULT 'medium',
    risk_description TEXT,
    impact_analysis TEXT,
    mitigation_measures TEXT,
    contingency_plan TEXT,
    risk_owner TEXT,
    risk_review_date TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- 幂等补列 + 回填（兼容旧库升级路径）
ALTER TABLE change_approvals       ADD COLUMN IF NOT EXISTS tenant_id INT NOT NULL DEFAULT 0;
ALTER TABLE change_approval_chains ADD COLUMN IF NOT EXISTS tenant_id INT NOT NULL DEFAULT 0;
UPDATE change_approvals ca
    SET tenant_id = c.tenant_id
    FROM changes c
    WHERE ca.change_id = c.id AND ca.tenant_id = 0;
UPDATE change_approval_chains cac
    SET tenant_id = c.tenant_id
    FROM changes c
    WHERE cac.change_id = c.id AND cac.tenant_id = 0;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_change_approvals_change_id ON change_approvals(change_id);
CREATE INDEX IF NOT EXISTS idx_change_approvals_tenant_id ON change_approvals(tenant_id);
CREATE INDEX IF NOT EXISTS idx_change_approvals_approver_id ON change_approvals(approver_id);
CREATE INDEX IF NOT EXISTS idx_change_approval_chains_change_id ON change_approval_chains(change_id);
CREATE INDEX IF NOT EXISTS idx_change_approval_chains_tenant_id ON change_approval_chains(tenant_id);
CREATE INDEX IF NOT EXISTS idx_change_approval_chains_approver_id ON change_approval_chains(approver_id);
CREATE INDEX IF NOT EXISTS idx_change_risk_assessments_change_id ON change_risk_assessments(change_id);
CREATE INDEX IF NOT EXISTS idx_change_risk_assessments_tenant_id ON change_risk_assessments(tenant_id);

COMMIT;
