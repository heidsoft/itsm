/**
 * Capabilities are enabled only when a tenant-aware, permission-protected
 * backend contract exists. Keep unsupported roadmap work invisible instead of
 * presenting controls that can only fail or mutate local mock state.
 */
export const PRODUCT_CAPABILITIES = {
  // 后端已注册 POST /api/v1/ai/rag/search（handlers/ai KnowledgeSearch），
  // ai-api.ts 已对齐契约（{results, degraded}），打开能力开关。
  aiKnowledgeSearch: true,
  advancedBatchOperations: false,
  changeClassification: false,
  collaborationAdvanced: false,
  knowledgeAdvancedActions: false,
  notificationTemplateManagement: false,
  notificationChannelManagement: false,
  priorityMatrix: false,
  advancedProblemActions: true,
  advancedReporting: false,
  genericTemplateMarketplace: false,
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
  { capability: 'advancedBatchOperations', file: 'batch-operations-api.ts', reason: 'Advanced batch orchestration is roadmap-only' },
  { capability: 'changeClassification', file: 'change-classification-api.ts', reason: 'Change classification/rule APIs are not registered' },
  { capability: 'changeClassification', file: 'change-api.ts', path: /\/changes\/templates\//, reason: 'Template instantiation route is not registered' },
  { capability: 'collaborationAdvanced', file: 'collaboration-api.ts', reason: 'Advanced comments, mentions and presence routes are not registered' },
  { capability: 'knowledgeAdvancedActions', file: 'knowledge-base-api.ts', reason: 'Advanced knowledge lifecycle actions are not registered' },
  { capability: 'notificationTemplateManagement', file: 'notification-preference-api.ts', reason: 'Preference reset/template application routes are not registered' },
  { capability: 'priorityMatrix', file: 'priority-matrix-api.ts', reason: 'Priority matrix backend is not registered' },
  { capability: 'advancedReporting', file: 'reports-api.ts', reason: 'Only read-only report summaries are supported by the backend' },
  { capability: 'genericTemplateMarketplace', file: 'template-api.ts', reason: 'Generic template marketplace is not registered; ticket templates use a separate supported API' },
  { capability: 'advancedTicketRelations', file: 'ticket-relations-api.ts', reason: 'Advanced relation analytics and batch routes are not registered' },
  { capability: 'rootCauseWorkflowActions', file: 'ticket-root-cause-api.ts', reason: 'Root-cause confirm/resolve routes are not registered' },
  // NOTE: workflowAnalytics is now enabled and the 4 legacy workflow-api.ts analytics/template
  // entrypoints (workflow-templates / workflows/:id/stats / node-stats / bottlenecks) were
  // neutralized to stop emitting unregistered paths (see workflow-api.ts). Their exemptions were
  // therefore removed from this list; canonical analytics use bpmn-dashboard-api.ts /
  // bpmn-monitoring-api.ts. The problem-relationships write endpoint is now registered
  // (router.go POST /api/v1/problem-relationships), so its exemption was removed as well.
] as const;
