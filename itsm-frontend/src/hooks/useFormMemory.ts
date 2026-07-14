import { useState, useEffect } from 'react';
import { FormInstance } from 'antd';
import { useWatch } from 'antd/es/form/Form';

interface UseFormMemoryOptions {
  /** 存储的键名，必须唯一 */
  storageKey: string;
  /** 是否自动保存表单值，默认 true */
  autoSave?: boolean;
  /** 哪些字段需要排除，不保存 */
  excludeFields?: string[];
  /** 表单提交成功后是否清除存储，默认 true */
  clearOnSubmit?: boolean;
}

/**
 * 表单记忆 Hook
 * 自动保存表单输入值到 localStorage，页面刷新后自动恢复
 */
export function useFormMemory<T = any>(
  form: FormInstance<T>,
  options: UseFormMemoryOptions
) {
  const {
    storageKey,
    autoSave = true,
    excludeFields = [],
    clearOnSubmit = true,
  } = options;

  const [savedValues, setSavedValues] = useState<Partial<T> | null>(null);

  // 加载已保存的表单值
  useEffect(() => {
    try {
      const stored = localStorage.getItem(storageKey);
      if (stored) {
        const parsed = JSON.parse(stored) as Partial<T>;
        setSavedValues(parsed);
        form.setFieldsValue(parsed as any);
      }
    } catch (e) {
      console.error('Failed to load saved form values:', e);
    }
  }, [form, storageKey]);

  // 自动保存表单值
  const allValues = useWatch([], form);

  useEffect(() => {
    if (!autoSave || !allValues) return;

    try {
      const filteredValues = { ...allValues } as Partial<T>;
      excludeFields.forEach((field) => {
        delete filteredValues[field as keyof T];
      });
      localStorage.setItem(storageKey, JSON.stringify(filteredValues));
    } catch (e) {
      console.error('Failed to save form values:', e);
    }
  }, [allValues, autoSave, excludeFields, storageKey]);

  // 清除保存的表单值
  const clearSavedValues = () => {
    try {
      localStorage.removeItem(storageKey);
      setSavedValues(null);
      form.resetFields();
    } catch (e) {
      console.error('Failed to clear saved form values:', e);
    }
  };

  // 表单提交成功后调用，清除存储
  const handleSubmitSuccess = () => {
    if (clearOnSubmit) {
      clearSavedValues();
    }
  };

  return {
    savedValues,
    clearSavedValues,
    handleSubmitSuccess,
  };
}
