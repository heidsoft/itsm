/**
 * 问题管理类型定义
 */

import { ProblemPriority, ProblemStatus } from '@/constants/problem';

// 问题实体接口
export interface Problem {
  id: number;
  title: string;
  description: string;
  status: ProblemStatus;
  priority: ProblemPriority;
  category: string;
  rootCause?: string;
  impact?: string;
  assigneeId?: number;
  createdBy: number;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

// 创建问题请求
export interface CreateProblemRequest {
  title: string;
  description: string;
  priority: ProblemPriority;
  category: string;
  rootCause?: string;
  impact?: string;
}

// 更新问题请求
export interface UpdateProblemRequest {
  title?: string;
  description?: string;
  status?: ProblemStatus;
  priority?: ProblemPriority;
  category?: string;
  rootCause?: string;
  impact?: string;
  assigneeId?: number;
}

// 列表查询参数
export interface ProblemQuery {
  page?: number;
  pageSize?: number;
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
  pageSize: number;
}

// 统计响应
export interface ProblemStats {
  total: number;
  open: number;
  inProgress: number;
  resolved: number;
  closed: number;
  highPriority: number;
}
