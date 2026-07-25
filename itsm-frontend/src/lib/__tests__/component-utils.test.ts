/**
 * Tests for component-utils.ts hooks
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

jest.mock('antd', () => ({
  message: { error: jest.fn(), warning: jest.fn(), success: jest.fn(), info: jest.fn() },
  notification: { error: jest.fn(), success: jest.fn(), warning: jest.fn(), info: jest.fn() },
}));

jest.mock('@/lib/env', () => ({
  logger: { error: jest.fn(), warn: jest.fn(), debug: jest.fn(), info: jest.fn() },
}));

jest.mock('lodash-es', () => ({
  debounce: (fn: any) => {
    const d = (...args: any[]) => fn(...args);
    d.cancel = jest.fn();
    return d;
  },
  throttle: (fn: any) => {
    const t = (...args: any[]) => fn(...args);
    t.cancel = jest.fn();
    return t;
  },
}));

// Mock matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: jest.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: jest.fn(),
    removeListener: jest.fn(),
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn(),
  })),
});

import { renderHook, act } from '@testing-library/react';
import {
  useLocalStorage,
  useSessionStorage,
  usePrevious,
  useIsFirstRender,
  useVirtualScroll,
  hotReloadComponent,
  componentUtils,
} from '../component-utils';

describe('Component Utils', () => {
  describe('useLocalStorage', () => {
    beforeEach(() => { localStorage.clear(); });

    it('should return initial value when no stored value', () => {
      const { result } = renderHook(() => useLocalStorage('test-key', 'default'));
      expect(result.current[0]).toBe('default');
    });

    it('should return stored value from localStorage', () => {
      localStorage.setItem('ls-key', JSON.stringify('stored'));
      const { result } = renderHook(() => useLocalStorage('ls-key', 'default'));
      expect(result.current[0]).toBe('stored');
    });

    it('should update value', () => {
      const { result } = renderHook(() => useLocalStorage('test-key', 'default'));
      act(() => { result.current[1]('new value'); });
      expect(result.current[0]).toBe('new value');
      expect(JSON.parse(localStorage.getItem('test-key')!)).toBe('new value');
    });

    it('should handle function updater', () => {
      const { result } = renderHook(() => useLocalStorage('num-key', 0));
      act(() => { result.current[1]((prev: number) => prev + 1); });
      expect(result.current[0]).toBe(1);
    });

    it('should handle invalid JSON in localStorage', () => {
      localStorage.setItem('bad-key', 'not json');
      const { result } = renderHook(() => useLocalStorage('bad-key', 'fallback'));
      expect(result.current[0]).toBe('fallback');
    });
  });

  describe('useSessionStorage', () => {
    beforeEach(() => { sessionStorage.clear(); });

    it('should return initial value', () => {
      const { result } = renderHook(() => useSessionStorage('ss-key', 'default'));
      expect(result.current[0]).toBe('default');
    });

    it('should update value', () => {
      const { result } = renderHook(() => useSessionStorage('ss-key', 'default'));
      act(() => { result.current[1]('updated'); });
      expect(result.current[0]).toBe('updated');
    });

    it('should handle invalid JSON', () => {
      sessionStorage.setItem('bad-ss', 'invalid');
      const { result } = renderHook(() => useSessionStorage('bad-ss', 'safe'));
      expect(result.current[0]).toBe('safe');
    });
  });

  describe('usePrevious', () => {
    it('should return undefined on first render', () => {
      const { result } = renderHook(() => usePrevious('hello'));
      expect(result.current).toBeUndefined();
    });

    it('should return previous value after update', () => {
      const { result, rerender } = renderHook(
        ({ value }) => usePrevious(value),
        { initialProps: { value: 'first' } }
      );
      rerender({ value: 'second' });
      expect(result.current).toBe('first');
    });
  });

  describe('useIsFirstRender', () => {
    it('should return true on first render', () => {
      const { result } = renderHook(() => useIsFirstRender());
      expect(result.current).toBe(true);
    });

    it('should return false on subsequent renders', () => {
      const { result, rerender } = renderHook(() => useIsFirstRender());
      rerender();
      expect(result.current).toBe(false);
    });
  });

  describe('useVirtualScroll', () => {
    it('should calculate visible items', () => {
      const items = Array.from({ length: 100 }, (_, i) => i);
      const { result } = renderHook(() => useVirtualScroll(items, 50, 500, 5));
      expect(result.current.visibleItems.length).toBeGreaterThan(0);
      expect(result.current.totalHeight).toBe(5000);
      expect(result.current.offsetY).toBeGreaterThanOrEqual(0);
    });

    it('should update visible items on scroll', () => {
      const items = Array.from({ length: 100 }, (_, i) => i);
      const { result } = renderHook(() => useVirtualScroll(items, 50, 500, 5));
      act(() => { result.current.setScrollTop(1000); });
      expect(result.current.visibleItems[0].index).toBeGreaterThan(0);
    });
  });

  describe('hotReloadComponent', () => {
    it('should not throw', () => {
      expect(() => hotReloadComponent('TestComponent')).not.toThrow();
    });
  });

  describe('componentUtils export', () => {
    it('should export all hooks', () => {
      expect(componentUtils.useDebounce).toBeDefined();
      expect(componentUtils.useThrottle).toBeDefined();
      expect(componentUtils.useDebouncedCallback).toBeDefined();
      expect(componentUtils.useThrottledCallback).toBeDefined();
      expect(componentUtils.useLocalStorage).toBeDefined();
      expect(componentUtils.useSessionStorage).toBeDefined();
      expect(componentUtils.useAsync).toBeDefined();
      expect(componentUtils.usePrevious).toBeDefined();
      expect(componentUtils.useIsFirstRender).toBeDefined();
      expect(componentUtils.useIsMounted).toBeDefined();
      expect(componentUtils.useWindowSize).toBeDefined();
      expect(componentUtils.useMediaQuery).toBeDefined();
      expect(componentUtils.useClickOutside).toBeDefined();
      expect(componentUtils.useKeyPress).toBeDefined();
      expect(componentUtils.useClipboard).toBeDefined();
      expect(componentUtils.useNotification).toBeDefined();
      expect(componentUtils.useFormValidation).toBeDefined();
      expect(componentUtils.useVirtualScroll).toBeDefined();
    });
  });
});
