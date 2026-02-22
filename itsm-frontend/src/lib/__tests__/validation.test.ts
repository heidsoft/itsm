/**
 * Validation 工具测试
 */

import { validators, Validator, ValidationRule } from '../validation';

describe('validators', () => {
  describe('required', () => {
    it('should return false for null', () => {
      expect(validators.required(null)).toBe(false);
    });

    it('should return false for undefined', () => {
      expect(validators.required(undefined)).toBe(false);
    });

    it('should return false for empty string', () => {
      expect(validators.required('')).toBe(false);
    });

    it('should return false for whitespace-only string', () => {
      expect(validators.required('   ')).toBe(false);
    });

    it('should return true for non-empty string', () => {
      expect(validators.required('test')).toBe(true);
    });

    it('should return true for number', () => {
      expect(validators.required(0)).toBe(true);
    });

    it('should return false for empty array', () => {
      expect(validators.required([])).toBe(false);
    });

    it('should return true for non-empty array', () => {
      expect(validators.required([1, 2, 3])).toBe(true);
    });

    it('should return true for object', () => {
      expect(validators.required({ key: 'value' })).toBe(true);
    });
  });

  describe('email', () => {
    it('should return true for valid email', () => {
      expect(validators.email('user@example.com')).toBe(true);
    });

    it('should return true for email with subdomain', () => {
      expect(validators.email('user@mail.example.com')).toBe(true);
    });

    it('should return false for email without @', () => {
      expect(validators.email('userexample.com')).toBe(false);
    });

    it('should return false for email without domain', () => {
      expect(validators.email('user@')).toBe(false);
    });

    it('should return false for email without username', () => {
      expect(validators.email('@example.com')).toBe(false);
    });

    it('should return false for email with spaces', () => {
      expect(validators.email('user @example.com')).toBe(false);
    });
  });

  describe('phone', () => {
    it('should return true for valid China mobile numbers', () => {
      expect(validators.phone('13800138000')).toBe(true);
      expect(validators.phone('13912345678')).toBe(true);
      expect(validators.phone('18798765432')).toBe(true);
    });

    it('should return false for invalid phone numbers', () => {
      expect(validators.phone('12800138000')).toBe(false);
      expect(validators.phone('1380013800')).toBe(false);
      expect(validators.phone('138001380000')).toBe(false);
      expect(validators.phone('abcdefghijk')).toBe(false);
    });
  });

  describe('minLength', () => {
    it('should return true for string at minimum length', () => {
      const minLength3 = validators.minLength(3);
      expect(minLength3('abc')).toBe(true);
    });

    it('should return true for string longer than minimum', () => {
      const minLength3 = validators.minLength(3);
      expect(minLength3('abcd')).toBe(true);
    });

    it('should return false for string shorter than minimum', () => {
      const minLength3 = validators.minLength(3);
      expect(minLength3('ab')).toBe(false);
    });
  });

  describe('maxLength', () => {
    it('should return true for string at maximum length', () => {
      const maxLength5 = validators.maxLength(5);
      expect(maxLength5('abcde')).toBe(true);
    });

    it('should return true for string shorter than maximum', () => {
      const maxLength5 = validators.maxLength(5);
      expect(maxLength5('abc')).toBe(true);
    });

    it('should return false for string longer than maximum', () => {
      const maxLength5 = validators.maxLength(5);
      expect(maxLength5('abcdef')).toBe(false);
    });
  });

  describe('minValue', () => {
    it('should return true for number at minimum value', () => {
      const min10 = validators.minValue(10);
      expect(min10(10)).toBe(true);
    });

    it('should return true for number greater than minimum', () => {
      const min10 = validators.minValue(10);
      expect(min10(15)).toBe(true);
    });

    it('should return false for number less than minimum', () => {
      const min10 = validators.minValue(10);
      expect(min10(5)).toBe(false);
    });
  });

  describe('maxValue', () => {
    it('should return true for number at maximum value', () => {
      const max100 = validators.maxValue(100);
      expect(max100(100)).toBe(true);
    });

    it('should return true for number less than maximum', () => {
      const max100 = validators.maxValue(100);
      expect(max100(50)).toBe(true);
    });

    it('should return false for number greater than maximum', () => {
      const max100 = validators.maxValue(100);
      expect(max100(150)).toBe(false);
    });
  });

  describe('url', () => {
    it('should return true for valid URL', () => {
      expect(validators.url('https://example.com')).toBe(true);
      expect(validators.url('http://example.com/path')).toBe(true);
      expect(validators.url('https://example.com/path?query=value')).toBe(true);
    });

    it('should return false for invalid URL', () => {
      expect(validators.url('not-a-url')).toBe(false);
    });

    it('should return true for ftp URL', () => {
      expect(validators.url('ftp://example.com')).toBe(true);
    });
  });

  describe('number', () => {
    it('should return true for numeric string', () => {
      expect(validators.number('123')).toBe(true);
      expect(validators.number('123.45')).toBe(true);
      expect(validators.number('-123')).toBe(true);
    });

    it('should return false for non-numeric string', () => {
      expect(validators.number('abc')).toBe(false);
      expect(validators.number('12abc')).toBe(false);
    });

    it('should return true for empty string (Number("") === 0)', () => {
      expect(validators.number('')).toBe(true);
    });
  });

  describe('integer', () => {
    it('should return true for integer string', () => {
      expect(validators.integer('123')).toBe(true);
      expect(validators.integer('-123')).toBe(true);
      expect(validators.integer('0')).toBe(true);
    });

    it('should return false for decimal string', () => {
      expect(validators.integer('123.45')).toBe(false);
    });
  });

  describe('positive', () => {
    it('should return true for positive numbers', () => {
      expect(validators.positive(1)).toBe(true);
      expect(validators.positive(100)).toBe(true);
    });

    it('should return false for zero', () => {
      expect(validators.positive(0)).toBe(false);
    });

    it('should return false for negative numbers', () => {
      expect(validators.positive(-1)).toBe(false);
    });
  });

  describe('nonNegative', () => {
    it('should return true for positive numbers', () => {
      expect(validators.nonNegative(1)).toBe(true);
    });

    it('should return true for zero', () => {
      expect(validators.nonNegative(0)).toBe(true);
    });

    it('should return false for negative numbers', () => {
      expect(validators.nonNegative(-1)).toBe(false);
    });
  });

  describe('date', () => {
    it('should return true for valid date string', () => {
      expect(validators.date('2024-01-15')).toBe(true);
      expect(validators.date('2024-01-15T10:30:00')).toBe(true);
      expect(validators.date('2024/01/15')).toBe(true);
    });

    it('should return false for invalid date string', () => {
      expect(validators.date('invalid-date')).toBe(false);
      expect(validators.date('')).toBe(false);
    });
  });

  describe('oneOf', () => {
    it('should return true for value in options', () => {
      const isActive = validators.oneOf(['active', 'pending', 'closed']);
      expect(isActive('active')).toBe(true);
      expect(isActive('pending')).toBe(true);
    });

    it('should return false for value not in options', () => {
      const isActive = validators.oneOf(['active', 'pending', 'closed']);
      expect(isActive('deleted')).toBe(false);
    });
  });
});

describe('Validator class', () => {
  interface TestForm extends Record<string, unknown> {
    username: string;
    email: string;
    age: number;
  }

  it('should validate valid data', () => {
    const validator = new Validator<TestForm>();

    validator.addRule('username', {
      validator: validators.required as (value: unknown) => boolean,
      message: 'Username is required',
    });

    validator.addRule('email', {
      validator: validators.email as (value: unknown) => boolean,
      message: 'Invalid email',
    });

    const result = validator.validate({
      username: 'testuser',
      email: 'test@example.com',
      age: 25,
    });

    expect(result.isValid).toBe(true);
    expect(Object.keys(result.errors)).toHaveLength(0);
  });

  it('should return errors for invalid data', () => {
    const validator = new Validator<TestForm>();

    validator.addRule('username', {
      validator: validators.required as (value: unknown) => boolean,
      message: 'Username is required',
    });

    validator.addRule('email', {
      validator: validators.email as (value: unknown) => boolean,
      message: 'Invalid email',
    });

    const result = validator.validate({
      username: '',
      email: 'invalid-email',
      age: 25,
    });

    expect(result.isValid).toBe(false);
    expect(result.errors.username).toContain('Username is required');
    expect(result.errors.email).toContain('Invalid email');
  });

  it('should support multiple rules per field', () => {
    const validator = new Validator<TestForm>();

    validator.addRules('username', [
      { validator: validators.required as (value: unknown) => boolean, message: 'Required' },
      { validator: validators.minLength(3) as (value: unknown) => boolean, message: 'Too short' },
    ]);

    const invalidResult = validator.validate({
      username: 'ab',
      email: 'test@example.com',
      age: 25,
    });

    expect(invalidResult.isValid).toBe(false);
    expect(invalidResult.errors.username).toContain('Too short');

    const validResult = validator.validate({
      username: 'abc',
      email: 'test@example.com',
      age: 25,
    });

    expect(validResult.isValid).toBe(true);
  });

  it('should validate single field', () => {
    const validator = new Validator<TestForm>();

    validator.addRule('email', {
      validator: validators.email as (value: unknown) => boolean,
      message: 'Invalid email',
    });

    const result = validator.validateField('email', 'invalid');

    expect(result.isValid).toBe(false);
    expect(result.errors).toContain('Invalid email');
  });

  it('should clear all rules', () => {
    const validator = new Validator<TestForm>();

    validator.addRule('username', {
      validator: validators.required as (value: unknown) => boolean,
      message: 'Required',
    });

    validator.clearRules();

    const result = validator.validate({
      username: '',
      email: '',
      age: 0,
    });

    expect(result.isValid).toBe(true);
  });

  it('should remove specific field rules', () => {
    const validator = new Validator<TestForm>();

    validator.addRule('username', {
      validator: validators.required as (value: unknown) => boolean,
      message: 'Required',
    });

    validator.addRule('email', {
      validator: validators.email as (value: unknown) => boolean,
      message: 'Invalid email',
    });

    validator.removeFieldRules('username');

    const result = validator.validate({
      username: '',
      email: 'invalid',
      age: 25,
    });

    expect(result.isValid).toBe(false);
    expect(result.errors.username).toBeUndefined();
    expect(result.errors.email).toContain('Invalid email');
  });
});
