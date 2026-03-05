/**
 * useSLAViolations - SLA 违规数据钩子测试
 */

import { renderHook, act } from '@testing-library/react';
import { useSLAViolations } from '../useSLAViolations';
import type { SLAViolation, SLAPolicy } from '../types';

// Mock API
const mockFetchViolations = jest.fn();
const mockUpdateViolation = jest.fn();
const mockAcknowledgeViolation = jest.fn();

const mockAPI = {
  fetchViolations: mockFetchViolations,
  updateViolation: mockUpdateViolation,
  acknowledgeViolation: mockAcknowledgeViolation,
};

describe('useSLAViolations', () => {
  const mockViolations: SLAViolation[] = [
    {
      id: 'violation-1',
      ticketId: 'ticket-1',
      policyId: 'policy-1',
      policyName: 'Response Time',
      breachedAt: '2024-01-20T10:00:00Z',
      currentValue: 120,
      targetValue: 60,
      slaMetric: 'response_time',
      status: 'open',
      severity: 'critical',
      assignedTo: 'user1',
      ticket: {
        id: 'ticket-1',
        title: 'Server Down',
        priority: 'critical',
      },
    },
    {
      id: 'violation-2',
      ticketId: 'ticket-2',
      policyId: 'policy-2',
      policyName: 'Resolution Time',
      breachedAt: '2024-01-20T11:00:00Z',
      currentValue: 240,
      targetValue: 120,
      slaMetric: 'resolution_time',
      status: 'acknowledged',
      severity: 'high',
      assignedTo: 'user2',
      ticket: {
        id: 'ticket-2',
        title: 'Database Slow',
        priority: 'high',
      },
    },
  ];

  const mockPolicies: SLAPolicy[] = [
    {
      id: 'policy-1',
      name: 'Response Time',
      metric: 'response_time',
      target: 60,
      unit: 'minutes',
      appliesTo: ['incident', 'problem'],
    },
    {
      id: 'policy-2',
      name: 'Resolution Time',
      metric: 'resolution_time',
      target: 120,
      unit: 'minutes',
      appliesTo: ['incident'],
    },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
    mockFetchViolations.mockResolvedValue(mockViolations);
  });

  it('应该初始加载违规数据', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    expect(result.current.violations).toHaveLength(2);
    expect(result.current.loading).toBe(false);
  });

  it('应该过滤违规状态', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    const openViolations = result.current.getViolationsByStatus('open');
    expect(openViolations).toHaveLength(1);
    expect(openViolations[0].status).toBe('open');
  });

  it('应该按严重程度过滤', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    const criticalViolations = result.current.getViolationsBySeverity('critical');
    expect(criticalViolations).toHaveLength(1);
    expect(criticalViolations[0].severity).toBe('critical');
  });

  it('应该获取违规统计', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    const stats = result.current.getStatistics();
    expect(stats.total).toBe(2);
    expect(stats.open).toBe(1);
    expect(stats.acknowledged).toBe(1);
    expect(stats.critical).toBe(1);
  });

  it('应该更新违规状态', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    await act(async () => {
      await result.current.updateViolationStatus('violation-1', 'resolved');
    });

    expect(mockUpdateViolation).toHaveBeenCalledWith('violation-1', { status: 'resolved' });
  });

  it('应该确认违规', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    await act(async () => {
      await result.current.acknowledgeViolation('violation-2', 'user1');
    });

    expect(mockAcknowledgeViolation).toHaveBeenCalledWith('violation-2', 'user1');
  });

  it('应该加载时显示加载状态', () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    expect(result.current.loading).toBe(false);

    act(() => {
      result.current.loadViolations();
    });

    expect(result.current.loading).toBe(true);
  });

  it('应该处理错误', async () => {
    mockFetchViolations.mockRejectedValue(new Error('Network error'));

    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    expect(result.current.error).toBe('Network error');
    expect(result.current.loading).toBe(false);
  });

  it('应该获取违规详情', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    const violation = result.current.getViolationById('violation-1');
    expect(violation).toBeDefined();
    expect(violation.id).toBe('violation-1');
  });

  it('应该获取不存在的违规详情返回 null', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    const violation = result.current.getViolationById('non-existent');
    expect(violation).toBeNull();
  });

  it('应该按工单 ID 过滤', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    const ticketViolations = result.current.getViolationsByTicketId('ticket-1');
    expect(ticketViolations).toHaveLength(1);
    expect(ticketViolations[0].ticketId).toBe('ticket-1');
  });

  it('应该获取违规趋势数据', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    const trend = result.current.getViolationTrend('day');
    expect(trend).toBeDefined();
    expect(typeof trend).toBe('object');
  });

  it('应该刷新数据', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    await act(async () => {
      await result.current.refresh();
    });

    expect(mockFetchViolations).toHaveBeenCalledTimes(2);
  });

  it('应该分页加载', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    await act(async () => {
      await result.current.loadNextPage();
    });

    expect(mockFetchViolations).toHaveBeenCalledWith(
      expect.objectContaining({
        page: 2,
      })
    );
  });

  it('应该批量确认违规', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    await act(async () => {
      await result.current.batchAcknowledge(['violation-1', 'violation-2'], 'admin');
    });

    expect(mockAcknowledgeViolation).toHaveBeenCalledWith('violation-1', 'admin');
    expect(mockAcknowledgeViolation).toHaveBeenCalledWith('violation-2', 'admin');
  });

  it('应该计算平均违规时长', async () => {
    const { result } = renderHook(() =>
      useSLAViolations(mockAPI)
    );

    await act(async () => {
      await result.current.loadViolations();
    });

    const avgDuration = result.current.getAverageDuration();
    expect(typeof avgDuration).toBe('number');
    expect(avgDuration).toBeGreaterThanOrEqual(0);
  });
});
