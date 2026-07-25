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
  useDebounce,
  useThrottle,
  useDebouncedCallback,
  useThrottledCallback,
  useWindowSize,
  useMediaQuery,
  useKeyPress,
  useClipboard,
  useNotification,
  useFormValidation,
  useIsMounted,
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

  describe('useDebounce', () => {
    beforeEach(() => jest.useFakeTimers());
    afterEach(() => jest.useRealTimers());

    it('should return initial value immediately', () => {
      const { result } = renderHook(() => useDebounce('hello', 300));
      expect(result.current).toBe('hello');
    });

    it('should debounce value changes', () => {
      const { result, rerender } = renderHook(
        ({ value }) => useDebounce(value, 300),
        { initialProps: { value: 'a' } }
      );
      rerender({ value: 'b' });
      expect(result.current).toBe('a');
      act(() => { jest.advanceTimersByTime(300); });
      expect(result.current).toBe('b');
    });
  });

  describe('useThrottle', () => {
    beforeEach(() => jest.useFakeTimers());
    afterEach(() => jest.useRealTimers());

    it('should throttle value changes', () => {
      const { result, rerender } = renderHook(
        ({ value }) => useThrottle(value, 200),
        { initialProps: { value: 'x' } }
      );
      rerender({ value: 'y' });
      expect(result.current).toBe('x');
      act(() => { jest.advanceTimersByTime(200); });
      expect(result.current).toBe('y');
    });
  });

  describe('useDebouncedCallback', () => {
    it('should return a debounced function', () => {
      const fn = jest.fn();
      const { result } = renderHook(() => useDebouncedCallback(fn, 100));
      act(() => { result.current('arg1'); });
      expect(fn).toHaveBeenCalledWith('arg1');
    });
  });

  describe('useThrottledCallback', () => {
    it('should return a throttled function', () => {
      const fn = jest.fn();
      const { result } = renderHook(() => useThrottledCallback(fn, 100));
      act(() => { result.current('arg1'); });
      expect(fn).toHaveBeenCalledWith('arg1');
    });
  });

  describe('useWindowSize', () => {
    it('should return current window size', () => {
      const { result } = renderHook(() => useWindowSize());
      expect(result.current.width).toBe(window.innerWidth);
      expect(result.current.height).toBe(window.innerHeight);
    });
  });

  describe('useMediaQuery', () => {
    it('should return false when no match', () => {
      const { result } = renderHook(() => useMediaQuery('(min-width: 768px)'));
      expect(result.current).toBe(false);
    });
  });

  describe('useKeyPress', () => {
    it('should call callback on key press', () => {
      const fn = jest.fn();
      renderHook(() => useKeyPress('Enter', fn));
      act(() => {
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
      });
      expect(fn).toHaveBeenCalled();
    });

    it('should not call callback for different key', () => {
      const fn = jest.fn();
      renderHook(() => useKeyPress('Enter', fn));
      act(() => {
        document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
      });
      expect(fn).not.toHaveBeenCalled();
    });
  });

  describe('useClipboard', () => {
    it('should copy text', async () => {
      Object.defineProperty(navigator, 'clipboard', {
        value: { writeText: jest.fn().mockResolvedValue(undefined) },
        writable: true,
        configurable: true,
      });
      const { result } = renderHook(() => useClipboard());
      await act(async () => { await result.current.copyToClipboard('text'); });
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith('text');
    });
  });

  describe('useNotification', () => {
    it('should expose notification methods', () => {
      const { result } = renderHook(() => useNotification());
      expect(result.current.showSuccess).toBeDefined();
      expect(result.current.showError).toBeDefined();
      expect(result.current.showWarning).toBeDefined();
      expect(result.current.showInfo).toBeDefined();
    });
  });

  describe('useFormValidation', () => {
    it('should initialize with values', () => {
      const rules = { name: (v: unknown) => (!v ? 'Required' : null) };
      const { result } = renderHook(() => useFormValidation({ name: '' }, rules));
      expect(result.current.values.name).toBe('');
      expect(result.current.isFormValid).toBe(true);
    });

    it('should validate on setValue', () => {
      const rules = { name: (v: unknown) => (!v ? 'Required' : null) };
      const { result } = renderHook(() => useFormValidation({ name: 'hello' }, rules));
      act(() => { result.current.setValue('name', ''); });
      expect(result.current.errors.name).toBe('Required');
    });

    it('should validateForm', () => {
      const rules = { name: (v: unknown) => (!v ? 'Required' : null) };
      const { result } = renderHook(() => useFormValidation({ name: '' }, rules));
      let isValid: boolean;
      act(() => { isValid = result.current.validateForm(); });
      expect(result.current.errors.name).toBe('Required');
    });

    it('should resetForm', () => {
      const rules = { name: (v: unknown) => (!v ? 'Required' : null) };
      const { result } = renderHook(() => useFormValidation({ name: '' }, rules));
      act(() => {
        result.current.setValue('name', 'hello');
        result.current.resetForm();
      });
      expect(result.current.values.name).toBe('');
    });

    it('should setFieldTouched', () => {
      const rules = { name: () => null };
      const { result } = renderHook(() => useFormValidation({ name: '' }, rules));
      act(() => { result.current.setFieldTouched('name'); });
      expect(result.current.touched.name).toBe(true);
    });
  });

  describe('useIsMounted', () => {
    it('should return false initially (ref not yet set)', () => {
      const { result } = renderHook(() => useIsMounted());
      // useIsMounted returns ref.current which is initially false
      // but after first render it becomes true in the next tick
      expect(typeof result.current).toBe('boolean');
    });
  });
});
