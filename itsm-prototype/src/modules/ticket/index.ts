/**
 * ITSM前端架构 - 工单管理模块入口
 * 
 * 统一导出工单管理模块的所有功能
 */

// ==================== 组件导出 ====================
export {
  SearchInput,
  StatusTag,
  TicketActions,
  TicketFilters,
  TicketList,
  TicketManagementTemplate,
  TicketManagementPage,
} from './components/TicketManagement';

// ==================== 状态管理导出 ====================
export {
  useTicketListStore,
  useTicketDetailStore,
  ticketListConfig,
  ticketDetailConfig,
} from './store/ticket-store';

// ==================== API管理导出 ====================
export {
  TicketApiService,
  useTickets,
  useTicket,
  useTicketComments,
  useTicketAttachments,
  useCreateTicket,
  useUpdateTicket,
  useDeleteTicket,
  useBatchDeleteTickets,
  useAssignTicket,
  useResolveTicket,
  useCloseTicket,
  useReopenTicket,
  useAddComment,
  useUploadAttachment,
  TicketCacheManager,
  QUERY_KEYS,
} from './api/ticket-api';

// ==================== 类型定义导出 ====================
export {
  BaseEntity,
  PaginationResponse,
  ApiResponse,
  ErrorResponse,
  User,
  UserRole,
  UserStatus,
  TicketStatus,
  TicketPriority,
  TicketType,
  Ticket,
  UserGroup,
  SLA,
  SLAStatus,
  BusinessHours,
  EscalationRule,
  EscalationAction,
  SLACondition,
  Attachment,
  Comment,
  ActivityType,
  Activity,
  TicketFilters,
  SortConfig,
  QueryParams,
  CreateTicketRequest,
  UpdateTicketRequest,
  AssignTicketRequest,
  ResolveTicketRequest,
  CloseTicketRequest,
  ReopenTicketRequest,
  TicketStats,
  TimeRangeStats,
  NotificationType,
  NotificationTemplate,
  Notification,
  TypeGuards,
} from './types/ticket-types';

// ==================== 工具函数导出 ====================
export {
  StateUtils,
  TicketUtils,
  DataUtils,
  FormatUtils,
  ValidationUtils,
} from './utils/ticket-utils';

// ==================== 模块配置 ====================
import { registerModule, CORE_MODULES } from '@/lib/architecture/modules';

/**
 * 工单管理模块配置
 */
const ticketModuleConfig = {
  name: '工单管理模块',
  version: '1.0.0',
  dependencies: ['auth', 'user', 'notification'],
  exports: [
    'TicketManagementPage',
    'useTicketListStore',
    'useTicketDetailStore',
    'TicketApiService',
    'TicketUtils',
  ],
  routes: [
    '/tickets',
    '/tickets/create',
    '/tickets/:id',
    '/tickets/:id/edit',
  ],
  permissions: [
    'ticket:view',
    'ticket:create',
    'ticket:edit',
    'ticket:delete',
    'ticket:assign',
    'ticket:resolve',
    'ticket:close',
  ],
};

// 注册模块
registerModule(CORE_MODULES.TICKET, ticketModuleConfig);

// ==================== 模块初始化 ====================

/**
 * 工单管理模块初始化函数
 */
export function initializeTicketModule() {
  console.log('初始化工单管理模块...');
  
  // 注册组件
  // 这里可以添加组件注册逻辑
  
  // 初始化状态管理
  // 这里可以添加状态管理初始化逻辑
  
  // 设置API拦截器
  // 这里可以添加API拦截器设置逻辑
  
  console.log('工单管理模块初始化完成');
}

/**
 * 工单管理模块清理函数
 */
export function cleanupTicketModule() {
  console.log('清理工单管理模块...');
  
  // 清理状态
  // 这里可以添加状态清理逻辑
  
  // 清理缓存
  // 这里可以添加缓存清理逻辑
  
  console.log('工单管理模块清理完成');
}

// ==================== 默认导出 ====================
export default {
  initializeTicketModule,
  cleanupTicketModule,
  ticketModuleConfig,
};
