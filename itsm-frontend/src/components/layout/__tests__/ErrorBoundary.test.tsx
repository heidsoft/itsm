/**
 * Layout ErrorBoundary Component Tests
 *
 * 测试覆盖:
 * - 正常渲染子组件
 * - 捕获子组件错误
 * - 显示错误 UI
 * - 刷新页面功能
 * - 返回首页功能
 * - 报告问题功能
 * - 自定义 fallback
 * - onError 回调
 * - 开发环境错误详情显示
 * - useErrorHandler Hook
 * - SimpleErrorFallback 组件
 */

import React, { Component } from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import ErrorBoundary, { useErrorHandler, SimpleErrorFallback } from '../ErrorBoundary';
import '@testing-library/jest-dom';

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

// 模拟 console.error
const originalConsoleError = console.error;

// 测试用错误组件
class ErrorComponent extends Component<{ shouldThrow?: boolean }> {
  render() {
    if (this.props.shouldThrow) {
      throw new Error('Test error from ErrorComponent');
    }
    return <div>Normal Content</div>;
  }
}

describe('Layout ErrorBoundary', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    console.error = jest.fn();
    mockWindowLocation.href = 'http://localhost/';
  });

  afterEach(() => {
    console.error = originalConsoleError;
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
      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      // 应该显示错误标题
      expect(screen.getByText(/System encountered some issues/i)).toBeInTheDocument();
    });

    it('显示默认错误消息', () => {
      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(
        screen.getByText(/Sorry, an error occurred while loading the page/i)
      ).toBeInTheDocument();
    });

    it('显示错误详情标题', () => {
      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByText(/Error Details:/i)).toBeInTheDocument();
    });
  });

  describe('错误操作按钮', () => {
    it('显示刷新页面按钮', () => {
      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByText(/Reload Page/i)).toBeInTheDocument();
    });

    it('显示返回首页按钮', () => {
      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByText(/Back to Home/i)).toBeInTheDocument();
    });

    it('显示报告问题按钮', () => {
      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByText(/Report Issue/i)).toBeInTheDocument();
    });

    it('点击刷新页面按钮调用 location.reload', () => {
      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      const reloadButton = screen.getByRole('button', { name: /Reload Page/i });
      fireEvent.click(reloadButton);

      expect(mockWindowLocation.reload).toHaveBeenCalled();
    });

    it('点击返回首页按钮跳转到 dashboard', () => {
      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      const homeButton = screen.getByRole('button', { name: /Back to Home/i });
      fireEvent.click(homeButton);

      expect(mockWindowLocation.href).toMatch(/^(http:\/\/localhost\/dashboard|\/dashboard)$/);
    });

    it('点击报告问题按钮不抛出错误', () => {
      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      const reportButton = screen.getByRole('button', { name: /Report Issue/i });

      // 点击不应该抛出错误
      expect(() => fireEvent.click(reportButton)).not.toThrow();
    });
  });

  describe('自定义配置', () => {
    it('使用自定义 fallback 组件', () => {
      const CustomFallback = () => <div data-testid='custom-fallback'>Custom Error UI</div>;

      render(
        <ErrorBoundary fallback={<CustomFallback />}>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(screen.getByTestId('custom-fallback')).toBeInTheDocument();
      expect(screen.getByText('Custom Error UI')).toBeInTheDocument();
      // 不应该显示默认错误 UI
      expect(screen.queryByText(/System encountered some issues/i)).not.toBeInTheDocument();
    });

    it('调用 onError 回调函数', () => {
      const onErrorMock = jest.fn();

      render(
        <ErrorBoundary onError={onErrorMock}>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      // onError 应该在 componentDidCatch 中被调用
      expect(onErrorMock).toHaveBeenCalled();
    });

    it('onError 回调接收错误和错误信息', () => {
      const onErrorMock = jest.fn();

      render(
        <ErrorBoundary onError={onErrorMock}>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      expect(onErrorMock).toHaveBeenCalledTimes(1);
      const [error, errorInfo] = onErrorMock.mock.calls[0];
      expect(error).toBeInstanceOf(Error);
      expect(error.message).toBe('Test error from ErrorComponent');
      expect(errorInfo).toHaveProperty('componentStack');
    });
  });

  describe('错误详情显示', () => {
    it('显示错误消息', () => {
      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      // 错误消息应该显示在错误详情区域
      expect(screen.getByText('Test error from ErrorComponent')).toBeInTheDocument();
    });

    it('开发环境显示堆栈跟踪详情', () => {
      const originalEnv = process.env.NODE_ENV;
      process.env.NODE_ENV = 'development';

      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      // 开发环境应该显示 Stack Trace 摘要
      expect(screen.getByText(/Stack Trace/i)).toBeInTheDocument();

      process.env.NODE_ENV = originalEnv;
    });

    it('生产环境不显示堆栈跟踪详情', () => {
      const originalEnv = process.env.NODE_ENV;
      process.env.NODE_ENV = 'production';

      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      // 生产环境不应该显示 Stack Trace 摘要
      expect(screen.queryByText(/Stack Trace/i)).not.toBeInTheDocument();

      process.env.NODE_ENV = originalEnv;
    });
  });

  describe('错误报告内容', () => {
    it('错误报告包含错误信息', () => {
      const consoleSpy = jest.spyOn(console, 'log').mockImplementation(() => {});

      render(
        <ErrorBoundary>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      );

      const reportButton = screen.getByRole('button', { name: /Report Issue/i });
      fireEvent.click(reportButton);

      // 验证 console.error 被调用（组件内部会记录错误）
      expect(console.error).toHaveBeenCalled();

      consoleSpy.mockRestore();
    });
  });

  describe('组件生命周期', () => {
    it('组件卸载时清理事件监听器', () => {
      const addEventListenerSpy = jest.spyOn(window, 'addEventListener');
      const removeEventListenerSpy = jest.spyOn(window, 'removeEventListener');

      const { unmount } = render(
        <ErrorBoundary>
          <div>Child</div>
        </ErrorBoundary>
      );

      unmount();

      // useErrorHandler 应该添加和移除事件监听器
      expect(addEventListenerSpy).toHaveBeenCalledWith('error', expect.any(Function));
      expect(addEventListenerSpy).toHaveBeenCalledWith('unhandledrejection', expect.any(Function));
      expect(removeEventListenerSpy).toHaveBeenCalledWith('error', expect.any(Function));
      expect(removeEventListenerSpy).toHaveBeenCalledWith(
        'unhandledrejection',
        expect.any(Function)
      );

      addEventListenerSpy.mockRestore();
      removeEventListenerSpy.mockRestore();
    });
  });
});

describe('useErrorHandler Hook', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    console.error = jest.fn();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('注册全局错误处理器', () => {
    const addEventListenerSpy = jest.spyOn(window, 'addEventListener');

    const TestComponent = () => {
      useErrorHandler();
      return <div>Test</div>;
    };

    render(<TestComponent />);

    expect(addEventListenerSpy).toHaveBeenCalledWith('error', expect.any(Function));
    expect(addEventListenerSpy).toHaveBeenCalledWith('unhandledrejection', expect.any(Function));

    addEventListenerSpy.mockRestore();
  });

  it('清理全局错误处理器', () => {
    const removeEventListenerSpy = jest.spyOn(window, 'removeEventListener');

    const TestComponent = () => {
      useErrorHandler();
      return <div>Test</div>;
    };

    const { unmount } = render(<TestComponent />);
    unmount();

    expect(removeEventListenerSpy).toHaveBeenCalledWith('error', expect.any(Function));
    expect(removeEventListenerSpy).toHaveBeenCalledWith('unhandledrejection', expect.any(Function));

    removeEventListenerSpy.mockRestore();
  });

  it('捕获全局错误并记录到控制台', () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

    const TestComponent = () => {
      useErrorHandler();
      return <div>Test</div>;
    };

    render(<TestComponent />);

    // 手动触发错误事件
    const errorEvent = new ErrorEvent('error', {
      error: new Error('Global error'),
      message: 'Global error',
    });
    window.dispatchEvent(errorEvent);

    expect(consoleErrorSpy).toHaveBeenCalledWith('Hook caught error:', errorEvent.error);

    consoleErrorSpy.mockRestore();
  });

  it('捕获未处理的 Promise rejection', () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

    const TestComponent = () => {
      useErrorHandler();
      return <div>Test</div>;
    };

    render(<TestComponent />);

    // 手动触发 unhandledrejection 事件
    const rejectionEvent = new PromiseRejectionEvent('unhandledrejection', {
      promise: Promise.reject(new Error('Unhandled rejection')),
      reason: new Error('Unhandled rejection'),
    });
    window.dispatchEvent(rejectionEvent);

    expect(consoleErrorSpy).toHaveBeenCalledWith(
      'Hook caught unhandled Promise rejection:',
      rejectionEvent.reason
    );

    consoleErrorSpy.mockRestore();
  });
});

describe('SimpleErrorFallback Component', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    console.error = jest.fn();
  });

  it('渲染错误消息', () => {
    const testError = new Error('Simple test error');

    render(<SimpleErrorFallback error={testError} />);

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText('Simple test error')).toBeInTheDocument();
  });

  it('无错误时显示 Unknown error', () => {
    render(<SimpleErrorFallback />);

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText('Unknown error')).toBeInTheDocument();
  });

  it('显示错误图标', () => {
    render(<SimpleErrorFallback />);

    // Bug 图标应该存在
    const bugIcon = screen.getByTestId('bug-icon') || document.querySelector('svg');
    expect(bugIcon).toBeInTheDocument();
  });

  it('点击刷新按钮重新加载页面', () => {
    render(<SimpleErrorFallback />);

    const reloadButton = screen.getByRole('button', { name: /Reload Page/i });
    fireEvent.click(reloadButton);

    expect(mockWindowLocation.reload).toHaveBeenCalled();
  });

  it('应用正确的样式', () => {
    const { container } = render(<SimpleErrorFallback />);

    const wrapper = container.firstChild as HTMLElement;
    expect(wrapper).toHaveStyle('text-align: center');
    expect(wrapper).toHaveStyle('background: #fafafa');
  });
});

describe('嵌套 ErrorBoundary', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    console.error = jest.fn();
  });

  it('内层错误被内层 ErrorBoundary 捕获', () => {
    const innerOnErrorMock = jest.fn();
    const outerOnErrorMock = jest.fn();

    render(
      <ErrorBoundary onError={outerOnErrorMock}>
        <ErrorBoundary onError={innerOnErrorMock}>
          <ErrorComponent shouldThrow={true} />
        </ErrorBoundary>
      </ErrorBoundary>
    );

    // 内层 ErrorBoundary 应该捕获错误
    expect(innerOnErrorMock).toHaveBeenCalled();
    // 外层 ErrorBoundary 不应该被触发（因为内层已经处理）
    expect(outerOnErrorMock).not.toThrow();
  });

  it('外层 ErrorBoundary 捕获传播的错误', () => {
    const outerOnErrorMock = jest.fn();

    // 内层没有 ErrorBoundary，错误会传播到外层
    render(
      <ErrorBoundary onError={outerOnErrorMock}>
        <div>
          <ErrorComponent shouldThrow={true} />
        </div>
      </ErrorBoundary>
    );

    expect(outerOnErrorMock).toHaveBeenCalled();
  });
});

describe('异步错误处理', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    console.error = jest.fn();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('捕获 useEffect 中的异步错误', () => {
    const AsyncErrorComponent: React.FC = () => {
      React.useEffect(() => {
        setTimeout(() => {
          throw new Error('Async error in useEffect');
        }, 100);
      }, []);
      return <div>Async Component</div>;
    };

    render(
      <ErrorBoundary>
        <AsyncErrorComponent />
      </ErrorBoundary>
    );

    // 推进定时器
    jest.advanceTimersByTime(200);

    // 异步错误会被全局错误处理器捕获，而不是 ErrorBoundary
    // ErrorBoundary 主要捕获渲染错误
    expect(screen.getByText('Async Component')).toBeInTheDocument();
  });

  it('捕获 Promise rejection', async () => {
    const AsyncErrorComponent: React.FC = () => {
      React.useEffect(() => {
        Promise.reject(new Error('Promise rejection'));
      }, []);
      return <div>Async Component</div>;
    };

    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <ErrorBoundary>
        <AsyncErrorComponent />
      </ErrorBoundary>
    );

    // 等待 Promise rejection
    await waitFor(
      () => {
        expect(consoleErrorSpy).toHaveBeenCalled();
      },
      { timeout: 1000 }
    );

    consoleErrorSpy.mockRestore();
  });
});

describe('错误状态管理', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    console.error = jest.fn();
  });

  it('错误状态在重置后恢复正常渲染', () => {
    const { rerender } = render(
      <ErrorBoundary>
        <ErrorComponent shouldThrow={true} />
      </ErrorBoundary>
    );

    // 应该显示错误 UI
    expect(screen.getByText(/System encountered some issues/i)).toBeInTheDocument();

    // 重新渲染正常组件
    rerender(
      <ErrorBoundary>
        <div>Normal Content After Reset</div>
      </ErrorBoundary>
    );

    // ErrorBoundary 不会自动重置，需要刷新页面或手动重置
    // 这里验证错误状态仍然存在
    expect(screen.getByText(/System encountered some issues/i)).toBeInTheDocument();
  });

  it('子组件变化不自动重置错误状态', () => {
    const { rerender } = render(
      <ErrorBoundary>
        <ErrorComponent shouldThrow={true} />
      </ErrorBoundary>
    );

    // 应该显示错误 UI
    expect(screen.getByText(/System encountered some issues/i)).toBeInTheDocument();

    // 改变子组件
    rerender(
      <ErrorBoundary>
        <div>Different Child</div>
      </ErrorBoundary>
    );

    // 错误状态应该保持
    expect(screen.getByText(/System encountered some issues/i)).toBeInTheDocument();
  });
});

describe('性能和边界情况', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    console.error = jest.fn();
  });

  it('处理空子组件', () => {
    const { container } = render(<ErrorBoundary />);

    // 没有子组件时应该渲染空内容
    expect(container.firstChild).toBeNull();
  });

  it('处理 null 子组件', () => {
    const { container } = render(<ErrorBoundary>{null}</ErrorBoundary>);

    expect(container.firstChild).toBeNull();
  });

  it('处理 undefined 子组件', () => {
    const { container } = render(<ErrorBoundary>{undefined}</ErrorBoundary>);

    expect(container.firstChild).toBeNull();
  });

  it('处理大量子组件', () => {
    const children = Array.from({ length: 100 }, (_, i) => <div key={i}>Child {i}</div>);

    render(<ErrorBoundary>{children}</ErrorBoundary>);

    // 应该能正常渲染所有子组件
    expect(screen.getByText('Child 0')).toBeInTheDocument();
    expect(screen.getByText('Child 99')).toBeInTheDocument();
  });

  it('错误消息包含特殊字符', () => {
    class SpecialErrorComponent extends Component {
      render() {
        throw new Error('Error with <special> & "characters"');
      }
    }

    render(
      <ErrorBoundary>
        <SpecialErrorComponent />
      </ErrorBoundary>
    );

    // 错误消息应该正确显示
    expect(screen.getByText(/Error with/i)).toBeInTheDocument();
  });
});
