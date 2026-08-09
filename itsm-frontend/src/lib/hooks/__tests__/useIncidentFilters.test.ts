import { renderHook, act } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';

import { useIncidentFilters } from '../useIncidentFilters';

const wrapper = ({ children }: { children: React.ReactNode }) =>
  React.createElement(App, null, children);

describe('useIncidentFilters', () => {
  it('starts with empty search and hidden filter panel', () => {
    const { result } = renderHook(() => useIncidentFilters(), { wrapper });

    expect(result.current.values).toEqual({ search: '' });
    expect(result.current.visible).toBe(false);
  });

  it('setSearch updates the search value', () => {
    const { result } = renderHook(() => useIncidentFilters(), { wrapper });

    act(() => result.current.setSearch('email outage'));
    expect(result.current.values.search).toBe('email outage');
  });

  it('setFilter updates individual filter fields', () => {
    const { result } = renderHook(() => useIncidentFilters(), { wrapper });

    act(() => result.current.setFilter({ status: 'open' }));
    expect(result.current.values.status).toBe('open');
    expect(result.current.values.search).toBe('');

    act(() => result.current.setFilter({ priority: 'high' }));
    expect(result.current.values.priority).toBe('high');
    expect(result.current.values.status).toBe('open');
  });

  it('toggleVisible flips visibility', () => {
    const { result } = renderHook(() => useIncidentFilters(), { wrapper });

    act(() => result.current.toggleVisible());
    expect(result.current.visible).toBe(true);

    act(() => result.current.toggleVisible());
    expect(result.current.visible).toBe(false);
  });

  it('reset clears all values back to initial', () => {
    const { result } = renderHook(() => useIncidentFilters(), { wrapper });

    act(() => {
      result.current.setSearch('foo');
      result.current.setFilter({ status: 'open', priority: 'high', source: 'email' });
      result.current.toggleVisible();
    });
    expect(result.current.values).toEqual({
      search: 'foo',
      status: 'open',
      priority: 'high',
      source: 'email',
    });
    expect(result.current.visible).toBe(true);

    act(() => result.current.reset());
    expect(result.current.values).toEqual({ search: '' });
    // visible is intentionally NOT reset — that's UI affordance, not filter value
    expect(result.current.visible).toBe(true);
  });
});
