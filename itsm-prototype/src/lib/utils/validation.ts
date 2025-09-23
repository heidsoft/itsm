// 验证结果接口
export interface ValidationResult {
  isValid: boolean;
  message?: string;
  errors?: string[];
}

// 验证规则接口
export interface ValidationRule {
  required?: boolean;
  minLength?: number;
  maxLength?: number;
  pattern?: RegExp;
  min?: number;
  max?: number;
  custom?: (value: unknown) => ValidationResult;
}

// 字段验证配置
export interface FieldValidation {
  [fieldName: string]: ValidationRule[];
}

// 表单验证配置
export interface FormValidationConfig {
  fields: FieldValidation;
  validateOnChange?: boolean;
  validateOnBlur?: boolean;
  stopOnFirstError?: boolean;
}

// 验证器类
export class Validator {
  // 必填验证
  static required(value: unknown): ValidationResult {
    const isEmpty = value === null || 
                   value === undefined || 
                   (typeof value === 'string' && value.trim() === '') ||
                   (Array.isArray(value) && value.length === 0);
    
    return {
      isValid: !isEmpty,
      message: isEmpty ? '此字段为必填项' : undefined
    };
  }

  // 长度验证
  static validateLength(value: unknown, min?: number, max?: number): ValidationResult {
    if (value === null || value === undefined) {
      return { isValid: true };
    }

    const length = typeof value === 'string' ? value.length : 
                  Array.isArray(value) ? value.length : 0;

    if (min !== undefined && length < min) {
      return {
        isValid: false,
        message: `长度不能少于${min}个字符`
      };
    }

    if (max !== undefined && length > max) {
      return {
        isValid: false,
        message: `长度不能超过${max}个字符`
      };
    }

    return { isValid: true };
  }

  // 正则表达式验证
  static pattern(value: unknown, regex: RegExp, message?: string): ValidationResult {
    if (value === null || value === undefined || value === '') {
      return { isValid: true };
    }

    const stringValue = String(value);
    const isValid = regex.test(stringValue);

    return {
      isValid,
      message: isValid ? undefined : (message || '格式不正确')
    };
  }

  // 数值范围验证
  static range(value: unknown, min?: number, max?: number): ValidationResult {
    if (value === null || value === undefined || value === '') {
      return { isValid: true };
    }

    const numValue = Number(value);
    if (isNaN(numValue)) {
      return {
        isValid: false,
        message: '必须是有效数字'
      };
    }

    if (min !== undefined && numValue < min) {
      return {
        isValid: false,
        message: `数值不能小于${min}`
      };
    }

    if (max !== undefined && numValue > max) {
      return {
        isValid: false,
        message: `数值不能大于${max}`
      };
    }

    return { isValid: true };
  }

  // 邮箱验证
  static email(value: unknown): ValidationResult {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return Validator.pattern(value, emailRegex, '请输入有效的邮箱地址');
  }

  // 手机号验证
  static phone(value: unknown): ValidationResult {
    const phoneRegex = /^1[3-9]\d{9}$/;
    return Validator.pattern(value, phoneRegex, '请输入有效的手机号码');
  }

  // URL验证
  static url(value: unknown): ValidationResult {
    if (value === null || value === undefined || value === '') {
      return { isValid: true };
    }

    try {
      new URL(String(value));
      return { isValid: true };
    } catch {
      return {
        isValid: false,
        message: '请输入有效的URL地址'
      };
    }
  }

  // 身份证验证
  static idCard(value: unknown): ValidationResult {
    const idCardRegex = /^[1-9]\d{5}(18|19|20)\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$/;
    return Validator.pattern(value, idCardRegex, '请输入有效的身份证号码');
  }
}

// 表单验证器类
export class FormValidator {
  private config: FormValidationConfig;
  private errors: Record<string, string[]> = {};

  constructor(config: FormValidationConfig) {
    this.config = config;
  }

  // 验证单个字段
  validateField(fieldName: string, value: unknown): ValidationResult {
    const rules = this.config.fields[fieldName];
    if (!rules) {
      return { isValid: true };
    }

    const errors: string[] = [];

    for (const rule of rules) {
      let result: ValidationResult;

      // 必填验证
      if (rule.required) {
        result = Validator.required(value);
        if (!result.isValid && result.message) {
          errors.push(result.message);
          if (this.config.stopOnFirstError) break;
        }
      }

      // 如果值为空且不是必填，跳过其他验证
      if ((value === null || value === undefined || value === '') && !rule.required) {
        continue;
      }

      // 长度验证
       if (rule.minLength !== undefined || rule.maxLength !== undefined) {
         result = Validator.validateLength(value, rule.minLength, rule.maxLength);
         if (!result.isValid && result.message) {
           errors.push(result.message);
           if (this.config.stopOnFirstError) break;
         }
       }

      // 正则验证
      if (rule.pattern) {
        result = Validator.pattern(value, rule.pattern);
        if (!result.isValid && result.message) {
          errors.push(result.message);
          if (this.config.stopOnFirstError) break;
        }
      }

      // 数值范围验证
      if (rule.min !== undefined || rule.max !== undefined) {
        result = Validator.range(value, rule.min, rule.max);
        if (!result.isValid && result.message) {
          errors.push(result.message);
          if (this.config.stopOnFirstError) break;
        }
      }

      // 自定义验证
      if (rule.custom) {
        result = rule.custom(value);
        if (!result.isValid && result.message) {
          errors.push(result.message);
          if (this.config.stopOnFirstError) break;
        }
      }
    }

    // 更新错误状态
    if (errors.length > 0) {
      this.errors[fieldName] = errors;
    } else {
      delete this.errors[fieldName];
    }

    return {
      isValid: errors.length === 0,
      errors: errors.length > 0 ? errors : undefined
    };
  }

  // 验证整个表单
  validateForm(data: Record<string, unknown>): ValidationResult {
    this.errors = {};
    const allErrors: string[] = [];

    for (const fieldName in this.config.fields) {
      const result = this.validateField(fieldName, data[fieldName]);
      if (!result.isValid && result.errors) {
        allErrors.push(...result.errors);
      }
    }

    return {
      isValid: allErrors.length === 0,
      errors: allErrors.length > 0 ? allErrors : undefined
    };
  }

  // 获取字段错误
  getFieldErrors(fieldName: string): string[] {
    return this.errors[fieldName] || [];
  }

  // 获取所有错误
  getAllErrors(): Record<string, string[]> {
    return { ...this.errors };
  }

  // 清除错误
  clearErrors(fieldName?: string): void {
    if (fieldName) {
      delete this.errors[fieldName];
    } else {
      this.errors = {};
    }
  }

  // 检查是否有错误
  hasErrors(fieldName?: string): boolean {
    if (fieldName) {
      return this.errors[fieldName]?.length > 0;
    }
    return Object.keys(this.errors).length > 0;
  }
}

// 预定义验证规则
export const ValidationRules = {
  // 用户名规则
  username: [
    { required: true },
    { minLength: 3, maxLength: 20 },
    { pattern: /^[a-zA-Z0-9_]+$/ }
  ],

  // 密码规则
  password: [
    { required: true },
    { minLength: 8, maxLength: 50 },
    { pattern: /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/ }
  ],

  // 邮箱规则
  email: [
    { required: true },
    { custom: Validator.email }
  ],

  // 手机号规则
  phone: [
    { required: true },
    { custom: Validator.phone }
  ],

  // 工单标题规则
  ticketTitle: [
    { required: true },
    { minLength: 5, maxLength: 100 }
  ],

  // 工单描述规则
  ticketDescription: [
    { required: true },
    { minLength: 10, maxLength: 2000 }
  ]
};

// 异步验证器接口
export interface AsyncValidator {
  validate: (value: unknown) => Promise<ValidationResult>;
  debounceMs?: number;
}

// 异步验证器类
export class AsyncFormValidator {
  private validators: Record<string, AsyncValidator> = {};
  private debounceTimers: Record<string, NodeJS.Timeout> = {};

  // 注册异步验证器
  registerValidator(fieldName: string, validator: AsyncValidator): void {
    this.validators[fieldName] = validator;
  }

  // 异步验证字段
  async validateFieldAsync(fieldName: string, value: unknown): Promise<ValidationResult> {
    const validator = this.validators[fieldName];
    if (!validator) {
      return { isValid: true };
    }

    // 清除之前的定时器
    if (this.debounceTimers[fieldName]) {
      clearTimeout(this.debounceTimers[fieldName]);
    }

    // 防抖处理
    return new Promise((resolve) => {
      const debounceMs = validator.debounceMs || 300;
      
      this.debounceTimers[fieldName] = setTimeout(async () => {
        try {
          const result = await validator.validate(value);
          resolve(result);
        } catch {
           resolve({
             isValid: false,
             message: '验证过程中发生错误'
           });
         }
      }, debounceMs);
    });
  }

  // 清理定时器
  cleanup(): void {
    Object.values(this.debounceTimers).forEach(timer => {
      clearTimeout(timer);
    });
    this.debounceTimers = {};
  }
}

// 常用异步验证器
export const AsyncValidators = {
  // 用户名唯一性验证
  uniqueUsername: {
    validate: async (value: unknown): Promise<ValidationResult> => {
      if (!value || typeof value !== 'string') {
        return { isValid: true };
      }

      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));
      
      // 这里应该调用实际的API
      const isUnique = !['admin', 'root', 'test'].includes(value.toLowerCase());
      
      return {
        isValid: isUnique,
        message: isUnique ? undefined : '用户名已存在'
      };
    },
    debounceMs: 500
  },

  // 邮箱唯一性验证
  uniqueEmail: {
    validate: async (value: unknown): Promise<ValidationResult> => {
      if (!value || typeof value !== 'string') {
        return { isValid: true };
      }

      // 先进行格式验证
      const formatResult = Validator.email(value);
      if (!formatResult.isValid) {
        return formatResult;
      }

      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));
      
      // 这里应该调用实际的API
      const isUnique = !value.includes('test@');
      
      return {
        isValid: isUnique,
        message: isUnique ? undefined : '邮箱已被注册'
      };
    },
    debounceMs: 500
  }
};

// 验证工具函数
export const ValidationUtils = {
  // 创建表单验证器
  createFormValidator: (fields: FieldValidation): FormValidator => {
    return new FormValidator({
      fields,
      validateOnChange: true,
      validateOnBlur: true,
      stopOnFirstError: false
    });
  },

  // 验证单个值
  validateValue: (value: unknown, rules: ValidationRule[]): ValidationResult => {
    const validator = new FormValidator({
      fields: { temp: rules }
    });
    return validator.validateField('temp', value);
  },

  // 合并验证结果
  mergeValidationResults: (...results: ValidationResult[]): ValidationResult => {
    const allErrors: string[] = [];
    let isValid = true;

    for (const result of results) {
      if (!result.isValid) {
        isValid = false;
        if (result.message) {
          allErrors.push(result.message);
        }
        if (result.errors) {
          allErrors.push(...result.errors);
        }
      }
    }

    return {
      isValid,
      errors: allErrors.length > 0 ? allErrors : undefined
    };
  }
};