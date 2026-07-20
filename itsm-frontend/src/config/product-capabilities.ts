/**
 * Capabilities are enabled only when a tenant-aware, permission-protected
 * backend contract exists. Keep unsupported roadmap work invisible instead of
 * presenting controls that can only fail or mutate local mock state.
 */
export const PRODUCT_CAPABILITIES = {
  aiKnowledgeSearch: false,
  advancedBatchOperations: false,
  changeClassification: false,
  collaborationAdvanced: false,
  knowledgeAdvancedActions: false,
  notificationTemplateManagement: false,
  notificationChannelManagement: false,
  priorityMatrix: false,
  advancedProblemActions: false,
  advancedReporting: false,
  genericTemplateMarketplace: false,
  advancedTicketRelations: false,
  rootCauseWorkflowActions: false,
  workflowAnalytics: false,
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
  { capability: 'aiKnowledgeSearch', file: 'ai-api.ts', path: /\/ai\/knowledge\/search$/, reason: 'AI knowledge search backend route is not registered' },
  { capability: 'advancedBatchOperations', file: 'batch-operations-api.ts', reason: 'Advanced batch orchestration is roadmap-only' },
  { capability: 'changeClassification', file: 'change-classification-api.ts', reason: 'Change classification/rule APIs are not registered' },
  { capability: 'changeClassification', file: 'change-api.ts', path: /\/changes\/templates\//, reason: 'Template instantiation route is not registered' },
  { capability: 'collaborationAdvanced', file: 'collaboration-api.ts', reason: 'Advanced comments, mentions and presence routes are not registered' },
  { capability: 'knowledgeAdvancedActions', file: 'knowledge-base-api.ts', reason: 'Advanced knowledge lifecycle actions are not registered' },
  { capability: 'notificationTemplateManagement', file: 'notification-preference-api.ts', reason: 'Preference reset/template application routes are not registered' },
  { capability: 'priorityMatrix', file: 'priority-matrix-api.ts', reason: 'Priority matrix backend is not registered' },
  { capability: 'advancedProblemActions', file: 'problem-api.ts', path: /\/(investigate|root-cause|solution|close|sla)$/, reason: 'Advanced problem lifecycle endpoints are not registered' },
  { capability: 'advancedProblemActions', file: 'problem-investigation.ts', path: /\/problem-relationships$/, reason: 'Problem relationship write endpoint is not registered' },
  { capability: 'advancedReporting', file: 'reports-api.ts', reason: 'Only read-only report summaries are supported by the backend' },
  { capability: 'genericTemplateMarketplace', file: 'template-api.ts', reason: 'Generic template marketplace is not registered; ticket templates use a separate supported API' },
  { capability: 'advancedTicketRelations', file: 'ticket-relations-api.ts', reason: 'Advanced relation analytics and batch routes are not registered' },
  { capability: 'rootCauseWorkflowActions', file: 'ticket-root-cause-api.ts', reason: 'Root-cause confirm/resolve routes are not registered' },
  { capability: 'workflowAnalytics', file: 'workflow-api.ts', path: /\/(workflow-templates|stats|node-stats|bottlenecks)(?:\/|$)/, reason: 'Workflow template and analytics routes are not registered' },
] as const;
