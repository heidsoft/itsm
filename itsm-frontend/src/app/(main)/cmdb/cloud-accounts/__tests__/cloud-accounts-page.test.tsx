/**
 * D53 回归：云账号弹窗文案
 * - 创建态：标签应为「凭据引用」，说明为“填写 secret://<tenant>/<name> 引用；留空表示暂不配置凭据”；
 *   修复前复用了编辑态文案「更新凭据引用 / 留空将保留现有凭据」，新增场景并不存在“现有凭据”。
 * - 编辑态：保留「更新凭据引用 / 留空将保留现有凭据」。
 */

import React from 'react';
import { App } from 'antd';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, fireEvent } from '@testing-library/react';
import CloudAccountPage from '../page';

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), replace: jest.fn(), prefetch: jest.fn() }),
}));

interface MockQueryState {
  data?: {
    items: Array<Record<string, unknown>>;
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
  };
  isPending: boolean;
  isError: boolean;
  error: null;
}

const successState: MockQueryState = {
  data: {
    items: [
      {
        id: 1,
        provider: 'aliyun',
        accountId: '1234567890123456',
        accountName: '生产账号',
        hasCredential: true,
        isActive: true,
        tenantId: 1,
      },
    ],
    total: 1,
    page: 1,
    pageSize: 20,
    totalPages: 1,
  },
  isPending: false,
  isError: false,
  error: null,
};

let mockQueryState: MockQueryState = successState;

jest.mock('@/lib/hooks/useCMDB', () => ({
  CMDB_KEYS: { all: ['cmdb'] },
  useCloudAccountsQuery: () => mockQueryState,
}));

jest.mock('@/lib/api/cmdb-api', () => ({
  CMDBApi: {
    createCloudAccount: jest.fn(),
    updateCloudAccount: jest.fn(),
    deleteCloudAccount: jest.fn(),
  },
}));

jest.mock('@/lib/api/http-client', () => ({
  serverErrorMessage: (_error: unknown, fallback: string) => fallback,
}));

const renderPage = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <App>
        <CloudAccountPage />
      </App>
    </QueryClientProvider>
  );
};

describe('云账号弹窗文案（D53 回归）', () => {
  beforeEach(() => {
    mockQueryState = successState;
  });

  it(
    '创建态用「凭据引用」，编辑态用「更新凭据引用」',
    async () => {
      renderPage();

      // 打开「新增云账号」
      fireEvent.click(screen.getByRole('button', { name: /新增云账号/ }));

      // 创建态：标签「凭据引用」+ 创建语义说明；不得出现编辑态标签
      await screen.findByText('凭据引用');
      expect(screen.queryByText('更新凭据引用')).toBeNull();
      await screen.findByText('填写 secret://<tenant>/<name> 引用；留空表示暂不配置凭据。');

      // 关闭创建弹窗（测试环境无 zh-CN locale，Modal 按钮为英文）
      fireEvent.click(screen.getByRole('button', { name: /cancel/i }));

      // 打开「编辑云账号」
      fireEvent.click(screen.getByLabelText('编辑云账号 生产账号'));

      // 编辑态：标签「更新凭据引用」+ 保留现有凭据说明
      await screen.findByText('更新凭据引用');
      await screen.findByText('留空将保留现有凭据；服务端不会返回已保存的凭据内容。');
    },
    // 整页 + 两个 antd Modal 的渲染与多次全文档 findBy 轮询较重，本用例为长耗时测试。
    60000
  );
});

describe('首屏加载态（D55 回归）', () => {
  it('拉取未完成时显示加载态，不得渲染「暂无数据 / 共 0 个账号」空态', () => {
    // 模拟 SSR/首帧：query 已发起但尚未返回（isPending=true，无 data）。
    // 修复前用 isLoading 判定，首帧 isFetching=false 被判为非加载中，
    // 直接渲染空表与「共 0 个账号」。
    mockQueryState = { data: undefined, isPending: true, isError: false, error: null };
    const { container } = renderPage();

    // 表格必须呈现 antd loading spin
    expect(container.querySelector('.ant-spin')).not.toBeNull();
    // 统计区显示「加载中…」，不得显示「共 0 个账号」
    expect(screen.getByText('加载中…')).toBeInTheDocument();
    expect(screen.queryByText(/共 \d+ 个账号/)).toBeNull();
    // 空态文案不得出现
    expect(screen.queryByText('暂无数据')).toBeNull();
  });
});
