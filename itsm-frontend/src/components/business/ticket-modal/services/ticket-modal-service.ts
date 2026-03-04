/**
 * Ticket Modal 服务层
 * 封装与工单模态框相关的 API 调用
 */

import type { TicketFormValues } from '../types';
import { ticketService } from '@/lib/services/ticket-service';

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
  // TODO: 替换为实际的 API 调用
  // return await UserApi.getUsers();
  return [
    { id: 1, name: 'Alice', role: 'IT Support', avatar: 'A' },
    { id: 2, name: 'Bob', role: 'Network Admin', avatar: 'B' },
    { id: 3, name: 'Charlie', role: 'Service Desk', avatar: 'C' },
  ];
};

/**
 * 获取工单模板列表
 */
export const fetchTicketTemplates = async (): Promise<any[]> => {
  // TODO: 替换为实际的 API 调用
  // return await TemplateApi.getTemplates();
  return [];
};
