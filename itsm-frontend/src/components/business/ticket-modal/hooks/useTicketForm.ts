/**
 * useTicketForm Hook
 * 管理分步表单的状态和逻辑
 */

import { useState, useCallback, useMemo } from 'react';
import type { FormInstance } from 'antd/es/form';
import type { TicketFormValues, UseTicketFormReturn, UseTicketFormConfig, User, TicketTemplate } from '../types';
import { TICKET_FORM_STEPS } from '../utils/ticket-form-utils';
import { submitTicket, updateTicket } from '../services/ticket-modal-service';

export const useTicketForm = ({
  form,
  onSubmit,
  loading = false,
  isEditing = false,
}: UseTicketFormConfig): UseTicketFormReturn => {
  const [currentStep, setCurrentStep] = useState(0);
  const [formData, setFormData] = useState<Record<string, any>>({});

  // 模拟用户列表（实际应从 API 获取）
  const userList: User[] = useMemo(
    () => [
      { id: 1, name: 'Alice', role: 'IT Support', avatar: 'A' },
      { id: 2, name: 'Bob', role: 'Network Admin', avatar: 'B' },
      { id: 3, name: 'Charlie', role: 'Service Desk', avatar: 'C' },
    ],
    []
  );

  // 模拟模板数据（实际应从 API 获取）
  const ticketTemplates: TicketTemplate[] = useMemo(
    () => [
      {
        id: 1,
        name: 'System Login Issue',
        type: 'incident' as any,
        category: 'System Access',
        priority: 'medium' as any,
        description: 'User unable to login to system, technical support needed',
        estimatedTime: '2 hours',
        sla: '4 hours',
      },
      {
        id: 2,
        name: 'Printer Malfunction',
        type: 'incident' as any,
        category: 'Hardware Equipment',
        priority: 'high' as any,
        description: 'Office printer not working properly',
        estimatedTime: '1 hour',
        sla: '2 hours',
      },
      {
        id: 3,
        name: 'Software Installation Request',
        type: 'service_request' as any,
        category: 'Software Services',
        priority: 'low' as any,
        description: 'Need to install new office software',
        estimatedTime: '30 minutes',
        sla: '4 hours',
      },
    ],
    []
  );

  /**
   * 获取当前步骤需要验证的字段
   */
  const getFieldsForStep = useCallback((step: number): string[] => {
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
  }, []);

  /**
   * 下一步
   */
  const handleNext = useCallback(async () => {
    try {
      const fieldsToValidate = getFieldsForStep(currentStep);
      await form.validateFields(fieldsToValidate);

      const values = form.getFieldsValue();
      setFormData(prev => ({ ...prev, ...values }));

      if (currentStep < TICKET_FORM_STEPS.length - 1) {
        setCurrentStep(prev => prev + 1);
      }
    } catch (error) {
      // 使用 message 提示验证失败
      console.error('Form validation failed:', error);
    }
  }, [currentStep, form, getFieldsForStep]);

  /**
   * 上一步
   */
  const handlePrev = useCallback(() => {
    if (currentStep > 0) {
      setCurrentStep(prev => prev - 1);
    }
  }, [currentStep]);

  /**
   * 提交表单
   */
  const handleSubmit = useCallback(async () => {
    try {
      const values = { ...formData, ...form.getFieldsValue() };

      // 调用 API
      if (isEditing) {
        await updateTicket(values.id as number, values);
      } else {
        await submitTicket(values);
      }

      // 调用外部 onSubmit
      await onSubmit(values);

      // 重置表单
      form.resetFields();
      setFormData({});
      setCurrentStep(0);
    } catch (error) {
      console.error('Submit failed:', error);
      throw error;
    }
  }, [formData, form, isEditing, onSubmit]);

  /**
   * 重置表单
   */
  const resetForm = useCallback(() => {
    form.resetFields();
    setFormData({});
    setCurrentStep(0);
  }, [form]);

  /**
   * 渲染步骤内容 - 简化的占位实现，将由独立组件替换
   */
  const renderStepContent = useCallback(() => {
    // TODO: 这里将被独立的步骤组件替换
    return null;
  }, []);

  return {
    currentStep,
    formData,
    steps: TICKET_FORM_STEPS,
    userList,
    ticketTemplates,
    handleNext,
    handlePrev,
    handleSubmit,
    resetForm,
    renderStepContent,
    getFieldsForStep,
  };
};
