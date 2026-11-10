import { renderHook, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';

import { useIncidentStats } from '../useIncidentStats';
import { IncidentAPI } from '@/lib/api/incident-api';

jest.mock('@/lib/api/incident-api', () => ({
  IncidentAPI: {
    getIncidentMetrics: jest.fn(),
  },
}));

const wrapper = ({ children }: { children: React.ReactNode }) =>
  React.createElement(App, null, children);

describe('useIncidentStats', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('returns an empty stats array before data arrives', () => {
    (IncidentAPI.getIncidentMetrics as jest.Mock).mockReturnValue(new Promise(() => {}));
    const { result } = renderHook(() => useIncidentStats(), { wrapper });

    expect(result.current.stats).toEqual([]);
    expect(result.current.loading).toBe(true);
    expect(result.current.metrics).toBeNull();
  });

  it('maps incident metrics to PageStats with icons', async () => {
    (IncidentAPI.getIncidentMetrics as jest.Mock).mockResolvedValue({
      totalIncidents: 42,
      openIncidents: 10,
      criticalIncidents: 3,
      avgResolutionTime: 7200, // 120 minutes
    });

    const { result } = renderHook(() => useIncidentStats(), { wrapper });

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.stats).toHaveLength(4);
    expect(result.current.stats[0]).toMatchObject({ label: "Total Incidents", value: 42 });
    expect(result.current.stats[1]).toMatchObject({ label: "Pending", value: 10 });
    expect(result.current.stats[2]).toMatchObject({ label: "Urgent", value: 3 });
    expect(result.current.stats[3]).toMatchObject({
      label: "Avg Resolution Time",
      value: 120,
      suffix: "minutes",
    });
    // Every entry has an icon
    expect(result.current.stats.every(s => s.icon !== undefined)).toBe(true);
  });

  it('falls back to empty stats when the API fails', async () => {
    (IncidentAPI.getIncidentMetrics as jest.Mock).mockRejectedValue(new Error('boom'));

    const { result } = renderHook(() => useIncidentStats(), { wrapper });

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.stats).toEqual([]);
  });
});
