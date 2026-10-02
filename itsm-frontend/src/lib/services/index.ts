/**
 * Services Index
 *
 * 统一的服务导出入口，提供类型安全的 API 调用。
 * 所有服务都继承自 BaseService，遵循一致的接口设计。
 */

// Base Service
export { BaseService, ApiError } from './base-service';
export type {
  PaginationParams,
  PaginatedResponse,
  ListParams,
} from './base-service';

// Ticket Service：唯一实现是 ./ticket-service（页面直接按路径导入，此处不再复制一层导出面）。
export { ticketService } from './ticket-service';
export type { TicketComment, TicketAttachment } from './ticket-service';
