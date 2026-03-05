/**
 * validation.ts - 验证工具类测试
 */

import { Validator, ValidationUtils } from '../validation';

describe('Validator class', () => {
  describe('required', () => {
    it('应该拒绝空字符串', () => {
      const result = Validator.required('');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('此字段为必填项');
    });

    it('应该拒绝 null 和 undefined', () => {
      expect(Validator.required(null).isValid).toBe(false);
      expect(Validator.required(undefined).isValid).toBe(false);
    });

    it('应该接受非空字符串', () => {
      const result = Validator.required('test');
      expect(result.isValid).toBe(true);
    });

    it('应该拒绝空数组', () => {
      expect(Validator.required([]).isValid).toBe(false);
    });

    it('应该接受非空数组', () => {
      expect(Validator.required(['item']).isValid).toBe(true);
    });

    it('应该拒绝只包含空格的字符串', () => {
      expect(Validator.required('   ').isValid).toBe(false);
    });
  });

  describe('validateLength', () => {
    it('应该验证最小长度', () => {
      const result = Validator.validateLength('hello', 5);
      expect(result.isValid).toBe(true);
    });

    it('应该拒绝太短的字符串', () => {
      const result = Validator.validateLength('hi', 5);
      expect(result.isValid).toBe(false);
      expect(result.message).toContain('至少');
    });

    it('应该验证最大长度', () => {
      const result = Validator.validateLength('hello', 3, 10);
      expect(result.isValid).toBe(true);
    });

    it('应该拒绝太长的字符串', () => {
      const result = Validator.validateLength('verylongstring', 0, 5);
      expect(result.isValid).toBe(false);
      expect(result.message).toContain('最多');
    });
  });

  describe('email', () => {
    it('应该验证有效的邮箱', () => {
      const result = Validator.email('test@example.com');
      expect(result.isValid).toBe(true);
    });

    it('应该拒绝无效的邮箱', () => {
      expect(Validator.email('invalid').isValid).toBe(false);
      expect(Validator.email('missing@domain').isValid).toBe(false);
      expect(Validator.email('@nodomain.com').isValid).toBe(false);
    });
  });

  describe('phone', () => {
    it('应该验证有效的电话号码', () => {
      expect(Validator.phone('+86-13800138000').isValid).toBe(true);
      expect(Validator.phone('13800138000').isValid).toBe(true);
    });

    it('应该拒绝无效的电话号码', () => {
      expect(Validator.phone('123').isValid).toBe(false);
      expect(Validator.phone('abcdef').isValid).toBe(false);
    });
  });

  describe('url', () => {
    it('应该验证有效的 URL', () => {
      expect(Validator.url('https://example.com').isValid).toBe(true);
      expect(Validator.url('http://localhost:3000').isValid).toBe(true);
    });

    it('应该拒绝无效的 URL', () => {
      expect(Validator.url('not-a-url').isValid).toBe(false);
    });
  });

  describe('date', () => {
    it('应该验证有效的日期', () => {
      expect(Validator.date('2024-01-15').isValid).toBe(true);
      expect(Validator.date('2024-01-15T10:00:00Z').isValid).toBe(true);
    });

    it('应该拒绝无效的日期', () => {
      expect(Validator.date('invalid').isValid).toBe(false);
    });
  });

  describe('pattern', () => {
    it('应该根据正则验证', () => {
      const alphaPattern = /^[a-zA-Z]+$/;
      expect(Validator.pattern('abc', alphaPattern).isValid).toBe(true);
      expect(Validator.pattern('123', alphaPattern).isValid).toBe(false);
    });
  });

  describe('range', () => {
    it('应该验证数值范围', () => {
      expect(Validator.range(5, 1, 10).isValid).toBe(true);
      expect(Validator.range(0, 1, 10).isValid).toBe(false);
      expect(Validator.range(11, 1, 10).isValid).toBe(false);
    });

    it('应该包括边界值', () => {
      expect(Validator.range(1, 1, 10).isValid).toBe(true);
      expect(Validator.range(10, 1, 10).isValid).toBe(true);
    });
  });

  describe('custom', () => {
    it('应该支持自定义验证', () => {
      const result = Validator.custom((value) => {
        return {
          isValid: value === 'valid',
          message: value === 'valid' ? undefined : '必须是 valid',
        };
      }, 'test');

      expect(result.isValid).toBe(false);
      expect(result.message).toBe('必须是 valid');
    });
  });

  describe('sanitize', () => {
    it('应该清理 HTML 标签', () => {
      const result = Validator.sanitize('<script>alert("xss")</script>');
      expect(result).toBe('');
    });

    it('应该清理危险内容', () => {
      const result = Validator.sanitize('Hello <b>World</b>');
      expect(result).toBe('Hello World');
    });
  });
});

describe('ValidationUtils', () => {
  describe('validateValue', () => {
    it('应该验证单个值', () => {
      const rules = [
        { required: true },
        { minLength: 5 },
      ];
      const result = ValidationUtils.validateValue('hello', rules);
      expect(result.isValid).toBe(true);
    });

    it('应该返回所有验证错误', () => {
      const rules = [
        { required: true },
        { minLength: 10 },
      ];
      const result = ValidationUtils.validateValue('hi', rules);
      expect(result.isValid).toBe(false);
      expect(result.errors).toHaveLength(2);
    });
  });

  describe('mergeValidationResults', () => {
    it('应该合并多个验证结果', () => {
      const result = ValidationUtils.mergeValidationResults(
        { isValid: false, message: '错误1' },
        { isValid: false, message: '错误2' }
      );
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('错误1');
      expect(result.errors).toContain('错误2');
    });

    it('应该合并所有错误', () => {
      const result = ValidationUtils.mergeValidationResults(
        { isValid: false, errors: ['e1', 'e2'] },
        { isValid: false, errors: ['e3'] }
      );
      expect(result.errors).toHaveLength(3);
    });

    it('全部通过应返回 true', () => {
      const result = ValidationUtils.mergeValidationResults(
        { isValid: true },
        { isValid: true }
      );
      expect(result.isValid).toBe(true);
    });
  });

  describe('createFormValidator', () => {
    it('应该创建表单验证器', () => {
      const validator = ValidationUtils.createFormValidator({
        fields: {
          email: [{ required: true }, { email: true }],
        },
      });

      const result = validator.validate({ email: 'test@example.com' });
      expect(result.email?.isValid).toBe(true);
    });

    it('应该验证整个表单', () => {
      const validator = ValidationUtils.createFormValidator({
        fields: {
          title: [{ required: true }, { minLength: 5 }],
        },
      });

      const result = validator.validate({ title: 'Hello World' });
      expect(result.title?.isValid).toBe(true);
    });

    it('应该返回字段级错误', () => {
      const validator = ValidationUtils.createFormValidator({
        fields: {
          title: [{ required: true }, { minLength: 10 }],
        },
      });

      const result = validator.validate({ title: 'Short' });
      expect(result.title?.isValid).toBe(false);
      expect(result.title?.errors).toContain('长度至少 10 个字符');
    });
  });
});

describe('Validator 链式调用', () => {
  it('应该支持链式验证', () => {
    const result = Validator.chain('hello')
      .required()
      .minLength(5)
      .maxLength(10)
      .result();

    expect(result.isValid).toBe(true);
  });

  it('链式验证应累积错误', () => {
    const result = Validator.chain('hi')
      .required()
      .minLength(5)
      .result();

    expect(result.isValid).toBe(false);
    expect(result.errors).toHaveLength(1);
  });

  it('应该支持自定义规则', () => {
    const result = Validator.chain('test')
      .custom((val) => val.includes('t'))
      .result();

    expect(result.isValid).toBe(true);
  });
});

describe('异步验证', () => {
  it('应该支持异步验证规则', async () => {
    const asyncRule = async (value: unknown) => {
      await new Promise(resolve => setTimeout(resolve, 10));
      return {
        isValid: value === 'valid',
        message: '无效值',
      };
    };

    const result = await Validator.custom(asyncRule, 'valid');
    expect(result.isValid).toBe(true);
  });
});
