import {
  validateRequiredProps,
  withDefaults,
  deepMergeDefaults,
  validatePropType,
  safeGet,
  safeGetNested,
  validateArrayProp,
  validateFunctionProp,
  createSafeHandler,
  sanitizeObjectProp,
} from '../props-validator';

describe('validateRequiredProps', () => {
  const origEnv = process.env.NODE_ENV;

  afterEach(() => {
    Object.defineProperty(process.env, 'NODE_ENV', { value: origEnv, writable: true });
  });

  it('returns true when all required props present', () => {
    const spy = jest.spyOn(console, 'error').mockImplementation(() => {});
    expect(validateRequiredProps({ a: 1, b: 'x' }, ['a', 'b'], 'Comp')).toBe(true);
    spy.mockRestore();
  });

  it('returns false and logs when props missing (production)', () => {
    Object.defineProperty(process.env, 'NODE_ENV', { value: 'production', writable: true });
    const spy = jest.spyOn(console, 'error').mockImplementation(() => {});
    expect(validateRequiredProps({ a: null, b: 'x' } as any, ['a'], 'Comp')).toBe(false);
    expect(spy).toHaveBeenCalled();
    spy.mockRestore();
  });

  it('throws in development mode when props missing', () => {
    Object.defineProperty(process.env, 'NODE_ENV', { value: 'development', writable: true });
    const spy = jest.spyOn(console, 'error').mockImplementation(() => {});
    expect(() => validateRequiredProps({ a: undefined } as any, ['a'], 'X')).toThrow();
    spy.mockRestore();
  });
});

describe('withDefaults', () => {
  it('merges partial props with defaults', () => {
    expect(withDefaults({ a: 1 }, { a: 0, b: 'hi' })).toEqual({ a: 1, b: 'hi' });
  });
});

describe('deepMergeDefaults', () => {
  it('deeply merges nested objects', () => {
    const result = deepMergeDefaults(
      { nested: { x: 10 } } as any,
      { nested: { x: 0, y: 5 }, flat: 'z' }
    );
    expect(result).toEqual({ nested: { x: 10, y: 5 }, flat: 'z' });
  });

  it('skips null/undefined values in props', () => {
    const result = deepMergeDefaults({ a: null } as any, { a: 1, b: 2 });
    expect(result).toEqual({ a: 1, b: 2 });
  });
});

describe('validatePropType', () => {
  it('returns true for correct type', () => {
    const spy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    expect(validatePropType('hello', 'string', 'name', 'C')).toBe(true);
    spy.mockRestore();
  });

  it('returns false and warns for wrong type', () => {
    const spy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    expect(validatePropType(123, 'string', 'name', 'C')).toBe(false);
    expect(spy).toHaveBeenCalled();
    spy.mockRestore();
  });

  it('detects arrays correctly', () => {
    const spy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    expect(validatePropType([1, 2], 'array', 'items', 'C')).toBe(true);
    spy.mockRestore();
  });
});

describe('safeGet', () => {
  it('returns value when present', () => {
    expect(safeGet({ name: 'Alice' }, 'name')).toBe('Alice');
  });
  it('returns defaultValue when obj is null', () => {
    expect(safeGet(null as unknown as Record<string, string>, 'name', 'fallback')).toBe('fallback');
  });
});

describe('safeGetNested', () => {
  it('navigates nested path', () => {
    expect(safeGetNested({ a: { b: 42 } }, 'a.b')).toBe(42);
  });
  it('returns default for broken path', () => {
    expect(safeGetNested({ a: null }, 'a.b.c', 'def')).toBe('def');
  });
});

describe('validateArrayProp', () => {
  it('returns true for valid array', () => {
    const spy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    expect(validateArrayProp([1, 2, 3], 'items', 'C', 1, 5)).toBe(true);
    spy.mockRestore();
  });
  it('returns false for non-array', () => {
    const spy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    expect(validateArrayProp('not array', 'items', 'C')).toBe(false);
    spy.mockRestore();
  });
  it('returns false when below minLength', () => {
    const spy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    expect(validateArrayProp([], 'items', 'C', 1)).toBe(false);
    spy.mockRestore();
  });
});

describe('validateFunctionProp', () => {
  it('returns true for function', () => {
    const spy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    expect(validateFunctionProp(() => {}, 'onClick', 'C')).toBe(true);
    spy.mockRestore();
  });
  it('returns false for non-function', () => {
    const spy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    expect(validateFunctionProp('not fn', 'onClick', 'C')).toBe(false);
    spy.mockRestore();
  });
});

describe('createSafeHandler', () => {
  it('calls handler when defined', () => {
    const fn = jest.fn(() => 'result');
    const safe = createSafeHandler(fn);
    expect(safe()).toBe('result');
  });
  it('calls fallback when handler undefined', () => {
    const fallback = jest.fn(() => 'fb');
    const safe = createSafeHandler(undefined, fallback);
    expect(safe()).toBe('fb');
  });
  it('catches errors in handler', () => {
    const spy = jest.spyOn(console, 'error').mockImplementation(() => {});
    const throwing = () => { throw new Error('oops'); };
    const safe = createSafeHandler(throwing as any);
    expect(() => safe()).not.toThrow();
    spy.mockRestore();
  });
});

describe('sanitizeObjectProp', () => {
  it('returns value if valid object', () => {
    const spy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    expect(sanitizeObjectProp({ x: 1 }, 'config', 'C', {})).toEqual({ x: 1 });
    spy.mockRestore();
  });
  it('returns default for non-object', () => {
    const spy = jest.spyOn(console, 'warn').mockImplementation(() => {});
    expect(sanitizeObjectProp('str', 'config', 'C', { y: 2 })).toEqual({ y: 2 });
    spy.mockRestore();
  });
});
