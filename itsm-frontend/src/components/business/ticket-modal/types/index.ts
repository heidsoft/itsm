/**
 * Ticket Modal 相关类型定义
 */

import type { FC, ReactNode } from 'react';
import type { Ticket, TicketStatus, TicketPriority, TicketType } from '@/lib/services/ticket-service';

// ============ Props 类型 ============

export interface TicketModalProps {
  visible: boolean;
  editingTicket: Ticket | null;
  onCancel: () => void;
  onSubmit: (values: TicketFormValues) => void;
  loading?: boolean;
}

export interface TicketTemplateModalProps {
  visible: boolean;
  onCancel: () => void;
}

export interface TicketFormProps {
  form: any; // Ant Design Form instance
  onSubmit: (values: TicketFormValues) => void;
  loading?: boolean;
  isEditing?: boolean;
}

// ============ 表单数据类型 ============

export interface TicketFormValues {
  title: string;
  description: string;
  type: TicketType;
  category: string;
  priority: TicketPriority;
  assignee_id?: number;
  estimated_time?: string; // 或使用 dayjs.Dayjs 类型
  attachments?: any[];
}

export interface TicketFormData {
  title?: string;
  description?: string;
  type?: TicketType;
  category?: string;
  priority?: TicketPriority;
  assignee_id?: number;
  estimated_time?: string;
}

// ============ 步骤配置类型 ============

export interface FormStep {
  title: string;
  description: string;
  content: string;
}

export interface StepConfig {
  steps: FormStep[];
  getFieldsForStep: (step: number) => string[];
}

// ============ 用户类型 ============

export interface User {
  id: number;
  name: string;
  role: string;
  avatar: string;
}

// ============ 工单模板类型 ============

export interface TicketTemplate {
  id: number;
  name: string;
  type: TicketType;
  category: string;
  priority: TicketPriority;
  description: string;
  estimatedTime: string;
  sla: string;
}

// ============ 工具函数返回类型 ============

export interface UseTicketFormReturn {
  currentStep: number;
  formData: TicketFormData;
  steps: FormStep[];
  userList: User[];
  ticketTemplates: TicketTemplate[];
  handleNext: () => Promise<void>;
  handlePrev: () => void;
  handleSubmit: () => Promise<void>;
  resetForm: () => void;
  renderStepContent: () => ReactNode;
  getFieldsForStep: (step: number) => string[];
}

export interface UseTicketFormConfig {
  form: any;
  onSubmit: (values: TicketFormValues) => void;
  loading?: boolean;
  isEditing?: boolean;
}
