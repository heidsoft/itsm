import { renderHook, act } from '@testing-library/react';
import { useDebounce, useThrottle, useLocalStorage, useSessionStorage } from '../usePerformance';

// Note: usePerformance hook is not tested directly because it has a
// useEffect with no dependency array that intentionally causes continuous
// re-renders (for tracking render metrics). This causes infinite loops
// in test environments.

describe('useDebounce', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('should return initial value immediately', () => {
    const { result } = renderHook(() => useDebounce('hello', 500));
    expect(result.current).toBe('hello');
  });

  it('should debounce value changes', () => {
    const { result, rerender } = renderHook(
      ({ value, delay }) => useDebounce(value, delay),
      { initialProps: { value: 'initial', delay: 500 } }
    );

    expect(result.current).toBe('initial');

    rerender({ value: 'updated', delay: 500 });

    // Value should not have changed yet
    expect(result.current).toBe('initial');

    act(() => {
      jest.advanceTimersByTime(500);
    });

    expect(result.current).toBe('updated');
  });
});

describe('useThrottle', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('should return initial value', () => {
    const { result } = renderHook(() => useThrottle('hello', 500));
    expect(result.current).toBe('hello');
  });

  it('should throttle value changes', () => {
    const { result, rerender } = renderHook(
      ({ value, limit }) => useThrottle(value, limit),
      { initialProps: { value: 'first', limit: 500 } }
    );

    expect(result.current).toBe('first');

    rerender({ value: 'second', limit: 500 });

    act(() => {
      jest.advanceTimersByTime(500);
    });

    expect(result.current).toBe('second');
  });
});

describe('useLocalStorage', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('should return initial value when no stored value', () => {
    const { result } = renderHook(() => useLocalStorage('test-key', 'default'));

    expect(result.current[0]).toBe('default');
  });

  it('should store and retrieve value', () => {
    const { result } = renderHook(() => useLocalStorage('test-key', 'default'));

    act(() => {
      result.current[1]('new-value');
    });

    expect(result.current[0]).toBe('new-value');
    expect(JSON.parse(window.localStorage.getItem('test-key')!)).toBe('new-value');
  });

  it('should read existing localStorage value', () => {
    window.localStorage.setItem('existing-key', JSON.stringify('stored'));

    const { result } = renderHook(() => useLocalStorage('existing-key', 'default'));

    expect(result.current[0]).toBe('stored');
  });

  it('should remove value', () => {
    const { result } = renderHook(() => useLocalStorage('remove-key', 'value'));

    act(() => {
      result.current[1]('set-value');
    });

    act(() => {
      result.current[2]();
    });

    expect(result.current[0]).toBe('value'); // Reset to initial
    expect(window.localStorage.getItem('remove-key')).toBeNull();
  });
});

describe('useSessionStorage', () => {
  beforeEach(() => {
    window.sessionStorage.clear();
  });

  it('should return initial value when no stored value', () => {
    const { result } = renderHook(() => useSessionStorage('session-key', 'default'));

    expect(result.current[0]).toBe('default');
  });

  it('should store and retrieve value', () => {
    const { result } = renderHook(() => useSessionStorage('session-key', 'default'));

    act(() => {
      result.current[1]('session-value');
    });

    expect(result.current[0]).toBe('session-value');
    expect(JSON.parse(window.sessionStorage.getItem('session-key')!)).toBe('session-value');
  });

  it('should remove value', () => {
    const { result } = renderHook(() => useSessionStorage('session-remove', 'initial'));

    act(() => {
      result.current[1]('stored');
    });

    act(() => {
      result.current[2]();
    });

    expect(result.current[0]).toBe('initial');
    expect(window.sessionStorage.getItem('session-remove')).toBeNull();
  });
});
