-- 创建 change_approvals 表
-- 用于存储变更审批记录（service/change_approval_service.go 引用此表名）

CREATE TABLE IF NOT EXISTS change_approvals (
    id BIGSERIAL PRIMARY KEY,
    change_id BIGINT NOT NULL,
    approver_id BIGINT NOT NULL,
    tenant_id BIGINT NOT NULL DEFAULT 1,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    comment TEXT,
    approved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_change_approvals_change_id ON change_approvals(change_id);
CREATE INDEX IF NOT EXISTS idx_change_approvals_approver_id ON change_approvals(approver_id);
CREATE INDEX IF NOT EXISTS idx_change_approvals_tenant_id ON change_approvals(tenant_id);
CREATE INDEX IF NOT EXISTS idx_change_approvals_status ON change_approvals(status);
