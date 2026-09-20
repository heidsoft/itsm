-- 清理遗留工作流表，ticket_categories 迁移到 BPMN workflow_definition_key
-- 遗留表 workflows/workflow_instances/workflow_tasks/workflow_versions 从未被 BPMN 引擎写入
-- 实际工作流由 process_definitions/process_instances/process_tasks 承载

BEGIN;

-- 1. 删除指向遗留 workflows 表的外键约束
ALTER TABLE ticket_categories DROP CONSTRAINT IF EXISTS ticket_categories_workflows_workflow;

-- 2. ticket_categories: 删除 workflow_id 列，新增 workflow_definition_key
ALTER TABLE ticket_categories DROP COLUMN IF EXISTS workflow_id;
ALTER TABLE ticket_categories ADD COLUMN IF NOT EXISTS workflow_definition_key VARCHAR(128) DEFAULT '';

-- 3. 删除遗留索引
DROP INDEX IF EXISTS wf_instance_tenant_idx;
DROP INDEX IF EXISTS wf_instance_entity_idx;
DROP INDEX IF EXISTS idx_workflow_tasks_tenant;
DROP INDEX IF EXISTS idx_workflow_versions_tenant;
DROP INDEX IF EXISTS idx_workflows_tenant;

-- 4. 删除遗留表（按依赖顺序：子表先删）
DROP TABLE IF EXISTS workflow_tasks;
DROP TABLE IF EXISTS workflow_versions;
DROP TABLE IF EXISTS workflow_instances;
DROP TABLE IF EXISTS workflows;

COMMIT;
