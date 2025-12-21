/**
 * 批量操作类型定义
 * 支持工单的批量分配、状态更新、字段更新、删除、导出等操作
 */

import type { TicketStatus, TicketPriority, TicketType } from './ticket';

// ==================== 批量操作类型枚举 ====================

export enum BatchOperationType {
  ASSIGN = 'assign',                     // 批量分配
  UPDATE_STATUS = 'update_status',       // 批量状态更新
  UPDATE_PRIORITY = 'update_priority',   // 批量优先级更新
  UPDATE_TYPE = 'update_type',           // 批量类型更新
  UPDATE_CATEGORY = 'update_category',   // 批量分类更新
  ADD_TAGS = 'add_tags',                 // 批量添加标签
  REMOVE_TAGS = 'remove_tags',           // 批量删除标签
  UPDATE_FIELDS = 'update_fields',       // 批量更新自定义字段
  DELETE = 'delete',                     // 批量删除
  ARCHIVE = 'archive',                   // 批量归档
  EXPORT = 'export',                     // 批量导出
  CLOSE = 'close',                       // 批量关闭
  REOPEN = 'reopen',                     // 批量重新打开
}

// ==================== 批量操作请求 ====================

export interface BatchOperationRequest {
  operationType: BatchOperationType;
  ticketIds: number[];                   // 工单IDs（最多1000个）
  data?: BatchOperationData;             // 操作数据
  comment?: string;                      // 批量操作说明
  notifyAssignees?: boolean;             // 是否通知处理人
}

export interface BatchOperationData {
  // 批量分配
  assigneeId?: number;
  teamId?: number;
  assignmentRule?: 'round_robin' | 'load_balance' | 'manual';
  
  // 批量状态更新
  status?: TicketStatus;
  resolution?: string;
  
  // 批量优先级更新
  priority?: TicketPriority;
  
  // 批量类型更新
  type?: TicketType;
  
  // 批量分类更新
  categoryId?: number;
  
  // 批量标签操作
  tags?: string[];
  
  // 批量字段更新
  customFields?: Record<string, any>;
  
  // 批量导出
  exportFormat?: 'excel' | 'csv' | 'pdf';
  exportFields?: string[];
  includeComments?: boolean;
  includeAttachments?: boolean;
  
  // 批量关闭
  closureReason?: string;
  satisfactionRequired?: boolean;
}

// ==================== 批量操作响应 ====================

export interface BatchOperationResponse {
  operationId: string;                   // 操作ID
  operationType: BatchOperationType;
  totalCount: number;                    // 总数
  successCount: number;                  // 成功数
  failedCount: number;                   // 失败数
  skippedCount: number;                  // 跳过数
  completedAt?: Date;                    // 完成时间
  errors: BatchOperationError[];         // 错误列表
  warnings: BatchOperationWarning[];     // 警告列表
  downloadUrl?: string;                  // 导出文件下载链接（导出操作）
}

export interface BatchOperationError {
  ticketId: number;
  ticketNumber: string;
  error: string;
  errorCode?: string;
  details?: any;
}

export interface BatchOperationWarning {
  ticketId: number;
  ticketNumber: string;
  warning: string;
  details?: any;
}

// ==================== 批量操作进度 ====================

export interface BatchOperationProgress {
  operationId: string;
  operationType: BatchOperationType;
  status: BatchOperationStatus;
  totalCount: number;
  processedCount: number;
  successCount: number;
  failedCount: number;
  percentage: number;                    // 百分比（0-100）
  startedAt: Date;
  estimatedCompletionTime?: Date;
  currentTicket?: {
    ticketId: number;
    ticketNumber: string;
  };
}

export enum BatchOperationStatus {
  PENDING = 'pending',                   // 等待中
  RUNNING = 'running',                   // 执行中
  PAUSED = 'paused',                     // 已暂停
  COMPLETED = 'completed',               // 已完成
  FAILED = 'failed',                     // 失败
  CANCELLED = 'cancelled',               // 已取消
}

// ==================== 批量操作日志 ====================

export interface BatchOperationLog {
  id: string;
  operationId: string;
  operationType: BatchOperationType;
  operatorId: number;
  operatorName: string;
  ticketIds: number[];
  totalCount: number;
  successCount: number;
  failedCount: number;
  status: BatchOperationStatus;
  startedAt: Date;
  completedAt?: Date;
  duration?: number;                     // 耗时（秒）
  comment?: string;
  data?: BatchOperationData;
  errors?: BatchOperationError[];
}

// ==================== 批量选择配置 ====================

export interface BatchSelectionConfig {
  maxSelectionCount?: number;            // 最大选择数量（默认1000）
  allowCrossPage?: boolean;              // 允许跨页选择
  selectAll?: boolean;                   // 全选（根据筛选条件）
  excludeIds?: number[];                 // 排除的工单IDs
}

// ==================== 批量验证 ====================

export interface BatchOperationValidation {
  isValid: boolean;
  canProceed: boolean;
  errors: ValidationError[];
  warnings: ValidationWarning[];
  affectedTickets: {
    ticketId: number;
    ticketNumber: string;
    currentStatus: TicketStatus;
    willChange: boolean;
    conflicts?: string[];
  }[];
}

export interface ValidationError {
  field: string;
  message: string;
  severity: 'error' | 'critical';
  ticketIds?: number[];
}

export interface ValidationWarning {
  field: string;
  message: string;
  ticketIds?: number[];
  canIgnore: boolean;
}

// ==================== 批量导出配置 ====================

export interface BatchExportConfig {
  format: 'excel' | 'csv' | 'pdf';
  fields: ExportField[];
  includeComments: boolean;
  includeAttachments: boolean;
  includeHistory: boolean;
  includeRelations: boolean;
  groupBy?: 'status' | 'priority' | 'assignee' | 'category';
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  fileName?: string;
}

export interface ExportField {
  fieldName: string;
  fieldLabel: string;
  includeInExport: boolean;
  order: number;
}

// ==================== 批量分配规则 ====================

export interface BatchAssignmentRule {
  type: 'round_robin' | 'load_balance' | 'skill_based' | 'manual';
  targetUserIds?: number[];
  targetTeamId?: number;
  skillTags?: string[];
  maxWorkload?: number;                  // 每人最大工作负载
  considerCurrentWorkload?: boolean;
  considerAvailability?: boolean;
}

export interface AssignmentResult {
  ticketId: number;
  assignedTo: number;
  assignedToName: string;
  currentWorkload: number;
  reason: string;
}

// ==================== 批量操作预览 ====================

export interface BatchOperationPreview {
  operationType: BatchOperationType;
  affectedCount: number;
  changes: BatchOperationChange[];
  estimatedDuration: number;             // 预计耗时（秒）
  risks: BatchOperationRisk[];
}

export interface BatchOperationChange {
  ticketId: number;
  ticketNumber: string;
  field: string;
  currentValue: any;
  newValue: any;
  willAffectSLA: boolean;
  willTriggerWorkflow: boolean;
  willSendNotification: boolean;
}

export interface BatchOperationRisk {
  level: 'low' | 'medium' | 'high' | 'critical';
  description: string;
  affectedTicketIds: number[];
  mitigation?: string;
}

// ==================== 批量操作统计 ====================

export interface BatchOperationStats {
  totalOperations: number;
  operationsByType: Record<BatchOperationType, number>;
  successRate: number;                   // 成功率（%）
  averageDuration: number;               // 平均耗时（秒）
  totalTicketsAffected: number;
  mostCommonErrors: {
    error: string;
    count: number;
  }[];
  operationsByUser: {
    userId: number;
    username: string;
    operationCount: number;
    successRate: number;
  }[];
  recentOperations: BatchOperationLog[];
}

// ==================== 批量操作权限 ====================

export interface BatchOperationPermissions {
  canBatchAssign: boolean;
  canBatchUpdateStatus: boolean;
  canBatchUpdatePriority: boolean;
  canBatchUpdateFields: boolean;
  canBatchDelete: boolean;
  canBatchExport: boolean;
  canBatchClose: boolean;
  maxBatchSize: number;                  // 最大批量操作数量
  allowedOperations: BatchOperationType[];
}

// ==================== 批量操作队列 ====================

export interface BatchOperationQueue {
  queueId: string;
  operations: QueuedOperation[];
  status: 'active' | 'paused' | 'completed';
  createdAt: Date;
  processedCount: number;
  remainingCount: number;
}

export interface QueuedOperation {
  id: string;
  operationType: BatchOperationType;
  ticketIds: number[];
  priority: 'low' | 'normal' | 'high';
  status: 'pending' | 'processing' | 'completed' | 'failed';
  scheduledAt?: Date;
  startedAt?: Date;
  completedAt?: Date;
}

// ==================== 批量操作确认 ====================

export interface BatchOperationConfirmation {
  operationType: BatchOperationType;
  ticketCount: number;
  requiresConfirmation: boolean;
  confirmationMessage: string;
  confirmationLevel: 'low' | 'medium' | 'high';
  mustTypeConfirmation?: string;         // 高危操作需要输入的确认文本
  warnings: string[];
  estimatedImpact: {
    affectedUsers: number;
    affectedTeams: number;
    slaImpact: number;
    workflowTriggers: number;
  };
}

// ==================== 工具类型 ====================

export type BatchOperationCallback = (progress: BatchOperationProgress) => void;

export type BatchOperationFilter = (ticketId: number) => boolean;

// ==================== 导出所有类型 ====================

export default BatchOperationRequest;

