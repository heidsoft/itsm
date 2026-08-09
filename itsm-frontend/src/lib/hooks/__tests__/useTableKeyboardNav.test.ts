import { renderHook } from '@testing-library/react';
import { useTableKeyboardNav } from '../useTableKeyboardNav';

interface Row {
  id: number;
  label: string;
}

const ROWS: Row[] = [
  { id: 1, label: 'one' },
  { id: 2, label: 'two' },
  { id: 3, label: 'three' },
];

function fireKey(key: string, target: EventTarget = document.body) {
  // jsdom does not focus body by default; ensure the event has a target the
  // hook recognizes as non-editable.
  Object.defineProperty(target, 'matches', {
    value: () => false,
    configurable: true,
  });
  window.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }));
}

describe('useTableKeyboardNav', () => {
  it('calls onNext when j is pressed', () => {
    const onNext = jest.fn();
    renderHook(() =>
      useTableKeyboardNav<Row>({
        rows: ROWS,
        activeIndex: 0,
        onActivate: jest.fn(),
        onNext,
        onPrev: jest.fn(),
      })
    );

    fireKey('j');
    expect(onNext).toHaveBeenCalledTimes(1);
  });

  it('calls onPrev when k is pressed', () => {
    const onPrev = jest.fn();
    renderHook(() =>
      useTableKeyboardNav<Row>({
        rows: ROWS,
        activeIndex: 1,
        onActivate: jest.fn(),
        onNext: jest.fn(),
        onPrev,
      })
    );

    fireKey('k');
    expect(onPrev).toHaveBeenCalledTimes(1);
  });

  it('calls onActivate with the active row when o is pressed', () => {
    const onActivate = jest.fn();
    renderHook(() =>
      useTableKeyboardNav<Row>({
        rows: ROWS,
        activeIndex: 1,
        onActivate,
        onNext: jest.fn(),
        onPrev: jest.fn(),
      })
    );

    fireKey('o');
    expect(onActivate).toHaveBeenCalledWith(ROWS[1], 1);
  });

  it('skips keys when target is an input', () => {
    const onNext = jest.fn();
    renderHook(() =>
      useTableKeyboardNav<Row>({
        rows: ROWS,
        activeIndex: 0,
        onActivate: jest.fn(),
        onNext,
        onPrev: jest.fn(),
      })
    );

    const input = document.createElement('input');
    document.body.appendChild(input);
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'j', bubbles: true }));
    expect(onNext).not.toHaveBeenCalled();
    document.body.removeChild(input);
  });

  it('clamps onActivate to the last row when activeIndex is out of range', () => {
    const onActivate = jest.fn();
    renderHook(() =>
      useTableKeyboardNav<Row>({
        rows: ROWS,
        activeIndex: 99,
        onActivate,
        onNext: jest.fn(),
        onPrev: jest.fn(),
      })
    );

    fireKey('o');
    expect(onActivate).toHaveBeenCalledWith(ROWS[ROWS.length - 1], ROWS.length - 1);
  });

  it('does nothing when rows is empty', () => {
    const onNext = jest.fn();
    renderHook(() =>
      useTableKeyboardNav<Row>({
        rows: [],
        activeIndex: 0,
        onActivate: jest.fn(),
        onNext,
        onPrev: jest.fn(),
      })
    );

    fireKey('j');
    expect(onNext).not.toHaveBeenCalled();
  });

  it('does nothing when enabled is false', () => {
    const onNext = jest.fn();
    renderHook(() =>
      useTableKeyboardNav<Row>({
        rows: ROWS,
        activeIndex: 0,
        onActivate: jest.fn(),
        onNext,
        onPrev: jest.fn(),
        enabled: false,
      })
    );

    fireKey('j');
    expect(onNext).not.toHaveBeenCalled();
  });

  it('ignores unrelated keys', () => {
    const onNext = jest.fn();
    const onPrev = jest.fn();
    const onActivate = jest.fn();
    renderHook(() =>
      useTableKeyboardNav<Row>({
        rows: ROWS,
        activeIndex: 0,
        onActivate,
        onNext,
        onPrev,
      })
    );

    fireKey('x');
    fireKey('Enter');
    fireKey('ArrowDown');
    expect(onNext).not.toHaveBeenCalled();
    expect(onPrev).not.toHaveBeenCalled();
    expect(onActivate).not.toHaveBeenCalled();
  });
});
