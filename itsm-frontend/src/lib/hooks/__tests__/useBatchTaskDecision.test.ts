import { renderHook, act, waitFor } from '@testing-library/react';
import { useBatchTaskDecision, type BatchDecisionResult } from '../useBatchTaskDecision';

const payload = { action: 'approve' as const, comment: '' };

describe('useBatchTaskDecision', () => {
  it('reports full success when every decision resolves', async () => {
    const submit = jest.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() => useBatchTaskDecision(submit));

    const outcome: BatchDecisionResult = await act(() => result.current.runBatch([1, 2, 3], payload));

    expect(submit).toHaveBeenCalledTimes(3);
    expect(outcome).toEqual({ total: 3, successCount: 3, failedIds: [], errors: {} });
  });

  it('keeps per-id detail on partial failure (Pattern D)', async () => {
    const submit = jest.fn((id: number) =>
      id === 2 ? Promise.reject(new Error('conflict')) : Promise.resolve()
    );
    const { result } = renderHook(() => useBatchTaskDecision(submit));

    const outcome = await act(() => result.current.runBatch([1, 2, 3], payload));

    expect(outcome.successCount).toBe(2);
    expect(outcome.failedIds).toEqual([2]);
    expect(outcome.errors[2]).toBe('conflict');
  });

  it('reports all ids failed when every decision rejects', async () => {
    const submit = jest.fn().mockRejectedValue(new Error('unavailable'));
    const { result } = renderHook(() => useBatchTaskDecision(submit));

    const outcome = await act(() => result.current.runBatch([1, 2], payload));

    expect(outcome.successCount).toBe(0);
    expect([...outcome.failedIds].sort()).toEqual([1, 2]);
  });

  it('caps in-flight requests at 5 (Pattern E)', async () => {
    let active = 0;
    let peak = 0;
    const submit = jest.fn(async () => {
      active += 1;
      peak = Math.max(peak, active);
      await new Promise((r) => setTimeout(r, 10));
      active -= 1;
    });
    const { result } = renderHook(() => useBatchTaskDecision(submit));

    await act(() => result.current.runBatch(Array.from({ length: 20 }, (_, i) => i + 1), payload));

    expect(submit).toHaveBeenCalledTimes(20);
    expect(peak).toBe(5);
  });

  it('returns an empty result for an empty selection without calling submit', async () => {
    const submit = jest.fn();
    const { result } = renderHook(() => useBatchTaskDecision(submit));

    const outcome = await act(() => result.current.runBatch([], payload));

    expect(submit).not.toHaveBeenCalled();
    expect(outcome.total).toBe(0);
    expect(outcome.successCount).toBe(0);
  });

  it('exposes running state during execution and a stable runBatch identity', async () => {
    let release: () => void = () => {};
    const submit = jest.fn(() => new Promise<void>((r) => { release = r; }));
    const { result } = renderHook(() => useBatchTaskDecision(submit));
    const firstIdentity = result.current.runBatch;

    let pending: Promise<BatchDecisionResult> = Promise.resolve({
      total: 0, successCount: 0, failedIds: [], errors: {},
    });
    act(() => {
      pending = result.current.runBatch([1], payload);
    });
    await waitFor(() => expect(result.current.running).toBe(true));

    release();
    await act(() => pending);
    expect(result.current.running).toBe(false);
    expect(result.current.runBatch).toBe(firstIdentity);
  });
});