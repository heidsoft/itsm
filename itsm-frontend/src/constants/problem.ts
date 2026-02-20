/**
 * 问题管理相关常量定义
 */

// 问题状态
export enum ProblemStatus {
    OPEN = 'open',
    IN_PROGRESS = 'in_progress',
    RESOLVED = 'resolved',
    CLOSED = 'closed',
}

// 状态描述映射
export const ProblemStatusLabels: Record<ProblemStatus, string> = {
    [ProblemStatus.OPEN]: '待处理',
    [ProblemStatus.IN_PROGRESS]: '处理中',
    [ProblemStatus.RESOLVED]: '已解决',
    [ProblemStatus.CLOSED]: '已关闭',
};

// 优先级
export enum ProblemPriority {
    LOW = 'low',
    MEDIUM = 'medium',
    HIGH = 'high',
    CRITICAL = 'critical',
}

export const ProblemPriorityLabels: Record<ProblemPriority, string> = {
    [ProblemPriority.LOW]: '低',
    [ProblemPriority.MEDIUM]: '中',
    [ProblemPriority.HIGH]: '高',
    [ProblemPriority.CRITICAL]: '极高',
};
