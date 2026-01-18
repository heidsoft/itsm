/**
 * 增强版 Ticket API
 * 集成统一的错误处理
 */

import { TicketApi } from './ticket-api';
import { handleApiRequest } from './base-api-handler';
import type {
  Ticket,
  TicketListResponse,
  CreateTicketRequest,
  GetTicketsParams,
} from './api-config';

/**
 * 增强版 Ticket API 类
 * 所有方法都包含统一的错误处理
 */
export class TicketApiEnhanced {
  /**
   * 获取工单列表
   * @param params 查询参数
   * @returns Promise<TicketListResponse>
   */
  static async getTickets(params?: GetTicketsParams & { [key: string]: unknown }) {
    return handleApiRequest(
      TicketApi.getTickets(params),
      {
        errorMessage: '获取工单列表失败，请稍后重试',
        silent: true, // 列表查询失败时不显示错误，由页面组件处理
      }
    );
  }

  /**
   * 创建工单
   * @param data 工单数据
   * @returns Promise<Ticket>
   */
  static async createTicket(data: CreateTicketRequest) {
    return handleApiRequest(
      TicketApi.createTicket(data),
      {
        errorMessage: '创建工单失败，请检查输入信息',
        showSuccess: true,
        successMessage: '工单创建成功',
      }
    );
  }

  /**
   * 获取工单详情
   * @param id 工单ID
   * @returns Promise<Ticket>
   */
  static async getTicket(id: number) {
    return handleApiRequest(
      TicketApi.getTicket(id),
      {
        errorMessage: '获取工单详情失败',
      }
    );
  }

  /**
   * 更新工单状态
   * @param id 工单ID
   * @param status 新状态
   * @returns Promise<Ticket>
   */
  static async updateTicketStatus(id: number, status: string) {
    return handleApiRequest(
      TicketApi.updateTicketStatus(id, status),
      {
        errorMessage: '更新工单状态失败',
        showSuccess: true,
        successMessage: '工单状态更新成功',
      }
    );
  }

  /**
   * 更新工单信息
   * @param id 工单ID
   * @param data 更新数据
   * @returns Promise<Ticket>
   */
  static async updateTicket(id: number, data: Partial<CreateTicketRequest>) {
    return handleApiRequest(
      TicketApi.updateTicket(id, data),
      {
        errorMessage: '更新工单失败',
        showSuccess: true,
        successMessage: '工单更新成功',
      }
    );
  }

  /**
   * 删除工单
   * @param id 工单ID
   * @returns Promise<void>
   */
  static async deleteTicket(id: number) {
    return handleApiRequest(
      TicketApi.deleteTicket(id),
      {
        errorMessage: '删除工单失败',
        showSuccess: true,
        successMessage: '工单删除成功',
      }
    );
  }

  /**
   * 批量删除工单
   * @param ids 工单ID数组
   * @returns Promise<{success: number; failed: number}>
   */
  static async batchDeleteTickets(ids: number[]) {
    return handleApiRequest(
      TicketApi.batchDeleteTickets(ids),
      {
        errorMessage: '批量删除工单失败',
        showSuccess: true,
        successMessage: `成功删除 ${ids.length} 个工单`,
      }
    );
  }

  /**
   * 分配工单
   * @param id 工单ID
   * @param assigneeId 指派人ID
   * @param comment 备注
   * @returns Promise<Ticket>
   */
  static async assignTicket(id: number, assigneeId: number, comment?: string) {
    return handleApiRequest(
      TicketApi.assignTicket(id, { assignee_id: assigneeId, comment }),
      {
        errorMessage: '分配工单失败',
        showSuccess: true,
        successMessage: '工单分配成功',
      }
    );
  }

  /**
   * 升级工单
   * @param id 工单ID
   * @param data 升级数据
   * @returns Promise<Ticket>
   */
  static async escalateTicket(
    id: number, 
    data: { 
      level: string; 
      reason: string; 
      assignee_id?: number 
    }
  ) {
    return handleApiRequest(
      TicketApi.escalateTicket(id, data),
      {
        errorMessage: '升级工单失败',
        showSuccess: true,
        successMessage: '工单升级成功',
      }
    );
  }

  /**
   * 解决工单
   * @param id 工单ID
   * @param data 解决方案数据
   * @returns Promise<Ticket>
   */
  static async resolveTicket(
    id: number, 
    data: { 
      solution: string; 
      resolution_code?: string 
    }
  ) {
    return handleApiRequest(
      TicketApi.resolveTicket(id, data),
      {
        errorMessage: '解决工单失败',
        showSuccess: true,
        successMessage: '工单已标记为已解决',
      }
    );
  }

  /**
   * 关闭工单
   * @param id 工单ID
   * @param data 关闭数据
   * @returns Promise<Ticket>
   */
  static async closeTicket(
    id: number, 
    data: { 
      close_notes?: string 
    }
  ) {
    return handleApiRequest(
      TicketApi.closeTicket(id, data),
      {
        errorMessage: '关闭工单失败',
        showSuccess: true,
        successMessage: '工单已关闭',
      }
    );
  }

  /**
   * 添加评论
   * @param id 工单ID
   * @param content 评论内容
   * @returns Promise
   */
  static async addComment(id: number, content: string) {
    return handleApiRequest(
      TicketApi.addComment(id, content),
      {
        errorMessage: '添加评论失败',
        showSuccess: true,
        successMessage: '评论添加成功',
      }
    );
  }

  /**
   * 审批工单
   * @param id 工单ID
   * @param data 审批数据
   * @returns Promise
   */
  static async approveTicket(
    id: number,
    data: {
      action: 'approve' | 'reject' | 'delegate';
      comment?: string;
      delegate_to_user_id?: number;
    }
  ) {
    const actionText = data.action === 'approve' ? '审批通过' : data.action === 'reject' ? '审批拒绝' : '转交';
    return handleApiRequest(
      TicketApi.approveTicket(id, { ...data, ticket_id: id }),
      {
        errorMessage: `${actionText}失败`,
        showSuccess: true,
        successMessage: `工单${actionText}成功`,
      }
    );
  }

  /**
   * 上传附件
   * @param id 工单ID
   * @param file 文件
   * @returns Promise
   */
  static async uploadAttachment(id: number, file: File) {
    return handleApiRequest(
      TicketApi.uploadTicketAttachment(id, file),
      {
        errorMessage: '上传附件失败',
        showSuccess: true,
        successMessage: '附件上传成功',
      }
    );
  }
}

// 默认导出增强版 API
export default TicketApiEnhanced;

