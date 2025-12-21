/**
 * 测试工具库
 * 提供常用的测试工具函数和Mock数据
 */

import { render, RenderOptions } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactElement } from 'react';
import antdTheme from '@/app/lib/antd-theme';

// 创建测试用的QueryClient
export const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        cacheTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });

// 测试包装器
const AllTheProviders = ({ children }: { children: React.ReactNode }) => {
  const queryClient = createTestQueryClient();

  return (
    <ConfigProvider theme={antdTheme}>
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    </ConfigProvider>
  );
};

// 自定义渲染函数
const customRender = (ui: ReactElement, options?: Omit<RenderOptions, 'wrapper'>) =>
  render(ui, { wrapper: AllTheProviders, ...options });

// 重新导出所有内容
export * from '@testing-library/react';
export { customRender as render };

// Mock数据生成器
export const mockData = {
  // 生成随机字符串
  string: (length: number = 10) => {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < length; i++) {
      result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result;
  },

  // 生成随机数字
  number: (min: number = 0, max: number = 100) => {
    return Math.floor(Math.random() * (max - min + 1)) + min;
  },

  // 生成随机布尔值
  boolean: () => Math.random() > 0.5,

  // 生成随机日期
  date: (start: Date = new Date(2020, 0, 1), end: Date = new Date()) => {
    return new Date(start.getTime() + Math.random() * (end.getTime() - start.getTime()));
  },

  // 生成随机邮箱
  email: () => `${mockData.string(8)}@${mockData.string(5)}.com`,

  // 生成随机手机号
  phone: () => `1${mockData.number(3, 9)}${mockData.number(10000000, 99999999)}`,

  // 生成随机数组
  array: (generator: () => any, length: number = 5) => {
    return Array.from({ length }, generator);
  },

  // 生成随机对象
  object: (generator: () => any) => generator(),
};

// Mock API响应
export const mockApiResponse = {
  success: (data: any) => ({
    code: 0,
    message: 'success',
    data,
    timestamp: new Date().toISOString(),
  }),

  error: (message: string = 'error', code: number = 1) => ({
    code,
    message,
    data: null,
    timestamp: new Date().toISOString(),
  }),

  paginated: (data: any[], page: number = 1, pageSize: number = 10) => ({
    code: 0,
    message: 'success',
    data: {
      items: data.slice((page - 1) * pageSize, page * pageSize),
      total: data.length,
      page,
      pageSize,
      totalPages: Math.ceil(data.length / pageSize),
    },
    timestamp: new Date().toISOString(),
  }),
};

// Mock工单数据
export const mockTickets = () =>
  mockData.array(
    () => ({
      id: mockData.string(8),
      title: `工单标题 ${mockData.string(10)}`,
      description: `工单描述 ${mockData.string(50)}`,
      status: ['open', 'in_progress', 'resolved', 'closed'][mockData.number(0, 3)],
      priority: ['low', 'medium', 'high', 'urgent'][mockData.number(0, 3)],
      category: ['incident', 'request', 'problem', 'change'][mockData.number(0, 3)],
      assignee: {
        id: mockData.string(6),
        name: `用户 ${mockData.string(5)}`,
        email: mockData.email(),
      },
      requester: {
        id: mockData.string(6),
        name: `请求者 ${mockData.string(5)}`,
        email: mockData.email(),
      },
      createdAt: mockData.date(),
      updatedAt: mockData.date(),
    }),
    20
  );

// Mock用户数据
export const mockUsers = () =>
  mockData.array(
    () => ({
      id: mockData.string(6),
      name: `用户 ${mockData.string(5)}`,
      email: mockData.email(),
      role: ['admin', 'user', 'agent', 'manager'][mockData.number(0, 3)],
      department: `部门 ${mockData.string(4)}`,
      status: ['active', 'inactive'][mockData.number(0, 1)],
      createdAt: mockData.date(),
    }),
    10
  );

// 性能测试工具
export const performanceUtils = {
  // 测量函数执行时间
  measureTime: async (fn: () => Promise<any> | any) => {
    const start = performance.now();
    const result = await fn();
    const end = performance.now();
    return { result, time: end - start };
  },

  // 模拟网络延迟
  delay: (ms: number) => new Promise(resolve => setTimeout(resolve, ms)),

  // 批量执行测试
  batchTest: async (tests: Array<() => Promise<any>>) => {
    const results = [];
    for (const test of tests) {
      const result = await performanceUtils.measureTime(test);
      results.push(result);
    }
    return results;
  },
};

// 测试辅助函数
export const testHelpers = {
  // 等待元素出现
  waitForElement: async (selector: string, timeout: number = 5000) => {
    const start = Date.now();
    while (Date.now() - start < timeout) {
      const element = document.querySelector(selector);
      if (element) return element;
      await new Promise(resolve => setTimeout(resolve, 100));
    }
    throw new Error(`Element ${selector} not found within ${timeout}ms`);
  },

  // 模拟用户输入
  simulateInput: (element: HTMLElement, value: string) => {
    const input = element as HTMLInputElement;
    input.value = value;
    input.dispatchEvent(new Event('input', { bubbles: true }));
    input.dispatchEvent(new Event('change', { bubbles: true }));
  },

  // 模拟点击
  simulateClick: (element: HTMLElement) => {
    element.dispatchEvent(new MouseEvent('click', { bubbles: true }));
  },
};
