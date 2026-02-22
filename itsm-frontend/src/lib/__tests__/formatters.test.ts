/**
 * Formatters 测试
 */

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

  it('should return Invalid Date for invalid date', () => {
    const invalidDate = 'invalid-date';
    const result = formatDateTime(invalidDate);
    expect(result).toBe('Invalid Date');
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
      'open': '打开',
      'in_progress': '进行中',
      'resolved': '已解决',
      'closed': '已关闭',
    };
    expect(mapLabel(map, 'open')).toBe('打开');
    expect(mapLabel(map, 'closed')).toBe('已关闭');
  });
});
