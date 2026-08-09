import { renderHook, act, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';

import { useIncidentsQuery } from '../useIncidentsQuery';
import { IncidentAPI } from '@/lib/api/incident-api';
import { UserApi } from '@/lib/api/user-api';

jest.mock('@/lib/api/incident-api', () => ({
  IncidentAPI: {
    listIncidents: jest.fn(),
  },
}));

jest.mock('@/lib/api/user-api', () => ({
  UserApi: {
    getUsers: jest.fn(),
  },
}));

// Antd App context wrapper so `App.useApp()` works inside the hook.
const wrapper = ({ children }: { children: React.ReactNode }) =>
  React.createElement(App, null, children);

const mockIncident = (overrides: Record<string, unknown> = {}) => ({
  id: 1,
  title: 'Test incident',
  description: '',
  status: 'open',
  priority: 'high',
  severity: 'major',
  source: 'web',
  type: 'incident',
  reporterId: 7,
  createdAt: '2026-08-01T00:00:00Z',
  updatedAt: '2026-08-01T00:00:00Z',
  ...overrides,
});

describe('useIncidentsQuery', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('starts empty and idle, then loads data on mount', async () => {
    (IncidentAPI.listIncidents as jest.Mock).mockResolvedValue({
      incidents: [mockIncident()],
      total: 1,
    });
    (UserApi.getUsers as jest.Mock).mockResolvedValue({
      users: [{ id: 7, name: 'Alice', username: 'alice' }],
    });

    const { result } = renderHook(() => useIncidentsQuery({ search: '' }), { wrapper });

    expect(result.current.loading).toBe(true);
    expect(result.current.incidents).toEqual([]);

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.incidents).toHaveLength(1);
    expect(result.current.incidents[0].reporter).toEqual({ id: 7, name: 'Alice' });
    expect(result.current.total).toBe(1);
    expect(result.current.loadError).toBe(false);
  });

  it('sets loadError=true on failure', async () => {
    (IncidentAPI.listIncidents as jest.Mock).mockRejectedValue(new Error('boom'));
    (UserApi.getUsers as jest.Mock).mockResolvedValue({ users: [] });

    const { result } = renderHook(() => useIncidentsQuery({ search: '' }), { wrapper });

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.loadError).toBe(true);
    expect(result.current.incidents).toEqual([]);
  });

  it('still returns data when user-name enrichment fails', async () => {
    (IncidentAPI.listIncidents as jest.Mock).mockResolvedValue({
      incidents: [mockIncident({ reporterId: 7 })],
      total: 1,
    });
    (UserApi.getUsers as jest.Mock).mockRejectedValue(new Error('users down'));

    const { result } = renderHook(() => useIncidentsQuery({ search: '' }), { wrapper });

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.incidents).toHaveLength(1);
    expect(result.current.incidents[0].reporter).toBeUndefined();
    expect(result.current.loadError).toBe(false);
  });

  it('refresh() re-fetches without losing existing data on error', async () => {
    (IncidentAPI.listIncidents as jest.Mock)
      .mockResolvedValueOnce({ incidents: [mockIncident()], total: 1 })
      .mockRejectedValueOnce(new Error('boom'));

    (UserApi.getUsers as jest.Mock).mockResolvedValue({ users: [] });

    const { result } = renderHook(() => useIncidentsQuery({ search: '' }), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.incidents).toHaveLength(1);

    await act(async () => {
      await result.current.refresh();
    });

    expect(result.current.loadError).toBe(true);
    expect(result.current.incidents).toHaveLength(1);
  });

  it('refetches when filter inputs change', async () => {
    (IncidentAPI.listIncidents as jest.Mock).mockResolvedValue({ incidents: [], total: 0 });
    (UserApi.getUsers as jest.Mock).mockResolvedValue({ users: [] });

    const { result, rerender } = renderHook(
      ({ status }: { status?: string }) => useIncidentsQuery({ search: '', status }),
      { wrapper, initialProps: { status: undefined } as { status?: string } }
    );

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(IncidentAPI.listIncidents).toHaveBeenCalledTimes(1);

    rerender({ status: 'open' });
    await waitFor(() => expect(IncidentAPI.listIncidents).toHaveBeenCalledTimes(2));
    expect(IncidentAPI.listIncidents).toHaveBeenLastCalledWith(
      expect.objectContaining({ status: 'open' })
    );
  });

  it('setPage() updates page and pageSize', async () => {
    (IncidentAPI.listIncidents as jest.Mock).mockResolvedValue({ incidents: [], total: 0 });
    (UserApi.getUsers as jest.Mock).mockResolvedValue({ users: [] });

    const { result } = renderHook(() => useIncidentsQuery({ search: '' }), { wrapper });
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => result.current.setPage(3, 25));
    expect(result.current.page).toBe(3);
    expect(result.current.pageSize).toBe(25);
  });
});
