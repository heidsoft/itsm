/**
 * AI 相关常量
 */

export const AIRoles = {
  USER: 'user',
  ASSISTANT: 'assistant',
  SYSTEM: 'system',
};

export const AIActionTypes = {
  CHAT: 'chat',
  ANALYZE: 'analyze',
  PREDICT: 'predict',
  ROOT_CAUSE: 'root_cause',
};

export const AIActionLabels = {
  [AIActionTypes.CHAT]: 'Intelligent conversation',
  [AIActionTypes.ANALYZE]: 'In-depth analysis',
  [AIActionTypes.PREDICT]: 'Trend Forecast',
  [AIActionTypes.ROOT_CAUSE]: 'Root Cause Analysis',
};
