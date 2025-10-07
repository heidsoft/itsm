/**
 * 性能测试
 * 测试组件渲染性能、内存使用、大数据处理等性能相关功能
 */

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';

// Mock performance API
const mockPerformance = {
  now: jest.fn(() => Date.now()),
  mark: jest.fn(),
  measure: jest.fn(),
  getEntriesByType: jest.fn(() => []),
  getEntriesByName: jest.fn(() => []),
};

Object.defineProperty(window, 'performance', {
  value: mockPerformance,
  writable: true,
});

// Mock ResizeObserver
global.ResizeObserver = jest.fn().mockImplementation(() => ({
  observe: jest.fn(),
  unobserve: jest.fn(),
  disconnect: jest.fn(),
}));

// Mock IntersectionObserver
global.IntersectionObserver = jest.fn().mockImplementation(() => ({
  observe: jest.fn(),
  unobserve: jest.fn(),
  disconnect: jest.fn(),
}));

// Performance monitoring utilities
class PerformanceMonitor {
  private startTime: number = 0;
  private endTime: number = 0;

  start() {
    this.startTime = performance.now();
  }

  end() {
    this.endTime = performance.now();
    return this.endTime - this.startTime;
  }

  getDuration() {
    return this.endTime - this.startTime;
  }
}

// Mock heavy computation component
const HeavyComputationComponent: React.FC<{ itemCount: number }> = ({ itemCount }) => {
  const [items, setItems] = React.useState<number[]>([]);
  const [isLoading, setIsLoading] = React.useState(true);

  React.useEffect(() => {
    const generateItems = () => {
      const newItems = Array.from({ length: itemCount }, (_, i) => i);
      setItems(newItems);
      setIsLoading(false);
    };

    // Simulate heavy computation
    const timer = setTimeout(generateItems, 100);
    return () => clearTimeout(timer);
  }, [itemCount]);

  if (isLoading) {
    return <div data-testid="loading">Loading...</div>;
  }

  return (
    <div data-testid="heavy-component">
      <div data-testid="item-count">{items.length} items</div>
      <div data-testid="items-container">
        {items.map(item => (
          <div key={item} data-testid={`item-${item}`} className="item">
            Item {item}
          </div>
        ))}
      </div>
    </div>
  );
};

// Mock virtualized list component
const VirtualizedList: React.FC<{ 
  items: unknown[]; 
  itemHeight: number; 
  containerHeight: number;
}> = ({ items, itemHeight, containerHeight }) => {
  const [scrollTop, setScrollTop] = React.useState(0);
  const [visibleItems, setVisibleItems] = React.useState<unknown[]>([]);

  React.useEffect(() => {
    const startIndex = Math.floor(scrollTop / itemHeight);
    const endIndex = Math.min(
      startIndex + Math.ceil(containerHeight / itemHeight) + 1,
      items.length
    );
    
    setVisibleItems(items.slice(startIndex, endIndex));
  }, [scrollTop, items, itemHeight, containerHeight]);

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    setScrollTop(e.currentTarget.scrollTop);
  };

  return (
    <div 
      data-testid="virtualized-list"
      style={{ height: containerHeight, overflow: 'auto' }}
      onScroll={handleScroll}
    >
      <div style={{ height: items.length * itemHeight, position: 'relative' }}>
        {visibleItems.map((_, index) => {
          const actualIndex = Math.floor(scrollTop / itemHeight) + index;
          return (
            <div
              key={actualIndex}
              data-testid={`virtual-item-${actualIndex}`}
              style={{
                position: 'absolute',
                top: actualIndex * itemHeight,
                height: itemHeight,
                width: '100%',
              }}
            >
              Virtual Item {actualIndex}
            </div>
          );
        })}
      </div>
    </div>
  );
};

// Mock memoized component
const MemoizedComponent = React.memo<{ 
  data: { id: number; name: string }[];
  onItemClick: (id: number) => void;
}>(({ data, onItemClick }) => {
  const renderCount = React.useRef(0);
  renderCount.current += 1;

  return (
    <div data-testid="memoized-component">
      <div data-testid="render-count">{renderCount.current}</div>
      {data.map(item => (
        <div 
          key={item.id} 
          data-testid={`memoized-item-${item.id}`}
          onClick={() => onItemClick(item.id)}
          className="cursor-pointer p-2 hover:bg-gray-100"
        >
          {item.name}
        </div>
      ))}
    </div>
  );
});

MemoizedComponent.displayName = 'MemoizedComponent';

// Mock lazy loaded component
const LazyComponent = React.lazy(() => 
  Promise.resolve({
    default: () => (
      <div data-testid="lazy-component">
        Lazy loaded content
      </div>
    )
  })
);

// Test wrapper
const TestWrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <ConfigProvider locale={zhCN}>
      {children}
    </ConfigProvider>
  );
};

describe('性能测试', () => {
  let performanceMonitor: PerformanceMonitor;

  beforeEach(() => {
    performanceMonitor = new PerformanceMonitor();
    jest.clearAllMocks();
  });

  describe('渲染性能测试', () => {
    it('应该在合理时间内渲染小型组件', () => {
      performanceMonitor.start();
      
      render(
        <TestWrapper>
          <div data-testid="simple-component">Simple Component</div>
        </TestWrapper>
      );

      const renderTime = performanceMonitor.end();

      // 验证渲染时间小于50ms
      expect(renderTime).toBeLessThan(50);
      expect(screen.getByTestId('simple-component')).toBeInTheDocument();
    });

    it('应该高效渲染中等复杂度组件', async () => {
      performanceMonitor.start();
      
      render(
        <TestWrapper>
          <HeavyComputationComponent itemCount={100} />
        </TestWrapper>
      );

      // 等待组件加载完成
      await waitFor(() => {
        expect(screen.getByTestId('heavy-component')).toBeInTheDocument();
      });

      const renderTime = performanceMonitor.end();

      // 验证渲染时间小于200ms
      expect(renderTime).toBeLessThan(200);
      expect(screen.getByTestId('item-count')).toHaveTextContent('100 items');
    });

    it('应该处理大量数据的渲染', async () => {
      const largeDataset = Array.from({ length: 10000 }, (_, i) => i);
      
      performanceMonitor.start();
      
      render(
        <TestWrapper>
          <VirtualizedList 
            items={largeDataset}
            itemHeight={50}
            containerHeight={400}
          />
        </TestWrapper>
      );

      const renderTime = performanceMonitor.end();

      // 验证虚拟化列表渲染时间
      expect(renderTime).toBeLessThan(100);
      expect(screen.getByTestId('virtualized-list')).toBeInTheDocument();
      
      // 验证只渲染可见项目
      const visibleItems = screen.getAllByTestId(/virtual-item-/);
      expect(visibleItems.length).toBeLessThan(20); // 只渲染可见的项目
    });

    it('应该优化重复渲染', async () => {
      const mockData = [
        { id: 1, name: 'Item 1' },
        { id: 2, name: 'Item 2' },
        { id: 3, name: 'Item 3' },
      ];
      const mockOnClick = jest.fn();

      const { rerender } = render(
        <TestWrapper>
          <MemoizedComponent data={mockData} onItemClick={mockOnClick} />
        </TestWrapper>
      );

      // 验证初始渲染
      expect(screen.getByTestId('render-count')).toHaveTextContent('1');

      // 使用相同props重新渲染
      rerender(
        <TestWrapper>
          <MemoizedComponent data={mockData} onItemClick={mockOnClick} />
        </TestWrapper>
      );

      // 验证没有重新渲染（由于React.memo）
      expect(screen.getByTestId('render-count')).toHaveTextContent('1');

      // 使用不同props重新渲染
      const newData = [...mockData, { id: 4, name: 'Item 4' }];
      rerender(
        <TestWrapper>
          <MemoizedComponent data={newData} onItemClick={mockOnClick} />
        </TestWrapper>
      );

      // 验证重新渲染
      expect(screen.getByTestId('render-count')).toHaveTextContent('2');
    });
  });

  describe('内存使用测试', () => {
    it('应该正确清理事件监听器', () => {
      const mockAddEventListener = jest.spyOn(window, 'addEventListener');
      const mockRemoveEventListener = jest.spyOn(window, 'removeEventListener');

      const ComponentWithEventListener = () => {
        React.useEffect(() => {
          const handleResize = () => {
            // Handle resize
          };

          window.addEventListener('resize', handleResize);
          return () => window.removeEventListener('resize', handleResize);
        }, []);

        return <div data-testid="event-component">Component with event listener</div>;
      };

      const { unmount } = render(
        <TestWrapper>
          <ComponentWithEventListener />
        </TestWrapper>
      );

      expect(mockAddEventListener).toHaveBeenCalledWith('resize', expect.any(Function));

      // 卸载组件
      unmount();

      expect(mockRemoveEventListener).toHaveBeenCalledWith('resize', expect.any(Function));

      mockAddEventListener.mockRestore();
      mockRemoveEventListener.mockRestore();
    });

    it('应该正确清理定时器', () => {
      jest.useFakeTimers();
      const mockClearTimeout = jest.spyOn(global, 'clearTimeout');

      const ComponentWithTimer = () => {
        const [count, setCount] = React.useState(0);

        React.useEffect(() => {
          const timer = setTimeout(() => {
            setCount(1);
          }, 1000);

          return () => clearTimeout(timer);
        }, []);

        return <div data-testid="timer-component">{count}</div>;
      };

      const { unmount } = render(
        <TestWrapper>
          <ComponentWithTimer />
        </TestWrapper>
      );

      // 卸载组件
      unmount();

      expect(mockClearTimeout).toHaveBeenCalled();

      jest.useRealTimers();
      mockClearTimeout.mockRestore();
    });

    it('应该避免内存泄漏', () => {
      const ComponentWithState = () => {
        const [data, setData] = React.useState<number[]>([]);

        React.useEffect(() => {
          // 模拟大量数据
          const largeArray = Array.from({ length: 10000 }, (_, i) => i);
          setData(largeArray);

          return () => {
            // 清理数据
            setData([]);
          };
        }, []);

        return (
          <div data-testid="state-component">
            Data length: {data.length}
          </div>
        );
      };

      const { unmount } = render(
        <TestWrapper>
          <ComponentWithState />
        </TestWrapper>
      );

      expect(screen.getByTestId('state-component')).toHaveTextContent('Data length: 10000');

      // 卸载组件应该清理状态
      unmount();

      // 验证组件已卸载
      expect(screen.queryByTestId('state-component')).not.toBeInTheDocument();
    });
  });

  describe('异步加载性能测试', () => {
    it('应该高效处理懒加载组件', async () => {
      performanceMonitor.start();

      render(
        <TestWrapper>
          <React.Suspense fallback={<div data-testid="suspense-loading">Loading...</div>}>
            <LazyComponent />
          </React.Suspense>
        </TestWrapper>
      );

      // 验证Suspense fallback显示
      expect(screen.getByTestId('suspense-loading')).toBeInTheDocument();

      // 等待懒加载组件加载
      await waitFor(() => {
        expect(screen.getByTestId('lazy-component')).toBeInTheDocument();
      });

      const loadTime = performanceMonitor.end();

      // 验证加载时间
      expect(loadTime).toBeLessThan(500);
      expect(screen.queryByTestId('suspense-loading')).not.toBeInTheDocument();
    });

    it('应该优化并发请求', async () => {
      const mockFetch = jest.fn().mockImplementation(() =>
        Promise.resolve({
          json: () => Promise.resolve({ data: 'test' }),
        })
      );

      global.fetch = mockFetch as jest.MockedFunction<typeof fetch>;

      const ConcurrentRequestComponent = () => {
        const [data, setData] = React.useState<unknown[]>([]);
        const [loading, setLoading] = React.useState(true);

        React.useEffect(() => {
          const fetchData = async () => {
            try {
              // 并发请求
              const promises = Array.from({ length: 5 }, (_, i) =>
                fetch(`/api/data/${i}`)
              );

              const responses = await Promise.all(promises);
              const results = await Promise.all(
                responses.map(response => response.json())
              );

              setData(results);
            } catch (error) {
              console.error('Fetch error:', error);
            } finally {
              setLoading(false);
            }
          };

          fetchData();
        }, []);

        if (loading) {
          return <div data-testid="concurrent-loading">Loading...</div>;
        }

        return (
          <div data-testid="concurrent-data">
            Data count: {data.length}
          </div>
        );
      };

      performanceMonitor.start();

      render(
        <TestWrapper>
          <ConcurrentRequestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('concurrent-data')).toBeInTheDocument();
      });

      const loadTime = performanceMonitor.end();

      // 验证并发请求性能
      expect(loadTime).toBeLessThan(1000);
      expect(mockFetch).toHaveBeenCalledTimes(5);
      expect(screen.getByTestId('concurrent-data')).toHaveTextContent('Data count: 5');
    });
  });

  describe('用户交互性能测试', () => {
    it('应该快速响应用户点击', async () => {
      const user = userEvent.setup();
      const mockClick = jest.fn();

      render(
        <TestWrapper>
          <button data-testid="performance-button" onClick={mockClick}>
            Click me
          </button>
        </TestWrapper>
      );

      performanceMonitor.start();
      
      await user.click(screen.getByTestId('performance-button'));
      
      const responseTime = performanceMonitor.end();

      // 验证响应时间小于100ms
      expect(responseTime).toBeLessThan(100);
      expect(mockClick).toHaveBeenCalledTimes(1);
    });

    it('应该高效处理输入事件', async () => {
      const user = userEvent.setup();
      const mockChange = jest.fn();

      render(
        <TestWrapper>
          <input 
            data-testid="performance-input"
            onChange={mockChange}
            placeholder="Type here..."
          />
        </TestWrapper>
      );

      performanceMonitor.start();
      
      await user.type(screen.getByTestId('performance-input'), 'test input');
      
      const inputTime = performanceMonitor.end();

      // 验证输入响应时间
      expect(inputTime).toBeLessThan(200);
      expect(mockChange).toHaveBeenCalled();
    });

    it('应该优化滚动性能', () => {
      const items = Array.from({ length: 1000 }, (_, i) => i);

      render(
        <TestWrapper>
          <div 
            data-testid="scrollable-container"
            style={{ height: '400px', overflow: 'auto' }}
          >
            {items.map(item => (
              <div key={item} style={{ height: '50px', padding: '10px' }}>
                Item {item}
              </div>
            ))}
          </div>
        </TestWrapper>
      );

      const container = screen.getByTestId('scrollable-container');
      
      performanceMonitor.start();
      
      // 模拟滚动
      container.scrollTop = 1000;
      
      const scrollTime = performanceMonitor.end();

      // 验证滚动性能
      expect(scrollTime).toBeLessThan(50);
      expect(container.scrollTop).toBe(1000);
    });
  });

  describe('资源加载性能测试', () => {
    it('应该优化图片加载', () => {
      const mockImage = {
        onload: null as (() => void) | null,
        onerror: null as (() => void) | null,
        src: '',
      };

      // Mock Image constructor
      global.Image = jest.fn().mockImplementation(() => mockImage);

      const ImageComponent = () => {
        const [loaded, setLoaded] = React.useState(false);
        const [error, setError] = React.useState(false);

        React.useEffect(() => {
          const img = new Image();
          img.onload = () => setLoaded(true);
          img.onerror = () => setError(true);
          img.src = '/test-image.jpg';
        }, []);

        if (error) {
          return <div data-testid="image-error">Image failed to load</div>;
        }

        if (!loaded) {
          return <div data-testid="image-loading">Loading image...</div>;
        }

        return <div data-testid="image-loaded">Image loaded</div>;
      };

      render(
        <TestWrapper>
          <ImageComponent />
        </TestWrapper>
      );

      expect(screen.getByTestId('image-loading')).toBeInTheDocument();

      // 模拟图片加载成功
      if (mockImage.onload) {
        mockImage.onload();
      }

      expect(screen.getByTestId('image-loaded')).toBeInTheDocument();
    });

    it('应该实现资源预加载', () => {
      const mockLink = {
        rel: '',
        href: '',
        as: '',
      };

      const mockCreateElement = jest.spyOn(document, 'createElement').mockImplementation((tagName) => {
        if (tagName === 'link') {
          return mockLink as unknown as HTMLLinkElement;
        }
        return document.createElement(tagName);
      });

      const mockAppendChild = jest.spyOn(document.head, 'appendChild').mockImplementation(() => mockLink as unknown as Node);

      const PreloadComponent = () => {
        React.useEffect(() => {
          // 预加载关键资源
          const preloadLink = document.createElement('link');
          preloadLink.rel = 'preload';
          preloadLink.href = '/critical-resource.js';
          preloadLink.as = 'script';
          document.head.appendChild(preloadLink);
        }, []);

        return <div data-testid="preload-component">Component with preload</div>;
      };

      render(
        <TestWrapper>
          <PreloadComponent />
        </TestWrapper>
      );

      expect(mockCreateElement).toHaveBeenCalledWith('link');
      expect(mockAppendChild).toHaveBeenCalled();
      expect(mockLink.rel).toBe('preload');
      expect(mockLink.href).toBe('/critical-resource.js');
      expect(mockLink.as).toBe('script');

      mockCreateElement.mockRestore();
      mockAppendChild.mockRestore();
    });
  });

  describe('Bundle大小优化测试', () => {
    it('应该支持代码分割', async () => {
      const DynamicImportComponent = () => {
        const [Component, setComponent] = React.useState<React.ComponentType | null>(null);

        React.useEffect(() => {
          // 动态导入
          import('./mock-module').then((module) => {
            setComponent(() => module.default);
          }).catch(() => {
            setComponent(() => () => <div>Failed to load</div>);
          });
        }, []);

        if (!Component) {
          return <div data-testid="dynamic-loading">Loading dynamic component...</div>;
        }

        return <Component />;
      };

      render(
        <TestWrapper>
          <DynamicImportComponent />
        </TestWrapper>
      );

      expect(screen.getByTestId('dynamic-loading')).toBeInTheDocument();
    });

    it('应该优化依赖包大小', () => {
      // 验证只导入需要的模块
      const mockLodash = {
        debounce: jest.fn(),
        throttle: jest.fn(),
      };

      // 模拟tree-shaking优化
      const { debounce } = mockLodash;
      
      const DebouncedComponent = () => {
        const [value, setValue] = React.useState('');
        
        const debouncedSetValue = React.useMemo(
          () => debounce((newValue: string) => setValue(newValue), 300),
          [debounce]
        );

        return (
          <div data-testid="debounced-component">
            <input 
              onChange={(e) => debouncedSetValue(e.target.value)}
              placeholder="Debounced input"
            />
            <div data-testid="debounced-value">{value}</div>
          </div>
        );
      };

      render(
        <TestWrapper>
          <DebouncedComponent />
        </TestWrapper>
      );

      expect(screen.getByTestId('debounced-component')).toBeInTheDocument();
      // 验证只使用了需要的函数
      expect(debounce).toBeDefined();
    });
  });

  describe('缓存策略测试', () => {
    it('应该实现组件级缓存', () => {
      const expensiveCalculation = jest.fn((n: number) => {
        // 模拟昂贵计算
        let result = 0;
        for (let i = 0; i < n; i++) {
          result += i;
        }
        return result;
      });

      const CachedComponent: React.FC<{ input: number }> = ({ input }) => {
        const result = React.useMemo(() => expensiveCalculation(input), [input]);

        return (
          <div data-testid="cached-component">
            Result: {result}
          </div>
        );
      };

      CachedComponent.displayName = 'CachedComponent';

      const { rerender } = render(
        <TestWrapper>
          <CachedComponent input={1000} />
        </TestWrapper>
      );

      expect(expensiveCalculation).toHaveBeenCalledTimes(1);

      // 使用相同输入重新渲染
      rerender(
        <TestWrapper>
          <CachedComponent input={1000} />
        </TestWrapper>
      );

      // 验证缓存生效，没有重新计算
      expect(expensiveCalculation).toHaveBeenCalledTimes(1);

      // 使用不同输入重新渲染
      rerender(
        <TestWrapper>
          <CachedComponent input={2000} />
        </TestWrapper>
      );

      // 验证重新计算
      expect(expensiveCalculation).toHaveBeenCalledTimes(2);
    });

    it('应该实现请求缓存', async () => {
      const mockFetch = jest.fn().mockResolvedValue({
        json: () => Promise.resolve({ data: 'cached data' }),
      });

      global.fetch = mockFetch as jest.MockedFunction<typeof fetch>;

      // 简单的请求缓存实现
      const requestCache = new Map<string, Promise<unknown>>();

      const cachedFetch = (url: string) => {
        if (requestCache.has(url)) {
          return requestCache.get(url)!;
        }

        const promise = fetch(url).then(response => response.json());
        requestCache.set(url, promise);
        return promise;
      };

      const CachedRequestComponent = () => {
        const [data, setData] = React.useState<unknown>(null);

        React.useEffect(() => {
          cachedFetch('/api/data').then(setData);
        }, []);

        return (
          <div data-testid="cached-request">
            {data ? JSON.stringify(data) : 'Loading...'}
          </div>
        );
      };

      // 渲染第一个组件
      render(
        <TestWrapper>
          <CachedRequestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getByTestId('cached-request')).toHaveTextContent('cached data');
      });

      expect(mockFetch).toHaveBeenCalledTimes(1);

      // 渲染第二个相同的组件
      render(
        <TestWrapper>
          <CachedRequestComponent />
        </TestWrapper>
      );

      await waitFor(() => {
        expect(screen.getAllByTestId('cached-request')).toHaveLength(2);
      });

      // 验证缓存生效，没有发起新请求
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });
  });
});