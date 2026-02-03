/**
 * 仪表盘页面测试
 * 测试仪表盘的核心功能
 */

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
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
  useSearchParams: () => new URLSearchParams(),
}));

// Mock Ant Design components
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
    Spin: ({ children, tip }: { children?: React.ReactNode; tip?: string }) => (
      <div data-testid='spin'>
        {tip || 'Loading...'}
        {children}
      </div>
    ),
    Skeleton: Object.assign(
      ({ active }: { active?: boolean }) => <div data-testid='skeleton' data-active={active} />,
      {
        Button: () => <div data-testid='skeleton-button' />,
        Avatar: () => <div data-testid='skeleton-avatar' />,
        Input: () => <div data-testid='skeleton-input' />,
        Image: () => <div data-testid='skeleton-image' />,
      }
    ),
    Card: ({ children, className }: { children?: React.ReactNode; className?: string }) => (
      <div data-testid='card' className={className}>
        {children}
      </div>
    ),
  };
});

// Mock dashboard hook with flexible return
let mockDashboardData = {
  kpiMetrics: [
    {
      id: 'total-tickets',
      title: '总工单数',
      value: 1250,
      unit: '个',
      color: '#3b82f6',
      trend: 'up',
      change: 12.5,
      changeType: 'increase',
      description: '总工单数',
    },
    {
      id: 'pending-tickets',
      title: '待处理工单',
      value: 320,
      unit: '个',
      color: '#f59e0b',
      trend: 'down',
      change: 8.3,
      changeType: 'decrease',
      description: '待处理',
    },
  ],
  ticketTrend: [],
  incidentDistribution: [],
  slaData: [],
  satisfactionData: [],
  quickActions: [],
  recentActivities: [],
};

let mockLoading = false;
let mockError: Error | null = null;

jest.mock('@/app/(main)/dashboard/hooks/useDashboardData', () => ({
  useDashboardData: () => ({
    data: mockLoading ? null : mockError ? null : mockDashboardData,
    loading: mockLoading,
    error: mockError ? mockError.message : null,
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
jest.mock('@/lib/store/auth-store', () => ({
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

// Mock Chart components
jest.mock('@/app/(main)/dashboard/components/TicketTrendChart', () => ({
  __esModule: true,
  default: () => <div data-testid='ticket-trend-chart'>TicketTrendChart</div>,
}));
jest.mock('@/app/(main)/dashboard/components/IncidentDistributionChart', () => ({
  __esModule: true,
  default: () => <div data-testid='incident-distribution-chart'>IncidentDistributionChart</div>,
}));
jest.mock('@/app/(main)/dashboard/components/SLAComplianceChart', () => ({
  __esModule: true,
  default: () => <div data-testid='sla-compliance-chart'>SLAComplianceChart</div>,
}));
jest.mock('@/app/(main)/dashboard/components/UserSatisfactionChart', () => ({
  __esModule: true,
  default: () => <div data-testid='user-satisfaction-chart'>UserSatisfactionChart</div>,
}));
jest.mock('@/app/(main)/dashboard/components/ResponseTimeChart', () => ({
  __esModule: true,
  default: () => <div data-testid='response-time-chart'>ResponseTimeChart</div>,
}));
jest.mock('@/app/(main)/dashboard/components/TeamWorkloadChart', () => ({
  __esModule: true,
  default: () => <div data-testid='team-workload-chart'>TeamWorkloadChart</div>,
}));
jest.mock('@/app/(main)/dashboard/components/PeakHoursChart', () => ({
  __esModule: true,
  default: () => <div data-testid='peak-hours-chart'>PeakHoursChart</div>,
}));
// KPICards is NOT mocked to test value rendering
jest.mock('@/app/(main)/dashboard/components/ChartsSection', () => ({
  ChartsSection: ({ children }: { children: React.ReactNode }) => (
    <div data-testid='charts-section'>{children}</div>
  ),
}));
jest.mock('@/app/(main)/dashboard/components/QuickActions', () => ({
  QuickActions: () => <div data-testid='quick-actions'>QuickActions</div>,
}));

// Mock Lucide icons
jest.mock('lucide-react', () => ({
  RefreshCw: () => <div data-testid='icon-refresh' />,
  Settings: () => <div data-testid='icon-settings' />,
  LayoutDashboard: () => <div data-testid='icon-dashboard' />,
  Zap: () => <div data-testid='icon-zap' />,
  LineChart: () => <div data-testid='icon-line-chart' />,
  TrendingUp: () => <div data-testid='icon-trending-up' />,
  ArrowUp: () => <div data-testid='icon-arrow-up' />,
  ArrowDown: () => <div data-testid='icon-arrow-down' />,
  Minus: () => <div data-testid='icon-minus' />,
  Clock: () => <div data-testid='icon-clock' />,
  CheckCircle: () => <div data-testid='icon-check-circle' />,
  AlertTriangle: () => <div data-testid='icon-alert-triangle' />,
  User: () => <div data-testid='icon-user' />,
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

import DashboardPage from '@/app/(main)/dashboard/page';

describe('DashboardPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Reset mock states
    mockLoading = false;
    mockError = null;
    mockDashboardData = {
      kpiMetrics: [
        {
          id: 'total-tickets',
          title: '总工单数',
          value: 1250,
          unit: '个',
          color: '#3b82f6',
          trend: 'up',
          change: 12.5,
          changeType: 'increase',
          description: '总工单数',
        },
        {
          id: 'pending-tickets',
          title: '待处理工单',
          value: 320,
          unit: '个',
          color: '#f59e0b',
          trend: 'down',
          change: 8.3,
          changeType: 'decrease',
          description: '待处理',
        },
      ],
      ticketTrend: [],
      incidentDistribution: [],
      slaData: [],
      satisfactionData: [],
      quickActions: [],
      recentActivities: [],
    };
  });

  describe('页面渲染', () => {
    it('应该正确渲染仪表盘页面', async () => {
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('card')).toBeInTheDocument();
      });
    });

    it('应该显示仪表盘标题', async () => {
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByText('ITSM 运营仪表盘')).toBeInTheDocument();
      });
    });
  });

  describe('KPI 指标卡片', () => {
    it('应该显示总工单数', async () => {
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        // Match 1,250 (formatted) or 1250 (raw)
        expect(screen.getByText(/1,250|1250/)).toBeInTheDocument();
      });
    });

    it('应该显示未解决工单数', async () => {
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByText(/320/)).toBeInTheDocument();
      });
    });
  });

  describe('加载状态', () => {
    it('应该显示加载状态', async () => {
      // Set loading state before rendering
      mockLoading = true;

      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getAllByTestId('skeleton').length).toBeGreaterThan(0);
      });
    });
  });

  describe('错误处理', () => {
    it('应该显示错误信息', async () => {
      // Set error state before rendering
      mockError = new Error('Failed to fetch dashboard data');

      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(
          screen.queryByText(/错误/i) || screen.queryByText(/Error/i) || screen.queryByText(/失败/i)
        ).toBeTruthy();
      });
    });
  });
});
