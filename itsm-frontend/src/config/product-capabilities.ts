/**
 * Capabilities are enabled only when a tenant-aware, permission-protected
 * backend contract exists. Keep unsupported roadmap work invisible instead of
 * presenting controls that can only fail or mutate local mock state.
 */
export const PRODUCT_CAPABILITIES = {
  knowledgeAdvancedActions: false,
  // change-api.ts 仍是在用的客户端，只有 /changes/templates/:id/instantiate 这条
  // 路径后端从未注册，所以开关保留。
  changeClassification: false,
  // 已开放：router/msp_routes.go 注册了 GET /api/v1/msp/allocations/history
  // （msp_allocation:read），查询按认证上下文的 MSP 租户收敛。
  // 解除原因与操作人没有落库字段，因此契约里也不出现这两个键。
  mspAllocationHistory: true,
  notificationTemplateManagement: false,
  notificationChannelManagement: false,
  advancedTicketRelations: false,
  rootCauseWorkflowActions: false,
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
  // 本清单只用于「文件仍在、路径未注册」的情形；实现已删除的能力不留在这里。
  // workflow-api.ts 的 analytics/template 入口已不再发出未注册路径，规范分析走
  // bpmn-dashboard-api.ts 与 bpmn-monitoring-api.ts。
] as const;
