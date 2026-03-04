/**
 * Ticket Modal 工具函数和常量
 */

import { TicketType, TicketPriority } from '@/lib/services/ticket-service';
import type { FormStep, TicketTemplate, User } from '../types';

// ============ 步骤配置常量 ============

export const TICKET_FORM_STEPS: FormStep[] = [
  {
    title: '基本信息',
    description: '填写工单标题和描述',
    content: 'step1',
  },
  {
    title: '分类设置',
    description: '选择类型和优先级',
    content: 'step2',
  },
  {
    title: '分配处理',
    description: '指定处理人和预计时间',
    content: 'step3',
  },
  {
    title: '确认提交',
    description: '检查信息并提交',
    content: 'step4',
  },
];

// ============ 验证规则 ============

export const TITLE_RULES = [
  { required: true, message: '请输入工单标题' },
];

export const DESCRIPTION_RULES = [
  { required: true, message: '请输入工单描述' },
  { min: 10, message: '描述至少需要 10 个字符' },
];

export const TYPE_RULES = [
  { required: true, message: '请选择工单类型' },
];

export const CATEGORY_RULES = [
  { required: true, message: '请选择工单分类' },
];

export const PRIORITY_RULES = [
  { required: true, message: '请选择优先级' },
];

// ============ 选项配置 ============

export const TICKET_TYPE_OPTIONS = [
  { value: TicketType.INCIDENT, label: '事件' },
  { value: TicketType.SERVICE_REQUEST, label: '服务请求' },
  { value: TicketType.PROBLEM, label: '问题' },
  { value: TicketType.CHANGE, label: '变更' },
];

export const PRIORITY_OPTIONS = [
  { value: TicketPriority.LOW, label: '低 - 一般问题，可延后处理' },
  { value: TicketPriority.MEDIUM, label: '中 - 正常问题，按序处理' },
  { value: TicketPriority.HIGH, label: '高 - 重要问题，优先处理' },
  { value: TicketPriority.URGENT, label: '紧急 - 严重影响业务，立即处理' },
];

export const CATEGORY_OPTIONS = [
  { value: 'System Access', label: '系统访问' },
  { value: 'Hardware Equipment', label: '硬件设备' },
  { value: 'Software Services', label: '软件服务' },
  { value: 'Network Services', label: '网络服务' },
];

// ============ 模拟用户列表（应该从 API 获取） ============

export const MOCK_USER_LIST: User[] = [
  { id: 1, name: 'Alice', role: 'IT Support', avatar: 'A' },
  { id: 2, name: 'Bob', role: 'Network Admin', avatar: 'B' },
  { id: 3, name: 'Charlie', role: 'Service Desk', avatar: 'C' },
];

// ============ 工单模板数据（应该从 API 获取） ============

export const MOCK_TICKET_TEMPLATES: TicketTemplate[] = [
  {
    id: 1,
    name: 'System Login Issue',
    type: TicketType.INCIDENT,
    category: 'System Access',
    priority: TicketPriority.MEDIUM,
    description: 'User unable to login to system, technical support needed',
    estimatedTime: '2 hours',
    sla: '4 hours',
  },
  {
    id: 2,
    name: 'Printer Malfunction',
    type: TicketType.INCIDENT,
    category: 'Hardware Equipment',
    priority: TicketPriority.HIGH,
    description: 'Office printer not working properly',
    estimatedTime: '1 hour',
    sla: '2 hours',
  },
  {
    id: 3,
    name: 'Software Installation Request',
    type: TicketType.SERVICE_REQUEST,
    category: 'Software Services',
    priority: TicketPriority.LOW,
    description: 'Need to install new office software',
    estimatedTime: '30 minutes',
    sla: '4 hours',
  },
];

// ============ 工具函数 ============

/**
 * 获取指定步骤需要验证的字段
 */
export const getFieldsForStep = (step: number): string[] => {
  switch (step) {
    case 0:
      return ['title', 'description'];
    case 1:
      return ['type', 'category', 'priority'];
    case 2:
      return []; // 可选字段
    default:
      return [];
  }
};

/**
 * 合并表单数据
 */
export const mergeFormData = (
  formData: Record<string, any>,
  formValues: Record<string, any>
): Record<string, any> => {
  return { ...formData, ...formValues };
};

/**
 * 重置表单数据
 */
export const resetFormData = (): Record<string, any> => {
  return {};
};
