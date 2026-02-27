/**
 * API Mock Service
 * API模拟服务 - 用于开发环境和测试
 */

import { vi } from 'vitest';

// ============================================
// Mock Data - 模拟数据
// ============================================

export const mockUsers = [
  { id: 1, username: 'admin', email: 'admin@example.com', role: 'admin' },
  { id: 2, username: 'user1', email: 'user1@example.com', role: 'user' },
];

export const mockTickets = [
  {
    id: 1,
    title: '无法登录系统',
    status: 'open',
    priority: 'high',
    category: 'Incident',
    requester: '张三',
    assignee: '李四',
    created_at: '2026-02-26T08:00:00Z',
    updated_at: '2026-02-26T10:00:00Z',
  },
  {
    id: 2,
    title: '请求新电脑',
    status: 'pending',
    priority: 'low',
    category: 'Service Request',
    requester: '王五',
    assignee: null,
    created_at: '2026-02-25T14:00:00Z',
    updated_at: '2026-02-25T14:00:00Z',
  },
];

export const mockIncidents = [
  {
    id: 1,
    title: '数据库连接失败',
    status: 'open',
    priority: 'critical',
    impact: 'high',
    category: '系统故障',
    assignee: '张三',
    created_at: '2026-02-26T08:00:00Z',
  },
];

export const mockProblems = [
  {
    id: 1,
    title: '数据库频繁死锁问题',
    status: 'investigation',
    priority: 'high',
    category: '数据库',
    assignee: '王五',
    created_at: '2026-02-20T10:00:00Z',
  },
];

export const mockChanges = [
  {
    id: 1,
    title: '数据库版本升级',
    status: 'pending_approval',
    change_type: 'standard',
    risk_level: 'medium',
    category: '数据库',
    assignee: '孙七',
    start_date: '2026-03-01',
    end_date: '2026-03-02',
    created_at: '2026-02-20T10:00:00Z',
  },
];

export const mockKnowledgeArticles = [
  {
    id: 1,
    title: '如何重置密码',
    content: '本文介绍如何重置IT系统密码...',
    category: '操作指南',
    tags: ['密码', '账户'],
    view_count: 150,
    like_count: 25,
    author: '管理员',
    created_at: '2026-01-15T10:00:00Z',
    is_published: true,
  },
];

export const mockSLA = [
  {
    id: 1,
    name: '紧急响应',
    response_time: 15,
    resolution_time: 60,
    priority: 'critical',
    status: 'active',
  },
];

// ============================================
// Mock API Handlers - 模拟API处理器
// ============================================

export const createMockHandlers = () => {
  const handlers: Record<string, Function> = {};

  // Tickets
  handlers['GET:/api/v1/tickets'] = () => ({
    code: 0,
    message: 'success',
    data: { tickets: mockTickets, total: mockTickets.length },
  });

  handlers['POST:/api/v1/tickets'] = (body: any) => ({
    code: 0,
    message: 'success',
    data: { id: 3, ...body },
  });

  handlers['GET:/api/v1/tickets/:id'] = () => ({
    code: 0,
    message: 'success',
    data: mockTickets[0],
  });

  handlers['PUT:/api/v1/tickets/:id'] = (body: any) => ({
    code: 0,
    message: 'success',
    data: { ...mockTickets[0], ...body },
  });

  handlers['DELETE:/api/v1/tickets/:id'] = () => ({
    code: 0,
    message: 'success',
  });

  // Incidents
  handlers['GET:/api/v1/incidents'] = () => ({
    code: 0,
    message: 'success',
    data: { incidents: mockIncidents, total: mockIncidents.length },
  });

  handlers['POST:/api/v1/incidents'] = (body: any) => ({
    code: 0,
    message: 'success',
    data: { id: 3, ...body },
  });

  // Problems
  handlers['GET:/api/v1/problems'] = () => ({
    code: 0,
    message: 'success',
    data: { problems: mockProblems, total: mockProblems.length },
  });

  handlers['POST:/api/v1/problems'] = (body: any) => ({
    code: 0,
    message: 'success',
    data: { id: 3, ...body },
  });

  // Changes
  handlers['GET:/api/v1/changes'] = () => ({
    code: 0,
    message: 'success',
    data: { changes: mockChanges, total: mockChanges.length },
  });

  handlers['POST:/api/v1/changes'] = (body: any) => ({
    code: 0,
    message: 'success',
    data: { id: 3, ...body },
  });

  // Knowledge Base
  handlers['GET:/api/v1/knowledge/articles'] = () => ({
    code: 0,
    message: 'success',
    data: { articles: mockKnowledgeArticles, total: mockKnowledgeArticles.length },
  });

  handlers['POST:/api/v1/knowledge/articles'] = (body: any) => ({
    code: 0,
    message: 'success',
    data: { id: 3, ...body },
  });

  // Auth
  handlers['POST:/api/v1/auth/login'] = (body: any) => ({
    code: 0,
    message: 'success',
    data: {
      token: 'mock-jwt-token',
      refresh_token: 'mock-refresh-token',
      user: mockUsers[0],
    },
  });

  handlers['POST:/api/v1/auth/refresh'] = () => ({
    code: 0,
    message: 'success',
    data: {
      token: 'new-mock-token',
      refresh_token: 'new-refresh-token',
    },
  });

  handlers['GET:/api/v1/auth/me'] = () => ({
    code: 0,
    message: 'success',
    data: mockUsers[0],
  });

  // Dashboard
  handlers['GET:/api/v1/dashboard'] = () => ({
    code: 0,
    message: 'success',
    data: {
      totalTickets: 150,
      openTickets: 25,
      resolvedToday: 12,
      avgResolutionTime: 4.5,
    },
  });

  // SLA
  handlers['GET:/api/v1/sla/definitions'] = () => ({
    code: 0,
    message: 'success',
    data: { sla: mockSLA, total: mockSLA.length },
  });

  return handlers;
};

// ============================================
// Mock Server - 模拟服务器
// ============================================

export class MockAPIServer {
  private handlers: Record<string, Function>;
  private delay: number;

  constructor(delay = 100) {
    this.handlers = createMockHandlers();
    this.delay = delay;
  }

  async handleRequest(method: string, url: string, body?: any): Promise<any> {
    // Add delay to simulate network
    await new Promise((resolve) => setTimeout(resolve, this.delay));

    // Find matching handler
    const key = `${method}:${url}`;
    let handler = this.handlers[key];

    // Try pattern matching
    if (!handler) {
      for (const handlerKey of Object.keys(this.handlers)) {
        const [handlerMethod, handlerPath] = handlerKey.split(':');
        if (handlerMethod === method) {
          const pattern = handlerPath.replace(/:(\w+)/g, '([^/]+)');
          const regex = new RegExp(`^${pattern}$`);
          if (regex.test(url)) {
            handler = this.handlers[handlerKey];
            break;
          }
        }
      }
    }

    if (handler) {
      return handler(body);
    }

    // Default 404 response
    return {
      code: 404,
      message: 'API endpoint not found',
    };
  }
}

// ============================================
// Mock Fetch - 模拟fetch
// ============================================

export const createMockFetch = (server?: MockAPIServer) => {
  const mockServer = server || new MockAPIServer();

  return vi.fn((url: string, options?: any) => {
    const method = options?.method || 'GET';
    const body = options?.body ? JSON.parse(options.body) : undefined;

    return mockServer.handleRequest(method, url, body);
  });
};

// ============================================
// Test Utilities - 测试工具
// ============================================

export const mockApiResponse = <T>(data: T, code = 0, message = 'success') => ({
  code,
  message,
  data,
});

export const createErrorResponse = (code: number, message: string) => ({
  code,
  message,
  data: null,
});

export const waitFor = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

export const retry = async <T>(
  fn: () => Promise<T>,
  options: { retries: number; delay: number } = { retries: 3, delay: 1000 }
): Promise<T> => {
  let lastError: Error | null = null;

  for (let i = 0; i < options.retries; i++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error as Error;
      if (i < options.retries - 1) {
        await waitFor(options.delay);
      }
    }
  }

  throw lastError;
};

// ============================================
// Export default instance
// ============================================

export default {
  mockUsers,
  mockTickets,
  mockIncidents,
  mockProblems,
  mockChanges,
  mockKnowledgeArticles,
  mockSLA,
  createMockHandlers,
  MockAPIServer,
  createMockFetch,
  mockApiResponse,
  createErrorResponse,
  waitFor,
  retry,
};
