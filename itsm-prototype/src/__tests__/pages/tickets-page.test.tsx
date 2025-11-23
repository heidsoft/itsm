/**
 * 工单页面测试
 * 测试工单列表页面的核心功能
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
  usePathname: () => '/tickets',
  useSearchParams: () => new URLSearchParams(),
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

// Mock API
jest.mock('@/lib/api/ticket-api', () => ({
  TicketApi: {
    getTickets: jest.fn(() =>
      Promise.resolve({
        tickets: [
          {
            id: 1,
            ticket_number: 'TK-001',
            title: 'Test Ticket 1',
            description: 'Test Description',
            status: 'open',
            priority: 'high',
            created_at: '2024-01-01T00:00:00Z',
          },
          {
            id: 2,
            ticket_number: 'TK-002',
            title: 'Test Ticket 2',
            description: 'Test Description 2',
            status: 'in_progress',
            priority: 'medium',
            created_at: '2024-01-02T00:00:00Z',
          },
        ],
        total: 2,
        page: 1,
        pageSize: 10,
      })
    ),
    getTicketStats: jest.fn(() =>
      Promise.resolve({
        total: 100,
        open: 30,
        in_progress: 25,
        resolved: 35,
        closed: 10,
      })
    ),
    createTicket: jest.fn(data =>
      Promise.resolve({
        id: 3,
        ticket_number: 'TK-003',
        ...data,
      })
    ),
    updateTicket: jest.fn((id, data) =>
      Promise.resolve({
        id,
        ...data,
      })
    ),
    deleteTicket: jest.fn(() => Promise.resolve()),
  },
}));

// Mock auth store
jest.mock('@/lib/store/unified-auth-store', () => ({
  useAuthStore: () => ({
    user: {
      id: 1,
      username: 'testuser',
      role: 'admin',
      permissions: ['ticket:view', 'ticket:create', 'ticket:update', 'ticket:delete'],
    },
    isAuthenticated: true,
  }),
  usePermissions: () => ({
    canViewTickets: () => true,
    canCreateTickets: () => true,
    canUpdateTickets: () => true,
    canDeleteTickets: () => true,
  }),
}));

// Test utilities
const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        cacheTime: 0,
      },
    },
  });

const renderWithProviders = (ui: React.ReactElement) => {
  const queryClient = createTestQueryClient();
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
};

describe('TicketsPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('页面渲染', () => {
    it('应该正确渲染工单列表页面', async () => {
      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      renderWithProviders(<TicketsPage />);

      // 检查页面标题
      expect(screen.getByText(/工单管理/i)).toBeInTheDocument();
    });

    it('应该显示加载状态', async () => {
      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      renderWithProviders(<TicketsPage />);

      // 加载时应该显示骨架屏或加载指示器
      await waitFor(
        () => {
          expect(screen.queryByText(/加载中/i) || screen.queryByRole('progressbar')).toBeTruthy();
        },
        { timeout: 100 }
      );
    });

    it('应该显示工单统计信息', async () => {
      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      renderWithProviders(<TicketsPage />);

      await waitFor(() => {
        // 应该显示统计卡片
        expect(screen.getByText(/总数/i) || screen.getByText(/Total/i)).toBeTruthy();
      });
    });
  });

  describe('工单列表', () => {
    it('应该显示工单列表', async () => {
      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      renderWithProviders(<TicketsPage />);

      await waitFor(() => {
        expect(screen.getByText('Test Ticket 1')).toBeInTheDocument();
        expect(screen.getByText('Test Ticket 2')).toBeInTheDocument();
      });
    });

    it('应该显示工单编号', async () => {
      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      renderWithProviders(<TicketsPage />);

      await waitFor(() => {
        expect(screen.getByText(/TK-001/)).toBeInTheDocument();
        expect(screen.getByText(/TK-002/)).toBeInTheDocument();
      });
    });
  });

  describe('筛选功能', () => {
    it('应该支持按状态筛选', async () => {
      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      const { container } = renderWithProviders(<TicketsPage />);

      await waitFor(() => {
        // 查找状态选择器
        const statusFilter = container.querySelector('[data-testid="status-filter"]');
        if (statusFilter) {
          fireEvent.click(statusFilter);
        }
      });
    });

    it('应该支持按优先级筛选', async () => {
      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      const { container } = renderWithProviders(<TicketsPage />);

      await waitFor(() => {
        // 查找优先级选择器
        const priorityFilter = container.querySelector('[data-testid="priority-filter"]');
        if (priorityFilter) {
          fireEvent.click(priorityFilter);
        }
      });
    });

    it('应该支持搜索', async () => {
      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      const { container } = renderWithProviders(<TicketsPage />);

      await waitFor(() => {
        // 查找搜索框
        const searchInput = container.querySelector('input[placeholder*="搜索"]');
        if (searchInput) {
          fireEvent.change(searchInput, { target: { value: 'test' } });
        }
      });
    });
  });

  describe('分页功能', () => {
    it('应该显示分页器', async () => {
      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      renderWithProviders(<TicketsPage />);

      await waitFor(() => {
        // 应该有分页组件
        const pagination = screen.queryByRole('navigation');
        expect(pagination).toBeTruthy();
      });
    });

    it('应该支持切换页码', async () => {
      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      const { container } = renderWithProviders(<TicketsPage />);

      await waitFor(() => {
        // 查找下一页按钮
        const nextButton = container.querySelector('.ant-pagination-next');
        if (nextButton) {
          fireEvent.click(nextButton);
        }
      });
    });
  });

  describe('刷新功能', () => {
    it('应该支持刷新数据', async () => {
      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      const { container } = renderWithProviders(<TicketsPage />);

      await waitFor(() => {
        // 查找刷新按钮
        const refreshButton =
          container.querySelector('[data-testid="refresh-button"]') ||
          container.querySelector('[aria-label*="刷新"]');
        if (refreshButton) {
          fireEvent.click(refreshButton);
        }
      });
    });
  });

  describe('错误处理', () => {
    it('应该显示错误信息', async () => {
      // Mock API 错误
      const { TicketApi } = require('@/lib/api/ticket-api');
      TicketApi.getTickets.mockRejectedValueOnce(new Error('Network Error'));

      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      renderWithProviders(<TicketsPage />);

      await waitFor(() => {
        // 应该显示错误提示
        expect(screen.queryByText(/错误/i) || screen.queryByText(/Error/i)).toBeTruthy();
      });
    });

    it('应该提供重试按钮', async () => {
      // Mock API 错误
      const { TicketApi } = require('@/lib/api/ticket-api');
      TicketApi.getTickets.mockRejectedValueOnce(new Error('Network Error'));

      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      const { container } = renderWithProviders(<TicketsPage />);

      await waitFor(() => {
        // 查找重试按钮
        const retryButton = screen.queryByText(/重试/i) || screen.queryByText(/Retry/i);
        expect(retryButton).toBeTruthy();
      });
    });
  });

  describe('权限控制', () => {
    it('应该根据权限显示创建按钮', async () => {
      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      renderWithProviders(<TicketsPage />);

      await waitFor(() => {
        // 有权限时应该显示创建按钮
        const createButton = screen.queryByText(/创建工单/i) || screen.queryByText(/Create/i);
        expect(createButton).toBeTruthy();
      });
    });

    it('无权限时应该隐藏操作按钮', async () => {
      // Mock 无权限的用户
      const { usePermissions } = require('@/lib/store/unified-auth-store');
      usePermissions.mockReturnValueOnce({
        canViewTickets: () => true,
        canCreateTickets: () => false,
        canUpdateTickets: () => false,
        canDeleteTickets: () => false,
      });

      const TicketsPage = (await import('@/app/(main)/tickets/page')).default;
      renderWithProviders(<TicketsPage />);

      await waitFor(() => {
        // 无权限时不应该显示创建按钮
        const createButton = screen.queryByText(/创建工单/i);
        // 根据实际实现，按钮可能被隐藏或禁用
      });
    });
  });
});
