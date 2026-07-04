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
  catalog_id: number;
  catalogId?: number;
  requester_id: number;
  requesterId?: number;
  status: ServiceRequestStatus | string;
  title?: string;
  reason?: string;
  form_data?: Record<string, any>;
  formData?: Record<string, any>;

  cost_center?: string;
  costCenter?: string;
  data_classification?: string;
  dataClassification?: string;
  needs_public_ip?: boolean;
  needsPublicIp?: boolean;
  source_ip_whitelist?: string[];
  sourceIpWhitelist?: string[];
  expire_at?: string;
  expireAt?: string;
  compliance_ack: boolean;
  complianceAck?: boolean;

  current_level: number;
  currentLevel?: number;
  total_levels: number;
  totalLevels?: number;
  created_at: string;
  createdAt?: string;
  updated_at: string;
  updatedAt?: string;

  approvals?: ServiceRequestApproval[];
  catalog?: ServiceCatalogRef; // 后端目前可能未填充，需注意
  requester?: RequesterRef; // 后端目前可能未填充，需注意
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
