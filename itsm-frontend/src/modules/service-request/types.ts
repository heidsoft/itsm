/**
 * 服务请求模块类型定义
 */

import { ServiceRequestStatus, ApprovalStatus, ApprovalStep, ApprovalAction } from './constants';

// 服务目录简要信息 (用于内嵌在服务请求中)
export interface ServiceCatalogRef {
    id: number;
    name: string;
    category: string;
    description?: string;
}

// 请求者简要信息
export interface RequesterRef {
    id: number;
    username: string;
    name: string;
    email: string;
    department?: string;
}

// 服务请求审批记录
export interface ServiceRequestApproval {
    id: number;
    service_request_id: number;
    level: number;
    step: ApprovalStep | string;
    status: ApprovalStatus | string;
    approver_id?: number;
    approver_name?: string;
    action?: ApprovalAction | string;
    comment?: string;
    created_at: string;
    processed_at?: string;

    // V1 新增字段
    timeout_hours?: number;
    due_at?: string;
    is_escalated?: boolean;
    delegated_to_id?: number;
    escalation_reason?: string;
}

// 服务请求实体 (对应后端 DTO)
export interface ServiceRequest {
    id: number;
    catalog_id: number;
    requester_id: number;
    status: ServiceRequestStatus | string;
    title?: string;
    reason?: string;
    form_data?: Record<string, any>;

    cost_center?: string;
    data_classification?: string; // public, internal, confidential
    needs_public_ip?: boolean;
    source_ip_whitelist?: string[];
    expire_at?: string;
    compliance_ack: boolean;

    current_level: number;
    total_levels: number;
    created_at: string;
    updated_at: string;

    approvals?: ServiceRequestApproval[];
    catalog?: ServiceCatalogRef;   // 后端目前可能未填充，需注意
    requester?: RequesterRef;      // 后端目前可能未填充，需注意
}

// 创建服务请求参数
export interface CreateServiceRequestRequest {
    catalog_id: number;
    title?: string;
    reason?: string;
    form_data?: Record<string, any>;

    cost_center?: string;
    data_classification?: string;
    needs_public_ip?: boolean;
    source_ip_whitelist?: string[];
    expire_at?: string;
    compliance_ack: boolean;
}

// 审批动作请求参数
export interface ServiceRequestApprovalActionRequest {
    action: 'approve' | 'reject';
    comment?: string;
}

// 列表查询参数
export interface ServiceRequestQuery {
    page?: number;
    size?: number;
    status?: ServiceRequestStatus;
    scope?: 'me' | 'all'; // me: 我的请求, all: 管理员查看所有
}

// 列表响应
export interface ServiceRequestListResponse {
    requests: ServiceRequest[];
    total: number;
    page: number;
    size: number;
}
