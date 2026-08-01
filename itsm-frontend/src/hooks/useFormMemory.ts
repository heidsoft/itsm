import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { FormInstance } from 'antd';
import { useWatch } from 'antd/es/form/Form';
import dayjs from 'dayjs';

interface UseFormMemoryOptions {
  /** 存储的键名，必须唯一 */
  storageKey: string;
  /** 是否自动保存表单值，默认 true */
  autoSave?: boolean;
  /** 是否启用草稿恢复，默认 true。编辑模式应设为 false */
  enabled?: boolean;
  /** 哪些字段需要排除，不保存 */
  excludeFields?: string[];
  /** 表单提交成功后是否清除存储，默认 true */
  clearOnSubmit?: boolean;
  /** 自动保存防抖延迟（毫秒），默认 500ms */
  debounceMs?: number;
}

// 需要特殊处理的日期字段名模式
const DATE_FIELD_PATTERNS = [
  /date/i,
  /time/i,
  /At$/,
  /Date$/,
  /Range$/,
  /Deadline$/,
  /Start$/,
  /End$/,
];

/**
 * 判断字段是否是日期类型
 */
function isDateField(fieldName: string): boolean {
  return DATE_FIELD_PATTERNS.some(pattern => pattern.test(fieldName));
}

/**
 * 序列化表单值，将 dayjs 对象转为 ISO 字符串
 */
function serializeFormValues(values: Record<string, any>): Record<string, any> {
  const result: Record<string, any> = {};
  for (const [key, value] of Object.entries(values)) {
    if (value && typeof value === 'object' && dayjs.isDayjs(value)) {
      result[key] = value.toISOString();
    } else if (Array.isArray(value) && value.length === 2 && dayjs.isDayjs(value[0])) {
      // 处理 RangePicker [dayjs, dayjs]
      result[key] = [value[0].toISOString(), value[1]?.toISOString()];
    } else {
      result[key] = value;
    }
  }
  return result;
}

/**
 * 反序列化表单值，将 ISO 字符串转回 dayjs 对象
 */
function deserializeFormValues(values: Record<string, any>): Record<string, any> {
  const result: Record<string, any> = {};
  for (const [key, value] of Object.entries(values)) {
    if (isDateField(key)) {
      if (typeof value === 'string') {
        result[key] = dayjs(value);
      } else if (Array.isArray(value) && value.length === 2) {
        result[key] = [dayjs(value[0]), dayjs(value[1])];
      } else {
        result[key] = value;
      }
    } else {
      result[key] = value;
    }
  }
  return result;
}

/**
 * 表单记忆 Hook
 * 自动保存表单输入值到 localStorage，页面刷新后自动恢复
 *
 * 修复：
 * - H-3: 添加防抖，避免每次按键都同步写 localStorage
 * - H-4: 添加 enabled 参数，编辑模式下禁用草稿恢复
 * - H-5: 正确处理 dayjs 对象的序列化/反序列化
 */
export function useFormMemory<T = any>(
  form: FormInstance<T>,
  options: UseFormMemoryOptions
) {
  const {
    storageKey,
    autoSave = true,
    enabled = true,
    excludeFields = [],
    clearOnSubmit = true,
    debounceMs = 500,
  } = options;

  const [savedValues, setSavedValues] = useState<Partial<T> | null>(null);
  const debounceTimerRef = useRef<NodeJS.Timeout | null>(null);

  // 将 excludeFields 转换为稳定的字符串引用
  const excludeFieldsKey = useMemo(() => JSON.stringify(excludeFields), [excludeFields]);

  // 加载已保存的表单值（H-4: 仅在 enabled 为 true 时恢复）
  useEffect(() => {
    if (!enabled) return;

    try {
      const stored = localStorage.getItem(storageKey);
      if (stored) {
        const parsed = deserializeFormValues(JSON.parse(stored)) as Partial<T>;
        setSavedValues(parsed);
        form.setFieldsValue(parsed as any);
      }
    } catch (e) {
      console.error('Failed to load saved form values:', e);
    }
  }, [form, storageKey, enabled]);

  // 自动保存表单值（H-3: 添加防抖）
  const allValues = useWatch([], form);

  useEffect(() => {
    if (!autoSave || !allValues || !enabled) return;

    // 清除之前的防抖定时器
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    // 设置新的防抖定时器
    debounceTimerRef.current = setTimeout(() => {
      try {
        const filteredValues = { ...allValues } as Record<string, any>;
        const parsedExcludeFields = JSON.parse(excludeFieldsKey);
        parsedExcludeFields.forEach((field: string) => {
          delete filteredValues[field];
        });

        // H-5: 序列化时正确处理 dayjs 对象
        const serialized = serializeFormValues(filteredValues);
        localStorage.setItem(storageKey, JSON.stringify(serialized));
      } catch (e) {
        console.error('Failed to save form values:', e);
      }
    }, debounceMs);

    // 清理定时器
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, [allValues, autoSave, enabled, excludeFieldsKey, storageKey, debounceMs]);

  // 组件卸载时清理定时器
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  // 清除保存的表单值
  const clearSavedValues = useCallback(() => {
    try {
      localStorage.removeItem(storageKey);
      setSavedValues(null);
      form.resetFields();
    } catch (e) {
      console.error('Failed to clear saved form values:', e);
    }
  }, [form, storageKey]);

  // 表单提交成功后调用，清除存储
  const handleSubmitSuccess = useCallback(() => {
    if (clearOnSubmit) {
      clearSavedValues();
    }
  }, [clearOnSubmit, clearSavedValues]);

  return {
    savedValues,
    clearSavedValues,
    handleSubmitSuccess,
  };
}
