/**
 * ChangeList Component Tests
 * Tests for the change list component including rendering, loading, pagination, and filtering
 */

import React from 'react';
import { render, screen, waitFor, fireEvent, within } from '@/lib/test-utils';
import userEvent from '@testing-library/user-event';

// Mock next/navigation first
const mockPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush }),
  usePathname: () => '/changes',
  useSearchParams: () => new URLSearchParams(),
}));

// Mock ChangeApi
jest.mock('@/lib/api/', () => ({
  ChangeApi: {
    getChanges: jest.fn().mockResolvedValue({ changes: [], total: 0 }),
    deleteChange: jest.fn(),
  },
}));

// Mock dayjs
jest.mock('dayjs', () => {
  const mockDate = {
    format: jest.fn(() => '2024-01-01 12:00'),
    isValid: () => true,
  };
  const mockDayjs: any = jest.fn(() => mockDate);
  mockDayjs.extend = jest.fn();
  mockDayjs.locale = jest.fn();
  return mockDayjs;
});

// Import after mocks
import { ChangeApi } from '@/lib/api/';
import { ChangeStatus, ChangeType, ChangePriority } from '@/constants/change';

// Test data
const mockChanges = [
  {
    id: 1,
    title: 'Server Upgrade',
    description: 'Upgrade production server',
    justification: 'Performance improvement',
    type: ChangeType.NORMAL,
    status: ChangeStatus.DRAFT,
    priority: ChangePriority.HIGH,
    impactScope: 'high' as const,
    riskLevel: 'medium' as const,
    createdBy: 1,
    createdByName: 'Admin User',
    tenantId: 1,
    implementationPlan: 'Step by step plan',
    rollbackPlan: 'Rollback procedure',
    affectedCis: [],
    relatedTickets: [],
    createdAt: '2024-01-01T10:00:00Z',
    updatedAt: '2024-01-01T10:00:00Z',
  },
  {
    id: 2,
    title: 'Database Migration',
    description: 'Migrate database to new server',
    justification: 'Scalability improvement',
    type: ChangeType.STANDARD,
    status: ChangeStatus.PENDING,
    priority: ChangePriority.MEDIUM,
    impactScope: 'medium' as const,
    riskLevel: 'high' as const,
    createdBy: 2,
    createdByName: 'John Doe',
    tenantId: 1,
    implementationPlan: 'Migration plan',
    rollbackPlan: 'Rollback procedure',
    affectedCis: [],
    relatedTickets: [],
    createdAt: '2024-01-02T10:00:00Z',
    updatedAt: '2024-01-02T10:00:00Z',
  },
  {
    id: 3,
    title: 'Emergency Patch',
    description: 'Apply security patch',
    justification: 'Security fix',
    type: ChangeType.EMERGENCY,
    status: ChangeStatus.APPROVED,
    priority: ChangePriority.CRITICAL,
    impactScope: 'high' as const,
    riskLevel: 'high' as const,
    createdBy: 1,
    createdByName: 'Admin User',
    tenantId: 1,
    implementationPlan: 'Patch procedure',
    rollbackPlan: 'Rollback procedure',
    affectedCis: [],
    relatedTickets: [],
    createdAt: '2024-01-03T10:00:00Z',
    updatedAt: '2024-01-03T10:00:00Z',
  },
];

const mockChangeListResponse = {
  changes: mockChanges,
  total: 3,
};

// Dynamic import to ensure mocks are applied before component loads
let ChangeList: React.FC<any>;

beforeAll(async () => {
  // Import the component after mocks are set up
  const module = await import('../ChangeList');
  ChangeList = module.default;
});

function renderChangeList(ui: React.ReactElement) {
  try {
    return render(ui);
  } catch (error) {
    if (error instanceof AggregateError) {
      const details = (error.errors || [])
        .map(e => (e instanceof Error ? e.stack || e.message : String(e)))
        .join('\n\n');
      throw new Error(details || 'AggregateError');
    }
    throw error;
  }
}

describe('ChangeList', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (ChangeApi.getChanges as jest.Mock).mockResolvedValue(mockChangeListResponse);
  });

  describe('List Rendering', () => {
    it('renders change list with data', async () => {
      // Suppress console errors for cleaner output
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('Server Upgrade')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );

        expect(screen.getByText('Database Migration')).toBeInTheDocument();
        expect(screen.getByText('Emergency Patch')).toBeInTheDocument();
      } finally {
        spy.mockRestore();
      }
    });

    it('renders table columns correctly', async () => {
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('Server Upgrade')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );

        // Check column headers
        const tableHeader = document.querySelector('.ant-table thead');
        expect(tableHeader).toBeTruthy();
        const headerQueries = within(tableHeader as HTMLElement);
        expect(headerQueries.getByText('ID')).toBeInTheDocument();
        expect(headerQueries.getByText('标题')).toBeInTheDocument();
        expect(headerQueries.getByText('模型')).toBeInTheDocument();
        expect(headerQueries.getByText('状态')).toBeInTheDocument();
        expect(headerQueries.getByText('优先级')).toBeInTheDocument();
        expect(headerQueries.getByText('创建人')).toBeInTheDocument();
        expect(headerQueries.getByText('创建时间')).toBeInTheDocument();
      } finally {
        spy.mockRestore();
      }
    });

    it('displays correct status labels', async () => {
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('草稿')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );

        expect(screen.getByText('待审批')).toBeInTheDocument();
        expect(screen.getByText('已批准')).toBeInTheDocument();
      } finally {
        spy.mockRestore();
      }
    });

    it('displays correct type labels', async () => {
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('普通变更')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );

        expect(screen.getByText('标准变更')).toBeInTheDocument();
        expect(screen.getByText('紧急变更')).toBeInTheDocument();
      } finally {
        spy.mockRestore();
      }
    });

    it('displays correct priority labels', async () => {
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('高')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );

        expect(screen.getByText('中')).toBeInTheDocument();
        expect(screen.getByText('极高')).toBeInTheDocument();
      } finally {
        spy.mockRestore();
      }
    });

    it('renders empty state when no data', async () => {
      (ChangeApi.getChanges as jest.Mock).mockResolvedValue({
        changes: [],
        total: 0,
      });

      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('暂无变更记录')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );

        expect(screen.getByText('创建第一个变更')).toBeInTheDocument();
      } finally {
        spy.mockRestore();
      }
    });

    it('renders header with title and description', async () => {
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('变更管理')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );

        expect(screen.getByText(/管理IT基础架构和服务的变更请求/)).toBeInTheDocument();
      } finally {
        spy.mockRestore();
      }
    });

    it('hides header when showHeader is false', async () => {
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList showHeader={false} />);

        await waitFor(
          () => {
            expect(screen.queryByText('变更管理')).not.toBeInTheDocument();
          },
          { timeout: 5000 }
        );
      } finally {
        spy.mockRestore();
      }
    });

    it('renders new change button', async () => {
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('新建变更')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );
      } finally {
        spy.mockRestore();
      }
    });
  });

  describe('Loading State', () => {
    it('shows loading spinner while fetching data', async () => {
      let resolvePromise: (value: unknown) => void;
      const loadingPromise = new Promise(resolve => {
        resolvePromise = resolve;
      });
      (ChangeApi.getChanges as jest.Mock).mockReturnValue(loadingPromise);

      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        // Check for loading indicator
        const loadingSpinner = document.querySelector('.ant-spin');
        expect(loadingSpinner).toBeInTheDocument();

        // Resolve the promise
        resolvePromise!(mockChangeListResponse);

        await waitFor(
          () => {
            expect(screen.getByText('Server Upgrade')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );
      } finally {
        spy.mockRestore();
      }
    });

    it('hides loading spinner after data is loaded', async () => {
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('Server Upgrade')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );

        const loadingSpinner = document.querySelector('.ant-spin-spinning');
        expect(loadingSpinner).not.toBeInTheDocument();
      } finally {
        spy.mockRestore();
      }
    });
  });

  describe('Pagination', () => {
    it('displays pagination info', async () => {
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText(/共 3 条记录/)).toBeInTheDocument();
          },
          { timeout: 5000 }
        );
      } finally {
        spy.mockRestore();
      }
    });

    it('calls API with correct page number when pagination changes', async () => {
      const manyChanges = Array.from({ length: 25 }, (_, i) => ({
        ...mockChanges[0],
        id: i + 1,
        title: `Change ${i + 1}`,
      }));

      (ChangeApi.getChanges as jest.Mock).mockResolvedValue({
        changes: manyChanges,
        total: 25,
      });

      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('Change 1')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );

        expect(ChangeApi.getChanges).toHaveBeenCalledWith(
          expect.objectContaining({ page: 1, pageSize: 10 })
        );

        (ChangeApi.getChanges as jest.Mock).mockClear();
        (ChangeApi.getChanges as jest.Mock).mockResolvedValue({
          changes: manyChanges.slice(10, 20),
          total: 25,
        });

        const page2Button = screen.getByTitle('2');
        fireEvent.click(page2Button);

        await waitFor(
          () => {
            expect(ChangeApi.getChanges).toHaveBeenCalledWith(expect.objectContaining({ page: 2 }));
          },
          { timeout: 5000 }
        );
      } finally {
        spy.mockRestore();
      }
    });
  });

  describe('Status Filter', () => {
    it('renders status filter dropdown', async () => {
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('Server Upgrade')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );

        const form = document.querySelector('form.ant-form');
        expect(form).toBeTruthy();
        expect(within(form as HTMLElement).getByText('状态')).toBeInTheDocument();
      } finally {
        spy.mockRestore();
      }
    });

    it('renders search input', async () => {
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('Server Upgrade')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );

        expect(screen.getByPlaceholderText('搜索标题')).toBeInTheDocument();
      } finally {
        spy.mockRestore();
      }
    });
  });

  describe('Actions', () => {
    it('navigates to detail page when view button is clicked', async () => {
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('Server Upgrade')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );

        const viewButtons = screen
          .getAllByRole('button')
          .filter(btn => btn.querySelector('.anticon-eye'));

        fireEvent.click(viewButtons[0]);

        expect(mockPush).toHaveBeenCalledWith('/changes/1');
      } finally {
        spy.mockRestore();
      }
    });

    it('navigates to new change page when new button is clicked', async () => {
      const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

      try {
        renderChangeList(<ChangeList />);

        await waitFor(
          () => {
            expect(screen.getByText('Server Upgrade')).toBeInTheDocument();
          },
          { timeout: 5000 }
        );

        const newButton = screen.getByText('新建变更');
        fireEvent.click(newButton);

        expect(mockPush).toHaveBeenCalledWith('/changes/new');
      } finally {
        spy.mockRestore();
      }
    });
  });
});
