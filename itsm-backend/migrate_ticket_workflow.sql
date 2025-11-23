-- 工单类型表
CREATE TABLE IF NOT EXISTS ticket_types (
    id SERIAL PRIMARY KEY,
    code VARCHAR(100) NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    icon VARCHAR(100),
    color VARCHAR(50),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    
    -- JSON字段
    custom_fields JSONB DEFAULT '[]',
    approval_enabled BOOLEAN DEFAULT FALSE,
    approval_workflow_id VARCHAR(100),
    approval_chain JSONB DEFAULT '[]',
    
    sla_enabled BOOLEAN DEFAULT FALSE,
    default_sla_id INTEGER,
    
    auto_assign_enabled BOOLEAN DEFAULT FALSE,
    assignment_rules JSONB DEFAULT '[]',
    
    notification_config JSONB,
    permission_config JSONB,
    
    -- 元数据
    created_by INTEGER NOT NULL,
    updated_by INTEGER,
    tenant_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    usage_count INTEGER DEFAULT 0,
    
    UNIQUE(code, tenant_id)
);

-- 索引
CREATE INDEX idx_ticket_types_tenant ON ticket_types(tenant_id);
CREATE INDEX idx_ticket_types_status ON ticket_types(status);
CREATE INDEX idx_ticket_types_code ON ticket_types(code);

-- 工单流转记录表
CREATE TABLE IF NOT EXISTS ticket_workflow_records (
    id SERIAL PRIMARY KEY,
    ticket_id INTEGER NOT NULL,
    action VARCHAR(50) NOT NULL,
    from_status VARCHAR(50),
    to_status VARCHAR(50),
    operator_id INTEGER NOT NULL,
    from_user_id INTEGER,
    to_user_id INTEGER,
    comment TEXT,
    reason TEXT,
    metadata JSONB,
    tenant_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_workflow_records_ticket ON ticket_workflow_records(ticket_id);
CREATE INDEX idx_workflow_records_tenant ON ticket_workflow_records(tenant_id);
CREATE INDEX idx_workflow_records_action ON ticket_workflow_records(action);
CREATE INDEX idx_workflow_records_created ON ticket_workflow_records(created_at DESC);

-- 工单审批记录表
CREATE TABLE IF NOT EXISTS ticket_approvals (
    id SERIAL PRIMARY KEY,
    ticket_id INTEGER NOT NULL,
    level INTEGER NOT NULL,
    level_name VARCHAR(200),
    approver_id INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    action VARCHAR(50),
    comment TEXT,
    processed_at TIMESTAMP,
    delegate_to_user_id INTEGER,
    tenant_id INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_approvals_ticket ON ticket_approvals(ticket_id);
CREATE INDEX idx_approvals_approver ON ticket_approvals(approver_id);
CREATE INDEX idx_approvals_status ON ticket_approvals(status);
CREATE INDEX idx_approvals_tenant ON ticket_approvals(tenant_id);

-- 工单抄送表
CREATE TABLE IF NOT EXISTS ticket_cc (
    id SERIAL PRIMARY KEY,
    ticket_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    added_by INTEGER NOT NULL,
    tenant_id INTEGER NOT NULL,
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    
    UNIQUE(ticket_id, user_id, tenant_id)
);

-- 索引
CREATE INDEX idx_ticket_cc_ticket ON ticket_cc(ticket_id);
CREATE INDEX idx_ticket_cc_user ON ticket_cc(user_id);
CREATE INDEX idx_ticket_cc_tenant ON ticket_cc(tenant_id);

-- 添加工单表的字段（如果不存在）
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS ticket_type_id INTEGER;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS resolution TEXT;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS resolution_category VARCHAR(100);
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMP;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS closed_at TIMESTAMP;

-- 添加外键约束
ALTER TABLE tickets ADD CONSTRAINT fk_ticket_type 
    FOREIGN KEY (ticket_type_id) REFERENCES ticket_types(id) ON DELETE SET NULL;

-- 注释
COMMENT ON TABLE ticket_types IS '工单类型定义表';
COMMENT ON TABLE ticket_workflow_records IS '工单流转记录表';
COMMENT ON TABLE ticket_approvals IS '工单审批记录表';
COMMENT ON TABLE ticket_cc IS '工单抄送表';

COMMENT ON COLUMN ticket_types.code IS '类型编码，租户内唯一';
COMMENT ON COLUMN ticket_types.custom_fields IS '自定义字段定义(JSON)';
COMMENT ON COLUMN ticket_types.approval_chain IS '审批链定义(JSON)';
COMMENT ON COLUMN ticket_types.assignment_rules IS '分配规则(JSON)';
COMMENT ON COLUMN ticket_types.usage_count IS '使用次数统计';

COMMENT ON COLUMN ticket_workflow_records.action IS '流转操作类型: accept, reject, withdraw, forward, cc, approve等';
COMMENT ON COLUMN ticket_workflow_records.metadata IS '附加元数据(JSON)';

COMMENT ON COLUMN ticket_approvals.level IS '审批级别';
COMMENT ON COLUMN ticket_approvals.status IS '审批状态: pending, approved, rejected, cancelled';

