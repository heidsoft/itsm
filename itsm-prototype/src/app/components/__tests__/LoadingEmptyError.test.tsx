import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import { LoadingEmptyError, LoadingEmptyErrorState } from '../ui/LoadingEmptyError';

// Mock Ant Design components
jest.mock('antd', () => ({
  Button: ({ children, onClick, icon, type, ...props }: {
    children: React.ReactNode;
    onClick?: () => void;
    icon?: React.ReactNode;
    type?: string;
    [key: string]: unknown;
  }) => (
    <button onClick={onClick} data-testid={`button-${type || 'default'}`} {...props}>
      {icon}
      <span>{children}</span>
    </button>
  ),
  Result: ({ title, subTitle, extra, icon }: {
    title: React.ReactNode;
    subTitle: React.ReactNode;
    extra: React.ReactNode;
    icon: React.ReactNode;
  }) => (
    <div data-testid="result">
      <div data-testid="result-icon">{icon}</div>
      <div data-testid="result-title">{title}</div>
      <div data-testid="result-subtitle">{subTitle}</div>
      <div data-testid="result-extra">{extra}</div>
    </div>
  ),
  Spin: ({ size }: { size?: string }) => (
    <div data-testid="spin" data-size={size}>
      Loading...
    </div>
  ),
  Typography: {
    Text: ({ children, type, className }: {
      children: React.ReactNode;
      type?: string;
      className?: string;
    }) => (
      <span data-testid="text" data-type={type} className={className}>
        {children}
      </span>
    ),
    Title: ({ children, level, className }: {
      children: React.ReactNode;
      level?: number;
      className?: string;
    }) => (
      <h1 data-testid="title" data-level={level} className={className}>
        {children}
      </h1>
    ),
  },
}));

// Mock Lucide React icons
jest.mock('lucide-react', () => ({
  RotateCcw: ({ size }: { size?: number }) => <span data-testid="rotate-ccw-icon" data-size={size}>↻</span>,
  Plus: ({ size }: { size?: number }) => <span data-testid="plus-icon" data-size={size}>+</span>,
  FileText: ({ size }: { size?: number }) => <span data-testid="file-text-icon" data-size={size}>📄</span>,
  AlertTriangle: ({ size, style }: { size?: number; style?: React.CSSProperties }) => (
    <span data-testid="alert-triangle-icon" data-size={size} style={style}>⚠️</span>
  ),
  User: ({ size }: { size?: number }) => <span data-testid="user-icon" data-size={size}>👤</span>,
  Database: ({ size }: { size?: number }) => <span data-testid="database-icon" data-size={size}>🗄️</span>,
  Settings: ({ size }: { size?: number }) => <span data-testid="settings-icon" data-size={size}>⚙️</span>,
}));

describe('LoadingEmptyError Component', () => {
  describe('Loading State', () => {
    it('should render loading spinner with default text', () => {
      render(<LoadingEmptyError state="loading" />);
      
      expect(screen.getByTestId('spin')).toBeInTheDocument();
      expect(screen.getByText('加载中...')).toBeInTheDocument();
    });

    it('should render loading spinner with custom text', () => {
      render(<LoadingEmptyError state="loading" loadingText="正在获取数据..." />);
      
      expect(screen.getByTestId('spin')).toBeInTheDocument();
      expect(screen.getByText('正在获取数据...')).toBeInTheDocument();
    });

    it('should apply custom minHeight and className', () => {
      const { container } = render(
        <LoadingEmptyError 
          state="loading" 
          minHeight={300} 
          className="custom-loading" 
        />
      );
      
      const loadingContainer = container.firstChild as HTMLElement;
      expect(loadingContainer).toHaveClass('custom-loading');
      expect(loadingContainer.style.minHeight).toBe('300px');
    });
  });

  describe('Empty State', () => {
    it('should render empty state with default configuration', () => {
      render(<LoadingEmptyError state="empty" />);
      
      expect(screen.getByTestId('title')).toBeInTheDocument();
      expect(screen.getByTestId('text')).toBeInTheDocument();
    });

    it('should render empty state with custom configuration', () => {
      const mockAction = jest.fn();
      const emptyConfig = {
        title: '没有工单',
        description: '当前没有工单数据',
        actionText: '创建工单',
        onAction: mockAction,
        showAction: true,
      };

      render(<LoadingEmptyError state="empty" empty={emptyConfig} />);
      
      expect(screen.getByText('没有工单')).toBeInTheDocument();
      expect(screen.getByText('当前没有工单数据')).toBeInTheDocument();
      
      const actionButton = screen.getByTestId('button-primary');
      expect(actionButton).toBeInTheDocument();
      expect(actionButton).toHaveTextContent('创建工单');
      
      fireEvent.click(actionButton);
      expect(mockAction).toHaveBeenCalledTimes(1);
    });

    it('should not render action button when showAction is false', () => {
      const emptyConfig = {
        title: '没有数据',
        actionText: '添加数据',
        showAction: false,
      };

      render(<LoadingEmptyError state="empty" empty={emptyConfig} />);
      
      expect(screen.queryByTestId('button-primary')).not.toBeInTheDocument();
    });

    it('should render custom icon when provided', () => {
      const customIcon = <span data-testid="custom-icon">🎯</span>;
      const emptyConfig = {
        title: '自定义空状态',
        icon: customIcon,
      };

      render(<LoadingEmptyError state="empty" empty={emptyConfig} />);
      
      expect(screen.getByTestId('custom-icon')).toBeInTheDocument();
    });
  });

  describe('Error State', () => {
    it('should render error state with default configuration', () => {
      render(<LoadingEmptyError state="error" />);
      
      expect(screen.getByTestId('result')).toBeInTheDocument();
      expect(screen.getByTestId('result-title')).toHaveTextContent('加载失败');
      expect(screen.getByTestId('alert-triangle-icon')).toBeInTheDocument();
    });

    it('should render error state with custom configuration and retry action', () => {
      const mockRetry = jest.fn();
      const errorConfig = {
        title: '网络错误',
        description: '请检查网络连接',
        actionText: '重新加载',
        onAction: mockRetry,
        showRetry: true,
      };

      render(<LoadingEmptyError state="error" error={errorConfig} />);
      
      expect(screen.getByText('网络错误')).toBeInTheDocument();
      expect(screen.getByText('请检查网络连接')).toBeInTheDocument();
      
      const retryButton = screen.getByTestId('button-primary');
      expect(retryButton).toBeInTheDocument();
      expect(retryButton).toHaveTextContent('重新加载');
      
      fireEvent.click(retryButton);
      expect(mockRetry).toHaveBeenCalledTimes(1);
    });

    it('should not render retry button when showRetry is false', () => {
      const errorConfig = {
        title: '错误',
        showRetry: false,
      };

      render(<LoadingEmptyError state="error" error={errorConfig} />);
      
      expect(screen.queryByTestId('button-primary')).not.toBeInTheDocument();
    });
  });

  describe('Success State', () => {
    it('should render children when state is success', () => {
      render(
        <LoadingEmptyError state="success">
          <div data-testid="success-content">Success Content</div>
        </LoadingEmptyError>
      );
      
      expect(screen.getByTestId('success-content')).toBeInTheDocument();
      expect(screen.getByText('Success Content')).toBeInTheDocument();
    });

    it('should render success configuration when provided', () => {
      const mockAction = jest.fn();
      const successConfig = {
        title: '操作成功',
        description: '数据已成功加载',
        actionText: '继续操作',
        onAction: mockAction,
        showAction: true,
      };

      render(<LoadingEmptyError state="success" success={successConfig} />);
      
      expect(screen.getByText('操作成功')).toBeInTheDocument();
      expect(screen.getByText('数据已成功加载')).toBeInTheDocument();
      
      const actionButton = screen.getByTestId('button-primary');
      expect(actionButton).toBeInTheDocument();
      
      fireEvent.click(actionButton);
      expect(mockAction).toHaveBeenCalledTimes(1);
    });

    it('should render both success config and children', () => {
      const successConfig = {
        title: '加载完成',
        description: '数据加载成功',
      };

      render(
        <LoadingEmptyError state="success" success={successConfig}>
          <div data-testid="children-content">Children Content</div>
        </LoadingEmptyError>
      );
      
      expect(screen.getByText('加载完成')).toBeInTheDocument();
      expect(screen.getByText('数据加载成功')).toBeInTheDocument();
      expect(screen.getByTestId('children-content')).toBeInTheDocument();
    });
  });

  describe('Custom Styling', () => {
    it('should apply custom className to different states', () => {
      const { rerender, container } = render(
        <LoadingEmptyError state="loading" className="custom-class" />
      );
      
      expect(container.firstChild).toHaveClass('custom-class');
      
      rerender(<LoadingEmptyError state="empty" className="empty-class" />);
      expect(container.firstChild).toHaveClass('empty-class');
      
      rerender(<LoadingEmptyError state="error" className="error-class" />);
      expect(container.firstChild).toHaveClass('error-class');
    });

    it('should apply custom minHeight to loading and empty states', () => {
      const { rerender, container } = render(
        <LoadingEmptyError state="loading" minHeight={400} />
      );
      
      const element = container.firstChild as HTMLElement;
      expect(element.style.minHeight).toBe('400px');
      
      rerender(<LoadingEmptyError state="empty" minHeight={500} />);
      expect((container.firstChild as HTMLElement).style.minHeight).toBe('500px');
    });
  });

  describe('Edge Cases', () => {
    it('should return null for invalid state', () => {
      const { container } = render(
        <LoadingEmptyError state={'invalid' as LoadingEmptyErrorState} />
      );
      
      expect(container.firstChild).toBeNull();
    });

    it('should handle missing onAction gracefully', () => {
      const emptyConfig = {
        title: '测试',
        actionText: '操作',
        // onAction is missing
      };

      render(<LoadingEmptyError state="empty" empty={emptyConfig} />);
      
      // Should not render button without onAction
      expect(screen.queryByTestId('button-primary')).not.toBeInTheDocument();
    });
  });
});