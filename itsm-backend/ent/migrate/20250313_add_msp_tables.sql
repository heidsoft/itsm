-- MSP 相关表创建

-- 1. 扩展 tenant 表
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS type VARCHAR(50) DEFAULT 'standard';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS parent_tenant_id INT;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS msp_provider_id INT;
ALTER TABLE tenants ADD CONSTRAINT fk_tenant_parent FOREIGN KEY (parent_tenant_id) REFERENCES tenants(id) ON DELETE SET NULL;
ALTER TABLE tenants ADD CONSTRAINT fk_tenant_msp_provider FOREIGN KEY (msp_provider_id) REFERENCES tenants(id) ON DELETE SET NULL;

-- 2. 扩展 user 表
ALTER TABLE users ADD COLUMN IF NOT EXISTS msp_role VARCHAR(50);
ALTER TABLE users ADD COLUMN IF NOT EXISTS assigned_by_msp_id INT;
ALTER TABLE users ADD CONSTRAINT fk_user_assigned_by_msp FOREIGN KEY (assigned_by_msp_id) REFERENCES users(id) ON DELETE SET NULL;

-- 3. 扩展 ticket 表
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS is_managed_by_msp BOOLEAN DEFAULT FALSE;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS msp_provider_id INT;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS managed_by_user_id INT;
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS msp_ticket_id VARCHAR(255);
ALTER TABLE tickets ADD CONSTRAINT fk_ticket_msp_provider FOREIGN KEY (msp_provider_id) REFERENCES tenants(id) ON DELETE SET NULL;
ALTER TABLE tickets ADD CONSTRAINT fk_ticket_managed_by_user FOREIGN KEY (managed_by_user_id) REFERENCES users(id) ON DELETE SET NULL;

-- 4. 创建 msp_allocations 表
CREATE TABLE IF NOT EXISTS msp_allocations (
    id SERIAL PRIMARY KEY,
    msp_user_id INT NOT NULL,
    customer_tenant_id INT NOT NULL,
    role VARCHAR(50) DEFAULT 'primary',
    assigned_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deassigned_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_msp_allocation_msp_user FOREIGN KEY (msp_user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_msp_allocation_customer_tenant FOREIGN KEY (customer_tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT uk_msp_allocation_active UNIQUE (msp_user_id, customer_tenant_id) WHERE deassigned_at IS NULL
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_msp_allocations_user ON msp_allocations(msp_user_id, deassigned_at);
CREATE INDEX IF NOT EXISTS idx_msp_allocations_customer ON msp_allocations(customer_tenant_id, deassigned_at);
CREATE INDEX IF NOT EXISTS idx_tickets_tenant_managed ON tickets(tenant_id, is_managed_by_msp);
CREATE INDEX IF NOT EXISTS idx_tickets_managed_by ON tickets(managed_by_user_id, status);
CREATE INDEX IF NOT EXISTS idx_tenants_msp_provider ON tenants(msp_provider_id, type);
