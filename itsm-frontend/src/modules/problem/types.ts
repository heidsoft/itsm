/**
 * 问题管理类型定义
 */

import { ProblemPriority, ProblemStatus } from "./constants";

// 问题实体接口
export interface Problem {
    id: number;
    title: string;
    description: string;
    status: ProblemStatus;
    priority: ProblemPriority;
    category: string;
    root_cause?: string;
    impact?: string;
    assignee_id?: number;
    created_by: number;
    tenant_id: number;
    created_at: string;
    updated_at: string;
}

// 创建问题请求
export interface CreateProblemRequest {
    title: string;
    description: string;
    priority: ProblemPriority;
    category: string;
    root_cause?: string;
    impact?: string;
}

// 更新问题请求
export interface UpdateProblemRequest {
    title?: string;
    description?: string;
    status?: ProblemStatus;
    priority?: ProblemPriority;
    category?: string;
    root_cause?: string;
    impact?: string;
    assignee_id?: number;
}

// 列表查询参数
export interface ProblemQuery {
    page?: number;
    page_size?: number;
    status?: string;
    priority?: string;
    category?: string;
    keyword?: string;
}

// 列表响应
export interface ProblemListResponse {
    problems: Problem[];
    total: number;
    page: number;
    page_size: number;
}

// 统计响应
export interface ProblemStats {
    total: number;
    open: number;
    in_progress: number;
    resolved: number;
    closed: number;
    high_priority: number;
}
