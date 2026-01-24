/**
 * 统一API类型定义导出
 *
 * @deprecated 此目录已弃用，请直接导入具体类型:
 * import { ApiResponse, PaginationResponse, Ticket } from '@/lib/api/types';
 */

export * from '../types';

// API URL配置
export const API_URLS = {
  // 认证相关
  LOGIN: '/api/v1/auth/login',
  LOGOUT: '/api/v1/auth/logout',
  REFRESH: '/api/v1/auth/refresh',
  PROFILE: '/api/v1/auth/profile',

  // 用户管理
  USERS: '/api/v1/users',
  USER: (id: string | number) => `/api/v1/users/${id}`,

  // 事件管理
  INCIDENTS: '/api/v1/incidents',
  INCIDENT: (id: string | number) => `/api/v1/incidents/${id}`,
  INCIDENT_COMMENTS: (id: string | number) => `/api/v1/incidents/${id}/comments`,
  INCIDENT_ATTACHMENTS: (id: string | number) => `/api/v1/incidents/${id}/attachments`,

  // 变更管理
  CHANGES: '/api/v1/changes',
  CHANGE: (id: string | number) => `/api/v1/changes/${id}`,

  // 服务管理
  SERVICES: '/api/v1/services',
  SERVICE: (id: string | number) => `/api/v1/services/${id}`,

  // 仪表板
  DASHBOARD: '/api/v1/dashboard',
  DASHBOARD_WIDGETS: '/api/v1/dashboard/widgets',

  // 工单管理
  TICKETS: '/api/v1/tickets',
  TICKET: (id: string | number) => `/api/v1/tickets/${id}`,

  // 知识库
  KNOWLEDGE: '/api/v1/knowledge',
  ARTICLE: (id: string | number) => `/api/v1/knowledge/${id}`,
} as const;
