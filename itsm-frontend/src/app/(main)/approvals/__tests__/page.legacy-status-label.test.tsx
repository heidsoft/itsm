import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * 审批中心「业务待办（只读参考）」Tab 的状态列中文映射。
 *
 * 2026-11-04 prod 实测缺陷：该列直接渲染后端原始状态串，页面上赫然显示
 * "pending" / "submitted"，而同页「流程待办」走的是中文映射——同页面两套标准。
 *
 * 修复要点：工单/变更/服务请求三套状态词表互不相同，必须按业务类型分派，
 * 否则会出现「服务请求的 submitted 被套用工单词表」这类错译。
 */

const mockListTasks = jest.fn();
const mockGet = jest.fn();

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), replace: jest.fn() }),
  usePathname: () => '/approvals',
  useSearchParams: () => new URLSearchParams(),
}));

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowApi: {
    listMyApprovalTasks: (...args: unknown[]) => mockListTasks(...args),
  },
}));

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: (...args: unknown[]) => mockGet(...args),
  },
}));

import ApprovalsPage from '../page';

const renderPage = () =>
  render(
    <App>
      <ApprovalsPage />
    </App>
  );

const switchToLegacyTab = async () => {
  const tab = await screen.findByRole('tab', { name: /业务待办/ });
  tab.click();
};

beforeEach(() => {
  jest.clearAllMocks();
  mockListTasks.mockResolvedValue({ items: [] });
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/api/v1/tickets')) {
      return Promise.resolve({
        items: [
          { id: 11, title: '工单甲', status: 'in_progress', priority: 'high', createdAt: '2026-11-01T00:00:00Z' },
        ],
      });
    }
    if (url.startsWith('/api/v1/changes')) {
      return Promise.resolve({
        items: [
          { id: 21, title: '变更乙', status: 'pending', priority: 'medium', createdAt: '2026-11-01T00:00:00Z' },
        ],
      });
    }
    if (url.startsWith('/api/v1/service-requests')) {
      return Promise.resolve({
        items: [
          { id: 31, title: '服务请求丙', status: 'submitted', priority: '', createdAt: '2026-11-01T00:00:00Z' },
        ],
      });
    }
    return Promise.resolve({ items: [] });
  });
});

describe('审批中心业务待办状态列', () => {
  it('三种业务类型的状态都渲染为中文，且不出现原始英文状态串', async () => {
    renderPage();
    await switchToLegacyTab();

    await waitFor(() => {
      expect(screen.getByText('处理中')).toBeInTheDocument();
    });

    // 工单 in_progress → 处理中
    expect(screen.getByText('处理中')).toBeInTheDocument();
    // 变更 pending → 待审批
    expect(screen.getByText('待审批')).toBeInTheDocument();
    // 服务请求 submitted → 已提交
    expect(screen.getByText('已提交')).toBeInTheDocument();

    // 原始英文状态串不得出现在状态列
    expect(screen.queryByText('in_progress')).not.toBeInTheDocument();
    expect(screen.queryByText('submitted')).not.toBeInTheDocument();
  });

  it('同一状态在不同业务类型下按各自词表翻译（service_request.submitted ≠ ticket.submitted）', async () => {
    mockGet.mockImplementation((url: string) => {
      if (url.startsWith('/api/v1/tickets')) {
        return Promise.resolve({
          items: [
            { id: 11, title: '工单甲', status: 'submitted', priority: '', createdAt: '2026-11-01T00:00:00Z' },
          ],
        });
      }
      if (url.startsWith('/api/v1/service-requests')) {
        return Promise.resolve({
          items: [
            { id: 31, title: '服务请求丙', status: 'submitted', priority: '', createdAt: '2026-11-01T00:00:00Z' },
          ],
        });
      }
      return Promise.resolve({ items: [] });
    });

    renderPage();
    await switchToLegacyTab();

    await waitFor(() => {
      expect(screen.getByText('已提交')).toBeInTheDocument();
    });
    // 工单词表里没有 submitted → 回退显示原值，这是可接受的（不误译优先于硬译）
    expect(screen.queryByText('待审批')).not.toBeInTheDocument();
  });
});
