/**
 * Tests for src/lib/utils/validation.ts
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

jest.mock('dompurify', () => ({
  sanitize: jest.fn((html: string, _opts?: any) => html.replace(/<[^>]*>/g, '')),
}));

import {
  Validator,
  ChainValidator,
  FormValidator,
  ValidationRules,
  AsyncFormValidator,
  AsyncValidators,
  ValidationUtils,
} from '../validation';

describe('utils/validation - Validator class', () => {
  describe('Validator.required', () => {
    it('returns invalid for null/undefined/empty string/empty array', () => {
      expect(Validator.required(null).isValid).toBe(false);
      expect(Validator.required(undefined).isValid).toBe(false);
      expect(Validator.required('').isValid).toBe(false);
      expect(Validator.required('   ').isValid).toBe(false);
      expect(Validator.required([]).isValid).toBe(false);
    });

    it('returns valid for non-empty values', () => {
      expect(Validator.required('hello').isValid).toBe(true);
      expect(Validator.required([1]).isValid).toBe(true);
      expect(Validator.required(0).isValid).toBe(true);
    });
  });

  describe('Validator.validateLength', () => {
    it('returns valid for null/undefined', () => {
      expect(Validator.validateLength(null, 1, 10).isValid).toBe(true);
      expect(Validator.validateLength(undefined, 1, 10).isValid).toBe(true);
    });

    it('validates string min length', () => {
      expect(Validator.validateLength('ab', 3).isValid).toBe(false);
      expect(Validator.validateLength('abc', 3).isValid).toBe(true);
    });

    it('validates string max length', () => {
      expect(Validator.validateLength('abcde', undefined, 3).isValid).toBe(false);
      expect(Validator.validateLength('abc', undefined, 3).isValid).toBe(true);
    });

    it('validates array length', () => {
      expect(Validator.validateLength([1], 2).isValid).toBe(false);
      expect(Validator.validateLength([1, 2], 2).isValid).toBe(true);
    });
  });

  describe('Validator.pattern', () => {
    it('returns valid for null/undefined/empty', () => {
      expect(Validator.pattern(null, /abc/).isValid).toBe(true);
      expect(Validator.pattern('', /abc/).isValid).toBe(true);
    });

    it('validates against regex', () => {
      expect(Validator.pattern('abc123', /^[a-z]+\d+$/).isValid).toBe(true);
      expect(Validator.pattern('ABC', /^[a-z]+$/).isValid).toBe(false);
    });

    it('uses custom message', () => {
      const result = Validator.pattern('bad', /^good$/, 'Must be good');
      expect(result.message).toBe('Must be good');
    });
  });

  describe('Validator.range', () => {
    it('returns valid for empty values', () => {
      expect(Validator.range(null, 0, 10).isValid).toBe(true);
      expect(Validator.range('', 0, 10).isValid).toBe(true);
    });

    it('returns invalid for NaN', () => {
      expect(Validator.range('abc', 0, 10).isValid).toBe(false);
    });

    it('validates min', () => {
      expect(Validator.range(3, 5).isValid).toBe(false);
      expect(Validator.range(5, 5).isValid).toBe(true);
    });

    it('validates max', () => {
      expect(Validator.range(11, undefined, 10).isValid).toBe(false);
      expect(Validator.range(10, undefined, 10).isValid).toBe(true);
    });
  });

  describe('Validator.email', () => {
    it('validates emails', () => {
      expect(Validator.email('user@example.com').isValid).toBe(true);
      expect(Validator.email('invalid').isValid).toBe(false);
    });
  });

  describe('Validator.phone', () => {
    it('validates Chinese phone numbers', () => {
      expect(Validator.phone('13800138000').isValid).toBe(true);
      expect(Validator.phone('+86-13800138000').isValid).toBe(true);
      expect(Validator.phone('12345').isValid).toBe(false);
    });
  });

  describe('Validator.url', () => {
    it('validates URLs', () => {
      expect(Validator.url('https://example.com').isValid).toBe(true);
      expect(Validator.url('not-a-url').isValid).toBe(false);
      expect(Validator.url('').isValid).toBe(true);
    });
  });

  describe('Validator.idCard', () => {
    it('validates ID card format', () => {
      expect(Validator.idCard('110101199003071234').isValid).toBe(true);
      expect(Validator.idCard('123').isValid).toBe(false);
    });
  });

  describe('Validator.date', () => {
    it('validates dates', () => {
      expect(Validator.date('2024-01-01').isValid).toBe(true);
      expect(Validator.date('not-a-date').isValid).toBe(false);
      expect(Validator.date('').isValid).toBe(true);
    });
  });

  describe('Validator.custom', () => {
    it('runs custom validator', () => {
      const fn = (v: unknown) => ({ isValid: v === 'ok' });
      expect(Validator.custom(fn, 'ok').isValid).toBe(true);
      expect(Validator.custom(fn, 'bad').isValid).toBe(false);
    });

    it('handles validator errors', () => {
      const fn = () => { throw new Error('fail'); };
      expect(Validator.custom(fn, 'x').isValid).toBe(false);
    });
  });

  describe('Validator.sanitize', () => {
    it('sanitizes HTML', () => {
      expect(Validator.sanitize('<script>alert(1)</script>Hello')).toBe('alert(1)Hello');
      expect(Validator.sanitize(null)).toBe('');
      expect(Validator.sanitize(undefined)).toBe('');
    });
  });

  describe('Validator.chain / createChainValidator', () => {
    it('chains validations', () => {
      const result = Validator.chain('test@example.com').required().email().result();
      expect(result.isValid).toBe(true);
    });

    it('collects multiple errors', () => {
      const result = Validator.chain('').required().minLength(5).result();
      expect(result.isValid).toBe(false);
      expect(result.errors!.length).toBeGreaterThanOrEqual(1);
    });
  });

  describe('Validator.validate', () => {
    it('validates value against rules', () => {
      const result = Validator.validate('', [{ required: true }]);
      expect(result.isValid).toBe(false);
    });
  });
});

describe('ChainValidator', () => {
  it('validates maxLength', () => {
    const result = new ChainValidator('toolong').maxLength(3).result();
    expect(result.isValid).toBe(false);
  });

  it('validates phone', () => {
    const result = new ChainValidator('13800138000').phone().result();
    expect(result.isValid).toBe(true);
  });

  it('validates custom', () => {
    const result = new ChainValidator('hello').custom((v) => ({
      isValid: v === 'hello',
    })).result();
    expect(result.isValid).toBe(true);
  });
});

describe('FormValidator', () => {
  it('validates single field', () => {
    const validator = new FormValidator({
      fields: { name: [{ required: true }, { minLength: 3 }] },
    });
    expect(validator.validateField('name', '').isValid).toBe(false);
    expect(validator.validateField('name', 'abc').isValid).toBe(true);
  });

  it('validates unknown field returns valid', () => {
    const validator = new FormValidator({ fields: {} });
    expect(validator.validateField('unknown', 'x').isValid).toBe(true);
  });

  it('validates entire form', () => {
    const validator = new FormValidator({
      fields: {
        email: [{ required: true }],
        name: [{ required: true }, { minLength: 2 }],
      },
    });
    const result = validator.validateForm({ email: '', name: 'A' });
    expect(result.isValid).toBe(false);
    expect(result.errors!.length).toBeGreaterThan(0);
  });

  it('stops on first error when configured', () => {
    const validator = new FormValidator({
      fields: { name: [{ required: true }, { minLength: 5 }, { maxLength: 3 }] },
      stopOnFirstError: true,
    });
    const result = validator.validateField('name', '');
    expect(result.errors!.length).toBe(1);
  });

  it('skips non-required empty fields', () => {
    const validator = new FormValidator({
      fields: { opt: [{ minLength: 5 }] },
    });
    expect(validator.validateField('opt', '').isValid).toBe(true);
  });

  it('validates pattern rule', () => {
    const validator = new FormValidator({
      fields: { code: [{ pattern: /^[A-Z]+$/ }] },
    });
    expect(validator.validateField('code', 'abc').isValid).toBe(false);
    expect(validator.validateField('code', 'ABC').isValid).toBe(true);
  });

  it('validates numeric range', () => {
    const validator = new FormValidator({
      fields: { age: [{ min: 0, max: 120 }] },
    });
    expect(validator.validateField('age', -1).isValid).toBe(false);
    expect(validator.validateField('age', 130).isValid).toBe(false);
    expect(validator.validateField('age', 25).isValid).toBe(true);
  });

  it('validates custom rule', () => {
    const validator = new FormValidator({
      fields: { x: [{ custom: (v) => ({ isValid: v === 'yes', message: 'Must be yes' }) }] },
    });
    expect(validator.validateField('x', 'no').isValid).toBe(false);
  });

  it('getFieldErrors / getAllErrors / clearErrors / hasErrors', () => {
    const validator = new FormValidator({
      fields: { a: [{ required: true }] },
    });
    validator.validateField('a', '');
    expect(validator.getFieldErrors('a').length).toBeGreaterThan(0);
    expect(validator.hasErrors('a')).toBe(true);
    expect(validator.hasErrors()).toBe(true);
    expect(Object.keys(validator.getAllErrors()).length).toBeGreaterThan(0);

    validator.clearErrors('a');
    expect(validator.hasErrors('a')).toBe(false);

    validator.validateField('a', '');
    validator.clearErrors();
    expect(validator.hasErrors()).toBe(false);
  });
});

describe('ValidationRules presets', () => {
  it('has username rules', () => {
    expect(ValidationRules.username.length).toBeGreaterThan(0);
  });
  it('has password rules', () => {
    expect(ValidationRules.password.length).toBeGreaterThan(0);
  });
  it('has email rules', () => {
    expect(ValidationRules.email.length).toBeGreaterThan(0);
  });
  it('has ticketTitle rules', () => {
    expect(ValidationRules.ticketTitle.length).toBeGreaterThan(0);
  });
});

describe('AsyncFormValidator', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it('validates field asynchronously', async () => {
    const asyncValidator = new AsyncFormValidator();
    asyncValidator.registerValidator('username', {
      validate: async (value) => ({ isValid: value !== 'taken' }),
      debounceMs: 100,
    });

    const promise = asyncValidator.validateFieldAsync('username', 'free');
    jest.advanceTimersByTime(200);
    const result = await promise;
    expect(result.isValid).toBe(true);
  });

  it('returns valid for unregistered field', async () => {
    const asyncValidator = new AsyncFormValidator();
    const result = await asyncValidator.validateFieldAsync('unknown', 'val');
    expect(result.isValid).toBe(true);
  });

  it('cleanup clears timers', () => {
    const asyncValidator = new AsyncFormValidator();
    asyncValidator.registerValidator('x', {
      validate: async () => ({ isValid: true }),
      debounceMs: 1000,
    });
    asyncValidator.validateFieldAsync('x', 'val');
    asyncValidator.cleanup(); // should not throw
  });
});

describe('AsyncValidators presets', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => jest.useRealTimers());

  it('uniqueUsername rejects reserved names', async () => {
    const promise = AsyncValidators.uniqueUsername.validate('admin');
    jest.advanceTimersByTime(600);
    const result = await promise;
    expect(result.isValid).toBe(false);
  });

  it('uniqueUsername accepts unique names', async () => {
    const promise = AsyncValidators.uniqueUsername.validate('uniqueUser123');
    jest.advanceTimersByTime(600);
    const result = await promise;
    expect(result.isValid).toBe(true);
  });

  it('uniqueUsername returns valid for empty', async () => {
    const result = await AsyncValidators.uniqueUsername.validate('');
    expect(result.isValid).toBe(true);
  });

  it('uniqueEmail rejects test@ emails', async () => {
    const promise = AsyncValidators.uniqueEmail.validate('test@example.com');
    jest.advanceTimersByTime(600);
    const result = await promise;
    expect(result.isValid).toBe(false);
  });

  it('uniqueEmail accepts valid unique emails', async () => {
    const promise = AsyncValidators.uniqueEmail.validate('user@example.com');
    jest.advanceTimersByTime(600);
    const result = await promise;
    expect(result.isValid).toBe(true);
  });

  it('uniqueEmail returns valid for empty', async () => {
    const result = await AsyncValidators.uniqueEmail.validate('');
    expect(result.isValid).toBe(true);
  });

  it('uniqueEmail rejects invalid format', async () => {
    const promise = AsyncValidators.uniqueEmail.validate('notanemail');
    jest.advanceTimersByTime(600);
    const result = await promise;
    expect(result.isValid).toBe(false);
  });
});

describe('ValidationUtils', () => {
  it('createFormValidator creates a validator', () => {
    const v = ValidationUtils.createFormValidator({ name: [{ required: true }] });
    expect(v).toBeInstanceOf(FormValidator);
  });

  it('validateValue validates a single value', () => {
    const result = ValidationUtils.validateValue('', [{ required: true }]);
    expect(result.isValid).toBe(false);
  });

  it('mergeValidationResults merges results', () => {
    const r1 = { isValid: false, message: 'err1' };
    const r2 = { isValid: true };
    const r3 = { isValid: false, errors: ['err2', 'err3'] };
    const merged = ValidationUtils.mergeValidationResults(r1, r2, r3);
    expect(merged.isValid).toBe(false);
    expect(merged.errors).toContain('err1');
    expect(merged.errors).toContain('err2');
  });

  it('mergeValidationResults returns valid when all valid', () => {
    const merged = ValidationUtils.mergeValidationResults(
      { isValid: true },
      { isValid: true }
    );
    expect(merged.isValid).toBe(true);
    expect(merged.errors).toBeUndefined();
  });
});
