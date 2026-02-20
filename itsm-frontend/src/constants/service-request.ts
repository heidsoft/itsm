/**
 * 服务请求模块常量定义
 */

// 服务请求状态枚举
export enum ServiceRequestStatus {
    SUBMITTED = 'submitted',                 // 已提交
    MANAGER_APPROVED = 'manager_approved',   // 经理已审批
    IT_APPROVED = 'it_approved',             // IT已审批
    SECURITY_APPROVED = 'security_approved', // 安全部门已审批
    REJECTED = 'rejected',                   // 已拒绝
    CANCELLED = 'cancelled',                 // 已取消
    COMPLETED = 'completed',                 // 已完成
}

// 审批状态枚举
export enum ApprovalStatus {
    PENDING = 'pending',   // 待审批
    APPROVED = 'approved', // 已通过
    REJECTED = 'rejected', // 已拒绝
}

// 审批步骤枚举
export enum ApprovalStep {
    MANAGER = 'manager',   // 经理审批
    IT = 'it',             // IT审批
    SECURITY = 'security', // 安全部门审批
}

// 审批动作枚举
export enum ApprovalAction {
    APPROVE = 'approve', // 同意
    REJECT = 'reject',   // 拒绝
}
