import { renderHook, act, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';

import { useIncidentBatchOps } from '../useIncidentBatchOps';
import { IncidentAPI } from '@/lib/api/incident-api';
import { UserApi } from '@/lib/api/user-api';

jest.mock('@/lib/api/incident-api', () => ({
  IncidentAPI: {
    resolveIncident: jest.fn(),
    closeIncident: jest.fn(),
    deleteIncident: jest.fn(),
    assignIncident: jest.fn(),
  },
}));

jest.mock('@/lib/api/user-api', () => ({
  UserApi: {
    getUsers: jest.fn(),
  },
}));

const wrapper = ({ children }: { children: React.ReactNode }) =>
  React.createElement(App, null, children);

describe('useIncidentBatchOps', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('returns an empty selection by default and 4 batch actions', () => {
    const { result } = renderHook(() => useIncidentBatchOps({ onAfterBatch: jest.fn() }), {
      wrapper,
    });

    expect(result.current.selectedRowKeys).toEqual([]);
    expect(result.current.batchActions).toHaveLength(4);
    expect(result.current.assignModalOpen).toBe(false);
  });

  it('runBatch reports success when all handlers resolve', async () => {
    const onAfterBatch = jest.fn().mockResolvedValue(undefined);
    (IncidentAPI.resolveIncident as jest.Mock).mockResolvedValue({});

    const { result } = renderHook(() => useIncidentBatchOps({ onAfterBatch }), { wrapper });

    await act(async () => {
      await result.current.runBatch(
        [1, 2],
        id => IncidentAPI.resolveIncident(id, { resolution: '已处理' }),
        'OK'
      );
    });

    expect(IncidentAPI.resolveIncident).toHaveBeenCalledTimes(2);
    expect(onAfterBatch).toHaveBeenCalledTimes(1);
    expect(result.current.selectedRowKeys).toEqual([]);
  });

  it('runBatch reports partial failure when some handlers reject', async () => {
    const onAfterBatch = jest.fn().mockResolvedValue(undefined);
    (IncidentAPI.deleteIncident as jest.Mock)
      .mockResolvedValueOnce({})
      .mockRejectedValueOnce(new Error('forbidden'));

    const { result } = renderHook(() => useIncidentBatchOps({ onAfterBatch }), { wrapper });

    await act(async () => {
      await result.current.runBatch([1, 2], id => IncidentAPI.deleteIncident(id), 'Deleted');
    });

    // Selection is cleared even on partial failure — the survivors are gone.
    expect(result.current.selectedRowKeys).toEqual([]);
    expect(onAfterBatch).toHaveBeenCalledTimes(1);
  });

  it('runBatch is a no-op when given an empty selection', async () => {
    const onAfterBatch = jest.fn();
    const { result } = renderHook(() => useIncidentBatchOps({ onAfterBatch }), { wrapper });

    await act(async () => {
      await result.current.runBatch([], jest.fn(), 'noop');
    });

    expect(onAfterBatch).not.toHaveBeenCalled();
  });

  it('openAssignModal lazily fetches the user list and opens the modal', async () => {
    (UserApi.getUsers as jest.Mock).mockResolvedValue({
      users: [
        { id: 1, name: 'Alice', username: 'alice' },
        { id: 2, name: 'Bob', username: 'bob' },
      ],
    });

    const { result } = renderHook(() => useIncidentBatchOps({ onAfterBatch: jest.fn() }), {
      wrapper,
    });

    expect(result.current.assignUserOptions).toEqual([]);

    await act(async () => {
      await result.current.openAssignModal();
    });

    expect(result.current.assignModalOpen).toBe(true);
    expect(result.current.assignUserOptions).toEqual([
      { label: 'Alice', value: 1 },
      { label: 'Bob', value: 2 },
    ]);

    // Subsequent opens do NOT re-fetch
    await act(async () => {
      await result.current.openAssignModal();
    });
    expect(UserApi.getUsers).toHaveBeenCalledTimes(1);
  });
});
