import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * 服务请求详情（/my-requests/:id）审批入口的资格门控。
 *
 * 2026-11-04 prod 实测缺陷：审批按钮此前只按请求状态渲染，任何能打开详情页的人
 * （包括发起人本人、不在本级审批人集合内的管理员）都会看到「通过/拒绝」，
 * 点下去必然失败——发起人自审命中「审批人集合只包含发起人本人」409，
 * 非审批人命中引擎归属校验，此前被 fmt.Errorf 包装吞成 500「操作失败」。
 *
 * 现在资格由后端下发（canApprove），前端据此决定是否渲染审批入口。
 */

const mockGetServiceRequest = jest.fn();

jest.mock('next/navigation', () => ({
  useParams: () => ({ requestId: '3' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('@/lib/api/service-catalog-api', () => ({
  ServiceCatalogApi: {
    getServiceRequest: (...args: unknown[]) => mockGetServiceRequest(...args),
    cancelServiceRequest: jest.fn(),
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

const detailWith = (status: string, canApprove?: boolean) => ({
  id: 3,
  status,
  title: '测试服务请求',
  reason: '申请原因',
  currentLevel: 1,
  totalLevels: 3,
  approvals: [],
  createdAt: '2026-10-31T10:00:00Z',
  ...(canApprove === undefined ? {} : { canApprove }),
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

describe('服务请求详情审批入口门控', () => {
  it('当前用户无审批资格时，不渲染「通过/拒绝」入口', async () => {
    mockGetServiceRequest.mockResolvedValue(detailWith('submitted', false));

    renderPage();

    await screen.findByText('测试服务请求');

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /^Approve$/ })).not.toBeInTheDocument();
    });
    expect(screen.queryByRole('button', { name: /^Rejected$/ })).not.toBeInTheDocument();
    expect(screen.queryByText("Approval operation")).not.toBeInTheDocument();
  });

  it('当前用户具备审批资格时，渲染「通过/拒绝」入口', async () => {
    mockGetServiceRequest.mockResolvedValue(detailWith('submitted', true));

    renderPage();

    const approveButton = await screen.findByRole('button', { name: /^Approve$/ });
    expect(approveButton).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^Rejected$/ })).toBeInTheDocument();
    expect(screen.getByText("Approval operation")).toBeInTheDocument();
  });

  it('后端未下发 canApprove 字段时（老响应），不渲染审批入口', async () => {
    // 缺字段必须按「无资格」处理：宁可少给入口，也不能给必然失败的入口。
    mockGetServiceRequest.mockResolvedValue(detailWith('submitted'));

    renderPage();

    await screen.findByText('测试服务请求');

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /^Approve$/ })).not.toBeInTheDocument();
    });
  });

  it('已离开审批阶段时，即便有资格也不渲染审批入口', async () => {
    mockGetServiceRequest.mockResolvedValue(detailWith('delivered', true));

    renderPage();

    await screen.findByText('测试服务请求');

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /^Approve$/ })).not.toBeInTheDocument();
    });
  });
});
