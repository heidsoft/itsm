/**
 * 服务请求详情（管理端 /service-requests/:id）审批入口的资格门控。
 *
 * 2026-11-04 prod 实测缺陷：组件原先只判断「当前级是否有 pending 行」，
 * 注释写明"依赖后端拦截"——但后端拦截返回的是 500「操作失败」（真实原因被吞），
 * 用户点下去只看到一个无信息量的报错。
 *
 * 现在以��端下发的 canApprove 为准：缺字段按无资格处理。
 */
import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import ServiceRequestDetail from '../ServiceRequestDetail';

const mockGetServiceRequest = jest.fn();
let mockId = '2';

jest.mock('next/navigation', () => ({
  useParams: () => ({ id: mockId }),
}));

jest.mock('@/lib/api/', () => ({
  ServiceRequestApi: {
    getServiceRequest: (...args: unknown[]) => mockGetServiceRequest(...args),
    applyApproval: jest.fn(),
    updateServiceRequestStatus: jest.fn(),
  },
}));

jest.mock('@/lib/api/service-request-api', () => ({
  serviceRequestAPI: {
    listProvisioningTasks: jest.fn().mockResolvedValue([]),
  },
}));

const renderDetail = () =>
  render(
    <App>
      <ServiceRequestDetail />
    </App>
  );

const pendingDetail = (canApprove?: boolean) => ({
  id: 2,
  status: 'submitted',
  title: '测试服务请求',
  reason: '申请原因',
  currentLevel: 1,
  totalLevels: 3,
  catalogId: 1,
  requesterId: 1,
  complianceAck: true,
  createdAt: '2026-11-04T10:00:00Z',
  approvals: [{ id: 1, level: 1, step: 'manager', status: 'pending' }],
  ...(canApprove === undefined ? {} : { canApprove }),
});

beforeEach(() => {
  mockId = '2';
  mockGetServiceRequest.mockReset();
});

describe('ServiceRequestDetail 审批入口门控', () => {
  it('后端未下发 canApprove 时不渲染审批按钮（不靠状态猜资格）', async () => {
    mockGetServiceRequest.mockResolvedValue(pendingDetail());

    renderDetail();

    await screen.findByText('测试服务请求');

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: "Approve" })).not.toBeInTheDocument();
    });
    expect(screen.queryByRole('button', { name: "Rejected" })).not.toBeInTheDocument();
  });

  it('canApprove=false 时不渲染审批按钮', async () => {
    mockGetServiceRequest.mockResolvedValue(pendingDetail(false));

    renderDetail();

    await screen.findByText('测试服务请求');

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: "Approve" })).not.toBeInTheDocument();
    });
  });

  it('canApprove=true 时渲染审批按钮', async () => {
    mockGetServiceRequest.mockResolvedValue(pendingDetail(true));

    renderDetail();

    const approveButton = await screen.findByRole('button', { name: "Approve" });
    expect(approveButton).toBeInTheDocument();
    expect(screen.getByRole('button', { name: "Rejected" })).toBeInTheDocument();
  });
});
