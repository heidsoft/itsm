/**
 * useTicketForm Hook
 * 管理分步表单的状态和逻辑
 */

import { useState, useCallback } from 'react';
import type { UseTicketFormReturn, UseTicketFormConfig } from '../types';
import { TICKET_FORM_STEPS } from '../utils/ticket-form-utils';
import { useTicketModalData } from './useTicketModalData';

export const useTicketForm = ({
  form,
  onSubmit,
  loading: _loading = false,
  isEditing: _isEditing = false,
}: UseTicketFormConfig): UseTicketFormReturn => {
  const [currentStep, setCurrentStep] = useState(0);
  const [formData, setFormData] = useState<Record<string, any>>({});

  const { users: userList, templates: ticketTemplates } = useTicketModalData();

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

      // Persistence belongs to the feature container/caller. This hook only
      // coordinates form state and awaits the controlled submit callback.
      await onSubmit(values);

      // 重置表单
      form.resetFields();
      setFormData({});
      setCurrentStep(0);
    } catch (error) {
      console.error('Submit failed:', error);
      throw error;
    }
  }, [formData, form, onSubmit]);

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
