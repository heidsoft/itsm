/**
 * Tests for validation.ts
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

import { validators, Validator } from '../validation';

describe('Validation Utilities', () => {
  describe('validators.required', () => {
    it('should return false for null/undefined', () => {
      expect(validators.required(null)).toBe(false);
      expect(validators.required(undefined)).toBe(false);
    });

    it('should return false for empty string', () => {
      expect(validators.required('')).toBe(false);
      expect(validators.required('   ')).toBe(false);
    });

    it('should return false for empty array', () => {
      expect(validators.required([])).toBe(false);
    });

    it('should return true for non-empty values', () => {
      expect(validators.required('hello')).toBe(true);
      expect(validators.required([1])).toBe(true);
      expect(validators.required(0)).toBe(true);
      expect(validators.required(false)).toBe(true);
    });
  });

  describe('validators.email', () => {
    it('should validate correct emails', () => {
      expect(validators.email('test@example.com')).toBe(true);
      expect(validators.email('user+tag@domain.org')).toBe(true);
    });

    it('should reject invalid emails', () => {
      expect(validators.email('notanemail')).toBe(false);
      expect(validators.email('@domain.com')).toBe(false);
      expect(validators.email('user@')).toBe(false);
    });
  });

  describe('validators.phone', () => {
    it('should validate Chinese phone numbers', () => {
      expect(validators.phone('13800138000')).toBe(true);
      expect(validators.phone('19912345678')).toBe(true);
    });

    it('should reject invalid phones', () => {
      expect(validators.phone('12345')).toBe(false);
      expect(validators.phone('02812345678')).toBe(false);
    });
  });

  describe('validators.minLength', () => {
    it('should validate minimum length', () => {
      const validate = validators.minLength(3);
      expect(validate('abc')).toBe(true);
      expect(validate('ab')).toBe(false);
    });
  });

  describe('validators.maxLength', () => {
    it('should validate maximum length', () => {
      const validate = validators.maxLength(5);
      expect(validate('abc')).toBe(true);
      expect(validate('abcdef')).toBe(false);
    });
  });

  describe('validators.minValue', () => {
    it('should validate minimum value', () => {
      const validate = validators.minValue(10);
      expect(validate(10)).toBe(true);
      expect(validate(9)).toBe(false);
    });
  });

  describe('validators.maxValue', () => {
    it('should validate maximum value', () => {
      const validate = validators.maxValue(100);
      expect(validate(100)).toBe(true);
      expect(validate(101)).toBe(false);
    });
  });

  describe('validators.url', () => {
    it('should validate URLs', () => {
      expect(validators.url('https://example.com')).toBe(true);
      expect(validators.url('http://localhost:3000')).toBe(true);
    });

    it('should reject invalid URLs', () => {
      expect(validators.url('not-a-url')).toBe(false);
    });
  });

  describe('validators.number', () => {
    it('should validate numbers', () => {
      expect(validators.number('123')).toBe(true);
      expect(validators.number('12.5')).toBe(true);
      expect(validators.number('-10')).toBe(true);
    });

    it('should reject non-numbers', () => {
      expect(validators.number('abc')).toBe(false);
      expect(validators.number('Infinity')).toBe(false);
    });
  });

  describe('validators.integer', () => {
    it('should validate integers', () => {
      expect(validators.integer('123')).toBe(true);
      expect(validators.integer('-10')).toBe(true);
    });

    it('should reject non-integers', () => {
      expect(validators.integer('12.5')).toBe(false);
    });
  });

  describe('validators.positive', () => {
    it('should validate positive numbers', () => {
      expect(validators.positive(1)).toBe(true);
      expect(validators.positive(0)).toBe(false);
      expect(validators.positive(-1)).toBe(false);
    });
  });

  describe('validators.nonNegative', () => {
    it('should validate non-negative numbers', () => {
      expect(validators.nonNegative(0)).toBe(true);
      expect(validators.nonNegative(1)).toBe(true);
      expect(validators.nonNegative(-1)).toBe(false);
    });
  });

  describe('validators.date', () => {
    it('should validate dates', () => {
      expect(validators.date('2024-01-01')).toBe(true);
      expect(validators.date('2024-12-31T23:59:59Z')).toBe(true);
    });

    it('should reject invalid dates', () => {
      expect(validators.date('not-a-date')).toBe(false);
    });
  });

  describe('validators.pattern', () => {
    it('should validate against regex', () => {
      const validate = validators.pattern(/^[A-Z]+$/);
      expect(validate('ABC')).toBe(true);
      expect(validate('abc')).toBe(false);
    });
  });

  describe('validators.oneOf', () => {
    it('should validate enum values', () => {
      const validate = validators.oneOf(['a', 'b', 'c']);
      expect(validate('a')).toBe(true);
      expect(validate('d')).toBe(false);
    });
  });

  describe('Validator class', () => {
    it('should add and validate rules', () => {
      const validator = new Validator<{ name: string; email: string }>();
      validator.addRule('name', { validator: (v) => validators.required(v), message: 'Name required' });
      validator.addRule('email', { validator: (v) => validators.email(v as string), message: 'Invalid email' });

      const result = validator.validate({ name: '', email: 'bad' } as any);
      expect(result.isValid).toBe(false);
      expect(result.errors['name']).toContain('Name required');
      expect(result.errors['email']).toContain('Invalid email');
    });

    it('should validate successfully', () => {
      const validator = new Validator<{ name: string }>();
      validator.addRule('name', { validator: (v) => validators.required(v), message: 'Required' });

      const result = validator.validate({ name: 'John' } as any);
      expect(result.isValid).toBe(true);
      expect(Object.keys(result.errors)).toHaveLength(0);
    });

    it('should validate single field', () => {
      const validator = new Validator<{ name: string }>();
      validator.addRule('name', { validator: (v) => validators.required(v), message: 'Required' });

      const result = validator.validateField('name', '' as any);
      expect(result.isValid).toBe(false);
      expect(result.errors).toContain('Required');
    });

    it('should add multiple rules at once', () => {
      const validator = new Validator<{ name: string }>();
      validator.addRules('name', [
        { validator: (v) => validators.required(v), message: 'Required' },
        { validator: (v) => validators.minLength(2)(v as string), message: 'Min 2' },
      ]);

      const result = validator.validate({ name: 'A' } as any);
      expect(result.errors['name']).toContain('Min 2');
    });

    it('should support method chaining', () => {
      const validator = new Validator<{ name: string }>();
      const result = validator
        .addRule('name', { validator: (v) => validators.required(v), message: 'Required' })
        .addRules('name', [{ validator: (v) => validators.minLength(2)(v as string), message: 'Min 2' }]);
      
      expect(result).toBe(validator);
    });

    it('should clear all rules', () => {
      const validator = new Validator<{ name: string }>();
      validator.addRule('name', { validator: (v) => validators.required(v), message: 'Required' });
      validator.clearRules();

      const result = validator.validate({ name: '' } as any);
      expect(result.isValid).toBe(true);
    });

    it('should remove field rules', () => {
      const validator = new Validator<{ name: string; email: string }>();
      validator.addRule('name', { validator: (v) => validators.required(v), message: 'Required' });
      validator.addRule('email', { validator: (v) => validators.email(v as string), message: 'Invalid' });
      validator.removeFieldRules('name');

      const result = validator.validate({ name: '', email: 'bad' } as any);
      expect(result.errors).not.toHaveProperty('name');
      expect(result.errors).toHaveProperty('email');
    });
  });
});
