/**
 * CIDetail 组件测试
 * 配置项详情页的主容器组件
 */

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CIDetail } from '../CIDetail';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

jest.mock('next/navigation', () => ({
  useParams: () => ({ id: 'ci-001' }),
  useRouter: () => ({
    push: jest.fn(),
    back: jest.fn(),
  }),
}));

jest.mock('../hooks/useCIDetail', () => ({
  useCIDetail: () => ({
    ci: {
      id: 'ci-001',
      name: 'Web Server 1',
      type: 'server',
      status: 'active',
      description: 'Production web server',
      ciTypeId: 'server',
    },
    loading: false,
    error: null,
    impactAnalysis: null,
    impactLoading: false,
    changeHistory: null,
    historyLoading: false,
    typeInfo: undefined,
    loadDetail: jest.fn(),
    loadImpactAnalysis: jest.fn(),
    loadChangeHistory: jest.fn(),
  }),
}));

describe('CIDetail', () => {
  it('应该正确渲染 CI 名称', async () => {
    render(<CIDetail />);

    await waitFor(() => {
      expect(screen.getByText('Web Server 1')).toBeInTheDocument();
    });
  });

  it('应该显示 CI 状态', async () => {
    render(<CIDetail />);

    await waitFor(() => {
      // 状态显示为中文标签
      expect(screen.getByText(/使用中/i)).toBeInTheDocument();
    });
  });

  it('应该包含标签页导航', async () => {
    render(<CIDetail />);

    await waitFor(() => {
      expect(screen.getByText(/基本信息/i)).toBeInTheDocument();
      expect(screen.getByText(/服务拓扑/i)).toBeInTheDocument();
      expect(screen.getByText(/关系管理/i)).toBeInTheDocument();
      expect(screen.getByText(/影响分析/i)).toBeInTheDocument();
      expect(screen.getByText(/变更历史/i)).toBeInTheDocument();
    });
  });

  it('应该显示返回按钮', async () => {
    render(<CIDetail />);

    await waitFor(() => {
      expect(screen.getByText(/返回列表/i)).toBeInTheDocument();
    });
  });
});
