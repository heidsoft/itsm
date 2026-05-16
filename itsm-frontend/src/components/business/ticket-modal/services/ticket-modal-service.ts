/**
 * Ticket Modal 服务层
 * 封装与工单模态框相关的 API 调用
 */

import type { TicketFormValues } from '../types';
import { ticketService } from '@/lib/services/ticket-service';
import { UserApi } from '@/lib/api/user-api';
import { TemplateApi } from '@/lib/api/template-api';

/**
 * 提交工单
 */
export const submitTicket = async (values: TicketFormValues): Promise<any> => {
  // 这里可以添加数据转换逻辑
  const payload = {
    ...values,
    // 如果有 estimated_time，转换为 appropriate format
    estimated_time: values.estimated_time || undefined,
  };

  return await ticketService.createTicket(payload);
};

/**
 * 更新工单
 */
export const updateTicket = async (
  ticketId: number,
  values: TicketFormValues
): Promise<any> => {
  const payload = {
    ...values,
    estimated_time: values.estimated_time || undefined,
  };

  return await ticketService.updateTicket(ticketId, payload);
};

/**
 * 获取用户列表（用于分配）
 */
export const fetchUserList = async (): Promise<any[]> => {
  try {
    const response = await UserApi.getUsers({ page: 1, page_size: 100 });
    return response.users || [];
  } catch (error) {
    console.error('Failed to fetch users:', error);
    return [];
  }
};

/**
 * 获取工单模板列表
 */
export const fetchTicketTemplates = async (): Promise<any[]> => {
  try {
    const response = await TemplateApi.getTemplates();
    return response.templates || [];
  } catch (error) {
    console.error('Failed to fetch templates:', error);
    return [];
  }
};
