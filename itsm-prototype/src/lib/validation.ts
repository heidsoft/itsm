// 数据验证工具库

// 基础验证函数
export const validators = {
  // 必填验证
  required: (value: unknown): boolean => {
    if (value === null || value === undefined) return false;
    if (typeof value === 'string') return value.trim().length > 0;
    if (Array.isArray(value)) return value.length > 0;
    return true;
  },

  // 邮箱验证
  email: (value: string): boolean => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(value);
  },

  // 手机号验证（中国大陆）
  phone: (value: string): boolean => {
    const phoneRegex = /^1[3-9]\d{9}$/;
    return phoneRegex.test(value);
  },

  // 长度验证
  minLength: (min: number) => (value: string): boolean => {
    return value.length >= min;
  },

  maxLength: (max: number) => (value: string): boolean => {
    return value.length <= max;
  },

  // 数字范围验证
  minValue: (min: number) => (value: number): boolean => {
    return value >= min;
  },

  maxValue: (max: number) => (value: number): boolean => {
    return value <= max;
  },

  // URL验证
  url: (value: string): boolean => {
    try {
      new URL(value);
      return true;
    } catch {
      return false;
    }
  },

  // 数字验证
  number: (value: string): boolean => {
    return !isNaN(Number(value)) && isFinite(Number(value));
  },

  // 整数验证
  integer: (value: string): boolean => {
    return Number.isInteger(Number(value));
  },

  // 正数验证
  positive: (value: number): boolean => {
    return value > 0;
  },

  // 非负数验证
  nonNegative: (value: number): boolean => {
    return value >= 0;
  },

  // 日期验证
  date: (value: string): boolean => {
    const date = new Date(value);
    return !isNaN(date.getTime());
  },

  // 正则表达式验证
  pattern: (regex: RegExp) => (value: string): boolean => {
    return regex.test(value);
  },

  // 枚举值验证
  oneOf: <T>(options: T[]) => (value: T): boolean => {
    return options.includes(value);
  },
};

// 验证规则接口
export interface ValidationRule {
  validator: (value: unknown) => boolean;
  message: string;
}

// 字段验证配置
export interface FieldValidation {
  [fieldName: string]: ValidationRule[];
}

// 验证结果接口
export interface ValidationResult {
  isValid: boolean;
  errors: Record<string, string[]>;
}

// 验证器类
export class Validator {
  private rules: FieldValidation = {};

  // 添加验证规则
  addRule(fieldName: string, rule: ValidationRule): this {
    if (!this.rules[fieldName]) {
      this.rules[fieldName] = [];
    }
    this.rules[fieldName].push(rule);
    return this;
  }

  // 添加多个验证规则
  addRules(fieldName: string, rules: ValidationRule[]): this {
    rules.forEach(rule => this.addRule(fieldName, rule));
    return this;
  }

  // 验证数据
  validate(data: Record<string, unknown>): ValidationResult {
    const errors: Record<string, string[]> = {};
    let isValid = true;

    Object.keys(this.rules).forEach(fieldName => {
      const fieldRules = this.rules[fieldName];
      const fieldValue = data[fieldName];
      const fieldErrors: string[] = [];

      fieldRules.forEach(rule => {
        if (!rule.validator(fieldValue)) {
          fieldErrors.push(rule.message);
          isValid = false;
        }
      });

      if (fieldErrors.length > 0) {
        errors[fieldName] = fieldErrors;
      }
    });

    return { isValid, errors };
  }

  // 验证单个字段
  validateField(fieldName: string, value: unknown): { isValid: boolean; errors: string[] } {
    const fieldRules = this.rules[fieldName] || [];
    const errors: string[] = [];
    let isValid = true;

    fieldRules.forEach(rule => {
      if (!rule.validator(value)) {
        errors.push(rule.message);
        isValid = false;
      }
    });

    return { isValid, errors };
  }

  // 清除规则
  clearRules(): this {
    this.rules = {};
    return this;
  }

  // 移除特定字段的规则
  removeFieldRules(fieldName: string): this {
    delete this.rules[fieldName];
    return this;
  }
}

// 常用验证规则构建器
export const createValidationRules = {
  // 必填字段
  required: (message = '此字段为必填项'): ValidationRule => ({
    validator: validators.required,
    message,
  }),

  // 邮箱字段
  email: (message = '请输入有效的邮箱地址'): ValidationRule => ({
    validator: (value: unknown) => {
      if (typeof value !== 'string') return false;
      return validators.email(value);
    },
    message,
  }),

  // 手机号字段
  phone: (message = '请输入有效的手机号码'): ValidationRule => ({
    validator: (value: unknown) => {
      if (typeof value !== 'string') return false;
      return validators.phone(value);
    },
    message,
  }),

  // 长度限制
  length: (min: number, max: number, message?: string): ValidationRule => ({
    validator: (value: unknown) => {
      if (typeof value !== 'string') return false;
      return value.length >= min && value.length <= max;
    },
    message: message || `长度必须在 ${min} 到 ${max} 个字符之间`,
  }),

  // 最小长度
  minLength: (min: number, message?: string): ValidationRule => ({
    validator: (value: unknown) => {
      if (typeof value !== 'string') return false;
      return value.length >= min;
    },
    message: message || `最少需要 ${min} 个字符`,
  }),

  // 最大长度
  maxLength: (max: number, message?: string): ValidationRule => ({
    validator: (value: unknown) => {
      if (typeof value !== 'string') return false;
      return value.length <= max;
    },
    message: message || `最多允许 ${max} 个字符`,
  }),

  // 数值范围
  range: (min: number, max: number, message?: string): ValidationRule => ({
    validator: (value: unknown) => {
      const num = Number(value);
      return !isNaN(num) && num >= min && num <= max;
    },
    message: message || `数值必须在 ${min} 到 ${max} 之间`,
  }),

  // 正数
  positive: (message = '必须是正数'): ValidationRule => ({
    validator: (value: unknown) => {
      const num = Number(value);
      return !isNaN(num) && validators.positive(num);
    },
    message,
  }),

  // 整数
  integer: (message = '必须是整数'): ValidationRule => ({
    validator: (value: unknown) => {
      return validators.integer(String(value));
    },
    message,
  }),

  // 枚举值
  oneOf: <T>(options: T[], message?: string): ValidationRule => ({
    validator: (value: unknown) => validators.oneOf(options)(value as T),
    message: message || `必须是以下值之一: ${options.join(', ')}`,
  }),

  // 自定义正则
  pattern: (regex: RegExp, message: string): ValidationRule => ({
    validator: (value: unknown) => {
      if (typeof value !== 'string') return false;
      return validators.pattern(regex)(value);
    },
    message,
  }),
};

// 数据清理工具
export const sanitizers = {
  // 去除首尾空格
  trim: (value: string): string => value.trim(),

  // 转换为小写
  toLowerCase: (value: string): string => value.toLowerCase(),

  // 转换为大写
  toUpperCase: (value: string): string => value.toUpperCase(),

  // 移除HTML标签
  stripHtml: (value: string): string => {
    return value.replace(/<[^>]*>/g, '');
  },

  // 转义HTML字符
  escapeHtml: (value: string): string => {
    const div = document.createElement('div');
    div.textContent = value;
    return div.innerHTML;
  },

  // 移除特殊字符（保留字母、数字、中文）
  removeSpecialChars: (value: string): string => {
    return value.replace(/[^\w\u4e00-\u9fa5]/g, '');
  },

  // 格式化手机号
  formatPhone: (value: string): string => {
    const cleaned = value.replace(/\D/g, '');
    if (cleaned.length === 11) {
      return cleaned.replace(/(\d{3})(\d{4})(\d{4})/, '$1-$2-$3');
    }
    return value;
  },

  // 格式化金额
  formatCurrency: (value: number, currency = '¥'): string => {
    return `${currency}${value.toLocaleString('zh-CN', { minimumFractionDigits: 2 })}`;
  },
};

// 表单验证Hook
import { useState, useCallback } from 'react';

export const useFormValidation = (initialData: Record<string, unknown> = {}) => {
  const [data, setData] = useState(initialData);
  const [errors, setErrors] = useState<Record<string, string[]>>({});
  const [isValidating, setIsValidating] = useState(false);

  const validator = new Validator();

  // 添加验证规则
  const addValidationRule = useCallback((fieldName: string, rule: ValidationRule) => {
    validator.addRule(fieldName, rule);
  }, [validator]);

  // 添加多个验证规则
  const addValidationRules = useCallback((fieldName: string, rules: ValidationRule[]) => {
    validator.addRules(fieldName, rules);
  }, [validator]);

  // 更新字段值
  const updateField = useCallback((fieldName: string, value: unknown) => {
    setData(prev => ({ ...prev, [fieldName]: value }));
    
    // 清除该字段的错误
    setErrors(prev => {
      const newErrors = { ...prev };
      delete newErrors[fieldName];
      return newErrors;
    });
  }, []);

  // 验证表单
  const validate = useCallback(async (): Promise<boolean> => {
    setIsValidating(true);
    
    try {
      const result = validator.validate(data);
      setErrors(result.errors);
      return result.isValid;
    } finally {
      setIsValidating(false);
    }
  }, [data, validator]);

  // 验证单个字段
  const validateField = useCallback((fieldName: string): boolean => {
    const result = validator.validateField(fieldName, data[fieldName]);
    
    setErrors(prev => ({
      ...prev,
      [fieldName]: result.errors,
    }));
    
    return result.isValid;
  }, [data, validator]);

  // 重置表单
  const reset = useCallback(() => {
    setData(initialData);
    setErrors({});
  }, [initialData]);

  // 获取字段错误
  const getFieldError = useCallback((fieldName: string): string | undefined => {
    const fieldErrors = errors[fieldName];
    return fieldErrors && fieldErrors.length > 0 ? fieldErrors[0] : undefined;
  }, [errors]);

  // 检查字段是否有错误
  const hasFieldError = useCallback((fieldName: string): boolean => {
    return !!(errors[fieldName] && errors[fieldName].length > 0);
  }, [errors]);

  return {
    data,
    errors,
    isValidating,
    updateField,
    validate,
    validateField,
    reset,
    getFieldError,
    hasFieldError,
    addValidationRule,
    addValidationRules,
  };
};