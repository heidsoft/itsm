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
    [AIActionTypes.CHAT]: '智能对话',
    [AIActionTypes.ANALYZE]: '深度分析',
    [AIActionTypes.PREDICT]: '趋势预测',
    [AIActionTypes.ROOT_CAUSE]: '根因分析',
};
