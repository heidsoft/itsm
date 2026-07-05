/**
 * 服务请求模块类型定义
 */

import type {
  ServiceRequestStatus,
  ApprovalStatus,
  ApprovalStep,
  ApprovalAction,
} from '@/constants/service-request';

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
  serviceRequestId: number;
  level: number;
  step: ApprovalStep | string;
  status: ApprovalStatus | string;
  approverId?: number;
  approverName?: string;
  action?: ApprovalAction | string;
  comment?: string;
  createdAt: string;
  processedAt?: string;

  // V1 新增字段
  timeoutHours?: number;
  dueAt?: string;
  isEscalated?: boolean;
  delegatedToId?: number;
  escalationReason?: string;
}

// 服务请求实体 (对应后端 DTO)
export interface ServiceRequest {
  id: number;
  catalogId: number;
  requesterId: number;
  status: ServiceRequestStatus | string;
  title?: string;
  reason?: string;
  formData?: Record<string, any>;

  costCenter?: string;
  dataClassification?: string;
  needsPublicIp?: boolean;
  sourceIpWhitelist?: string[];
  expireAt?: string;
  complianceAck: boolean;

  currentLevel: number;
  totalLevels: number;
  createdAt: string;
  updatedAt: string;

  approvals?: ServiceRequestApproval[];
  catalog?: ServiceCatalogRef; // 后端目前可能未填充，需注意
  requester?: RequesterRef; // 后端目前可能未填充，需注意
}

// 创建服务请求参数
export interface CreateServiceRequestRequest {
  catalogId: number;
  title?: string;
  reason?: string;
  formData?: Record<string, any>;

  costCenter?: string;
  dataClassification?: string;
  needsPublicIp?: boolean;
  sourceIpWhitelist?: string[];
  expireAt?: string;
  complianceAck: boolean;
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
