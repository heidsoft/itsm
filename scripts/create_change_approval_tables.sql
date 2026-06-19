-- Create change approval tables (not managed by Ent ORM)
-- These tables store approval records and approval chains for change management
-- Run this script after initial migration

BEGIN;

-- Change approval records table
CREATE TABLE IF NOT EXISTS change_approvals (
    id SERIAL PRIMARY KEY,
    change_id INT NOT NULL REFERENCES changes(id) ON DELETE CASCADE,
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
    level INT NOT NULL,
    approver_id INT NOT NULL,
    role TEXT DEFAULT 'approver',
    status TEXT DEFAULT 'pending',
    is_required BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_change_approvals_change_id ON change_approvals(change_id);
CREATE INDEX IF NOT EXISTS idx_change_approvals_approver_id ON change_approvals(approver_id);
CREATE INDEX IF NOT EXISTS idx_change_approval_chains_change_id ON change_approval_chains(change_id);
CREATE INDEX IF NOT EXISTS idx_change_approval_chains_approver_id ON change_approval_chains(approver_id);

COMMIT;
