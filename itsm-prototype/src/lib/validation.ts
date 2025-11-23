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

export class Validator<T extends Record<string, unknown>> {

  private rules: FieldValidation<T> = {};



  // 添加验证规则

  addRule<K extends keyof T>(fieldName: K, rule: ValidationRule): this {

    if (!this.rules[fieldName]) {

      this.rules[fieldName] = [];

    }

    this.rules[fieldName].push(rule);

    return this;

  }



  // 添加多个验证规则

  addRules<K extends keyof T>(fieldName: K, rules: ValidationRule[]): this {

    rules.forEach(rule => this.addRule(fieldName, rule));

    return this;

  }



  // 验证数据

  validate(data: T): ValidationResult {

    const errors: Record<string, string[]> = {};

    let isValid = true;



    Object.keys(this.rules).forEach(fieldName => {

      const fieldRules = this.rules[fieldName as keyof T];

      const fieldValue = data[fieldName as keyof T];

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

  validateField<K extends keyof T>(fieldName: K, value: T[K]): { isValid: boolean; errors: string[] } {

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

  removeFieldRules<K extends keyof T>(fieldName: K): this {

    delete this.rules[fieldName];

    return this;

  }

}



...



// 表单验证Hook

import { useState, useCallback } from 'react';



export const useFormValidation = <T extends Record<string, unknown>>(initialData: T) => {

  const [data, setData] = useState<T>(initialData);

  const [errors, setErrors] = useState<Record<string, string[]>>({});

  const [isValidating, setIsValidating] = useState(false);



  const validator = new Validator<T>();



  // 添加验证规则

  const addValidationRule = useCallback(<K extends keyof T>(fieldName: K, rule: ValidationRule) => {

    validator.addRule(fieldName, rule);

  }, [validator]);



  // 添加多个验证规则

  const addValidationRules = useCallback(<K extends keyof T>(fieldName: K, rules: ValidationRule[]) => {

    validator.addRules(fieldName, rules);

  }, [validator]);



  // 更新字段值

  const updateField = useCallback(<K extends keyof T>(fieldName: K, value: T[K]) => {

    setData(prev => ({ ...prev, [fieldName]: value }));

    

    // 清除该字段的错误

    setErrors(prev => {

      const newErrors = { ...prev };

      delete newErrors[fieldName as string];

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

  const validateField = useCallback((fieldName: keyof T): boolean => {

    const result = validator.validateField(fieldName, data[fieldName]);

    

    setErrors(prev => ({

      ...prev,

      [fieldName as string]: result.errors,

    }));

    

    return result.isValid;

  }, [data, validator]);



  // 重置表单

  const reset = useCallback(() => {

    setData(initialData);

    setErrors({});

  }, [initialData]);



  // 获取字段错误

  const getFieldError = useCallback((fieldName: keyof T): string | undefined => {

    const fieldErrors = errors[fieldName as string];

    return fieldErrors && fieldErrors.length > 0 ? fieldErrors[0] : undefined;

  }, [errors]);



  // 检查字段是否有错误

  const hasFieldError = useCallback((fieldName: keyof T): boolean => {

    return !!(errors[fieldName as string] && errors[fieldName as string].length > 0);

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



...



// 字段验证配置

export interface FieldValidation<T> {

  [fieldName: keyof T]: ValidationRule[];

}
