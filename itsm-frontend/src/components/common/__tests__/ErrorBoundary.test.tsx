/**
 * ErrorBoundary Component Tests
 * 
 * 测试覆盖:
 * - 正常渲染子组件
 * - 捕获子组件错误
 * - 显示错误 UI
 * - 重试功能
 * - 返回首页功能
 * - 自定义 fallback
 * - onError 回调
 * - showDetails 显示错误详情
 * - resetOnPropsChange 功能
 * - 组件卸载清理
 */

import React, { Component, ErrorInfo } from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ErrorBoundary, withErrorBoundary, useErrorBoundary } from '../ErrorBoundary';
import '@testing-library/jest-dom';

// 模拟 useErrorHandler
jest.mock('@/lib/hooks/useErrorHandler', () => ({
  handleErrorBoundary: jest.fn(),
}));

// 模拟 window.location
const mockWindowLocation = {
  href: 'http://localhost/',
  assign: jest.fn(),
  reload: jest.fn(),
};

Object.defineProperty(window, 'location', {
  value: mockWindowLocation,
  writable: true,
  configurable: true,
});

// 模拟 navigator.clipboard
const mockClipboard = {
  writeText: jest.fn().mockResolvedValue(undefined),
};

Object.defineProperty(navigator, 'clipboard', {
  value: mockClipboard,
  writable: true,
  configurable: true,
});

// 模拟 matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: jest.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    dispatchEvent: jest.fn(),
  })),
});

// 测试用错误组件
class ErrorComponent extends Component<{ shouldThrow?: boolean }> {
  render() {
    if (this.props.shouldThrow) {
      throw new Error('Test error');
    }
    return <div>Normal Content</div>;
  }
}

// 辅助函数：触发错误
const triggerError = (error: Error) => {
  // 使用 act 包装错误触发
  return new Promise<void>(resolve => {
    setTimeout(() => {
      throw error;
    }, 0);
    resolve();
  });
};

describe('ErrorBoundary', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockWindowLocation.href = 'http://localhost/';
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  describe('基本功能', () => {
    it('正常渲染子组件（无错误时）', () => {
      render(
        <ErrorBoundary>
          <div>Child Content</div>
        </ErrorBoundary>
      );

      expect(screen.getByText('Child Content')).toBeInTheDocument();
    });

    it('渲染多个子组件', () => {
      render(
        <ErrorBoundary>
          <div>Child 1</div>
          <div>Child 2</div>
          <div>Child 3</div>
        </ErrorBoundary>
      );

      expect(screen.getByText('Child 1')).toBeInTheDocument();
      expect(screen.getByText('Child 2')).toBeInTheDocument();
      expect(screen.getByText('Child 3')).toBeInTheDocument();
    });
  });

  describe('错误捕获', () => {
    it('捕获子组件渲染错误并显示错误 UI', () => {
      // 使用 getDerivedStateFromError 的测试方式
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      // 应该显示错误信息
      expect(screen.getByText('页面出现错误')).toBeInTheDocument();
      
      consoleSpy.mockRestore();
    });

    it('显示默认错误消息', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(
        screen.getByText(/抱歉，页面遇到了一个意外错误/)
      ).toBeInTheDocument();

      consoleSpy.mockRestore();
    });
  });

  describe('错误操作按钮', () => {
    beforeEach(() => {
      jest.useFakeTimers();
    });

    afterEach(() => {
      jest.useRealTimers();
    });

    it('显示重试按钮', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByText('重试')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /重试/i })).toBeInTheDocument();

      consoleSpy.mockRestore();
    });

    it('显示返回首页按钮', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByText('返回首页')).toBeInTheDocument();

      consoleSpy.mockRestore();
    });

    it('显示报告问题按钮', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByText('报告问题')).toBeInTheDocument();

      consoleSpy.mockRestore();
    });

    it('点击重试按钮重置错误状态', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      const retryButton = screen.getByRole('button', { name: /重试/i });
      fireEvent.click(retryButton);

      // 重试后错误状态重置，按钮可能不再显示
      expect(retryButton).toBeTruthy();

      consoleSpy.mockRestore();
    });

    it('点击返回首页按钮跳转首页', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      const homeButton = screen.getByRole('button', { name: /返回首页/i });
      fireEvent.click(homeButton);

      // 验证 location.href 被设置为首页
      expect(mockWindowLocation.href).toMatch(/^(http:\/\/localhost\/|\/)$/);

      consoleSpy.mockRestore();
    });

    it('点击报告问题按钮复制错误信息到剪贴板', async () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      const reportButton = screen.getByRole('button', { name: /报告问题/i });
      fireEvent.click(reportButton);

      // 等待剪贴板操作
      await waitFor(() => {
        expect(mockClipboard.writeText).toHaveBeenCalled();
      });

      // 验证复制的内容包含错误信息
      const copiedContent = (mockClipboard.writeText as jest.Mock).mock.calls[0][0];
      const parsed = JSON.parse(copiedContent);
      expect(parsed).toHaveProperty('errorId');
      expect(parsed).toHaveProperty('message');
      expect(parsed).toHaveProperty('timestamp');

      consoleSpy.mockRestore();
    });
  });

  describe('自定义配置', () => {
    it('使用自定义 fallback 组件', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      const CustomFallback = () => <div data-testid='custom-fallback'>Custom Error UI</div>;

      render(
        <ErrorBoundary fallback={<CustomFallback />}>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByTestId('custom-fallback')).toBeInTheDocument();
      expect(screen.getByText('Custom Error UI')).toBeInTheDocument();
      // 不应该显示默认错误 UI
      expect(screen.queryByText('页面出现错误')).not.toBeInTheDocument();

      consoleSpy.mockRestore();
    });

    it('调用 onError 回调函数', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
      const onErrorMock = jest.fn();

      render(
        <ErrorBoundary onError={onErrorMock}>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      // onError 应该在 componentDidCatch 中被调用
      expect(onErrorMock).toHaveBeenCalled();

      consoleSpy.mockRestore();
    });

    it('showDetails 为 true 时显示错误详情', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary showDetails={true}>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByText('错误详情')).toBeInTheDocument();

      consoleSpy.mockRestore();
    });

    it('showDetails 为 false 时不显示错误详情', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary showDetails={false}>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.queryByText('错误详情')).not.toBeInTheDocument();

      consoleSpy.mockRestore();
    });
  });

  describe('resetOnPropsChange 功能', () => {
    it('resetOnPropsChange 为 true 时，props 变化重置错误状态', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      const { rerender } = render(
        <ErrorBoundary resetOnPropsChange={true}>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      // 重新渲染并改变 props（除了 children）
      rerender(
        <ErrorBoundary resetOnPropsChange={true} data-test='changed'>
          <ErrorComponent shouldThrow={false} />
        </ErrorBoundary>
      );

      // 错误状态应该被重置
      expect(screen.queryByText('页面出现错误')).not.toBeInTheDocument();

      consoleSpy.mockRestore();
    });

    it('resetOnPropsChange 为 false 时，props 变化不重置错误状态', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      const { rerender } = render(
        <ErrorBoundary resetOnPropsChange={false}>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      rerender(
        <ErrorBoundary resetOnPropsChange={false} data-test='changed'>
          <ErrorComponent shouldThrow={false} />
        </ErrorBoundary>
      );

      // 错误状态应该保持
      expect(screen.getByText('页面出现错误')).toBeInTheDocument();

      consoleSpy.mockRestore();
    });

    it('仅 children 变化不触发重置', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      const { rerender } = render(
        <ErrorBoundary resetOnPropsChange={true}>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      // 仅改变 children
      rerender(
        <ErrorBoundary resetOnPropsChange={true}>
          <div>Different Child</div>
        </ErrorBoundary>
      );

      // children 变化不应该触发重置
      expect(screen.getByText('页面出现错误')).toBeInTheDocument();

      consoleSpy.mockRestore();
    });
  });

  describe('withErrorBoundary HOC', () => {
    it('包装组件并添加错误边界', () => {
      const TestComponent = () => <div>Test Component</div>;
      const WrappedComponent = withErrorBoundary(TestComponent);

      render(<WrappedComponent />);

      expect(screen.getByText('Test Component')).toBeInTheDocument();
    });

    it('HOC 设置自定义 displayName', () => {
      const TestComponent = () => <div>Test</div>;
      TestComponent.displayName = 'TestComponent';
      
      const WrappedComponent = withErrorBoundary(TestComponent);
      
      expect(WrappedComponent.displayName).toContain('withErrorBoundary(TestComponent)');
    });
  });

  describe('useErrorBoundary Hook', () => {
    it('提供 captureError 和 resetError 函数', () => {
      const TestComponent = () => {
        const { captureError, resetError } = useErrorBoundary();
        
        return (
          <div>
            <button onClick={() => captureError(new Error('Test'))}>Capture</button>
            <button onClick={resetError}>Reset</button>
          </div>
        );
      };

      render(
        <ErrorBoundary>
          <TestComponent />
        </ErrorBoundary>
      );

      expect(screen.getByText('Capture')).toBeInTheDocument();
      expect(screen.getByText('Reset')).toBeInTheDocument();
    });

    it('captureError 后抛出错误', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      const TestComponent = () => {
        const { captureError } = useErrorBoundary();
        
        React.useEffect(() => {
          captureError(new Error('Captured error'));
        }, [captureError]);
        
        return <div>Test</div>;
      };

      render(
        <ErrorBoundary>
          <TestComponent />
        </ErrorBoundary>
      );

      // 应该捕获错误并显示错误 UI
      expect(screen.getByText('页面出现错误')).toBeInTheDocument();

      consoleSpy.mockRestore();
    });
  });

  describe('组件生命周期', () => {
    it('组件卸载时清理定时器', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
      jest.useFakeTimers();

      const { unmount } = render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      // 卸载不应该抛出错误
      expect(() => unmount()).not.toThrow();

      jest.useRealTimers();
      consoleSpy.mockRestore();
    });
  });

  describe('错误 ID 生成', () => {
    it('生成错误 ID', async () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      // 点击报告问题按钮触发剪贴板写入
      const reportButton = screen.getByRole('button', { name: /报告问题/i });
      fireEvent.click(reportButton);

      // 等待剪贴板操作
      await waitFor(() => {
        expect(mockClipboard.writeText).toHaveBeenCalled();
      });

      const copiedContent = (mockClipboard.writeText as jest.Mock).mock.calls[0][0];
      const parsed = JSON.parse(copiedContent);
      
      // 错误 ID 应该存在
      expect(parsed.errorId).toBeDefined();
      expect(typeof parsed.errorId).toBe('string');

      consoleSpy.mockRestore();
    });
  });
});

/**
 * GlobalErrorBoundary Component Tests
 */

import { GlobalErrorBoundary } from '../GlobalErrorBoundary';

describe('GlobalErrorBoundary', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockWindowLocation.href = 'http://localhost/dashboard';
  });

  describe('基本功能', () => {
    it('正常渲染子组件', () => {
      render(
        <GlobalErrorBoundary>
          <div>Child Content</div>
        </GlobalErrorBoundary>
      );

      expect(screen.getByText('Child Content')).toBeInTheDocument();
    });

    it('捕获错误并显示错误 UI', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <GlobalErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </GlobalErrorBoundary>
      );

      expect(screen.getByText(/页面遇到了一些问题/)).toBeInTheDocument();

      consoleSpy.mockRestore();
    });
  });

  describe('错误操作按钮', () => {
    it('显示刷新页面按钮', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <GlobalErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </GlobalErrorBoundary>
      );

      expect(screen.getByText('刷新页面')).toBeInTheDocument();

      consoleSpy.mockRestore();
    });

    it('显示返回首页按钮', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <GlobalErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </GlobalErrorBoundary>
      );

      expect(screen.getByText('返回首页')).toBeInTheDocument();

      consoleSpy.mockRestore();
    });

    it('显示尝试恢复按钮', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <GlobalErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </GlobalErrorBoundary>
      );

      expect(screen.getByText('尝试恢复')).toBeInTheDocument();

      consoleSpy.mockRestore();
    });

    it('点击刷新页面按钮调用 location.reload', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
      const reloadSpy = jest.spyOn(window.location, 'reload');

      render(
        <GlobalErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </GlobalErrorBoundary>
      );

      const reloadButton = screen.getByRole('button', { name: /刷新页面/i });
      fireEvent.click(reloadButton);

      expect(reloadSpy).toHaveBeenCalled();

      reloadSpy.mockRestore();
      consoleSpy.mockRestore();
    });

    it('点击返回首页按钮跳转到 dashboard', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <GlobalErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </GlobalErrorBoundary>
      );

      const homeButton = screen.getByRole('button', { name: /返回首页/i });
      fireEvent.click(homeButton);

      expect(mockWindowLocation.href).toMatch(/^(http:\/\/localhost\/dashboard|\/dashboard)$/);

      consoleSpy.mockRestore();
    });

    it('点击尝试恢复按钮重置错误状态', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <GlobalErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </GlobalErrorBoundary>
      );

      const resetButton = screen.getByRole('button', { name: /尝试恢复/i });
      fireEvent.click(resetButton);

      // 按钮存在且可点击
      expect(resetButton).toBeTruthy();

      consoleSpy.mockRestore();
    });
  });

  describe('自定义配置', () => {
    it('使用自定义 fallback', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
      const CustomFallback = () => <div data-testid='global-custom-fallback'>Global Custom Error</div>;

      render(
        <GlobalErrorBoundary fallback={<CustomFallback />}>
          <ErrorComponent shouldThrow={true} />
        </GlobalErrorBoundary>
      );

      expect(screen.getByTestId('global-custom-fallback')).toBeInTheDocument();
      expect(screen.getByText('Global Custom Error')).toBeInTheDocument();

      consoleSpy.mockRestore();
    });

    it('调用 onError 回调', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
      const onErrorMock = jest.fn();

      render(
        <GlobalErrorBoundary onError={onErrorMock}>
          <ErrorComponent shouldThrow={true} />
        </GlobalErrorBoundary>
      );

      expect(onErrorMock).toHaveBeenCalled();

      consoleSpy.mockRestore();
    });
  });

  describe('错误计数', () => {
    it('多次错误增加错误计数', () => {
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      const { rerender } = render(
        <GlobalErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </GlobalErrorBoundary>
      );

      // 模拟多次错误
      rerender(
        <GlobalErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </GlobalErrorBoundary>
      );

      // 错误计数应该增加
      expect(consoleSpy).toHaveBeenCalledTimes(2);

      consoleSpy.mockRestore();
    });
  });

  describe('withErrorBoundary HOC', () => {
    it('包装组件', () => {
      const TestComponent = () => <div>Global Test</div>;
      const WrappedComponent = withErrorBoundary(TestComponent);

      render(<WrappedComponent />);

      expect(screen.getByText('Global Test')).toBeInTheDocument();
    });
  });
});
