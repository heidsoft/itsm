import { useEffect, useRef } from 'react';

interface TableKeyboardNavOptions<T> {
  /** Current rows; navigation is a no-op when empty. */
  readonly rows: readonly T[];
  /** Index of the row currently focused by keyboard navigation. */
  readonly activeIndex: number;
  /** Called when the user activates the active row (default key: `o`). */
  readonly onActivate: (row: T, index: number) => void;
  /** Move the active index by +1 (default key: `j`). */
  readonly onNext: () => void;
  /** Move the active index by -1 (default key: `k`). */
  readonly onPrev: () => void;
  /** Disable the listener entirely (e.g. when a modal is open). */
  readonly enabled?: boolean;
}

const NAV_KEYS = ['j', 'k', 'o'] as const;
type NavKey = (typeof NAV_KEYS)[number];

function isEditableTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  if (target.matches('input, textarea, select')) return true;
  return target.isContentEditable;
}

/**
 * Adds vim-style j/k/o navigation to a table. Skips key events when the user
 * is typing in an input/textarea/select/contenteditable, so search fields
 * still accept `j` and `k` as text.
 *
 * Latest handlers are kept in a ref so the listener never re-binds when
 * callers pass new closures, which was the bug in the original effect.
 */
export function useTableKeyboardNav<T>({
  rows,
  activeIndex,
  onActivate,
  onNext,
  onPrev,
  enabled = true,
}: TableKeyboardNavOptions<T>): void {
  const handlersRef = useRef({ onActivate, onNext, onPrev });
  handlersRef.current = { onActivate, onNext, onPrev };

  useEffect(() => {
    if (!enabled || rows.length === 0) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (isEditableTarget(event.target)) return;
      if (!(NAV_KEYS as readonly string[]).includes(event.key)) return;

      const key = event.key as NavKey;
      switch (key) {
        case 'j':
          event.preventDefault();
          handlersRef.current.onNext();
          return;
        case 'k':
          event.preventDefault();
          handlersRef.current.onPrev();
          return;
        case 'o': {
          event.preventDefault();
          const safeIndex = Math.min(Math.max(activeIndex, 0), rows.length - 1);
          const row = rows[safeIndex];
          if (row !== undefined) {
            handlersRef.current.onActivate(row, safeIndex);
          }
          return;
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [enabled, rows, activeIndex]);
}
