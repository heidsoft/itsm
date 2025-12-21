/**
 * 变更管理类型定义
 */

import {
    ChangeStatus, ChangeType,
    ChangePriority, ChangeImpact, ChangeRisk
} from "./constants";

// 变更实体
export interface Change {
    id: number;
    title: string;
    description: string;
    justification: string;
    type: ChangeType;
    status: ChangeStatus;
    priority: ChangePriority;
    impact_scope: ChangeImpact;
    risk_level: ChangeRisk;
    assignee_id?: number;
    assignee_name?: string;
    created_by: number;
    created_by_name: string;
    tenant_id: number;
    planned_start_date?: string;
    planned_end_date?: string;
    actual_start_date?: string;
    actual_end_date?: string;
    implementation_plan: string;
    rollback_plan: string;
    affected_cis?: string[];
    related_tickets?: string[];
    created_at: string;
    updated_at: string;
}

// 审批记录
export interface ApprovalRecord {
    id: number;
    change_id: number;
    approver_id: number;
    approver_name: string;
    status: ChangeStatus;
    comment?: string;
    approved_at?: string;
    created_at: string;
}

// 审批链/工作流项
export interface ApprovalChainItem {
    id: number;
    level: number;
    approver_id: number;
    approver_name: string;
    role: string;
    status: string;
    is_required: boolean;
}

// 风险评估
export interface RiskAssessment {
    id: number;
    change_id: number;
    risk_level: ChangeRisk;
    risk_description: string;
    impact_analysis: string;
    mitigation_measures: string;
    contingency_plan?: string;
    risk_owner: string;
    risk_review_date?: string;
}

// 创建变更请求
export interface CreateChangeRequest {
    title: string;
    description: string;
    justification: string;
    type: ChangeType;
    priority: ChangePriority;
    impact_scope: ChangeImpact;
    risk_level: ChangeRisk;
    planned_start_date?: string;
    planned_end_date?: string;
    implementation_plan: string;
    rollback_plan: string;
    affected_cis?: string[];
    related_tickets?: string[];
}

// 列表查询参数
export interface ChangeQuery {
    page?: number;
    page_size?: number;
    status?: string;
    type?: string;
    search?: string;
}

// 列表响应
export interface ChangeListResponse {
    changes: Change[];
    total: number;
    page: number;
    size: number;
}

// 统计响应
export interface ChangeStats {
    total: number;
    pending: number;
    approved: number;
    in_progress: number;
    completed: number;
    rolled_back: number;
    rejected: number;
    cancelled: number;
}
