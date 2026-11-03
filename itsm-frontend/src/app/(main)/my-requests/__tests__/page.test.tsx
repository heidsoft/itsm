import React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * 我的请求（/my-requests）回归（D17）：
 * 1. 关键字必须透传给后端（keyword 查询参数），页面不再对当前页做本地二次过滤；
 * 2. 状态筛选计数必须是服务端 total，而不是当前页条数；
 * 3. deps 变化引发的重取必须有 requestId 守卫，旧响应不得覆盖新结果。
 */

jest.mock('@/lib/api/service-catalog-api', () => {
  const getServiceRequests = jest.fn();
  return {
    ServiceCatalogApi: { getServiceRequests },
    __getServiceRequests: getServiceRequests,
  };
});

import MyRequestsPage from '../page';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';

const mockGetRequests = ServiceCatalogApi.getServiceRequests as jest.Mock;

const makeRequest = (id: number, name: string, status = 'submitted') => ({
  id,
  catalogId: id,
  requesterId: 1,
  status,
  reason: '申请原因',
  createdAt: '2026-10-01T08:00:00Z',
  catalog: { id, name, category: '计算资源', description: '描述' },
});

// 服务端各状态真实总数：全部 7、已提交 3、交付中 2、已交付 1、已拒绝 1
const COUNTS: Record<string, number> = {
  all: 7,
  submitted: 3,
  provisioning: 2,
  delivered: 1,
  rejected: 1,
};

type ListQuery = {
  page: number;
  pageSize: number;
  status?: string;
  keyword?: string;
};

const countsFor = (query: ListQuery) => {
  const items = query.status ? COUNTS[query.status] : COUNTS.all;
  return Promise.resolve({
    items: [],
    total: items ?? 0,
    page: 1,
    pageSize: 1,
    totalPages: 1,
  });
};

beforeEach(() => {
  jest.clearAllMocks();
});

const renderPage = () =>
  render(
    <App>
      <MyRequestsPage />
    </App>
  );

const buttonByPrefix = (prefix: string) =>
  screen.getAllByRole('button').find(button => (button.textContent || '').startsWith(prefix));

describe('我的请求服务端过滤与计数（D17）', () => {
  it('状态计数使用服务端 total，而不是当前页条数', async () => {
    mockGetRequests.mockImplementation((query: ListQuery) => {
      if (query.pageSize === 1) return countsFor(query);
      return Promise.resolve({
        items: [makeRequest(1, '云主机'), makeRequest(2, '数据库')],
        total: COUNTS.all,
        page: 1,
        pageSize: 10,
        totalPages: 1,
      });
    });

    renderPage();

    await waitFor(() => expect(screen.getByText('云主机')).toBeInTheDocument());

    // 当前页只有 2 条，但“全部”必须显示服务端 7 条
    expect(buttonByPrefix('全部')?.textContent).toContain('(7)');
    expect(buttonByPrefix('已提交')?.textContent).toContain('(3)');
    expect(buttonByPrefix('已交付')?.textContent).toContain('(1)');
  });

  it('搜索关键字经防抖后透传给后端，并用服务端返回结果渲染', async () => {
    mockGetRequests.mockImplementation((query: ListQuery) => {
      if (query.pageSize === 1) return countsFor(query);
      if (query.keyword) {
        return Promise.resolve({
          items: [makeRequest(2, '数据库')],
          total: 1,
          page: 1,
          pageSize: 10,
          totalPages: 1,
        });
      }
      return Promise.resolve({
        items: [makeRequest(1, '云主机'), makeRequest(2, '数据库')],
        total: 2,
        page: 1,
        pageSize: 10,
        totalPages: 1,
      });
    });

    renderPage();
    await waitFor(() => expect(screen.getByText('云主机')).toBeInTheDocument());

    fireEvent.change(screen.getByPlaceholderText('搜索服务名称或描述...'), {
      target: { value: '数据库' },
    });

    await waitFor(() => {
      const listCall = mockGetRequests.mock.calls
        .map(call => call[0] as ListQuery)
        .find(query => query.pageSize === 10 && query.keyword === '数据库');
      expect(listCall).toBeDefined();
    });

    await waitFor(() => {
      expect(screen.getByText('数据库')).toBeInTheDocument();
      expect(screen.queryByText('云主机')).not.toBeInTheDocument();
    });
  });

  it('旧请求晚返回时不得覆盖新筛选结果（requestId 竞态守卫）', async () => {
    const listResolvers: Array<{ query: ListQuery; resolve: (value: unknown) => void }> = [];

    mockGetRequests.mockImplementation((query: ListQuery) => {
      if (query.pageSize === 1) return countsFor(query);
      return new Promise(resolve => {
        listResolvers.push({ query, resolve });
      });
    });

    renderPage();

    // 挂起首个列表请求（status 未传 => 全部）
    await waitFor(() => expect(listResolvers).toHaveLength(1));
    expect(listResolvers[0].query.status).toBeUndefined();

    // 切换为“已交付”，触发第二个请求并先返回
    fireEvent.click(buttonByPrefix('已交付') as HTMLElement);
    await waitFor(() => expect(listResolvers).toHaveLength(2));

    await act(async () => {
      listResolvers[1].resolve({
        items: [makeRequest(99, '已交付请求', 'delivered')],
        total: 30,
        page: 1,
        pageSize: 10,
        totalPages: 3,
      });
    });
    expect(screen.getByText('已交付请求')).toBeInTheDocument();
    expect(document.body.textContent).toContain('共 30 条记录');

    // 旧的“全部”请求此时才返回，必须被守卫丢弃（含其 total/分页，不得回退页面数据范围）
    await act(async () => {
      listResolvers[0].resolve({
        items: [makeRequest(1, '过期全部请求', 'delivered')],
        total: 7,
        page: 1,
        pageSize: 10,
        totalPages: 1,
      });
    });
    expect(screen.queryByText('过期全部请求')).not.toBeInTheDocument();
    expect(screen.getByText('已交付请求')).toBeInTheDocument();
    expect(document.body.textContent).toContain('共 30 条记录');
  });
});
