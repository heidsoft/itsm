-- =====================================================
-- 审批工作流配置同步脚本
-- 执行时间: 2026-05-01
-- 目的: 同步 BPMN 中的审批节点与 approval_workflows 表
-- =====================================================
-- 1. 查看当前审批工作流配置
SELECT id,
    name,
    nodes
FROM approval_workflows
WHERE tenant_id = 1;
-- 2. 更新 P0/P1 事件审批配置（与 BPMN 对齐）
UPDATE approval_workflows
SET nodes = '[
  {
    "level": 1,
    "name": "自动分配",
    "type": "service_task",
    "action": "assign_incident",
    "timeout": 10,
    "approver_type": "system"
  },
  {
    "level": 2,
    "name": "主管审批",
    "type": "approval",
    "timeout": 60,
    "approver_type": "manager",
    "bpmn_node_id": "Activity_ManagerApproval"
  },
  {
    "level": 3,
    "name": "初步诊断",
    "type": "user_task",
    "action": "update_incident",
    "timeout": 30,
    "approver_type": "engineer"
  },
  {
    "level": 4,
    "name": "问题定位网关",
    "type": "exclusive_gateway",
    "condition_variable": "problem_identified"
  },
  {
    "level": 5,
    "name": "立即解决/自动升级",
    "type": "decision",
    "timeout": 120
  },
  {
    "level": 6,
    "name": "专家团队处理",
    "type": "user_task",
    "timeout": 240,
    "approver_type": "expert"
  },
  {
    "level": 7,
    "name": "通知相关方",
    "type": "service_task",
    "action": "notify",
    "timeout": 10
  },
  {
    "level": 8,
    "name": "事件关闭",
    "type": "user_task",
    "action": "close_incident",
    "timeout": 30
  }
]'
WHERE id = 1
    AND tenant_id = 1;
-- 3. 验证更新后的配置
SELECT id,
    name,
    nodes
FROM approval_workflows
WHERE tenant_id = 1;
-- =====================================================
-- 注意: PostgreSQL MCP 运行在只读模式，需要手动执行
-- 或通过 hasura/其他数据库客户端执行
-- =====================================================