/**
 * useI18n Hook 测试
 */

import { renderHook, act } from '@testing-library/react';
import { useI18n } from '../useI18n';

// Mock user-preferences
const mockGet = jest.fn(() => ({ language: 'zh-CN' }));
const mockUpdate = jest.fn();
const mockSubscribe = jest.fn(() => jest.fn());

jest.mock('@/lib/user-preferences', () => ({
  userPreferences: {
    get: (...args: unknown[]) => mockGet(...args),
    update: (...args: unknown[]) => mockUpdate(...args),
    subscribe: (...args: unknown[]) => mockSubscribe(...args),
  },
}));

describe('useI18n', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGet.mockReturnValue({ language: 'zh-CN' });
    mockSubscribe.mockReturnValue(jest.fn());
  });

  describe('initial state', () => {
    it('should initialize with zh-CN language by default', () => {
      const { result } = renderHook(() => useI18n());
      expect(result.current.language).toBe('zh-CN');
    });

    it('should initialize with en-US when preference is set', () => {
      mockGet.mockReturnValueOnce({ language: 'en-US' });

      const { result } = renderHook(() => useI18n());
      expect(result.current.language).toBe('en-US');
    });
  });

  describe('t() function', () => {
    it('should return translation for valid key', () => {
      const { result } = renderHook(() => useI18n());
      const translation = result.current.t('common.ok');
      expect(typeof translation).toBe('string');
    });

    it('should return key when translation not found', () => {
      const { result } = renderHook(() => useI18n());
      const translation = result.current.t('nonexistent.key');
      expect(translation).toBe('nonexistent.key');
    });

    it('should return key when placeholder params provided for unknown key', () => {
      const { result } = renderHook(() => useI18n());
      // When translation key doesn't exist, returns the key itself
      const translation = result.current.t('nonexistent.key', { count: 5 });
      expect(translation).toBe('nonexistent.key');
    });

    it('should handle nested keys', () => {
      const { result } = renderHook(() => useI18n());
      const translation = result.current.t('menu.dashboard');
      expect(typeof translation).toBe('string');
    });
  });

  describe('changeLanguage()', () => {
    it('should update language preference', () => {
      const { result } = renderHook(() => useI18n());

      act(() => {
        result.current.changeLanguage('en-US');
      });

      expect(mockUpdate).toHaveBeenCalledWith({ language: 'en-US' });
    });

    it('should switch to zh-CN', () => {
      mockGet.mockReturnValueOnce({ language: 'en-US' });

      const { result } = renderHook(() => useI18n());

      act(() => {
        result.current.changeLanguage('zh-CN');
      });

      expect(mockUpdate).toHaveBeenCalledWith({ language: 'zh-CN' });
    });
  });
});
