/**
 * useSLAStatistics - SLA 统计钩子测试
 */

import { renderHook, act } from '@testing-library/react';
import { useSLAStatistics } from '../useSLAStatistics';
import type { SLAStatistics, SLAMetric, SLAPolicy } from '../types';

// Mock API
const mockFetchStatistics = jest.fn();
const mockFetchMetrics = jest.fn();
const mockFetchPolicies = jest.fn();

const mockAPI = {
  fetchStatistics: mockFetchStatistics,
  fetchMetrics: mockFetchMetrics,
  fetchPolicies: mockFetchPolicies,
};

describe('useSLAStatistics', () => {
  const mockStats: SLAStatistics = {
    totalPolicies: 5,
    activePolicies: 4,
    totalViolations: 25,
    openViolations: 8,
    resolvedViolations: 17,
    avgResponseTime: 45,
    avgResolutionTime: 120,
    complianceRate: 87.5,
    trend: 'improving',
    updatedAt: '2024-01-20T10:00:00Z',
  };

  const mockMetrics: SLAMetric[] = [
    {
      id: 'metric-1',
      name: 'Response Time',
      value: 45,
      target: 60,
      unit: 'minutes',
      compliance: 85,
    },
    {
      id: 'metric-2',
      name: 'Resolution Time',
      value: 120,
      target: 120,
      unit: 'minutes',
      compliance: 92,
    },
  ];

  const mockPolicies: SLAPolicy[] = [
    {
      id: 'policy-1',
      name: 'Critical Response',
      metric: 'response_time',
      target: 30,
      appliesTo: ['incident'],
    },
    {
      id: 'policy-2',
      name: 'Standard Resolution',
      metric: 'resolution_time',
      target: 240,
      appliesTo: ['incident', 'problem'],
    },
  ];

  beforeEach(() => {
    jest.clearAllMocks();
    mockFetchStatistics.mockResolvedValue(mockStats);
    mockFetchMetrics.mockResolvedValue(mockMetrics);
    mockFetchPolicies.mockResolvedValue(mockPolicies);
  });

  it('应该加载统计数据', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadStatistics();
    });

    expect(result.current.statistics).toEqual(mockStats);
    expect(result.current.loading).toBe(false);
  });

  it('应该加载指标数据', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadMetrics();
    });

    expect(result.current.metrics).toHaveLength(2);
    expect(result.current.metrics[0].name).toBe('Response Time');
  });

  it('应该加载策略数据', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadPolicies();
    });

    expect(result.current.policies).toHaveLength(2);
  });

  it('应该同时加载所有数据', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadAll();
    });

    expect(mockFetchStatistics).toHaveBeenCalled();
    expect(mockFetchMetrics).toHaveBeenCalled();
    expect(mockFetchPolicies).toHaveBeenCalled();
  });

  it('应该计算总体合规率', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadStatistics();
    });

    expect(result.current.complianceRate).toBe(87.5);
  });

  it('应该返回值低于目标的指标', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadMetrics();
    });

    const failingMetrics = result.current.getFailingMetrics();
    expect(failingMetrics).toHaveLength(0); // 所有指标都合规
  });

  it('应该返回值高于目标的指标（时间类）', async () => {
    const failingStats: SLAStatistics = {
      ...mockStats,
      avgResponseTime: 75, // 超过目标 60
      avgResolutionTime: 150, // 超过目标 120
    };

    mockFetchStatistics.mockResolvedValue(failingStats);

    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadStatistics();
    });

    expect(result.current.statistics.avgResponseTime).toBe(75);
    expect(result.current.statistics.avgResolutionTime).toBe(150);
  });

  it('应该获取趋势数据', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadStatistics();
    });

    expect(result.current.trend).toBe('improving');
  });

  it('应该计算每个策略的违规率', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadAll();
    });

    const policyStats = result.current.getPolicyStatistics();
    expect(policyStats).toBeDefined();
    expect(policyStats.length).toBeGreaterThan(0);
  });

  it('应该获取最差性能的指标', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadMetrics();
    });

    const worst = result.current.getWorstPerformingMetric();
    expect(worst).toBeDefined();
    expect(worst.compliance).toBeLessThanOrEqual(100);
  });

  it('应该处理加载错误', async () => {
    mockFetchStatistics.mockRejectedValue(new Error('API error'));

    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadStatistics();
    });

    expect(result.current.error).toBe('API error');
    expect(result.current.loading).toBe(false);
  });

  it('应该刷新统计数据', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadStatistics();
    });

    await act(async () => {
      await result.current.refresh();
    });

    expect(mockFetchStatistics).toHaveBeenCalledTimes(2);
  });

  it('应该导出统计数据', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadAll();
    });

    const exportData = result.current.exportStatistics();
    expect(exportData).toBeDefined();
    expect(typeof exportData).toBe('object');
  });

  it('应该获取策略详情', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadPolicies();
    });

    const policy = result.current.getPolicyById('policy-1');
    expect(policy).toBeDefined();
    expect(policy.name).toBe('Critical Response');
  });

  it('应该支持时间范围过滤', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadStatistics();
    });

    const filtered = result.current.getTimeRangeStatistics('7d');
    expect(filtered).toBeDefined();
  });

  it('应该计算改进建议', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadAll();
    });

    const recommendations = result.current.getRecommendations();
    expect(Array.isArray(recommendations)).toBe(true);
    expect(recommendations.length).toBeGreaterThan(0);
  });

  it('应该显示数据最后更新时间', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadStatistics();
    });

    expect(result.current.lastUpdated).toBeDefined();
    expect(result.current.lastUpdated).toContain('2024');
  });

  it('应该判断是否需要关注', async () => {
    const poorStats: SLAStatistics = {
      ...mockStats,
      complianceRate: 65,
      trend: 'declining',
    };

    mockFetchStatistics.mockResolvedValue(poorStats);

    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadStatistics();
    });

    expect(result.current.needsAttention).toBe(true);
  });

  it('应该获取关键指标摘要', async () => {
    const { result } = renderHook(() =>
      useSLAStatistics(mockAPI)
    );

    await act(async () => {
      await result.current.loadAll();
    });

    const summary = result.current.getSummary();
    expect(summary).toHaveProperty('complianceRate');
    expect(summary).toHaveProperty('totalViolations');
    expect(summary).toHaveProperty('trend');
  });
});
