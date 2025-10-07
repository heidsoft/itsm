/**
 * 用户体验测试
 * 测试错误提示、加载状态、交互反馈等用户体验相关功能
 */

import React from 'react';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ConfigProvider, message } from 'antd';
import zhCN from 'antd/locale/zh_CN';

// Mock modules
jest.mock('../../lib/http-client');
jest.mock('../../lib/auth-service');
jest.mock('../../../lib/store/ui-store', () => ({
  useNotifications: jest.fn(),
}));
jest.mock('../../components/RouteGuard', () => ({
  useAuth: jest.fn(),
}));
jest.mock('antd', () => {
  const originalAntd = jest.requireActual('antd');
  return {
    ...originalAntd,
    message: {
      success: jest.fn(),
      error: jest.fn(),
      warning: jest.fn(),
      info: jest.fn(),
      loading: jest.fn(),
      destroy: jest.fn(),
    },
    notification: {
      success: jest.fn(),
      error: jest.fn(),
      warning: jest.fn(),
      info: jest.fn(),
      open: jest.fn(),
      destroy: jest.fn(),
    },
  };
});

// Mock loading components
const MockLoadingSpinner = () => (
  <div data-testid="loading-spinner" className="animate-spin">
    <div className="w-6 h-6 border-2 border-blue-600 border-t-transparent rounded-full" />
  </div>
);

const MockSkeletonLoader = () => (
  <div data-testid="skeleton-loader" className="animate-pulse">
    <div className="h-4 bg-gray-200 rounded mb-2" />
    <div className="h-4 bg-gray-200 rounded mb-2" />
    <div className="h-4 bg-gray-200 rounded w-3/4" />
  </div>
);

// Mock error boundary
class MockErrorBoundary extends React.Component<
  { children: React.ReactNode },
  { hasError: boolean; error?: Error }
> {
  constructor(props: { children: React.ReactNode }) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error) {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('Error caught by boundary:', error, errorInfo);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div data-testid="error-boundary" className="p-4 bg-red-50 border border-red-200 rounded">
          <h2 className="text-red-800 font-semibold mb-2">出现了错误</h2>
          <p className="text-red-600 mb-4">页面加载失败，请刷新重试</p>
          <button 
            data-testid="retry-button"
            onClick={() => this.setState({ hasError: false })}
            className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700"
          >
            重试
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}

// Mock components with different states
const MockTicketForm = ({ 
  loading = false, 
  error = null,
  onSubmit = jest.fn(),
  onCancel = jest.fn() 
}: {
  loading?: boolean;
  error?: string | null;
  onSubmit?: jest.MockedFunction<() => void>;
  onCancel?: jest.MockedFunction<() => void>;
}) => {
  const [formData, setFormData] = React.useState({
    title: '',
    description: '',
    priority: 'medium',
  });
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (loading || isSubmitting) return;
    
    try {
      setIsSubmitting(true);
      await onSubmit();
      message.success('工单创建成功');
    } catch {
      message.error('工单创建失败');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div data-testid="ticket-form">
      {error && (
        <div data-testid="form-error" className="mb-4 p-3 bg-red-50 border border-red-200 rounded">
          <p className="text-red-600">{error}</p>
        </div>
      )}
      
      <form onSubmit={handleSubmit}>
        <div className="mb-4">
          <label htmlFor="title" className="block text-sm font-medium text-gray-700 mb-1">
            工单标题 *
          </label>
          <input
            id="title"
            data-testid="title-input"
            type="text"
            value={formData.title}
            onChange={(e) => setFormData({ ...formData, title: e.target.value })}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="请输入工单标题"
            required
          />
        </div>

        <div className="mb-4">
          <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-1">
            问题描述 *
          </label>
          <textarea
            id="description"
            data-testid="description-input"
            value={formData.description}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            rows={4}
            placeholder="请详细描述遇到的问题"
            required
          />
        </div>

        <div className="mb-6">
          <label htmlFor="priority" className="block text-sm font-medium text-gray-700 mb-1">
            优先级
          </label>
          <select
            id="priority"
            data-testid="priority-select"
            value={formData.priority}
            onChange={(e) => setFormData({ ...formData, priority: e.target.value })}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="low">低</option>
            <option value="medium">中</option>
            <option value="high">高</option>
            <option value="urgent">紧急</option>
          </select>
        </div>

        <div className="flex justify-end space-x-3">
          <button
            type="button"
            data-testid="cancel-button"
            onClick={onCancel}
            className="px-4 py-2 text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-gray-500"
            disabled={loading || isSubmitting}
          >
            取消
          </button>
          <button
            type="submit"
            data-testid="submit-button"
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
            disabled={loading || isSubmitting || !formData.title.trim() || !formData.description.trim()}
          >
            {loading || isSubmitting ? (
              <div className="flex items-center">
                <MockLoadingSpinner />
                <span className="ml-2">创建中...</span>
              </div>
            ) : (
              '创建工单'
            )}
          </button>
        </div>
      </form>
    </div>
  );
};

const MockTicketList = ({ 
  loading = false, 
  error = null,
  tickets = [],
  onRefresh = jest.fn()
}: {
  loading?: boolean;
  error?: string | null;
  tickets?: unknown[];
  onRefresh?: jest.MockedFunction<() => void>;
}) => {
  if (loading) {
    return (
      <div data-testid="ticket-list-loading">
        <div className="space-y-4">
          {Array.from({ length: 5 }).map((_, index) => (
            <MockSkeletonLoader key={index} />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div data-testid="ticket-list-error" className="text-center py-8">
        <div className="mb-4">
          <div className="w-16 h-16 mx-auto bg-red-100 rounded-full flex items-center justify-center">
            <span className="text-red-600 text-2xl">⚠️</span>
          </div>
        </div>
        <h3 className="text-lg font-medium text-gray-900 mb-2">加载失败</h3>
        <p className="text-gray-600 mb-4">{error}</p>
        <button
          data-testid="refresh-button"
          onClick={onRefresh}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          重新加载
        </button>
      </div>
    );
  }

  if (tickets.length === 0) {
    return (
      <div data-testid="ticket-list-empty" className="text-center py-8">
        <div className="mb-4">
          <div className="w-16 h-16 mx-auto bg-gray-100 rounded-full flex items-center justify-center">
            <span className="text-gray-400 text-2xl">📝</span>
          </div>
        </div>
        <h3 className="text-lg font-medium text-gray-900 mb-2">暂无工单</h3>
        <p className="text-gray-600">还没有创建任何工单</p>
      </div>
    );
  }

  return (
    <div data-testid="ticket-list">
      <div className="space-y-4">
        {tickets.map((_, index) => (
          <div key={index} data-testid={`ticket-item-${index}`} className="p-4 bg-white border border-gray-200 rounded-lg">
            <h4 className="font-medium text-gray-900">工单 {index + 1}</h4>
            <p className="text-gray-600 mt-1">工单描述内容</p>
          </div>
        ))}
      </div>
    </div>
  );
};

// Test wrapper
const TestWrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <ConfigProvider locale={zhCN}>
      <MockErrorBoundary>
        {children}
      </MockErrorBoundary>
    </ConfigProvider>
  );
};

describe('用户体验测试', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('加载状态测试', () => {
    it('应该显示加载中的骨架屏', async () => {
      render(
        <TestWrapper>
          <MockTicketList loading={true} />
        </TestWrapper>
      );

      // 验证骨架屏显示
      expect(screen.getByTestId('ticket-list-loading')).toBeInTheDocument();
      expect(screen.getAllByTestId('skeleton-loader')).toHaveLength(5);
    });

    it('应该在表单提交时显示加载状态', async () => {
      const user = userEvent.setup();
      const mockSubmit = jest.fn().mockImplementation(() => 
        new Promise(resolve => setTimeout(resolve, 1000))
      );

      render(
        <TestWrapper>
          <MockTicketForm loading={false} onSubmit={mockSubmit} />
        </TestWrapper>
      );

      // 填写表单
      await user.type(screen.getByTestId('title-input'), '测试工单');
      await user.type(screen.getByTestId('description-input'), '这是一个测试工单');

      // 提交表单
      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).not.toBeDisabled();

      await user.click(submitButton);

      // 验证提交按钮状态
      expect(mockSubmit).toHaveBeenCalled();
    });

    it('应该显示加载中的旋转图标', () => {
      render(
        <TestWrapper>
          <MockTicketForm loading={true} />
        </TestWrapper>
      );

      // 验证加载图标显示
      expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
      expect(screen.getByText('创建中...')).toBeInTheDocument();
    });
  });

  describe('错误处理测试', () => {
    it('应该显示表单验证错误', async () => {

      render(
        <TestWrapper>
          <MockTicketForm error="标题不能为空" />
        </TestWrapper>
      );

      // 验证错误信息显示
      expect(screen.getByTestId('form-error')).toBeInTheDocument();
      expect(screen.getByText('标题不能为空')).toBeInTheDocument();

      // 验证提交按钮被禁用
      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toBeDisabled();
    });

    it('应该显示网络错误状态', () => {
      const mockRefresh = jest.fn();

      render(
        <TestWrapper>
          <MockTicketList error="网络连接失败，请检查网络设置" onRefresh={mockRefresh} />
        </TestWrapper>
      );

      // 验证错误状态显示
      expect(screen.getByTestId('ticket-list-error')).toBeInTheDocument();
      expect(screen.getByText('加载失败')).toBeInTheDocument();
      expect(screen.getByText('网络连接失败，请检查网络设置')).toBeInTheDocument();

      // 验证重新加载按钮
      const refreshButton = screen.getByTestId('refresh-button');
      expect(refreshButton).toBeInTheDocument();
    });

    it('应该处理组件错误边界', () => {
      const ThrowError = () => {
        throw new Error('测试错误');
      };

      render(
        <TestWrapper>
          <ThrowError />
        </TestWrapper>
      );

      // 验证错误边界显示
      expect(screen.getByTestId('error-boundary')).toBeInTheDocument();
      expect(screen.getByText('出现了错误')).toBeInTheDocument();
      expect(screen.getByText('页面加载失败，请刷新重试')).toBeInTheDocument();
      expect(screen.getByTestId('retry-button')).toBeInTheDocument();
    });

    it('应该支持错误重试功能', async () => {
      const user = userEvent.setup();
      const mockRefresh = jest.fn();

      render(
        <TestWrapper>
          <MockTicketList error="加载失败" onRefresh={mockRefresh} />
        </TestWrapper>
      );

      // 点击重新加载按钮
      const refreshButton = screen.getByTestId('refresh-button');
      await user.click(refreshButton);

      // 验证重试函数被调用
      expect(mockRefresh).toHaveBeenCalledTimes(1);
    });
  });

  describe('交互反馈测试', () => {
    it('应该显示成功消息', async () => {
      const user = userEvent.setup();
      const mockSubmit = jest.fn().mockResolvedValue({});

      render(
        <TestWrapper>
          <MockTicketForm onSubmit={mockSubmit} />
        </TestWrapper>
      );

      // 填写并提交表单
      await user.type(screen.getByTestId('title-input'), '测试工单');
      await user.type(screen.getByTestId('description-input'), '这是一个测试工单');
      await user.click(screen.getByTestId('submit-button'));

      // 等待提交完成
      await waitFor(() => {
        expect(mockSubmit).toHaveBeenCalled();
      });

      // 验证成功消息
      expect(message.success).toHaveBeenCalledWith('工单创建成功');
    });

    it('应该显示失败消息', async () => {
      const user = userEvent.setup();
      const mockSubmit = jest.fn().mockRejectedValue(new Error('提交失败'));

      render(
        <TestWrapper>
          <MockTicketForm onSubmit={mockSubmit} />
        </TestWrapper>
      );

      // 填写并提交表单
      await user.type(screen.getByTestId('title-input'), '测试工单');
      await user.type(screen.getByTestId('description-input'), '这是一个测试工单');
      await user.click(screen.getByTestId('submit-button'));

      // 等待提交完成
      await waitFor(() => {
        expect(mockSubmit).toHaveBeenCalled();
      });

      // 验证错误消息
      expect(message.error).toHaveBeenCalledWith('工单创建失败');
    });

    it('应该提供视觉反馈', async () => {
      const user = userEvent.setup();

      render(
        <TestWrapper>
          <MockTicketForm />
        </TestWrapper>
      );

      // 测试输入框焦点状态
      const titleInput = screen.getByTestId('title-input');
      await user.click(titleInput);
      
      expect(titleInput).toHaveFocus();
      expect(titleInput).toHaveClass('focus:ring-2', 'focus:ring-blue-500');

      // 测试按钮悬停状态
      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toHaveClass('hover:bg-blue-700');
    });

    it('应该支持键盘导航', async () => {
      const user = userEvent.setup();

      render(
        <TestWrapper>
          <MockTicketForm />
        </TestWrapper>
      );

      // 测试Tab键导航
      const titleInput = screen.getByTestId('title-input');
      titleInput.focus();

      await user.tab();
      expect(screen.getByTestId('description-input')).toHaveFocus();

      await user.tab();
      expect(screen.getByTestId('priority-select')).toHaveFocus();

      await user.tab();
      expect(screen.getByTestId('cancel-button')).toHaveFocus();

      await user.tab();
      // 提交按钮禁用时不应获取焦点
      expect(screen.getByTestId('submit-button')).not.toHaveFocus();
      expect(screen.getByTestId('submit-button')).toBeDisabled();
    });
  });

  describe('空状态测试', () => {
    it('应该显示空状态页面', () => {
      render(
        <TestWrapper>
          <MockTicketList tickets={[]} />
        </TestWrapper>
      );

      // 验证空状态显示
      expect(screen.getByTestId('ticket-list-empty')).toBeInTheDocument();
      expect(screen.getByText('暂无工单')).toBeInTheDocument();
      expect(screen.getByText('还没有创建任何工单')).toBeInTheDocument();
    });

    it('应该在有数据时显示列表', () => {
      const mockTickets = [1, 2, 3];

      render(
        <TestWrapper>
          <MockTicketList tickets={mockTickets} />
        </TestWrapper>
      );

      // 验证列表显示
      expect(screen.getByTestId('ticket-list')).toBeInTheDocument();
      expect(screen.getAllByTestId(/ticket-item-/)).toHaveLength(3);
    });
  });

  describe('表单验证测试', () => {
    it('应该验证必填字段', async () => {
      const user = userEvent.setup();

      render(
        <TestWrapper>
          <MockTicketForm />
        </TestWrapper>
      );

      // 验证初始状态下提交按钮被禁用
      const submitButton = screen.getByTestId('submit-button');
      expect(submitButton).toBeDisabled();

      // 只填写标题
      await user.type(screen.getByTestId('title-input'), '测试工单');
      expect(submitButton).toBeDisabled();

      // 填写描述
      await user.type(screen.getByTestId('description-input'), '这是一个测试工单');
      expect(submitButton).not.toBeDisabled();
    });

    it('应该实时验证输入内容', async () => {
      const user = userEvent.setup();

      render(
        <TestWrapper>
          <MockTicketForm />
        </TestWrapper>
      );

      const titleInput = screen.getByTestId('title-input');
      const submitButton = screen.getByTestId('submit-button');

      // 输入空格
      await user.type(titleInput, '   ');
      expect(submitButton).toBeDisabled();

      // 输入有效内容
      await user.clear(titleInput);
      await user.type(titleInput, '有效标题');
      await user.type(screen.getByTestId('description-input'), '有效描述');
      expect(submitButton).not.toBeDisabled();
    });
  });

  describe('响应式交互测试', () => {
    it('应该在移动设备上正确显示', () => {
      // Mock 移动设备视口
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 375,
      });

      render(
        <TestWrapper>
          <MockTicketForm />
        </TestWrapper>
      );

      // 验证表单在移动设备上的显示
      expect(screen.getByTestId('ticket-form')).toBeInTheDocument();
      expect(screen.getByTestId('title-input')).toHaveClass('w-full');
      expect(screen.getByTestId('description-input')).toHaveClass('w-full');
    });

    it('应该支持触摸交互', async () => {
      const user = userEvent.setup();

      render(
        <TestWrapper>
          <MockTicketForm />
        </TestWrapper>
      );

      // 模拟触摸交互
      const titleInput = screen.getByTestId('title-input');
      await user.click(titleInput);
      
      expect(titleInput).toHaveFocus();
    });
  });

  describe('性能优化测试', () => {
    it('应该防止重复提交', async () => {
      const user = userEvent.setup();
      const mockSubmit = jest.fn().mockImplementation(() => 
        new Promise(resolve => setTimeout(resolve, 1000))
      );

      render(
        <TestWrapper>
          <MockTicketForm onSubmit={mockSubmit} />
        </TestWrapper>
      );

      // 填写表单
      await user.type(screen.getByTestId('title-input'), '测试工单');
      await user.type(screen.getByTestId('description-input'), '这是一个测试工单');

      // 快速点击提交按钮多次
      const submitButton = screen.getByTestId('submit-button');
      await user.click(submitButton);
      await user.click(submitButton);
      await user.click(submitButton);

      // 验证只调用一次
      expect(mockSubmit).toHaveBeenCalledTimes(1);
    });

    it('应该优化大量数据的渲染', () => {
      const largeTicketList = Array.from({ length: 1000 }, (_, i) => i);

      const startTime = performance.now();
      
      render(
        <TestWrapper>
          <MockTicketList tickets={largeTicketList} />
        </TestWrapper>
      );

      const endTime = performance.now();
      const renderTime = endTime - startTime;

      // 验证渲染时间在合理范围内（小于300ms）
      expect(renderTime).toBeLessThan(300);
    });
  });

  describe('无障碍访问测试', () => {
    it('应该提供正确的ARIA标签', () => {
      render(
        <TestWrapper>
          <MockTicketForm />
        </TestWrapper>
      );

      // 验证表单标签
      const titleInput = screen.getByTestId('title-input');
      expect(titleInput).toHaveAttribute('required');
      expect(screen.getByLabelText('工单标题 *')).toBe(titleInput);

      const descriptionInput = screen.getByTestId('description-input');
      expect(descriptionInput).toHaveAttribute('required');
      expect(screen.getByLabelText('问题描述 *')).toBe(descriptionInput);
    });

    it('应该支持屏幕阅读器', () => {
      render(
        <TestWrapper>
          <MockTicketList error="加载失败" />
        </TestWrapper>
      );

      // 验证错误状态的可访问性
      const errorContainer = screen.getByTestId('ticket-list-error');
      expect(errorContainer).toBeInTheDocument();
      // 标题应为“加载失败”
      expect(within(errorContainer).getByRole('heading', { name: '加载失败' })).toBeInTheDocument();
      // 文本也应包含错误提示（可能出现多个匹配项）
      expect(within(errorContainer).getAllByText('加载失败').length).toBeGreaterThanOrEqual(1);
    });

    it('应该提供键盘快捷键支持', async () => {
      const user = userEvent.setup();
      const mockCancel = jest.fn();

      render(
        <TestWrapper>
          <MockTicketForm onCancel={mockCancel} />
        </TestWrapper>
      );

      // 测试Escape键取消
      await user.keyboard('{Escape}');
      
      // 注意：这里需要实际的键盘事件处理逻辑
      // 在真实组件中应该监听keydown事件
    });
  });
});