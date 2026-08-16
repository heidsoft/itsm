/**
 * Formatters 测试
 */

// jest.setup.js 为 Ant Design DatePicker 全局 mock 了 dayjs（isValid 恒 true），
// 本文件需要真实解析行为（invalid/零值检测），因此用 requireActual 覆盖 mock。
type DayjsFactory = typeof import('dayjs');

jest.mock('dayjs', () => {
  const actual = jest.requireActual('dayjs');
  const realDayjs = jest.fn((date?: string) => actual(date)) as jest.Mock & DayjsFactory;
  realDayjs.extend = actual.extend;
  realDayjs.locale = actual.locale;
  realDayjs.unix = actual.unix;
  realDayjs.isDayjs = actual.isDayjs;
  return realDayjs;
});

import { formatDateTime, mapLabel } from '../formatters';

describe('formatDateTime', () => {
  it('should format valid ISO date string', () => {
    const result = formatDateTime('2024-01-15T10:30:00Z');
    expect(result).toBeTruthy();
    expect(typeof result).toBe('string');
  });

  it('should return empty string for undefined', () => {
    expect(formatDateTime(undefined)).toBe('');
  });

  it('should return empty string for null', () => {
    expect(formatDateTime(null)).toBe('');
  });

  it('should return empty string for empty string', () => {
    expect(formatDateTime('')).toBe('');
  });

  it('should return empty string for invalid date', () => {
    const invalidDate = 'invalid-date';
    const result = formatDateTime(invalidDate);
    expect(result).toBe('');
  });

  it('should handle ISO format with milliseconds', () => {
    const result = formatDateTime('2024-01-15T10:30:00.123Z');
    expect(result).toBeTruthy();
    expect(typeof result).toBe('string');
  });

  it('should handle date-only format', () => {
    const result = formatDateTime('2024-01-15');
    expect(result).toBeTruthy();
    expect(typeof result).toBe('string');
  });
});

describe('mapLabel', () => {
  it('should return mapped value when key exists', () => {
    const map = { ACTIVE: '激活', INACTIVE: '未激活' };
    expect(mapLabel(map, 'ACTIVE')).toBe('激活');
    expect(mapLabel(map, 'INACTIVE')).toBe('未激活');
  });

  it('should return original value when key does not exist', () => {
    const map = { ACTIVE: '激活' };
    expect(mapLabel(map, 'INACTIVE')).toBe('INACTIVE');
  });

  it('should return empty string for undefined value', () => {
    const map = { ACTIVE: '激活' };
    expect(mapLabel(map, undefined)).toBe('');
  });

  it('should return empty string for null value', () => {
    const map = { ACTIVE: '激活' };
    expect(mapLabel(map, null as any)).toBe('');
  });

  it('should return empty string for empty string value', () => {
    const map = { ACTIVE: '激活' };
    expect(mapLabel(map, '')).toBe('');
  });

  it('should handle empty map', () => {
    expect(mapLabel({}, 'ACTIVE')).toBe('ACTIVE');
  });

  it('should handle complex map', () => {
    const map = {
      open: '打开',
      inProgress: '进行中',
      resolved: '已解决',
      closed: '已关闭',
    };
    expect(mapLabel(map, 'open')).toBe('打开');
    expect(mapLabel(map, 'closed')).toBe('已关闭');
  });
});
