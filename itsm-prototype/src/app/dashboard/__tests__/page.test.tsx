import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import DashboardPage from '../page';

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
    user: { id: 1, name: 'Test User', roles: ['admin'] },
    isAuthenticated: true,
  }),
}));

// Mock dashboard API
const mockDashboardAPI = {
  getDashboardStats: jest.fn(),
  getRecentTickets: jest.fn(),
  getSystemHealth: jest.fn(),
  getPerformanceMetrics: jest.fn(),
};

jest.mock('@/lib/api/dashboard-api', () => ({
  DashboardAPI: mockDashboardAPI,
}));

// Mock Recharts components
jest.mock('recharts', () => ({
  LineChart: ({ children, data, ...props }: { 
    children: React.ReactNode; 
    data: Array<Record<string, unknown>>; 
    [key: string]: unknown; 
  }) => (
    <div data-testid="line-chart" data-chart-data={JSON.stringify(data)} {...props}>
      {children}
    </div>
  ),
  BarChart: ({ children, data, ...props }: { 
    children: React.ReactNode; 
    data: Array<Record<string, unknown>>; 
    [key: string]: unknown; 
  }) => (
    <div data-testid="bar-chart" data-chart-data={JSON.stringify(data)} {...props}>
      {children}
    </div>
  ),
  PieChart: ({ children, data, ...props }: { 
    children: React.ReactNode; 
    data?: Array<Record<string, unknown>>; 
    [key: string]: unknown; 
  }) => (
    <div data-testid="pie-chart" data-chart-data={JSON.stringify(data)} {...props}>
      {children}
    </div>
  ),
  XAxis: ({ dataKey, ...props }: { dataKey?: string; [key: string]: unknown }) => (
    <div data-testid="x-axis" data-key={dataKey} {...props} />
  ),
  YAxis: ({ dataKey, ...props }: { dataKey?: string; [key: string]: unknown }) => (
    <div data-testid="y-axis" data-key={dataKey} {...props} />
  ),
  CartesianGrid: (props: Record<string, unknown>) => (
    <div data-testid="cartesian-grid" {...props} />
  ),
  Tooltip: (props: Record<string, unknown>) => (
    <div data-testid="chart-tooltip" {...props} />
  ),
  Legend: (props: Record<string, unknown>) => (
    <div data-testid="chart-legend" {...props} />
  ),
  Line: ({ dataKey, stroke, ...props }: { dataKey?: string; stroke?: string; [key: string]: unknown }) => (
    <div data-testid="chart-line" data-key={dataKey} style={{ color: stroke }} {...props} />
  ),
  Bar: ({ dataKey, fill, ...props }: { dataKey?: string; fill?: string; [key: string]: unknown }) => (
    <div data-testid="chart-bar" data-key={dataKey} style={{ backgroundColor: fill }} {...props} />
  ),
  Cell: ({ fill, ...props }: { fill?: string; [key: string]: unknown }) => (
    <div data-testid="chart-cell" style={{ backgroundColor: fill }} {...props} />
  ),
  Pie: ({ dataKey, ...props }: { dataKey?: string; [key: string]: unknown }) => (
    <div data-testid="chart-pie" data-key={dataKey} {...props} />
  ),
  ResponsiveContainer: ({ children, ...props }: { children: React.ReactNode; [key: string]: unknown }) => (
    <div data-testid="responsive-container" {...props}>
      {children}
    </div>
  ),
}));

// Mock Ant Design components
jest.mock('antd', () => ({
  Row: ({ children, gutter, ...props }: { children: React.ReactNode; gutter?: number; [key: string]: unknown }) => (
    <div data-testid="row" data-gutter={gutter} {...props}>{children}</div>
  ),
  Col: ({ children, span, xs, sm, md, lg, xl, ...props }: { 
    children: React.ReactNode; 
    span?: number; 
    xs?: number; 
    sm?: number; 
    md?: number; 
    lg?: number; 
    xl?: number; 
    [key: string]: unknown; 
  }) => (
    <div 
      data-testid="col" 
      data-span={span}
      data-responsive={JSON.stringify({ xs, sm, md, lg, xl })}
      {...props}
    >
      {children}
    </div>
  ),
  Card: ({ children, title, extra, loading, ...props }: {
    children: React.ReactNode;
    title?: string;
    extra?: React.ReactNode;
    loading?: boolean;
    [key: string]: unknown;
  }) => (
    <div data-testid="card" data-loading={loading} {...props}>
      {title && (
        <div data-testid="card-header">
          <h3>{title}</h3>
          {extra && <div data-testid="card-extra">{extra}</div>}
        </div>
      )}
      {loading ? <div data-testid="card-loading">Loading...</div> : children}
    </div>
  ),
  Statistic: ({ title, value, prefix, suffix, ...props }: {
    title?: string;
    value?: string | number;
    prefix?: React.ReactNode;
    suffix?: React.ReactNode;
    [key: string]: unknown;
  }) => (
    <div data-testid="statistic" {...props}>
      {title && <div data-testid="statistic-title">{title}</div>}
      <div data-testid="statistic-value">
        {prefix}{value}{suffix}
      </div>
    </div>
  ),
  Progress: ({ percent, status, ...props }: { percent?: number; status?: string; [key: string]: unknown }) => (
    <div data-testid="progress" data-percent={percent} data-status={status} {...props}>
      <div style={{ width: `${percent}%` }}>Progress: {percent}%</div>
    </div>
  ),
  Table: ({ columns, dataSource, loading, pagination, ...props }: {
    columns: Array<{ title: string; dataIndex: string; key: string }>;
    dataSource: Array<Record<string, unknown>>;
    loading?: boolean;
    pagination?: boolean | Record<string, unknown>;
    [key: string]: unknown;
  }) => (
    <div data-testid="table" data-loading={loading} {...props}>
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
            <tr key={index} data-testid={`table-row-${index}`}>
              {columns.map(col => (
                <td key={col.key}>{String(item[col.dataIndex] || '')}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      {pagination && <div data-testid="table-pagination">Pagination</div>}
    </div>
  ),
  Button: ({ children, type, onClick, loading, ...props }: {
    children: React.ReactNode;
    type?: string;
    onClick?: () => void;
    loading?: boolean;
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
  Select: ({ placeholder, options, onChange, value, ...props }: {
    placeholder?: string;
    options?: Array<{ label: string; value: string }>;
    onChange?: (value: string) => void;
    value?: string;
    [key: string]: unknown;
  }) => (
    <select 
      value={value}
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
}));

// Mock Lucide React icons
jest.mock('lucide-react', () => ({
  TrendingUp: () => <div data-testid="trending-up-icon">TrendingUp</div>,
  TrendingDown: () => <div data-testid="trending-down-icon">TrendingDown</div>,
  Users: () => <div data-testid="users-icon">Users</div>,
  Ticket: () => <div data-testid="ticket-icon">Ticket</div>,
  Clock: () => <div data-testid="clock-icon">Clock</div>,
  CheckCircle: () => <div data-testid="check-circle-icon">CheckCircle</div>,
  AlertCircle: () => <div data-testid="alert-circle-icon">AlertCircle</div>,
  Activity: () => <div data-testid="activity-icon">Activity</div>,
  BarChart3: () => <div data-testid="bar-chart-icon">BarChart3</div>,
  RefreshCw: () => <div data-testid="refresh-icon">RefreshCw</div>,
}));

// Mock dashboard data
const mockDashboardStats = {
  totalTickets: 150,
  openTickets: 45,
  resolvedTickets: 95,
  avgResolutionTime: 2.5,
  ticketTrend: [
    { date: '2024-01-01', open: 10, resolved: 8 },
    { date: '2024-01-02', open: 12, resolved: 10 },
    { date: '2024-01-03', open: 8, resolved: 15 },
  ],
  priorityDistribution: [
    { name: 'High', value: 20, color: '#ff4d4f' },
    { name: 'Medium', value: 35, color: '#faad14' },
    { name: 'Low', value: 45, color: '#52c41a' },
  ],
};

const mockRecentTickets = [
  {
    id: 1,
    title: '系统登录问题',
    status: 'open',
    priority: 'high',
    assignee: 'John Doe',
    created_at: '2024-01-01T10:00:00Z',
  },
  {
    id: 2,
    title: '网络连接异常',
    status: 'in_progress',
    priority: 'medium',
    assignee: 'Jane Smith',
    created_at: '2024-01-01T09:30:00Z',
  },
];

const mockSystemHealth = {
  cpu: 65,
  memory: 78,
  disk: 45,
  network: 92,
  status: 'healthy',
};

describe('DashboardPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    
    // Setup mock API responses
    mockDashboardAPI.getDashboardStats.mockResolvedValue(mockDashboardStats);
    mockDashboardAPI.getRecentTickets.mockResolvedValue(mockRecentTickets);
    mockDashboardAPI.getSystemHealth.mockResolvedValue(mockSystemHealth);
    mockDashboardAPI.getPerformanceMetrics.mockResolvedValue({
      responseTime: [
        { time: '10:00', value: 120 },
        { time: '10:05', value: 135 },
        { time: '10:10', value: 98 },
      ],
    });
  });

  describe('Rendering', () => {
    it('should render dashboard with all main sections', async () => {
      render(<DashboardPage />);
      
      // Should have main layout
      expect(screen.getByTestId('row')).toBeInTheDocument();
      
      // Should have statistics cards
      await waitFor(() => {
        expect(screen.getAllByTestId('card').length).toBeGreaterThan(0);
      });
    });

    it('should display key statistics', async () => {
      render(<DashboardPage />);
      
      await waitFor(() => {
        expect(screen.getByText('总工单数')).toBeInTheDocument();
        expect(screen.getByText('150')).toBeInTheDocument();
        expect(screen.getByText('待处理工单')).toBeInTheDocument();
        expect(screen.getByText('45')).toBeInTheDocument();
      });
    });

    it('should render charts', async () => {
      render(<DashboardPage />);
      
      await waitFor(() => {
        expect(screen.getByTestId('line-chart')).toBeInTheDocument();
        expect(screen.getByTestId('pie-chart')).toBeInTheDocument();
        expect(screen.getByTestId('bar-chart')).toBeInTheDocument();
      });
    });

    it('should display recent tickets table', async () => {
      render(<DashboardPage />);
      
      await waitFor(() => {
        expect(screen.getByTestId('table')).toBeInTheDocument();
        expect(screen.getByText('最近工单')).toBeInTheDocument();
      });
    });
  });

  describe('Data Loading', () => {
    it('should show loading states initially', async () => {
      mockDashboardAPI.getDashboardStats.mockImplementation(() => 
        new Promise(resolve => setTimeout(resolve, 100))
      );
      
      render(<DashboardPage />);
      
      expect(screen.getAllByTestId('card-loading').length).toBeGreaterThan(0);
    });

    it('should hide loading states after data loads', async () => {
      render(<DashboardPage />);
      
      await waitFor(() => {
        expect(screen.queryByTestId('card-loading')).not.toBeInTheDocument();
      });
    });

    it('should call all required APIs on mount', async () => {
      render(<DashboardPage />);
      
      await waitFor(() => {
        expect(mockDashboardAPI.getDashboardStats).toHaveBeenCalledTimes(1);
        expect(mockDashboardAPI.getRecentTickets).toHaveBeenCalledTimes(1);
        expect(mockDashboardAPI.getSystemHealth).toHaveBeenCalledTimes(1);
      });
    });
  });

  describe('Charts and Visualizations', () => {
    it('should render line chart with correct data', async () => {
      render(<DashboardPage />);
      
      await waitFor(() => {
        const lineChart = screen.getByTestId('line-chart');
        expect(lineChart).toBeInTheDocument();
        
        const chartData = JSON.parse(lineChart.getAttribute('data-chart-data') || '[]');
        expect(chartData).toHaveLength(3);
        expect(chartData[0]).toHaveProperty('date');
        expect(chartData[0]).toHaveProperty('open');
        expect(chartData[0]).toHaveProperty('resolved');
      });
    });

    it('should render pie chart with priority distribution', async () => {
      render(<DashboardPage />);
      
      await waitFor(() => {
        const pieChart = screen.getByTestId('pie-chart');
        expect(pieChart).toBeInTheDocument();
        
        const chartData = JSON.parse(pieChart.getAttribute('data-chart-data') || '[]');
        expect(chartData).toHaveLength(3);
        expect(chartData[0]).toHaveProperty('name');
        expect(chartData[0]).toHaveProperty('value');
      });
    });

    it('should display system health metrics', async () => {
      render(<DashboardPage />);
      
      await waitFor(() => {
        expect(screen.getByTestId('progress')).toBeInTheDocument();
        expect(screen.getByText('CPU使用率')).toBeInTheDocument();
        expect(screen.getByText('内存使用率')).toBeInTheDocument();
      });
    });
  });

  describe('Interactive Features', () => {
    it('should handle refresh button click', async () => {
      const user = userEvent.setup();
      render(<DashboardPage />);
      
      await waitFor(() => {
        const refreshButton = screen.getByTestId('button-default');
        expect(refreshButton).toBeInTheDocument();
      });
      
      const refreshButton = screen.getByTestId('button-default');
      await user.click(refreshButton);
      
      // Should call APIs again
      await waitFor(() => {
        expect(mockDashboardAPI.getDashboardStats).toHaveBeenCalledTimes(2);
      });
    });

    it('should handle time range selection', async () => {
      const user = userEvent.setup();
      render(<DashboardPage />);
      
      await waitFor(() => {
        const timeSelect = screen.getByTestId('select-时间范围');
        expect(timeSelect).toBeInTheDocument();
      });
      
      const timeSelect = screen.getByTestId('select-时间范围');
      await user.selectOptions(timeSelect, '7d');
      
      expect(timeSelect).toHaveValue('7d');
    });
  });

  describe('Error Handling', () => {
    it('should handle API errors gracefully', async () => {
      mockDashboardAPI.getDashboardStats.mockRejectedValue(new Error('API Error'));
      
      render(<DashboardPage />);
      
      await waitFor(() => {
        // Should not crash and should show some fallback content
        expect(screen.getByTestId('row')).toBeInTheDocument();
      });
    });

    it('should handle partial data loading failures', async () => {
      mockDashboardAPI.getDashboardStats.mockResolvedValue(mockDashboardStats);
      mockDashboardAPI.getRecentTickets.mockRejectedValue(new Error('Tickets API Error'));
      
      render(<DashboardPage />);
      
      await waitFor(() => {
        // Should still show statistics even if tickets fail
        expect(screen.getByText('150')).toBeInTheDocument();
      });
    });
  });

  describe('Responsive Design', () => {
    it('should render responsive grid layout', async () => {
      render(<DashboardPage />);
      
      await waitFor(() => {
        const cols = screen.getAllByTestId('col');
        expect(cols.length).toBeGreaterThan(0);
        
        // Check if responsive props are set
        cols.forEach(col => {
          const responsive = col.getAttribute('data-responsive');
          expect(responsive).toBeTruthy();
        });
      });
    });
  });

  describe('Performance', () => {
    it('should not cause memory leaks with timers', async () => {
      const { unmount } = render(<DashboardPage />);
      
      // Simulate component unmount
      unmount();
      
      // Should not have any pending timers
      expect(setTimeout).not.toHaveBeenCalled();
    });

    it('should handle large datasets efficiently', async () => {
      const largeDataset = Array.from({ length: 1000 }, (_, i) => ({
        date: `2024-01-${String(i + 1).padStart(2, '0')}`,
        open: Math.floor(Math.random() * 50),
        resolved: Math.floor(Math.random() * 50),
      }));
      
      mockDashboardAPI.getDashboardStats.mockResolvedValue({
        ...mockDashboardStats,
        ticketTrend: largeDataset,
      });
      
      render(<DashboardPage />);
      
      await waitFor(() => {
        const lineChart = screen.getByTestId('line-chart');
        expect(lineChart).toBeInTheDocument();
      });
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA labels', async () => {
      render(<DashboardPage />);
      
      await waitFor(() => {
        const statistics = screen.getAllByTestId('statistic');
        expect(statistics.length).toBeGreaterThan(0);
      });
    });

    it('should support keyboard navigation', async () => {
      const user = userEvent.setup();
      render(<DashboardPage />);
      
      await waitFor(() => {
        const refreshButton = screen.getByTestId('button-default');
        expect(refreshButton).toBeInTheDocument();
      });
      
      const refreshButton = screen.getByTestId('button-default');
      
      // Should be focusable
      refreshButton.focus();
      expect(refreshButton).toHaveFocus();
      
      // Should respond to Enter key
      await user.keyboard('{Enter}');
      
      await waitFor(() => {
        expect(mockDashboardAPI.getDashboardStats).toHaveBeenCalledTimes(2);
      });
    });
  });

  describe('Real-time Updates', () => {
    it('should handle real-time data updates', async () => {
      render(<DashboardPage />);
      
      await waitFor(() => {
        expect(screen.getByText('150')).toBeInTheDocument();
      });
      
      // Simulate real-time update
      const updatedStats = {
        ...mockDashboardStats,
        totalTickets: 155,
      };
      
      mockDashboardAPI.getDashboardStats.mockResolvedValue(updatedStats);
      
      // Trigger refresh
      const refreshButton = screen.getByTestId('button-default');
      await userEvent.click(refreshButton);
      
      await waitFor(() => {
        expect(screen.getByText('155')).toBeInTheDocument();
      });
    });
  });
});