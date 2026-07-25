/**
 * Tests for utils.ts
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

jest.mock('clsx', () => ({
  clsx: (...args: any[]) => args.filter(Boolean).join(' '),
}));

jest.mock('tailwind-merge', () => ({
  twMerge: (str: string) => str,
}));

import {
  cn,
  formatDate,
  formatFileSize,
  generateId,
  debounce,
  throttle,
  deepClone,
  isEmpty,
  formatNumber,
  truncateText,
  getFileExtension,
  isValidEmail,
  isValidPhone,
  snakeToCamel,
  keysToCamelCase,
  getApiData,
  extractListData,
  extractTotal,
  getUrlParams,
  buildQueryString,
} from '../utils';

describe('Utils', () => {
  describe('cn', () => {
    it('should merge class names', () => {
      const result = cn('a', 'b');
      expect(result).toContain('a');
      expect(result).toContain('b');
    });

    it('should handle falsy values', () => {
      const result = cn('a', undefined, null, 'b');
      expect(result).toContain('a');
      expect(result).toContain('b');
    });
  });

  describe('formatDate', () => {
    it('should return relative time for recent dates', () => {
      const now = new Date();
      const result = formatDate(now);
      expect(result).toBe('刚刚');
    });

    it('should return minutes ago', () => {
      const fiveMinAgo = new Date(Date.now() - 5 * 60 * 1000);
      const result = formatDate(fiveMinAgo);
      expect(result).toContain('分钟前');
    });

    it('should return hours ago', () => {
      const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000);
      const result = formatDate(twoHoursAgo);
      expect(result).toContain('小时前');
    });

    it('should return invalid date for bad input', () => {
      const result = formatDate('not-a-date');
      expect(result).toBe('无效日期');
    });

    it('should handle string date input', () => {
      const result = formatDate('2020-01-01');
      expect(typeof result).toBe('string');
    });
  });

  describe('formatFileSize', () => {
    it('should return 0 B for zero', () => {
      expect(formatFileSize(0)).toBe('0 B');
    });

    it('should format bytes', () => {
      expect(formatFileSize(500)).toBe('500 B');
    });

    it('should format KB', () => {
      expect(formatFileSize(1024)).toBe('1 KB');
    });

    it('should format MB', () => {
      expect(formatFileSize(1024 * 1024)).toBe('1 MB');
    });

    it('should format GB', () => {
      expect(formatFileSize(1024 * 1024 * 1024)).toBe('1 GB');
    });
  });

  describe('generateId', () => {
    it('should generate string of default length', () => {
      const id = generateId();
      expect(id.length).toBe(8);
    });

    it('should generate string of specified length', () => {
      const id = generateId(16);
      expect(id.length).toBe(16);
    });

    it('should generate unique values', () => {
      const id1 = generateId();
      const id2 = generateId();
      expect(id1).not.toBe(id2);
    });
  });

  describe('debounce', () => {
    jest.useFakeTimers();

    it('should delay execution', () => {
      const fn = jest.fn();
      const debounced = debounce(fn, 100);

      debounced();
      expect(fn).not.toHaveBeenCalled();

      jest.advanceTimersByTime(100);
      expect(fn).toHaveBeenCalledTimes(1);
    });

    it('should reset timer on subsequent calls', () => {
      const fn = jest.fn();
      const debounced = debounce(fn, 100);

      debounced();
      jest.advanceTimersByTime(50);
      debounced();
      jest.advanceTimersByTime(50);
      expect(fn).not.toHaveBeenCalled();

      jest.advanceTimersByTime(50);
      expect(fn).toHaveBeenCalledTimes(1);
    });

    afterAll(() => {
      jest.useRealTimers();
    });
  });

  describe('throttle', () => {
    jest.useFakeTimers();

    it('should execute immediately first time', () => {
      const fn = jest.fn();
      const throttled = throttle(fn, 100);

      throttled();
      expect(fn).toHaveBeenCalledTimes(1);
    });

    it('should not execute again during limit', () => {
      const fn = jest.fn();
      const throttled = throttle(fn, 100);

      throttled();
      throttled();
      throttled();
      expect(fn).toHaveBeenCalledTimes(1);
    });

    afterAll(() => {
      jest.useRealTimers();
    });
  });

  describe('deepClone', () => {
    it('should clone primitive values', () => {
      expect(deepClone(5)).toBe(5);
      expect(deepClone('hello')).toBe('hello');
      expect(deepClone(null)).toBeNull();
    });

    it('should clone arrays', () => {
      const arr = [1, 2, [3, 4]];
      const cloned = deepClone(arr);
      expect(cloned).toEqual(arr);
      expect(cloned).not.toBe(arr);
    });

    it('should clone objects', () => {
      const obj = { a: 1, b: { c: 2 } };
      const cloned = deepClone(obj);
      expect(cloned).toEqual(obj);
      expect(cloned).not.toBe(obj);
      expect(cloned.b).not.toBe(obj.b);
    });

    it('should clone Date objects', () => {
      const date = new Date('2024-01-01');
      const cloned = deepClone(date);
      expect(cloned.getTime()).toBe(date.getTime());
      expect(cloned).not.toBe(date);
    });
  });

  describe('isEmpty', () => {
    it('should return true for null/undefined', () => {
      expect(isEmpty(null)).toBe(true);
      expect(isEmpty(undefined)).toBe(true);
    });

    it('should return true for empty string', () => {
      expect(isEmpty('')).toBe(true);
      expect(isEmpty('   ')).toBe(true);
    });

    it('should return true for empty array', () => {
      expect(isEmpty([])).toBe(true);
    });

    it('should return true for empty object', () => {
      expect(isEmpty({})).toBe(true);
    });

    it('should return false for non-empty values', () => {
      expect(isEmpty('hello')).toBe(false);
      expect(isEmpty([1])).toBe(false);
      expect(isEmpty({ a: 1 })).toBe(false);
      expect(isEmpty(0)).toBe(false);
    });
  });

  describe('formatNumber', () => {
    it('should format with default options', () => {
      expect(formatNumber(1000)).toBe('1,000');
    });

    it('should format with decimals', () => {
      expect(formatNumber(1234.567, { decimals: 2 })).toBe('1,234.57');
    });

    it('should format with prefix/suffix', () => {
      expect(formatNumber(100, { prefix: '$', suffix: ' USD' })).toBe('$100 USD');
    });

    it('should use custom separator', () => {
      expect(formatNumber(1000000, { separator: '.' })).toBe('1.000.000');
    });
  });

  describe('truncateText', () => {
    it('should not truncate short text', () => {
      expect(truncateText('hello', 10)).toBe('hello');
    });

    it('should truncate long text', () => {
      expect(truncateText('hello world', 8)).toBe('hello...');
    });

    it('should use custom suffix', () => {
      expect(truncateText('hello world', 8, '…')).toBe('hello w…');
    });
  });

  describe('getFileExtension', () => {
    it('should get extension', () => {
      expect(getFileExtension('file.txt')).toBe('txt');
      expect(getFileExtension('archive.tar.gz')).toBe('gz');
    });

    it('should return empty for no extension', () => {
      expect(getFileExtension('noextension')).toBe('');
    });
  });

  describe('isValidEmail', () => {
    it('should validate correct emails', () => {
      expect(isValidEmail('test@example.com')).toBe(true);
      expect(isValidEmail('user.name@domain.org')).toBe(true);
    });

    it('should reject invalid emails', () => {
      expect(isValidEmail('notanemail')).toBe(false);
      expect(isValidEmail('@domain.com')).toBe(false);
      expect(isValidEmail('user@')).toBe(false);
    });
  });

  describe('isValidPhone', () => {
    it('should validate correct Chinese phone numbers', () => {
      expect(isValidPhone('13800138000')).toBe(true);
      expect(isValidPhone('19912345678')).toBe(true);
    });

    it('should reject invalid numbers', () => {
      expect(isValidPhone('12345')).toBe(false);
      expect(isValidPhone('02812345678')).toBe(false);
    });
  });

  describe('snakeToCamel', () => {
    it('should convert snake_case to camelCase', () => {
      expect(snakeToCamel('hello_world')).toBe('helloWorld');
      expect(snakeToCamel('foo_bar_baz')).toBe('fooBarBaz');
    });

    it('should handle single word', () => {
      expect(snakeToCamel('hello')).toBe('hello');
    });
  });

  describe('keysToCamelCase', () => {
    it('should convert object keys', () => {
      const result = keysToCamelCase({ first_name: 'John', last_name: 'Doe' });
      expect(result).toEqual({ firstName: 'John', lastName: 'Doe' });
    });

    it('should handle nested objects', () => {
      const result = keysToCamelCase({ user_info: { first_name: 'John' } } as any);
      expect(result).toEqual({ userInfo: { firstName: 'John' } });
    });

    it('should handle null/non-object', () => {
      expect(keysToCamelCase(null as any)).toBeNull();
    });
  });

  describe('getApiData', () => {
    it('should get data by camelCase key', () => {
      const response = { totalCount: 10 };
      expect(getApiData(response, 'total_count' as any)).toBe(10);
    });

    it('should get data by snake_case key', () => {
      const response = { total_count: 10 };
      expect(getApiData(response, 'total_count' as any)).toBe(10);
    });

    it('should return undefined for missing key', () => {
      expect(getApiData({}, 'missing' as any)).toBeUndefined();
    });
  });

  describe('extractListData', () => {
    it('should extract items from response', () => {
      const response = { items: [1, 2, 3] };
      expect(extractListData(response)).toEqual([1, 2, 3]);
    });

    it('should extract from nested data.items', () => {
      const response = { data: { items: [1, 2, 3] } };
      expect(extractListData(response)).toEqual([1, 2, 3]);
    });

    it('should extract array data directly', () => {
      const response = { data: [1, 2, 3] };
      expect(extractListData(response)).toEqual([1, 2, 3]);
    });

    it('should return empty array for no data', () => {
      expect(extractListData({})).toEqual([]);
    });
  });

  describe('extractTotal', () => {
    it('should extract total from response', () => {
      expect(extractTotal({ total: 100 })).toBe(100);
    });

    it('should extract from nested data.total', () => {
      expect(extractTotal({ data: { total: 50 } })).toBe(50);
    });

    it('should return 0 for missing total', () => {
      expect(extractTotal({})).toBe(0);
    });
  });

  describe('getUrlParams', () => {
    it('should parse URL params', () => {
      const params = getUrlParams('http://example.com?a=1&b=2');
      expect(params).toEqual({ a: '1', b: '2' });
    });

    it('should return empty for no params', () => {
      const params = getUrlParams('http://example.com');
      expect(params).toEqual({});
    });
  });

  describe('buildQueryString', () => {
    it('should build query string', () => {
      const result = buildQueryString({ a: '1', b: '2' });
      expect(result).toContain('a=1');
      expect(result).toContain('b=2');
    });

    it('should skip null/undefined/empty values', () => {
      const result = buildQueryString({ a: '1', b: null, c: undefined, d: '' });
      expect(result).toBe('a=1');
    });
  });
});
