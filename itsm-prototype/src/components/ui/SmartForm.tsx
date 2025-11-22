'use client';

import React, { useEffect, useState, useCallback } from 'react';
import { Form, FormProps, FormInstance, message, Alert } from 'antd';
import { cn } from '@/lib/utils';
import confetti from 'canvas-confetti';

export interface SmartFormProps extends FormProps {
  /**
   * 是否启用自动保存
   * @default false
   */
  autoSave?: boolean;
  
  /**
   * 自动保存的防抖延迟（毫秒）
   * @default 2000
   */
  autoSaveDelay?: number;
  
  /**
   * 自动保存的存储键
   */
  autoSaveKey?: string;
  
  /**
   * 是否在onChange时验证
   * @default true
   */
  validateOnChange?: boolean;
  
  /**
   * 是否显示提交成功动画
   * @default true
   */
  successAnimation?: boolean;
  
  /**
   * 是否显示必填字段提示
   * @default true
   */
  showRequiredHint?: boolean;
  
  /**
   * 自定义成功消息
   */
  successMessage?: string;
  
  /**
   * 自定义错误消息
   */
  errorMessage?: string;
  
  /**
   * 表单提交前的校验回调
   */
  onBeforeSubmit?: (values: any) => boolean | Promise<boolean>;
  
  /**
   * 自动保存回调
   */
  onAutoSave?: (values: any) => void;
}

/**
 * 智能表单组件
 * 
 * 特性：
 * - 自动保存草稿
 * - 实时验证
 * - 提交成功动画
 * - 错误聚合显示
 * - 必填字段提示
 * - 防抖处理
 * 
 * @example
 * ```tsx
 * <SmartForm
 *   autoSave
 *   autoSaveKey="ticket-form"
 *   onFinish={handleSubmit}
 *   validateOnChange
 * >
 *   <Form.Item name="title" label="标题" required>
 *     <Input />
 *   </Form.Item>
 * </SmartForm>
 * ```
 */
export const SmartForm: React.FC<SmartFormProps> = ({
  autoSave = false,
  autoSaveDelay = 2000,
  autoSaveKey = 'form-draft',
  validateOnChange = true,
  successAnimation = true,
  showRequiredHint = true,
  successMessage = '提交成功！',
  errorMessage = '提交失败，请检查表单内容后重试',
  onBeforeSubmit,
  onAutoSave,
  onFinish,
  onValuesChange,
  children,
  form: externalForm,
  ...props
}) => {
  const [internalForm] = Form.useForm();
  const form = externalForm || internalForm;
  
  const [submitting, setSubmitting] = useState(false);
  const [errors, setErrors] = useState<string[]>([]);
  const [saveTimer, setSaveTimer] = useState<NodeJS.Timeout>();
  const [lastSaved, setLastSaved] = useState<Date>();

  // 加载草稿
  useEffect(() => {
    if (autoSave && typeof window !== 'undefined') {
      try {
        const draft = localStorage.getItem(autoSaveKey);
        if (draft) {
          const parsedDraft = JSON.parse(draft);
          form.setFieldsValue(parsedDraft.values);
          setLastSaved(new Date(parsedDraft.timestamp));
        }
      } catch (error) {
        console.error('加载草稿失败:', error);
      }
    }
  }, [autoSave, autoSaveKey, form]);

  // 自动保存
  const saveFormValues = useCallback((values: any) => {
    if (!autoSave || typeof window === 'undefined') return;
    
    try {
      const draft = {
        values,
        timestamp: new Date().toISOString(),
      };
      localStorage.setItem(autoSaveKey, JSON.stringify(draft));
      setLastSaved(new Date());
      
      if (onAutoSave) {
        onAutoSave(values);
      }
      
      // 显示保存提示
      message.success({
        content: '草稿已自动保存',
        duration: 1,
        style: { marginTop: '10vh' },
      });
    } catch (error) {
      console.error('保存草稿失败:', error);
    }
  }, [autoSave, autoSaveKey, onAutoSave]);

  // 表单值变化处理
  const handleValuesChange = (changedValues: any, allValues: any) => {
    // 清除之前的定时器
    if (saveTimer) {
      clearTimeout(saveTimer);
    }

    // 设置新的定时器
    if (autoSave) {
      const timer = setTimeout(() => {
        saveFormValues(allValues);
      }, autoSaveDelay);
      setSaveTimer(timer);
    }

    // 调用外部回调
    if (onValuesChange) {
      onValuesChange(changedValues, allValues);
    }
  };

  // 表单提交处理
  const handleFinish = async (values: any) => {
    setErrors([]);
    setSubmitting(true);

    try {
      // 提交前校验
      if (onBeforeSubmit) {
        const canSubmit = await onBeforeSubmit(values);
        if (!canSubmit) {
          setSubmitting(false);
          return;
        }
      }

      // 执行提交
      await onFinish?.(values);

      // 成功动画
      if (successAnimation) {
        confetti({
          particleCount: 100,
          spread: 70,
          origin: { y: 0.6 },
        });
      }

      // 成功提示
      message.success(successMessage);

      // 清除草稿
      if (autoSave && typeof window !== 'undefined') {
        localStorage.removeItem(autoSaveKey);
        setLastSaved(undefined);
      }
    } catch (error: any) {
      console.error('表单提交失败:', error);
      message.error(errorMessage);
      
      // 如果错误对象包含验证信息，显示详细错误
      if (error.errors && Array.isArray(error.errors)) {
        setErrors(error.errors);
      }
    } finally {
      setSubmitting(false);
    }
  };

  // 表单提交失败处理
  const handleFinishFailed = (errorInfo: any) => {
    const errorMessages = errorInfo.errorFields.map(
      (field: any) => `${field.name.join('.')}: ${field.errors.join(', ')}`
    );
    setErrors(errorMessages);
    
    // 滚动到第一个错误字段
    if (errorInfo.errorFields.length > 0) {
      form.scrollToField(errorInfo.errorFields[0].name);
    }
    
    message.error('请检查表单内容');
  };

  // 清除草稿
  const clearDraft = useCallback(() => {
    if (typeof window !== 'undefined') {
      localStorage.removeItem(autoSaveKey);
      setLastSaved(undefined);
      message.info('草稿已清除');
    }
  }, [autoSaveKey]);

  return (
    <div className="smart-form-wrapper">
      {/* 必填字段提示 */}
      {showRequiredHint && (
        <Alert
          message="带有 * 标记的字段为必填项"
          type="info"
          showIcon
          closable
          style={{ marginBottom: 16 }}
        />
      )}

      {/* 错误信息汇总 */}
      {errors.length > 0 && (
        <Alert
          message="表单验证失败"
          description={
            <ul className="mb-0 pl-4">
              {errors.map((error, index) => (
                <li key={index}>{error}</li>
              ))}
            </ul>
          }
          type="error"
          showIcon
          closable
          onClose={() => setErrors([])}
          style={{ marginBottom: 16 }}
        />
      )}

      {/* 草稿保存状态 */}
      {autoSave && lastSaved && (
        <div className="flex items-center justify-between mb-4 p-2 bg-blue-50 rounded-lg">
          <span className="text-sm text-blue-600">
            草稿已保存于 {lastSaved.toLocaleTimeString()}
          </span>
          <button
            type="button"
            onClick={clearDraft}
            className="text-sm text-blue-600 hover:text-blue-800 underline"
          >
            清除草稿
          </button>
        </div>
      )}

      <Form
        form={form}
        onFinish={handleFinish}
        onFinishFailed={handleFinishFailed}
        onValuesChange={handleValuesChange}
        validateTrigger={validateOnChange ? 'onChange' : 'onBlur'}
        scrollToFirstError
        layout="vertical"
        requiredMark="optional"
        {...props}
      >
        {typeof children === 'function' 
          ? children({ form, submitting }) 
          : children}
      </Form>
    </div>
  );
};

/**
 * 智能表单字段
 * 
 * 增强的Form.Item，包含更友好的提示和验证
 */
export interface SmartFormFieldProps {
  name: string;
  label: string;
  required?: boolean;
  tooltip?: string;
  helpText?: string;
  rules?: any[];
  children: React.ReactNode;
  validateDebounce?: number;
}

export const SmartFormField: React.FC<SmartFormFieldProps> = ({
  name,
  label,
  required = false,
  tooltip,
  helpText,
  rules = [],
  children,
  validateDebounce = 500,
}) => {
  const finalRules = [
    ...(required ? [{ required: true, message: `请输入${label}` }] : []),
    ...rules,
  ];

  return (
    <Form.Item
      name={name}
      label={
        <span className="flex items-center gap-1">
          {label}
          {required && <span className="text-red-500">*</span>}
          {tooltip && (
            <span
              className="text-gray-400 text-xs cursor-help"
              title={tooltip}
            >
              ⓘ
            </span>
          )}
        </span>
      }
      rules={finalRules}
      validateDebounce={validateDebounce}
      help={helpText}
      className="smart-form-field"
    >
      {children}
    </Form.Item>
  );
};

export default SmartForm;

