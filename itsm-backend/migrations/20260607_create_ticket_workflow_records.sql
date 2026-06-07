-- 创建 ticket_workflow_records 表（工单流转记录）
-- 用于追踪工单的状态变更、分配变更、审批等操作历史

CREATE TABLE IF NOT EXISTS ticket_workflow_records (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL,
    action VARCHAR(50) NOT NULL,
    from_status VARCHAR(50),
    to_status VARCHAR(50),
    operator_id BIGINT NOT NULL,
    from_user_id BIGINT,
    to_user_id BIGINT,
    comment TEXT,
    reason TEXT,
    metadata JSONB DEFAULT '{}',
    tenant_id BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 索引：按工单ID快速查询流转历史
CREATE INDEX IF NOT EXISTS idx_ticket_workflow_records_ticket_id
    ON ticket_workflow_records(ticket_id);

-- 索引：按租户+创建时间查询
CREATE INDEX IF NOT EXISTS idx_ticket_workflow_records_tenant_created
    ON ticket_workflow_records(tenant_id, created_at DESC);

-- 索引：按操作类型查询
CREATE INDEX IF NOT EXISTS idx_ticket_workflow_records_action
    ON ticket_workflow_records(action);

-- 索引：按操作人查询
CREATE INDEX IF NOT EXISTS idx_ticket_workflow_records_operator
    ON ticket_workflow_records(operator_id);
