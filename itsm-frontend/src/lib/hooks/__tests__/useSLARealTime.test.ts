import { renderHook, act, waitFor } from '@testing-library/react';
import { useSLARealTime, formatSLARemainingTime, getSLAStatusColor } from '../useSLARealTime';

describe('useSLARealTime', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    // Mock document.visibilityState
    Object.defineProperty(document, 'visibilityState', {
      writable: true,
      value: 'visible',
    });
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('should return initial state', () => {
    const onRefresh = jest.fn().mockResolvedValue(undefined);

    const { result } = renderHook(() => useSLARealTime({ enabled: false }, onRefresh));

    expect(result.current.isRefreshing).toBe(false);
    expect(result.current.lastRefresh).toBeNull();
    expect(result.current.isPaused).toBe(false);
    expect(result.current.isPageVisible).toBe(true);
  });

  it('should calculate interval based on priority', () => {
    const onRefresh = jest.fn().mockResolvedValue(undefined);

    const { result } = renderHook(() =>
      useSLARealTime({ highestPriority: 'urgent', enabled: false }, onRefresh)
    );

    expect(result.current.currentInterval).toBe(5000);
  });

  it('should use 5s interval when hasAtRiskTickets is true', () => {
    const onRefresh = jest.fn().mockResolvedValue(undefined);

    const { result } = renderHook(() =>
      useSLARealTime({ hasAtRiskTickets: true, enabled: false }, onRefresh)
    );

    expect(result.current.currentInterval).toBe(5000);
  });

  it('should use custom interval when provided', () => {
    const onRefresh = jest.fn().mockResolvedValue(undefined);

    const { result } = renderHook(() =>
      useSLARealTime({ customInterval: 15000, enabled: false }, onRefresh)
    );

    expect(result.current.currentInterval).toBe(15000);
  });

  it('should pause and resume', () => {
    const onRefresh = jest.fn().mockResolvedValue(undefined);

    const { result } = renderHook(() => useSLARealTime({ enabled: false }, onRefresh));

    act(() => {
      result.current.pause();
    });
    expect(result.current.isPaused).toBe(true);

    act(() => {
      result.current.resume();
    });
    expect(result.current.isPaused).toBe(false);
  });

  it('should call refreshNow manually', async () => {
    jest.useRealTimers();
    const onRefresh = jest.fn().mockResolvedValue(undefined);

    const { result } = renderHook(() => useSLARealTime({ enabled: false }, onRefresh));

    await act(async () => {
      await result.current.refreshNow();
    });

    expect(onRefresh).toHaveBeenCalled();
    expect(result.current.lastRefresh).not.toBeNull();
  });

  // Pattern A: requestIdRef guard tests (H1 stale-overwrite prevention)
  // Isolated describe block — race tests use real timers exclusively.
});

describe('useSLARealTime requestId guard', () => {
  beforeEach(() => {
    jest.useRealTimers();
    Object.defineProperty(document, 'visibilityState', {
      writable: true,
      value: 'visible',
    });
  });

  it('should NOT overwrite lastRefresh when stale slow response arrives later', async () => {
    let slowResolve!: () => void;
    const slow = new Promise<void>(r => {
      slowResolve = r;
    });
    const onRefresh = jest
      .fn()
      .mockImplementationOnce(() => slow)
      .mockResolvedValue(undefined);

    const { result } = renderHook(() => useSLARealTime({ enabled: false }, onRefresh));

    // First refreshNow: triggers slow onRefresh
    const p1 = result.current.refreshNow();
    // Second refreshNow: triggers fast (mockResolvedValue default)
    const p2 = result.current.refreshNow();

    await act(async () => {
      await p2;
    });

    await waitFor(() => {
      expect(result.current.lastRefresh).not.toBeNull();
    });
    const fastRefresh = result.current.lastRefresh!;

    await act(async () => {
      slowResolve();
      await p1;
    });

    expect(result.current.lastRefresh).toBe(fastRefresh);
  });

  it('should NOT surface stale error after fresh success', async () => {
    let slowReject!: (e: Error) => void;
    const slow = new Promise<void>((_, rj) => {
      slowReject = rj;
    });
    const onRefresh = jest
      .fn()
      .mockImplementationOnce(() => slow)
      .mockResolvedValue(undefined);
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

    const { result } = renderHook(() => useSLARealTime({ enabled: false }, onRefresh));

    const p1 = result.current.refreshNow();
    const p2 = result.current.refreshNow();

    await act(async () => {
      await p2;
    });

    await waitFor(() => {
      expect(result.current.lastRefresh).not.toBeNull();
    });
    const fastRefresh = result.current.lastRefresh!;

    await act(async () => {
      slowReject(new Error('stale sla error'));
      await p1;
    });

    const errorCalls = consoleErrorSpy.mock.calls.filter(
      c => typeof c[0] === 'string' && c[0].includes('stale sla error')
    );
    expect(errorCalls).toHaveLength(0);
    expect(result.current.lastRefresh).toBe(fastRefresh);

    consoleErrorSpy.mockRestore();
  });

  it('should NOT flip isRefreshing off for stale requests', async () => {
    let slowResolve!: () => void;
    const slow = new Promise<void>(r => {
      slowResolve = r;
    });
    const onRefresh = jest
      .fn()
      .mockImplementationOnce(() => slow)
      .mockResolvedValue(undefined);

    const { result } = renderHook(() => useSLARealTime({ enabled: false }, onRefresh));

    const p1 = result.current.refreshNow();
    const p2 = result.current.refreshNow();

    await act(async () => {
      await p2;
    });

    await waitFor(() => {
      expect(result.current.isRefreshing).toBe(false);
    });

    await act(async () => {
      slowResolve();
      await p1;
    });

    expect(result.current.isRefreshing).toBe(false);
  });
});

describe('formatSLARemainingTime', () => {
  it('should format overdue time', () => {
    const result = formatSLARemainingTime(-3600);
    expect(result.isOverdue).toBe(true);
    expect(result.riskLevel).toBe('critical');
    expect(result.text).toContain('超时');
  });

  it('should format less than 30 min as critical', () => {
    const result = formatSLARemainingTime(600); // 10 minutes
    expect(result.riskLevel).toBe('critical');
    expect(result.isOverdue).toBe(false);
  });

  it('should format 30-60 min as warning', () => {
    const result = formatSLARemainingTime(2400); // 40 minutes
    expect(result.riskLevel).toBe('warning');
  });

  it('should format more than 24h as normal', () => {
    const result = formatSLARemainingTime(100000); // > 24h
    expect(result.riskLevel).toBe('normal');
    expect(result.text).toContain('天');
  });
});

describe('getSLAStatusColor', () => {
  it('should return red for critical', () => {
    expect(getSLAStatusColor('critical')).toBe('#cf1322');
  });

  it('should return orange for warning', () => {
    expect(getSLAStatusColor('warning')).toBe('#fa8c16');
  });

  it('should return green for normal', () => {
    expect(getSLAStatusColor('normal')).toBe('#52c41a');
  });
});
