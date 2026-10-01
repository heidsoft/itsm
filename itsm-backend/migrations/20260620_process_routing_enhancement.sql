-- ITSM Process Routing Enhancement Migration
-- Date: 2026-06-20
-- Description: Add multi-dimensional routing fields to process_bindings table

-- Step 1: Add new columns to process_bindings table
ALTER TABLE process_bindings 
ADD COLUMN IF NOT EXISTS department_id INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS team_id INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS scenario VARCHAR(100) DEFAULT '',
ADD COLUMN IF NOT EXISTS category VARCHAR(50) DEFAULT '',
ADD COLUMN IF NOT EXISTS conditions JSONB DEFAULT '{}',
ADD COLUMN IF NOT EXISTS approval_chain_id VARCHAR(100) DEFAULT '',
ADD COLUMN IF NOT EXISTS sla_policy_id VARCHAR(100) DEFAULT '',
ADD COLUMN IF NOT EXISTS overrides JSONB DEFAULT '{}';

-- Step 2: Create composite index for efficient routing queries
CREATE INDEX IF NOT EXISTS idx_process_binding_routing 
ON process_bindings(tenant_id, business_type, is_active, department_id, team_id, scenario);

-- Step 3: Create index for department-specific queries
CREATE INDEX IF NOT EXISTS idx_process_binding_department 
ON process_bindings(tenant_id, department_id, is_active);

-- Step 4: Create index for scenario-based queries
CREATE INDEX IF NOT EXISTS idx_process_binding_scenario 
ON process_bindings(tenant_id, scenario, is_active);

-- Step 5: Create domain configuration table for inheritance support
CREATE TABLE IF NOT EXISTS domain_configs (
    id BIGSERIAL PRIMARY KEY,
    config_key VARCHAR(255) NOT NULL,
    config_type VARCHAR(255) NOT NULL,
    config_value JSONB NOT NULL DEFAULT '{}',
    inherit_mode VARCHAR(50) NOT NULL DEFAULT 'inherit',
    tenant_id INTEGER NOT NULL DEFAULT 0,
    department_id INTEGER NOT NULL DEFAULT 0,
    team_id INTEGER NOT NULL DEFAULT 0,
    parent_config_id INTEGER,
    version INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT true,
    description VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT domain_configs_tenant_id_check CHECK (tenant_id >= 0),
    CONSTRAINT domain_configs_scope_unique UNIQUE (tenant_id, config_type, config_key, department_id, team_id)
);

CREATE INDEX IF NOT EXISTS idx_domain_config_lookup
ON domain_configs(tenant_id, config_type, config_key);

CREATE INDEX IF NOT EXISTS idx_domain_config_scope
ON domain_configs(tenant_id, department_id, team_id);

-- Step 6: Backfill existing data (set defaults for new columns)
UPDATE process_bindings 
SET 
    department_id = COALESCE(department_id, 0),
    team_id = COALESCE(team_id, 0),
    scenario = COALESCE(scenario, ''),
    category = COALESCE(category, ''),
    conditions = COALESCE(conditions, '{}'),
    approval_chain_id = COALESCE(approval_chain_id, ''),
    sla_policy_id = COALESCE(sla_policy_id, ''),
    overrides = COALESCE(overrides, '{}')
WHERE department_id IS NULL 
   OR team_id IS NULL 
   OR scenario IS NULL 
   OR category IS NULL;

-- Step 7（原“sample routing rules”INSERT 批次）已于 2026-10-01 移除：这批演示绑定
-- 在全新安装上必然中断初始化，而且从来没有按设计生效过。
--
-- 1) 其中四个 process_definition_key 在仓库里没有任何可部署载体：
--    incident_general_flow / change_emergency_flow / release_test_flow /
--    expense_approval_flow 既不在 service/bpmn/*.bpmn（BPMNTemplateService
--    LoadAndDeployTemplates 的唯一来源），也不在 pkg/seeder 的 workflow_templates
--    清单里；service/incident_service.go:2161 就注明“incident_general_flow 不存在”。
--    pkg/seeder/initialization_adapter.go:571 verifyWorkflowTemplates 要求每条 active
--    绑定都能找到 active process_definition，于是 workflow-core 组件校验失败、整组
--    事务回滚，全新私有部署起不来（实测 itsm-init exit 1）。
--    存量安装看不到这个问题：本迁移日期早于 adoptionCutoff（2026-09-08），在已有
--    安装上只被记账、从不执行。
-- 2) 迁移在 seed 之前执行（实测 itsm-init：绑定写入 13:27:01，identity-rbac 组件
--    13:27:02 才开始），所以 (SELECT id FROM departments WHERE code='OPS') 恒为
--    NULL，落库的 department_id 全是 NULL，原本想要的按部门路由一条都没成立。
-- 3) 全部语句硬编码 tenant_id = 1，与 MSP/多租户基线冲突。
--
-- 归属：流程绑定由 Go 清单负责——pkg/seeder/seeder.go:678 内置 ProcessBindings
-- （6 条，key 全部对应 service/bpmn/*.bpmn）经 seedProcessBindings 按基线租户幂等
-- 写入；部门级绑定属于 service/bpmn_process_binding_service.go
-- getDepartmentDefaultBindings 的运行期能力。守卫见
-- migration/fresh_install_reference_test.go。

-- Verification: Count routing rules
DO $$
DECLARE
    total_count INTEGER;
    dept_count INTEGER;
    global_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO total_count FROM process_bindings WHERE is_active = true;
    SELECT COUNT(*) INTO dept_count FROM process_bindings WHERE department_id > 0 AND is_active = true;
    SELECT COUNT(*) INTO global_count FROM process_bindings WHERE department_id = 0 AND is_active = true;
    
    RAISE NOTICE 'Migration completed:';
    RAISE NOTICE '  Total active bindings: %', total_count;
    RAISE NOTICE '  Department-specific bindings: %', dept_count;
    RAISE NOTICE '  Global bindings: %', global_count;
END $$;
