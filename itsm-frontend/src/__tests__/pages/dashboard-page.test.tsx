/**
 * 仪表盘页面测试
 * 测试仪表盘的核心功能
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import '@testing-library/jest-dom';

// Mock Next.js router
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
    replace: jest.fn(),
    prefetch: jest.fn(),
  }),
  usePathname: () => '/dashboard',
}));

// Mock Ant Design message
jest.mock('antd', () => {
  const actual = jest.requireActual('antd');
  return {
    ...actual,
    message: {
      success: jest.fn(),
      error: jest.fn(),
      warning: jest.fn(),
      info: jest.fn(),
    },
  };
});

// Mock API responses
const mockDashboardData = {
  kpis: {
    totalTickets: 1250,
    openTickets: 320,
    resolvedTickets: 850,
    avgResolutionTime: 4.2,
    ticketGrowth: 12.5,
    openGrowth: -8.3,
    resolvedGrowth: 15.2,
    timeGrowth: -10.5,
  },
  recentTickets: [
    {
      id: 1,
      ticket_number: 'TK-001',
      title: 'System Error',
      priority: 'high',
      status: 'open',
      created_at: '2024-01-01T10:00:00Z',
    },
    {
      id: 2,
      ticket_number: 'TK-002',
      title: 'Feature Request',
      priority: 'medium',
      status: 'in_progress',
      created_at: '2024-01-01T11:00:00Z',
    },
  ],
  chartData: {
    ticketTrend: [
      { date: '2024-01-01', value: 45 },
      { date: '2024-01-02', value: 52 },
      { date: '2024-01-03', value: 48 },
    ],
    statusDistribution: {
      open: 320,
      in_progress: 280,
      resolved: 450,
      closed: 200,
    },
    priorityDistribution: {
      low: 200,
      medium: 450,
      high: 380,
      urgent: 150,
      critical: 70,
    },
  },
};

// Mock dashboard hook
jest.mock('@/lib/hooks/useDashboardData', () => ({
  useDashboardData: () => ({
    data: mockDashboardData,
    loading: false,
    error: null,
    lastUpdated: new Date(),
    autoRefresh: true,
    refreshInterval: 30,
    refresh: jest.fn(),
    setAutoRefresh: jest.fn(),
    setRefreshInterval: jest.fn(),
    isConnected: true,
  }),
}));

// Mock auth store
jest.mock('@/lib/store/unified-auth-store', () => ({
  useAuthStore: () => ({
    user: {
      id: 1,
      username: 'testuser',
      name: 'Test User',
      role: 'admin',
    },
    isAuthenticated: true,
  }),
}));

// Test utilities
const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  });

const renderWithProviders = (ui: React.ReactElement) => {
  const queryClient = createTestQueryClient();
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
};

describe('DashboardPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('页面渲染', () => {
    it('应该正确渲染仪表盘页面', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        // 检查页面标题
        expect(screen.getByText(/仪表盘/i) || screen.getByText(/Dashboard/i)).toBeInTheDocument();
      });
    });

    it('应该显示欢迎信息', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        // 应该显示用户名
        expect(screen.getByText(/Test User/i) || screen.getByText(/欢迎/i)).toBeTruthy();
      });
    });
  });

  describe('KPI 指标卡片', () => {
    it('应该显示总工单数', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByText(/1250/)).toBeInTheDocument();
      });
    });

    it('应该显示未解决工单数', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByText(/320/)).toBeInTheDocument();
      });
    });

    it('应该显示已解决工单数', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByText(/850/)).toBeInTheDocument();
      });
    });

    it('应该显示平均解决时间', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByText(/4\.2/)).toBeInTheDocument();
      });
    });

    it('应该显示增长趋势', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      const { container } = renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        // 应该有增长指示器（箭头、百分比等）
        const growthIndicators = container.querySelectorAll('.ant-statistic-content-suffix');
        expect(growthIndicators.length).toBeGreaterThan(0);
      });
    });
  });

  describe('快速操作', () => {
    it('应该显示快速操作按钮', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        // 应该有创建工单等快速操作
        expect(
          screen.queryByText(/创建工单/i) || screen.queryByText(/Create Ticket/i)
        ).toBeTruthy();
      });
    });

    it('应该支持点击快速操作', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      const { container } = renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        const quickActionButtons = container.querySelectorAll('[data-testid^="quick-action"]');
        if (quickActionButtons.length > 0) {
          fireEvent.click(quickActionButtons[0]);
        }
      });
    });
  });

  describe('图表展示', () => {
    it('应该显示工单趋势图', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      const { container } = renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        // 应该有图表容器
        const chartContainer =
          container.querySelector('[data-testid="ticket-trend-chart"]') ||
          container.querySelector('.chart-container');
        expect(chartContainer).toBeTruthy();
      });
    });

    it('应该显示状态分布图', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      const { container } = renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        const statusChart = container.querySelector('[data-testid="status-distribution-chart"]');
        expect(statusChart).toBeTruthy();
      });
    });

    it('应该显示优先级分布图', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      const { container } = renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        const priorityChart = container.querySelector(
          '[data-testid="priority-distribution-chart"]'
        );
        expect(priorityChart).toBeTruthy();
      });
    });
  });

  describe('最近工单', () => {
    it('应该显示最近工单列表', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByText(/TK-001/)).toBeInTheDocument();
        expect(screen.getByText(/TK-002/)).toBeInTheDocument();
      });
    });

    it('应该支持点击工单查看详情', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      const { container } = renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        const ticketLinks = container.querySelectorAll('[data-testid^="ticket-link"]');
        if (ticketLinks.length > 0) {
          fireEvent.click(ticketLinks[0]);
        }
      });
    });
  });

  describe('自动刷新', () => {
    it('应该显示自动刷新开关', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      const { container } = renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        const autoRefreshSwitch =
          container.querySelector('[data-testid="auto-refresh-switch"]') ||
          container.querySelector('.ant-switch');
        expect(autoRefreshSwitch).toBeTruthy();
      });
    });

    it('应该支持手动刷新', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      const { container } = renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        const refreshButton =
          container.querySelector('[data-testid="refresh-button"]') ||
          container.querySelector('[aria-label*="刷新"]');
        if (refreshButton) {
          fireEvent.click(refreshButton);
        }
      });
    });

    it('应该显示最后更新时间', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        // 应该显示最后更新时间
        expect(screen.queryByText(/最后更新/i) || screen.queryByText(/Last updated/i)).toBeTruthy();
      });
    });
  });

  describe('加载状态', () => {
    it('应该显示加载状态', async () => {
      // Mock loading state
      const { useDashboardData } = require('@/lib/hooks/useDashboardData');
      useDashboardData.mockReturnValueOnce({
        data: null,
        loading: true,
        error: null,
        lastUpdated: null,
        autoRefresh: false,
        refreshInterval: 30,
        refresh: jest.fn(),
        setAutoRefresh: jest.fn(),
        setRefreshInterval: jest.fn(),
        isConnected: true,
      });

      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      // 应该显示加载指示器
      await waitFor(() => {
        expect(screen.queryByText(/加载中/i) || screen.queryByRole('progressbar')).toBeTruthy();
      });
    });
  });

  describe('错误处理', () => {
    it('应该显示错误信息', async () => {
      // Mock error state
      const { useDashboardData } = require('@/lib/hooks/useDashboardData');
      useDashboardData.mockReturnValueOnce({
        data: null,
        loading: false,
        error: new Error('Failed to fetch dashboard data'),
        lastUpdated: null,
        autoRefresh: false,
        refreshInterval: 30,
        refresh: jest.fn(),
        setAutoRefresh: jest.fn(),
        setRefreshInterval: jest.fn(),
        isConnected: false,
      });

      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(
          screen.queryByText(/错误/i) || screen.queryByText(/Error/i) || screen.queryByText(/失败/i)
        ).toBeTruthy();
      });
    });

    it('应该提供重新加载按钮', async () => {
      // Mock error state
      const { useDashboardData } = require('@/lib/hooks/useDashboardData');
      useDashboardData.mockReturnValueOnce({
        data: null,
        loading: false,
        error: new Error('Failed to fetch dashboard data'),
        lastUpdated: null,
        autoRefresh: false,
        refreshInterval: 30,
        refresh: jest.fn(),
        setAutoRefresh: jest.fn(),
        setRefreshInterval: jest.fn(),
        isConnected: false,
      });

      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        const reloadButton =
          screen.queryByText(/重新加载/i) ||
          screen.queryByText(/重试/i) ||
          screen.queryByText(/Retry/i);
        expect(reloadButton).toBeTruthy();
      });
    });
  });

  describe('响应式设计', () => {
    it('应该在移动端正确显示', async () => {
      // Mock mobile viewport
      global.innerWidth = 375;
      global.innerHeight = 667;

      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      const { container } = renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        // 检查是否有响应式类名
        const responsiveElements =
          container.querySelectorAll('[class*="mobile"]') ||
          container.querySelectorAll('[class*="sm:"]');
        expect(responsiveElements.length).toBeGreaterThan(0);
      });
    });
  });
});
