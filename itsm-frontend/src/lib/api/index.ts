/**
 * API模块统一导出
 * 提供所有API客户端的集中访问点
 */

// 基础类
export { BaseApi } from './base-api';
export type { BaseApiV2, ApiError, ApiResult, CrudApiInterface, PaginatedApiInterface, BatchApiInterface } from './base-api-v2';
export { httpClient } from './http-client';

// 类型定义
export type {
  ApiResponse,
  PaginationResponse,
  ListQueryParams,
  BatchOperationRequest,
  BatchOperationResponse,
  RequestOptions,
  UploadProgress,
  FileUploadOptions,
  ApiErrorCode,
} from './types';

// ==================== 核心业务API ====================

// 工单
export { TicketApi } from './ticket-api';
export { TicketApprovalApi } from './ticket-approval-api';
export { TicketAssignmentApi } from './ticket-assignment-api';
export { TicketAttachmentApi } from './ticket-attachment-api';
export { TicketCommentApi } from './ticket-comment-api';
export { TicketNotificationApi } from './ticket-notification-api';
export { TicketRatingApi } from './ticket-rating-api';
export { TicketAnalyticsApi } from './ticket-analytics-api';
export { TicketViewApi } from './ticket-view-api';
export { TicketRelationsApi } from './ticket-relations-api';
export { TicketAutomationRuleApi } from './ticket-automation-rule-api';
export { TicketPredictionApi } from './ticket-prediction-api';
export { TicketRootCauseApi } from './ticket-root-cause-api';

// 事件
export { IncidentAPI } from './incident-api';

// 变更
export { ChangeApi } from './change-api';
export { ChangeClassificationApi } from './change-classification-api';

// 服务请求
export { serviceRequestAPI } from './service-request-api';
export { ServiceCatalogApi } from './service-catalog-api';

// 知识库
export { KnowledgeApi } from './knowledge-api';
export { KnowledgeBaseApi } from './knowledge-base-api';

// ==================== 系统管理API ====================

// 用户和权限
export { UserApi } from './user-api';
export { RoleAPI } from './role-api';
export { AuthAPI } from './auth-api';
export { TenantAPI } from './tenant-api';

// 配置管理
export { CMDBApi } from './cmdb-api';
export { SystemConfigAPI } from './system-config-api';

// SLA
export { SLAApi } from './sla-api';

// ==================== 工作流API ====================

export { WorkflowApi } from './workflow-api';

// ==================== 仪表盘和报表API ====================

export { DashboardAPI } from './dashboard-api';
export { ReportsApi } from './reports-api';

// ==================== AI相关API ====================

export * from './ai-api';

// ==================== 其他功能API ====================

export { BatchOperationsApi } from './batch-operations-api';
export { CollaborationApi } from './collaboration-api';
export { globalSearchApi } from './global-search-api';
export { listAuditLogs, type AuditLog, type ListAuditLogsParams, type ListAuditLogsResponse } from './auditlog-api';
export { PriorityMatrixApi } from './priority-matrix-api';
export { TemplateApi } from './template-api';

// ==================== 工具函数 ====================

import type { PaginationResponse } from './types';

/**
 * 延迟函数
 */
export const delay = (ms: number): Promise<void> =>
  new Promise((resolve) => setTimeout(resolve, ms));

/**
 * 重试函数
 */
export const retry = async <T>(
  fn: () => Promise<T>,
  maxRetries: number = 3,
  delayMs: number = 1000
): Promise<T> => {
  let lastError: Error | undefined;
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error as Error;
      if (i < maxRetries - 1) {
        await delay(delayMs * (i + 1));
      }
    }
  }
  throw lastError;
};

/**
 * 创建空分页响应
 */
export const emptyPagination = <T>(data: T[] = []): PaginationResponse<T> => ({
  data,
  total: 0,
  page: 1,
  page_size: 10,
  total_pages: 0,
});

/**
 * 检查API响应是否成功
 */
export const isSuccess = (code: number): boolean => code === 0;

/**
 * 获取错误消息
 */
export const getErrorMessage = (message: string, defaultMessage: string = '操作失败'): string =>
  message || defaultMessage;
