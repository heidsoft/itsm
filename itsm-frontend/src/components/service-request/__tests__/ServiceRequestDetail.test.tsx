/**
 * 服务请求详情遇到非数字 ID 不请求后端。
 *
 * 2026-10-31 生产实测：服务请求列表空态按钮曾跳 `/service-requests/new`，当时仓库
 * 没有这个页面，Next 把 `new` 当动态段，`Number("new")` 得到 NaN，随后发出
 * `GET /api/v1/service-requests/NaN`。创建页现已补上，但手输地址或旧书签仍可能
 * 落进 `[id]`，详情页必须自己兜住非数字 ID。
 */
import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import ServiceRequestDetail from '../ServiceRequestDetail';

const mockGetServiceRequest = jest.fn();
const mockUpdateServiceRequestStatus = jest.fn();
let mockId = 'new';

jest.mock('next/navigation', () => ({
  useParams: () => ({ id: mockId }),
}));

jest.mock('@/lib/api/', () => ({
  ServiceRequestApi: {
    getServiceRequest: (...args: unknown[]) => mockGetServiceRequest(...args),
    applyApproval: jest.fn(),
    updateServiceRequestStatus: (...args: unknown[]) => mockUpdateServiceRequestStatus(...args),
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

describe('ServiceRequestDetail', () => {
  beforeEach(() => {
    mockId = 'new';
    mockGetServiceRequest.mockReset();
    mockUpdateServiceRequestStatus.mockReset();
  });

  it('ID 不是正整数时直接显示未找到，不请求后端', async () => {
    renderDetail();

    expect(await screen.findByText("Request not found")).toBeInTheDocument();
    expect(mockGetServiceRequest).not.toHaveBeenCalled();
  });

  it('正整数 ID 仍按数字请求详情', async () => {
    mockId = '3';
    mockGetServiceRequest.mockResolvedValue({
      id: 3,
      title: '测试服务请求',
      catalog: { name: 'Customer service', category: '支持' },
      status: 'submitted',
      currentLevel: 1,
      totalLevels: 3,
      approvals: [],
      createdAt: '2026-10-31T10:00:00Z',
    });

    renderDetail();

    await waitFor(() => expect(mockGetServiceRequest).toHaveBeenCalledWith(3));
    expect(await screen.findByText('测试服务请求')).toBeInTheDocument();
    expect(screen.getByText('Account & Permissions')).toBeInTheDocument();
    expect(screen.queryByText('支持')).not.toBeInTheDocument();
  });

  it('待审批状态下可取消，交付中不可取消', async () => {
    mockId = '3';
    mockGetServiceRequest.mockResolvedValue({
      id: 3,
      title: '测试服务请求',
      status: 'submitted',
      currentLevel: 1,
      totalLevels: 3,
      approvals: [],
      createdAt: '2026-10-31T10:00:00Z',
    });
    mockUpdateServiceRequestStatus.mockResolvedValue({});

    renderDetail();

    fireEvent.click(await screen.findByRole('button', { name: /Cancel request/ }));
    fireEvent.change(screen.getByPlaceholderText("Reason for cancellation (optional)"), {
      target: { value: '需求取消' },
    });
    fireEvent.click(screen.getByRole('button', { name: "Confirm cancellation" }));

    await waitFor(() =>
      expect(mockUpdateServiceRequestStatus).toHaveBeenCalledWith(3, 'cancelled', '需求取消')
    );
  });

  it('交付中不显示取消入口', async () => {
    mockId = '3';
    mockGetServiceRequest.mockResolvedValue({
      id: 3,
      title: '测试服务请求',
      status: 'provisioning',
      currentLevel: 3,
      totalLevels: 3,
      approvals: [],
      createdAt: '2026-10-31T10:00:00Z',
    });

    renderDetail();

    expect(await screen.findByText('测试服务请求')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /Cancel request/ })).not.toBeInTheDocument();
  });
});
