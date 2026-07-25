import { renderHook, act } from '@testing-library/react';
import { useGlobalSearch } from '../useGlobalSearch';

describe('useGlobalSearch', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should return initial closed state', () => {
    const { result } = renderHook(() => useGlobalSearch());

    expect(result.current.isOpen).toBe(false);
  });

  it('should open the search', () => {
    const { result } = renderHook(() => useGlobalSearch());

    act(() => {
      result.current.open();
    });

    expect(result.current.isOpen).toBe(true);
  });

  it('should close the search', () => {
    const { result } = renderHook(() => useGlobalSearch());

    act(() => {
      result.current.open();
    });

    act(() => {
      result.current.close();
    });

    expect(result.current.isOpen).toBe(false);
  });

  it('should toggle the search', () => {
    const { result } = renderHook(() => useGlobalSearch());

    act(() => {
      result.current.toggle();
    });
    expect(result.current.isOpen).toBe(true);

    act(() => {
      result.current.toggle();
    });
    expect(result.current.isOpen).toBe(false);
  });

  it('should toggle on Ctrl+K', () => {
    const { result } = renderHook(() => useGlobalSearch());

    act(() => {
      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
        bubbles: true,
      });
      window.dispatchEvent(event);
    });

    expect(result.current.isOpen).toBe(true);
  });

  it('should toggle on Meta+K (Mac)', () => {
    const { result } = renderHook(() => useGlobalSearch());

    act(() => {
      const event = new KeyboardEvent('keydown', {
        key: 'k',
        metaKey: true,
        bubbles: true,
      });
      window.dispatchEvent(event);
    });

    expect(result.current.isOpen).toBe(true);
  });

  it('should not toggle on plain K key', () => {
    const { result } = renderHook(() => useGlobalSearch());

    act(() => {
      const event = new KeyboardEvent('keydown', {
        key: 'k',
        bubbles: true,
      });
      window.dispatchEvent(event);
    });

    expect(result.current.isOpen).toBe(false);
  });
});
