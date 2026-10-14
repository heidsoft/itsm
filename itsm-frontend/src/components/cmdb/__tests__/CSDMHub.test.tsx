/**
 * CSDMHub 工作台回归测试
 *
 * 锁定两件事：
 * 1. stats 契约字段：后端 GET /cmdb/cis/stats 返回 data.total（dto.CIStatsResponse），
 *    曾因读取 totalCount/totalCIs 兜底键导致「配置项总数」恒为 0。
 * 2. 管理面入口按权限展示：无 cmdb:write 时不得渲染跳 /admin/cmdb-types 的入口，
 *    否则 cmdb:read 用户（如 it_admin）会被导航进 403 页。
 */

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';

import { CSDMHub } from '../CSDMHub';
import { CMDBApi } from '@/lib/api/cmdb-api';
import { useAuthStore } from '@/lib/store/auth-store';

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('@/lib/i18n/useI18n', () => {
  // t 身份必须跨渲染稳定，否则 CSDMHub 的 useCallback(load) 每次渲染都会变化，
  // useEffect 无限重跑（真实 provider 的 t 是稳定的）。
  const t = (key: string) => key;
  return { useI18n: () => ({ t }) };
});

jest.mock('@/lib/hooks/useCapabilities', () => ({
  useCapabilities: () => ({ capabilities: [], allows: () => false }),
}));

jest.mock('@/lib/api/cmdb-api', () => ({
  CMDBApi: {
    getCMDBStats: jest.fn(),
    getCITypes: jest.fn(),
  },
}));

const mockGetCMDBStats = CMDBApi.getCMDBStats as jest.Mock;
const mockGetCITypes = CMDBApi.getCITypes as jest.Mock;

const mockStats = () => {
  mockGetCMDBStats.mockResolvedValue({
    total: 14,
    statusDistribution: { online: 10 },
    typeDistribution: { server: 5 },
    environmentDistribution: { prod: 8 },
    criticalityDistribution: { high: 2 },
  });
  mockGetCITypes.mockResolvedValue([{ id: 1 }, { id: 2 }, { id: 3 }]);
};

describe('CSDMHub', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    useAuthStore.setState({ user: null });
  });

  it('从 stats.total 读取配置项总数，而不是兜底键', async () => {
    mockStats();

    render(
      <App>
        <CSDMHub />
      </App>
    );

    await waitFor(() => {
      expect(screen.getByText('14')).toBeInTheDocument();
    });
    expect(screen.getByText('3')).toBeInTheDocument();
    expect(mockGetCMDBStats).toHaveBeenCalledTimes(1);
  });

  it('无 cmdb:write 时不渲染 CI 类型维护入口', async () => {
    mockStats();

    render(
      <App>
        <CSDMHub />
      </App>
    );

    await waitFor(() => {
      expect(screen.getByText('14')).toBeInTheDocument();
    });
    expect(screen.queryByText('cmdb.hub.maintainTypeTemplates')).not.toBeInTheDocument();
  });

  it('持有 cmdb:write 时渲染 CI 类型维护入口', async () => {
    mockStats();
    useAuthStore.setState({
      user: {
        id: 1,
        username: 'admin1',
        email: 'admin1@example.com',
        name: 'Admin',
        role: 'admin',
        permissions: ['cmdb:write'],
      },
    });

    render(
      <App>
        <CSDMHub />
      </App>
    );

    await waitFor(() => {
      expect(screen.getByText('14')).toBeInTheDocument();
    });
    expect(screen.getByText('cmdb.hub.maintainTypeTemplates')).toBeInTheDocument();
  });
});
