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
      <div data-testid="spin">{tip || 'Loading...'}{children}</div>
    ),
    Card: ({ children, className }: { children?: React.ReactNode; className?: string }) => (
      <div data-testid="card" className={className}>{children}</div>
    ),
  };
});

// Mock dashboard hook with flexible return
let mockDashboardData = {
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
};

let mockLoading = false;
let mockError: Error | null = null;

jest.mock('@/lib/hooks/useDashboardData', () => ({
  useDashboardData: () => ({
    data: mockLoading ? null : mockError ? null : mockDashboardData,
    loading: mockLoading,
    error: mockError,
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
    // Reset mock states
    mockLoading = false;
    mockError = null;
    mockDashboardData = {
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
      ],
    };
  });

  describe('页面渲染', () => {
    it('应该正确渲染仪表盘页面', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('card')).toBeInTheDocument();
      });
    });

    it('应该显示欢迎信息', async () => {
      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
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
  });

  describe('加载状态', () => {
    it('应该显示加载状态', async () => {
      // Set loading state before rendering
      mockLoading = true;

      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(screen.getByTestId('spin')).toBeInTheDocument();
      });
    });
  });

  describe('错误处理', () => {
    it('应该显示错误信息', async () => {
      // Set error state before rendering
      mockError = new Error('Failed to fetch dashboard data');

      const DashboardPage = (await import('@/app/(main)/dashboard/page')).default;
      renderWithProviders(<DashboardPage />);

      await waitFor(() => {
        expect(
          screen.queryByText(/错误/i) || screen.queryByText(/Error/i) || screen.queryByText(/失败/i)
        ).toBeTruthy();
      });
    });
  });
});
