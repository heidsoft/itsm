import { renderHook, act } from '@testing-library/react';
import { useRowSelection } from '../useRowSelection';

describe('useRowSelection', () => {
  it('starts empty', () => {
    const { result } = renderHook(() => useRowSelection<number>());
    expect(result.current.count).toBe(0);
    expect(result.current.toArray()).toEqual([]);
  });

  it('toggle adds then removes', () => {
    const { result } = renderHook(() => useRowSelection<number>());

    act(() => result.current.toggle(1));
    expect(result.current.count).toBe(1);
    expect(result.current.isSelected(1)).toBe(true);

    act(() => result.current.toggle(1));
    expect(result.current.count).toBe(0);
    expect(result.current.isSelected(1)).toBe(false);
  });

  it('select is idempotent', () => {
    const { result } = renderHook(() => useRowSelection<number>());

    act(() => result.current.select(7));
    act(() => result.current.select(7));
    expect(result.current.count).toBe(1);
    expect(result.current.toArray()).toEqual([7]);
  });

  it('deselect is idempotent', () => {
    const { result } = renderHook(() => useRowSelection<number>());

    act(() => result.current.select(3));
    act(() => result.current.deselect(3));
    act(() => result.current.deselect(3));
    expect(result.current.count).toBe(0);
  });

  it('clear empties the set', () => {
    const { result } = renderHook(() => useRowSelection<number>());

    act(() => {
      result.current.select(1);
      result.current.select(2);
    });
    expect(result.current.count).toBe(2);

    act(() => result.current.clear());
    expect(result.current.count).toBe(0);
  });

  it('setMany replaces the entire selection', () => {
    const { result } = renderHook(() => useRowSelection<number>());

    act(() => {
      result.current.select(1);
      result.current.setMany([10, 20, 30]);
    });
    expect(result.current.count).toBe(3);
    expect(result.current.isSelected(1)).toBe(false);
    expect(result.current.isSelected(10)).toBe(true);
  });

  it('works with string keys', () => {
    const { result } = renderHook(() => useRowSelection<string>());

    act(() => {
      result.current.toggle('a');
      result.current.toggle('b');
    });
    expect(result.current.toArray().sort()).toEqual(['a', 'b']);
  });
});
