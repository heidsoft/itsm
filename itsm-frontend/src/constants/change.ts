/**
 * 变更管理相关常量定义
 */

// 变更状态
export enum ChangeStatus {
    DRAFT = 'draft',
    PENDING = 'pending',
    APPROVED = 'approved',
    REJECTED = 'rejected',
    IN_PROGRESS = 'in_progress',
    COMPLETED = 'completed',
    ROLLED_BACK = 'rolled_back',
    CANCELLED = 'cancelled',
}

export const ChangeStatusLabels: Record<ChangeStatus, string> = {
    [ChangeStatus.DRAFT]: '草稿',
    [ChangeStatus.PENDING]: '待审批',
    [ChangeStatus.APPROVED]: '已批准',
    [ChangeStatus.REJECTED]: '已拒绝',
    [ChangeStatus.IN_PROGRESS]: '实施中',
    [ChangeStatus.COMPLETED]: '已完成',
    [ChangeStatus.ROLLED_BACK]: '已回滚',
    [ChangeStatus.CANCELLED]: '已取消',
};

// 变更类型
export enum ChangeType {
    NORMAL = 'normal',
    STANDARD = 'standard',
    EMERGENCY = 'emergency',
}

export const ChangeTypeLabels: Record<ChangeType, string> = {
    [ChangeType.NORMAL]: '普通变更',
    [ChangeType.STANDARD]: '标准变更',
    [ChangeType.EMERGENCY]: '紧急变更',
};

// 优先级
export enum ChangePriority {
    LOW = 'low',
    MEDIUM = 'medium',
    HIGH = 'high',
    CRITICAL = 'critical',
}

export const ChangePriorityLabels: Record<ChangePriority, string> = {
    [ChangePriority.LOW]: '低',
    [ChangePriority.MEDIUM]: '中',
    [ChangePriority.HIGH]: '高',
    [ChangePriority.CRITICAL]: '极高',
};

// 影响范围
export enum ChangeImpact {
    LOW = 'low',
    MEDIUM = 'medium',
    HIGH = 'high',
}

export const ChangeImpactLabels: Record<ChangeImpact, string> = {
    [ChangeImpact.LOW]: '低',
    [ChangeImpact.MEDIUM]: '中',
    [ChangeImpact.HIGH]: '高',
};

// 风险等级
export enum ChangeRisk {
    LOW = 'low',
    MEDIUM = 'medium',
    HIGH = 'high',
}

export const ChangeRiskLabels: Record<ChangeRisk, string> = {
    [ChangeRisk.LOW]: '低',
    [ChangeRisk.MEDIUM]: '中',
    [ChangeRisk.HIGH]: '高',
};
