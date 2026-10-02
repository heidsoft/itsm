/**
 * Capabilities are enabled only when a tenant-aware, permission-protected
 * backend contract exists. Keep unsupported roadmap work invisible instead of
 * presenting controls that can only fail or mutate local mock state.
 */
export const PRODUCT_CAPABILITIES = {
  // 后端已注册 POST /api/v1/ai/rag/search（handlers/ai KnowledgeSearch），
  // ai-api.ts 已对齐契约（{results, degraded}），打开能力开关。
  aiKnowledgeSearch: true,
  // E3-2（2026-10-02）：advancedBatchOperations / collaborationAdvanced /
  // priorityMatrix / advancedReporting / genericTemplateMarketplace 五个开关随其
  // 前端实现一起删除。实测从 src/app/**/(page|layout).tsx 建 import 闭包，
  // 这六条链（api client + hook + 组件目录）没有任何路由可达，对应的
  // DISABLED_API_CONTRACTS 表项因此成为无用豁免；能力重新开放的前提是后端先注册
  // 路由并新建可达入口，而不是把这些不可达实现恢复回来。
  knowledgeAdvancedActions: false,
  // change-api.ts 仍是在用的客户端，只有 /changes/templates/:id/instantiate 这一条
  // 路径后端从未注册，因此该开关与它的豁免一起保留。
  changeClassification: false,
  // E3-1（2026-10-02）：router/msp_routes.go 已注册
  // GET /api/v1/msp/allocations/history（RequireMSPPermission msp_allocation:read），
  // handlers/msp 的历史查询按认证上下文的 MSP 租户收敛，返回标准 items 信封。
  // 响应只有 msp_allocations 真实存在的列（assigned_at/deassigned_at/role/员工/客户）；
  // 解除原因与操作人没有落库字段，因此契约里也不出现这两个键。
  mspAllocationHistory: true,
  notificationTemplateManagement: false,
  notificationChannelManagement: false,
  advancedProblemActions: true,
  advancedTicketRelations: false,
  rootCauseWorkflowActions: false,
  // P1-6：后端已完成 BPMN 监控/仪表盘/瓶颈分析服务实现：
  //   controller/bpmn_monitoring_controller.go 注册 /api/v1/bpmn/monitoring/*
  //   controller/bpmn_dashboard_controller.go 注册 /api/v1/bpmn/dashboard/* (含 /bottlenecks)
  //   service/bpmn_monitoring_service.go + service/bpmn_metrics_service.go 有单元测试
  // 前端使用 bpmn-monitoring-api.ts / bpmn-dashboard-api.ts 而非 workflow-api.ts。
  // capability 已打开；workflow-api.ts 中 4 个遗留分析/模板入口已改为不再发起未注册请求
  // （getTemplates/getNodeStats/getBottleneckAnalysis 返回空，getWorkflowStats 返回零快照），
  // 因此原有 4 条 workflowAnalytics 豁免表项已一并从 DISABLED_API_CONTRACTS 移除。
  workflowAnalytics: true,
} as const;

export type ProductCapability = keyof typeof PRODUCT_CAPABILITIES;

export function hasProductCapability(capability: ProductCapability): boolean {
  return PRODUCT_CAPABILITIES[capability];
}

export interface DisabledApiContract {
  capability: ProductCapability;
  file: string;
  path?: RegExp;
  reason: string;
}

/** Explicit audit allow-list for roadmap clients that are disabled in UI. */
export const DISABLED_API_CONTRACTS: readonly DisabledApiContract[] = [
  { capability: 'changeClassification', file: 'change-api.ts', path: /\/changes\/templates\//, reason: 'Template instantiation route is not registered' },
  { capability: 'knowledgeAdvancedActions', file: 'knowledge-base-api.ts', reason: 'Advanced knowledge lifecycle actions are not registered' },
  { capability: 'notificationTemplateManagement', file: 'notification-preference-api.ts', reason: 'Preference reset/template application routes are not registered' },
  { capability: 'advancedTicketRelations', file: 'ticket-relations-api.ts', reason: 'Advanced relation analytics and batch routes are not registered' },
  { capability: 'rootCauseWorkflowActions', file: 'ticket-root-cause-api.ts', reason: 'Root-cause confirm/resolve routes are not registered' },
  // NOTE: workflowAnalytics is now enabled and the 4 legacy workflow-api.ts analytics/template
  // entrypoints (workflow-templates / workflows/:id/stats / node-stats / bottlenecks) were
  // neutralized to stop emitting unregistered paths (see workflow-api.ts). Their exemptions were
  // therefore removed from this list; canonical analytics use bpmn-dashboard-api.ts /
  // bpmn-monitoring-api.ts. The problem-relationships write endpoint is now registered
  // (router.go POST /api/v1/problem-relationships), so its exemption was removed as well.
  // NOTE(E3-2): batch-operations-api / change-classification-api / collaboration-api /
  // priority-matrix-api / reports-api / template-api 连同其 hook 与组件目录一起删除
  // （实测从路由入口建 import 闭包不可达），因此这五条豁免不再存在。此清单只用于
  // 「文件仍在、路径未注册」的情形；实现已不存在的能力不应继续留豁免。
] as const;
