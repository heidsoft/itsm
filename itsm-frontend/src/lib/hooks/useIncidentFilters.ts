'use client';

import { useCallback, useState } from 'react';

export interface IncidentFilterValues {
  readonly search: string;
  readonly status?: string;
  readonly priority?: string;
  readonly source?: string;
}

export interface UseIncidentFiltersResult {
  readonly values: IncidentFilterValues;
  readonly visible: boolean;
  readonly setSearch: (next: string) => void;
  readonly setFilter: (next: Partial<Omit<IncidentFilterValues, 'search'>>) => void;
  readonly toggleVisible: () => void;
  readonly reset: () => void;
}

const INITIAL_VALUES: IncidentFilterValues = {
  search: '',
};

/**
 * Owns the five pieces of UI state behind the search/filter controls on the
 * incidents page: keyword, status, priority, source, plus the visibility
 * flag of the advanced filter row.
 *
 * Pure state — does not refetch. The caller passes `values` to
 * `useIncidentsQuery`, which owns fetch lifecycle.
 */
export function useIncidentFilters(): UseIncidentFiltersResult {
  const [values, setValues] = useState<IncidentFilterValues>(INITIAL_VALUES);
  const [visible, setVisible] = useState(false);

  const setSearch = useCallback((next: string) => {
    setValues(prev => ({ ...prev, search: next }));
  }, []);

  const setFilter = useCallback((next: Partial<Omit<IncidentFilterValues, 'search'>>) => {
    setValues(prev => ({ ...prev, ...next }));
  }, []);

  const toggleVisible = useCallback(() => {
    setVisible(prev => !prev);
  }, []);

  const reset = useCallback(() => {
    setValues(INITIAL_VALUES);
  }, []);

  return { values, visible, setSearch, setFilter, toggleVisible, reset };
}
