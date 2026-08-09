import { useCallback, useMemo, useState } from 'react';

export interface UseRowSelectionResult<TKey extends string | number> {
  readonly selectedIds: ReadonlySet<TKey>;
  readonly count: number;
  readonly isSelected: (id: TKey) => boolean;
  readonly toggle: (id: TKey) => void;
  readonly select: (id: TKey) => void;
  readonly deselect: (id: TKey) => void;
  readonly clear: () => void;
  readonly setMany: (ids: Iterable<TKey>) => void;
  readonly toArray: () => TKey[];
}

/**
 * Manages row-selection state for a table without prop-drilling a Set through
 * the container. Returns immutable Set wrappers plus ergonomic helpers, so
 * callers never need to spread/copy a Set manually.
 */
export function useRowSelection<TKey extends string | number>(): UseRowSelectionResult<TKey> {
  const [selectedIds, setSelectedIds] = useState<ReadonlySet<TKey>>(() => new Set());

  const toggle = useCallback((id: TKey) => {
    setSelectedIds(prev => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const select = useCallback((id: TKey) => {
    setSelectedIds(prev => {
      if (prev.has(id)) return prev;
      const next = new Set(prev);
      next.add(id);
      return next;
    });
  }, []);

  const deselect = useCallback((id: TKey) => {
    setSelectedIds(prev => {
      if (!prev.has(id)) return prev;
      const next = new Set(prev);
      next.delete(id);
      return next;
    });
  }, []);

  const clear = useCallback(() => {
    setSelectedIds(prev => (prev.size === 0 ? prev : new Set()));
  }, []);

  const setMany = useCallback((ids: Iterable<TKey>) => {
    setSelectedIds(new Set(ids));
  }, []);

  const toArray = useCallback(() => Array.from(selectedIds), [selectedIds]);

  const isSelected = useCallback((id: TKey) => selectedIds.has(id), [selectedIds]);

  return useMemo(
    () => ({
      selectedIds,
      count: selectedIds.size,
      isSelected,
      toggle,
      select,
      deselect,
      clear,
      setMany,
      toArray,
    }),
    [selectedIds, isSelected, toggle, select, deselect, clear, setMany, toArray]
  );
}
