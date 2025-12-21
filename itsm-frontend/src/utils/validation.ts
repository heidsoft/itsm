/**
 * 表单验证工具函数
 * Form Validation Utilities
 */

// 邮箱验证
export const validateEmail = (email: string): boolean => {
  const emailRegex = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/;
  return emailRegex.test(email);
};

// 手机号验证（中国）
export const validatePhone = (phone: string): boolean => {
  const phoneRegex = /^1[3-9]\d{9}$/;
  return phoneRegex.test(phone);
};

// 身份证验证（中国）
export const validateIdCard = (idCard: string): boolean => {
  const idCardRegex = /(^\d{15}$)|(^\d{18}$)|(^\d{17}(\d|X|x)$)/;
  return idCardRegex.test(idCard);
};

// URL验证
export const validateURL = (url: string): boolean => {
  try {
    new URL(url);
    return true;
  } catch {
    return false;
  }
};

// IP地址验证
export const validateIP = (ip: string): boolean => {
  const ipRegex = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
  return ipRegex.test(ip);
};

// 密码强度验证
export const validatePasswordStrength = (password: string): {
  isValid: boolean;
  strength: 'weak' | 'medium' | 'strong';
  message: string;
} => {
  if (password.length < 6) {
    return { isValid: false, strength: 'weak', message: '密码长度至少6位' };
  }

  let strength = 0;
  
  // 检查长度
  if (password.length >= 8) strength += 1;
  if (password.length >= 12) strength += 1;
  
  // 检查包含小写字母
  if (/[a-z]/.test(password)) strength += 1;
  
  // 检查包含大写字母
  if (/[A-Z]/.test(password)) strength += 1;
  
  // 检查包含数字
  if (/[0-9]/.test(password)) strength += 1;
  
  // 检查包含特殊字符
  if (/[^a-zA-Z0-9]/.test(password)) strength += 1;

  if (strength <= 2) {
    return { isValid: true, strength: 'weak', message: '密码强度：弱' };
  } else if (strength <= 4) {
    return { isValid: true, strength: 'medium', message: '密码强度：中' };
  } else {
    return { isValid: true, strength: 'strong', message: '密码强度：强' };
  }
};

// 必填验证
export const validateRequired = (value: any, fieldName: string = '此字段'): string | undefined => {
  if (value === null || value === undefined || value === '' || (Array.isArray(value) && value.length === 0)) {
    return `${fieldName}不能为空`;
  }
  return undefined;
};

// 长度验证
export const validateLength = (
  value: string,
  min?: number,
  max?: number,
  fieldName: string = '此字段'
): string | undefined => {
  if (min !== undefined && value.length < min) {
    return `${fieldName}长度不能小于${min}个字符`;
  }
  if (max !== undefined && value.length > max) {
    return `${fieldName}长度不能大于${max}个字符`;
  }
  return undefined;
};

// 数值范围验证
export const validateRange = (
  value: number,
  min?: number,
  max?: number,
  fieldName: string = '此字段'
): string | undefined => {
  if (min !== undefined && value < min) {
    return `${fieldName}不能小于${min}`;
  }
  if (max !== undefined && value > max) {
    return `${fieldName}不能大于${max}`;
  }
  return undefined;
};

// 自定义正则验证
export const validatePattern = (
  value: string,
  pattern: RegExp,
  message: string = '格式不正确'
): string | undefined => {
  if (!pattern.test(value)) {
    return message;
  }
  return undefined;
};

// 表单验证规则构造器
export const createValidator = (rules: {
  required?: boolean;
  email?: boolean;
  phone?: boolean;
  url?: boolean;
  minLength?: number;
  maxLength?: number;
  min?: number;
  max?: number;
  pattern?: { regex: RegExp; message: string };
  custom?: (value: any) => string | undefined;
  fieldName?: string;
}) => {
  return (value: any): string | undefined => {
    const fieldName = rules.fieldName || '此字段';

    // 必填验证
    if (rules.required) {
      const error = validateRequired(value, fieldName);
      if (error) return error;
    }

    // 如果值为空且非必填，跳过其他验证
    if (!value && !rules.required) return undefined;

    // 邮箱验证
    if (rules.email && !validateEmail(value)) {
      return `${fieldName}格式不正确`;
    }

    // 手机号验证
    if (rules.phone && !validatePhone(value)) {
      return `${fieldName}格式不正确`;
    }

    // URL验证
    if (rules.url && !validateURL(value)) {
      return `${fieldName}格式不正确`;
    }

    // 长度验证
    if (typeof value === 'string') {
      const lengthError = validateLength(value, rules.minLength, rules.maxLength, fieldName);
      if (lengthError) return lengthError;
    }

    // 数值范围验证
    if (typeof value === 'number') {
      const rangeError = validateRange(value, rules.min, rules.max, fieldName);
      if (rangeError) return rangeError;
    }

    // 正则验证
    if (rules.pattern && typeof value === 'string') {
      const patternError = validatePattern(value, rules.pattern.regex, rules.pattern.message);
      if (patternError) return patternError;
    }

    // 自定义验证
    if (rules.custom) {
      const customError = rules.custom(value);
      if (customError) return customError;
    }

    return undefined;
  };
};

// Ant Design Form 规则适配器
export const toAntdRules = (validator: (value: any) => string | undefined) => {
  return [
    {
      validator: async (_: any, value: any) => {
        const error = validator(value);
        if (error) {
          return Promise.reject(new Error(error));
        }
        return Promise.resolve();
      },
    },
  ];
};

// 批量验证
export const validateFields = (
  data: Record<string, any>,
  validators: Record<string, (value: any) => string | undefined>
): { isValid: boolean; errors: Record<string, string> } => {
  const errors: Record<string, string> = {};
  let isValid = true;

  Object.keys(validators).forEach(key => {
    const error = validators[key](data[key]);
    if (error) {
      errors[key] = error;
      isValid = false;
    }
  });

  return { isValid, errors };
};

