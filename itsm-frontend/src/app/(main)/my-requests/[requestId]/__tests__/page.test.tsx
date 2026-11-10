import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * 服务请求详情（/my-requests/:id）取消动作回归。
 *
 * 后端状态机只允许 submitted / manager_approved / it_approved /
 * security_approved / failed 直接转 cancelled；交付中的请求取消会被 409。
 * 页面必须按同一口径决定是否显示取消入口，避免用户点了才收到冲突。
 */

const mockGetServiceRequest = jest.fn();
const mockCancelServiceRequest = jest.fn();

jest.mock('next/navigation', () => ({
  useParams: () => ({ requestId: '3' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('@/lib/api/service-catalog-api', () => ({
  ServiceCatalogApi: {
    getServiceRequest: (...args: unknown[]) => mockGetServiceRequest(...args),
    cancelServiceRequest: (...args: unknown[]) => mockCancelServiceRequest(...args),
  },
}));

jest.mock('@/lib/api/service-request-api', () => ({
  serviceRequestAPI: {
    listProvisioningTasks: jest.fn().mockResolvedValue([]),
    startProvisioning: jest.fn(),
    executeProvisioningTask: jest.fn(),
    retryProvisioningTask: jest.fn(),
    resolveProvisioningTask: jest.fn(),
  },
}));

import MyRequestDetailPage from '../page';

const detailWith = (status: string) => ({
  id: 3,
  status,
  title: '测试服务请求',
  reason: '申请原因',
  currentLevel: 1,
  totalLevels: 3,
  approvals: [],
  createdAt: '2026-10-31T10:00:00Z',
});

const renderPage = () =>
  render(
    <App>
      <MyRequestDetailPage />
    </App>
  );

beforeEach(() => {
  jest.clearAllMocks();
});

describe('服务请求详情取消动作', () => {
  it('待审批状态下取消，提交原因并刷新详情', async () => {
    mockGetServiceRequest.mockResolvedValue(detailWith('submitted'));
    mockCancelServiceRequest.mockResolvedValue(undefined);

    renderPage();

    const cancelButton = await screen.findByRole('button', { name: /Cancel request/ });
    fireEvent.click(cancelButton);

    expect(mockCancelServiceRequest).not.toHaveBeenCalled();

    const reasonInput = await screen.findByPlaceholderText("For example: requirements change, apply for larger specifications instead");
    fireEvent.change(reasonInput, { target: { value: '需求变更' } });
    fireEvent.click(screen.getByRole('button', { name: "Confirm cancellation" }));

    await waitFor(() => expect(mockCancelServiceRequest).toHaveBeenCalledWith(3, '需求变更'));
    await waitFor(() => expect(mockGetServiceRequest).toHaveBeenCalledTimes(2));
  });

  it.each(['provisioning', 'delivered', 'rejected', 'cancelled'])(
    '%s 状态不显示取消入口',
    async status => {
      mockGetServiceRequest.mockResolvedValue(detailWith(status));

      renderPage();

      expect(await screen.findByText('测试服务请求')).toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /Cancel request/ })).not.toBeInTheDocument();
    }
  );
});
