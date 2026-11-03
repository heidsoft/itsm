import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * 服务请求列表页的创建入口。
 *
 * 此前列表非空时页面没有任何新建入口，用户只能从服务目录绕行；
 * 头部按钮必须指向真实存在的 `/service-requests/new`，且无写权限时不得出现。
 */

const mockPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush }),
}));

jest.mock('@/components/service-request/ServiceRequestList', () => ({
  __esModule: true,
  default: () => null,
}));

jest.mock('@/lib/api/service-request-api', () => ({
  serviceRequestAPI: {
    getPendingApprovals: jest.fn().mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 20,
      totalPages: 0,
    }),
    getUserServiceRequests: jest.fn().mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 100,
      totalPages: 0,
    }),
  },
}));

let mockPermissions: Record<string, boolean> = {};
jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => ({
    hasPermission: (resource: string, action: string) =>
      Boolean(mockPermissions[`${resource}:${action}`]),
  }),
}));

import ServiceRequestsPage from '../page';
import { serviceRequestAPI } from '@/lib/api/service-request-api';

async function renderPage() {
  render(
    <App>
      <ServiceRequestsPage />
    </App>
  );
  await waitFor(() => expect(serviceRequestAPI.getUserServiceRequests).toHaveBeenCalled());
}

describe('服务请求列表创建入口', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockPermissions = {};
  });

  it('有写权限时展示新建按钮并跳转到选服务页', async () => {
    mockPermissions = { 'service_request:write': true };
    await renderPage();

    const button = screen.getByRole('button', { name: /新建服务请求/ });
    await userEvent.click(button);

    expect(mockPush).toHaveBeenCalledWith('/service-requests/new');
  });

  it('无写权限时不展示新建按钮', async () => {
    await renderPage();

    expect(screen.queryByRole('button', { name: /新建服务请求/ })).not.toBeInTheDocument();
  });
});
