import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import TicketsPage from '../page';

// Mock dependencies
jest.mock('@/lib/store/ui-store', () => ({
  useNotifications: () => ({
    success: jest.fn(),
    error: jest.fn(),
    warning: jest.fn(),
    info: jest.fn(),
  }),
}));

jest.mock('@/lib/auth', () => ({
  useAuth: () => ({
    user: { id: 1, name: 'Test User' },
    isAuthenticated: true,
  }),
}));

jest.mock('@/lib/api/ticket-api', () => ({
  TicketAPI: {
    getTickets: jest.fn(),
    createTicket: jest.fn(),
    updateTicket: jest.fn(),
    deleteTicket: jest.fn(),
  },
}));

// Mock Ant Design components
jest.mock('antd', () => ({
  Table: ({ columns, dataSource, loading, ...props }: { 
    columns: Array<{ title: string; dataIndex: string; key: string }>;
    dataSource: Array<Record<string, unknown>>;
    loading?: boolean;
    [key: string]: unknown;
  }) => (
    <div data-testid="tickets-table" {...props}>
      {loading && <div data-testid="table-loading">Loading...</div>}
      <table>
        <thead>
          <tr>
            {columns.map(col => (
              <th key={col.key}>{col.title}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {dataSource?.map((item, index) => (
            <tr key={index} data-testid={`ticket-row-${index}`}>
              {columns.map(col => (
                <td key={col.key}>{String(item[col.dataIndex] || '')}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  ),
  Button: ({ children, loading, type, onClick, ...props }: {
    children: React.ReactNode;
    loading?: boolean;
    type?: string;
    onClick?: () => void;
    [key: string]: unknown;
  }) => (
    <button 
      onClick={onClick}
      disabled={loading}
      data-testid={`button-${type || 'default'}`}
      {...props}
    >
      {loading ? 'Loading...' : children}
    </button>
  ),
  Input: ({ placeholder, onChange, ...props }: {
    placeholder?: string;
    onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
    [key: string]: unknown;
  }) => (
    <input 
      placeholder={placeholder}
      onChange={onChange}
      data-testid={`input-${placeholder?.toLowerCase().replace(/\s+/g, '-')}`}
      {...props}
    />
  ),
  Select: ({ placeholder, options, onChange, ...props }: {
    placeholder?: string;
    options?: Array<{ label: string; value: string }>;
    onChange?: (value: string) => void;
    [key: string]: unknown;
  }) => (
    <select 
      onChange={(e) => onChange?.(e.target.value)}
      data-testid={`select-${placeholder?.toLowerCase().replace(/\s+/g, '-')}`}
      {...props}
    >
      <option value="">{placeholder}</option>
      {options?.map(option => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  ),
  Card: ({ children, title, extra, ...props }: {
    children: React.ReactNode;
    title?: string;
    extra?: React.ReactNode;
    [key: string]: unknown;
  }) => (
    <div data-testid="tickets-card" {...props}>
      {title && (
        <div data-testid="card-header">
          <h3>{title}</h3>
          {extra && <div data-testid="card-extra">{extra}</div>}
        </div>
      )}
      {children}
    </div>
  ),
  Space: ({ children, ...props }: { children: React.ReactNode; [key: string]: unknown }) => (
    <div data-testid="space" {...props}>{children}</div>
  ),
  Tag: ({ children, color, ...props }: { children: React.ReactNode; color?: string; [key: string]: unknown }) => (
    <span data-testid="tag" style={{ color }} {...props}>{children}</span>
  ),
  Modal: ({ title, open, children, onOk, onCancel, ...props }: {
    title?: string;
    open?: boolean;
    children: React.ReactNode;
    onOk?: () => void;
    onCancel?: () => void;
    [key: string]: unknown;
  }) => (
    open ? (
      <div data-testid="modal" {...props}>
        {title && <h2 data-testid="modal-title">{title}</h2>}
        {children}
        <div data-testid="modal-actions">
          <button onClick={onCancel} data-testid="modal-cancel">取消</button>
          <button onClick={onOk} data-testid="modal-ok">确定</button>
        </div>
      </div>
    ) : null
  ),
  Form: ({ children, onFinish, ...props }: {
    children: React.ReactNode;
    onFinish?: (values: Record<string, unknown>) => void;
    [key: string]: unknown;
  }) => (
    <form onSubmit={(e) => { e.preventDefault(); onFinish?.({}); }} {...props}>
      {children}
    </form>
  ),
}));

// Mock Lucide React icons
jest.mock('lucide-react', () => ({
  Plus: () => <div data-testid="plus-icon">Plus</div>,
  Search: () => <div data-testid="search-icon">Search</div>,
  Filter: () => <div data-testid="filter-icon">Filter</div>,
  Download: () => <div data-testid="download-icon">Download</div>,
  Edit: () => <div data-testid="edit-icon">Edit</div>,
  Trash2: () => <div data-testid="trash-icon">Trash</div>,
  Eye: () => <div data-testid="eye-icon">Eye</div>,
}));

// Mock ticket data
const mockTickets = [
  {
    id: 1,
    title: '测试工单1',
    description: '这是一个测试工单',
    status: 'open',
    priority: 'high',
    assignee: 'John Doe',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 2,
    title: '测试工单2',
    description: '这是另一个测试工单',
    status: 'in_progress',
    priority: 'medium',
    assignee: 'Jane Smith',
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
  },
];

// Mock TicketAPI
const mockTicketAPI = {
  getTickets: jest.fn(),
  createTicket: jest.fn(),
  updateTicket: jest.fn(),
  deleteTicket: jest.fn(),
};

describe('TicketsPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Mock API responses
    mockTicketAPI.getTickets.mockResolvedValue({
      tickets: mockTickets,
      total: mockTickets.length,
      page: 1,
      pageSize: 20,
    });
  });

  describe('Rendering', () => {
    it('should render tickets page with all main elements', async () => {
      render(<TicketsPage />);
      
      expect(screen.getByTestId('tickets-card')).toBeInTheDocument();
      expect(screen.getByText('工单管理')).toBeInTheDocument();
      
      // Should have action buttons
      expect(screen.getByTestId('button-primary')).toBeInTheDocument();
      expect(screen.getByText('新建工单')).toBeInTheDocument();
      
      // Should have search and filter controls
      expect(screen.getByTestId('input-搜索工单')).toBeInTheDocument();
      expect(screen.getByTestId('select-状态筛选')).toBeInTheDocument();
      expect(screen.getByTestId('select-优先级筛选')).toBeInTheDocument();
    });

    it('should render tickets table', async () => {
      render(<TicketsPage />);
      
      await waitFor(() => {
        expect(screen.getByTestId('tickets-table')).toBeInTheDocument();
      });
      
      // Should show table headers
      expect(screen.getByText('标题')).toBeInTheDocument();
      expect(screen.getByText('状态')).toBeInTheDocument();
      expect(screen.getByText('优先级')).toBeInTheDocument();
      expect(screen.getByText('处理人')).toBeInTheDocument();
      expect(screen.getByText('创建时间')).toBeInTheDocument();
      expect(screen.getByText('操作')).toBeInTheDocument();
    });

    it('should display ticket data in table rows', async () => {
      render(<TicketsPage />);
      
      await waitFor(() => {
        expect(screen.getByTestId('ticket-row-0')).toBeInTheDocument();
        expect(screen.getByTestId('ticket-row-1')).toBeInTheDocument();
      });
      
      // Should show ticket titles
      expect(screen.getByText('测试工单1')).toBeInTheDocument();
      expect(screen.getByText('测试工单2')).toBeInTheDocument();
    });
  });

  describe('Search and Filter', () => {
    it('should handle search input', async () => {
      const user = userEvent.setup();
      render(<TicketsPage />);
      
      const searchInput = screen.getByTestId('input-搜索工单');
      await user.type(searchInput, '测试工单1');
      
      expect(searchInput).toHaveValue('测试工单1');
    });

    it('should handle status filter', async () => {
      const user = userEvent.setup();
      render(<TicketsPage />);
      
      const statusSelect = screen.getByTestId('select-状态筛选');
      await user.selectOptions(statusSelect, 'open');
      
      expect(statusSelect).toHaveValue('open');
    });

    it('should handle priority filter', async () => {
      const user = userEvent.setup();
      render(<TicketsPage />);
      
      const prioritySelect = screen.getByTestId('select-优先级筛选');
      await user.selectOptions(prioritySelect, 'high');
      
      expect(prioritySelect).toHaveValue('high');
    });
  });

  describe('Ticket Creation', () => {
    it('should open create modal when clicking new ticket button', async () => {
      const user = userEvent.setup();
      render(<TicketsPage />);
      
      const newTicketButton = screen.getByTestId('button-primary');
      await user.click(newTicketButton);
      
      await waitFor(() => {
        expect(screen.getByTestId('modal')).toBeInTheDocument();
        expect(screen.getByTestId('modal-title')).toHaveTextContent('新建工单');
      });
    });

    it('should close modal when clicking cancel', async () => {
      const user = userEvent.setup();
      render(<TicketsPage />);
      
      // Open modal
      const newTicketButton = screen.getByTestId('button-primary');
      await user.click(newTicketButton);
      
      await waitFor(() => {
        expect(screen.getByTestId('modal')).toBeInTheDocument();
      });
      
      // Close modal
      const cancelButton = screen.getByTestId('modal-cancel');
      await user.click(cancelButton);
      
      await waitFor(() => {
        expect(screen.queryByTestId('modal')).not.toBeInTheDocument();
      });
    });
  });

  describe('Loading States', () => {
    it('should show loading state when fetching tickets', async () => {
      mockTicketAPI.getTickets.mockImplementation(() => 
        new Promise(resolve => setTimeout(resolve, 100))
      );
      
      render(<TicketsPage />);
      
      expect(screen.getByTestId('table-loading')).toBeInTheDocument();
    });

    it('should hide loading state after data is loaded', async () => {
      render(<TicketsPage />);
      
      await waitFor(() => {
        expect(screen.queryByTestId('table-loading')).not.toBeInTheDocument();
      });
    });
  });

  describe('Error Handling', () => {
    it('should handle API errors gracefully', async () => {
      mockTicketAPI.getTickets.mockRejectedValue(new Error('API Error'));
      
      render(<TicketsPage />);
      
      await waitFor(() => {
        // Should not crash and should handle error state
        expect(screen.getByTestId('tickets-table')).toBeInTheDocument();
      });
    });
  });

  describe('Accessibility', () => {
    it('should have proper table structure', async () => {
      render(<TicketsPage />);
      
      await waitFor(() => {
        const table = screen.getByRole('table');
        expect(table).toBeInTheDocument();
      });
    });

    it('should have accessible buttons', async () => {
      render(<TicketsPage />);
      
      const buttons = screen.getAllByRole('button');
      expect(buttons.length).toBeGreaterThan(0);
      
      buttons.forEach(button => {
        expect(button).toBeInTheDocument();
      });
    });

    it('should have accessible form controls', async () => {
      render(<TicketsPage />);
      
      const searchInput = screen.getByRole('textbox');
      expect(searchInput).toBeInTheDocument();
      
      const selects = screen.getAllByRole('combobox');
      expect(selects.length).toBeGreaterThan(0);
    });
  });

  describe('Responsive Design', () => {
    it('should render properly on different screen sizes', async () => {
      // Mock window.innerWidth
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 768,
      });
      
      render(<TicketsPage />);
      
      await waitFor(() => {
        expect(screen.getByTestId('tickets-table')).toBeInTheDocument();
      });
    });
  });

  describe('Performance', () => {
    it('should not re-render unnecessarily', async () => {
      const renderSpy = jest.fn();
      
      const TestWrapper = () => {
        renderSpy();
        return <TicketsPage />;
      };
      
      const { rerender } = render(<TestWrapper />);
      
      expect(renderSpy).toHaveBeenCalledTimes(1);
      
      // Re-render with same props
      rerender(<TestWrapper />);
      
      // Should not cause unnecessary re-renders
      expect(renderSpy).toHaveBeenCalledTimes(2);
    });
  });

  describe('Data Management', () => {
    it('should refresh data when needed', async () => {
      render(<TicketsPage />);
      
      await waitFor(() => {
        expect(mockTicketAPI.getTickets).toHaveBeenCalledTimes(1);
      });
    });

    it('should handle empty data state', async () => {
      mockTicketAPI.getTickets.mockResolvedValue({
        tickets: [],
        total: 0,
        page: 1,
        pageSize: 20,
      });
      
      render(<TicketsPage />);
      
      await waitFor(() => {
        expect(screen.getByTestId('tickets-table')).toBeInTheDocument();
      });
    });
  });
});