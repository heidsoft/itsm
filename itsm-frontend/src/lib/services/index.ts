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

// Ticket Service
export { TicketService, ticketService } from './ticket-service-v2';
export type {
  CreateTicketParams,
  UpdateTicketParams,
  TicketQueryParams,
  TicketStats,
  TicketSLAInfo,
  TicketComment,
  TicketAttachment,
  TicketActivity,
} from './ticket-service-v2';
