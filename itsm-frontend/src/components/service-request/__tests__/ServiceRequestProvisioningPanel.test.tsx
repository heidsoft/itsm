import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * 交付面板回归：动作按钮按 service_request:provision 展示，
 * 到达重试上限后只留人工判定，避免用户点到注定 409 的按钮。
 */

const mockList = jest.fn();
const mockStart = jest.fn();
const mockExecute = jest.fn();
const mockRetry = jest.fn();
const mockResolve = jest.fn();
let mockCanProvision = true;

jest.mock('@/lib/api/service-request-api', () => ({
  serviceRequestAPI: {
    listProvisioningTasks: (...args: unknown[]) => mockList(...args),
    startProvisioning: (...args: unknown[]) => mockStart(...args),
    executeProvisioningTask: (...args: unknown[]) => mockExecute(...args),
    retryProvisioningTask: (...args: unknown[]) => mockRetry(...args),
    resolveProvisioningTask: (...args: unknown[]) => mockResolve(...args),
  },
}));

jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => ({ hasPermission: () => mockCanProvision }),
}));

import ServiceRequestProvisioningPanel from '../ServiceRequestProvisioningPanel';

const task = (overrides: Record<string, unknown> = {}) => ({
  id: 11,
  serviceRequestId: 3,
  provider: 'aliyun',
  resourceType: 'ecs',
  status: 'failed',
  attemptCount: 5,
  maxAttempts: 5,
  errorMessage: 'quota exceeded',
  createdAt: '2026-10-31T10:00:00Z',
  updatedAt: '2026-10-31T10:00:00Z',
  ...overrides,
});

const renderPanel = (status: string, onChanged = jest.fn()) =>
  render(
    <App>
      <ServiceRequestProvisioningPanel requestId={3} status={status} onChanged={onChanged} />
    </App>
  );

beforeEach(() => {
  jest.clearAllMocks();
  mockCanProvision = true;
  mockList.mockResolvedValue([]);
});

describe('ServiceRequestProvisioningPanel', () => {
  it('安全审批通过后按权限显示启动交付，并回调刷新', async () => {
    mockStart.mockResolvedValue({ task: task({ status: 'pending' }) });
    const onChanged = jest.fn();

    renderPanel('security_approved', onChanged);

    fireEvent.click(await screen.findByRole('button', { name: /启动交付/ }));

    await waitFor(() => expect(mockStart).toHaveBeenCalledWith(3));
    await waitFor(() => expect(onChanged).toHaveBeenCalled());
  });

  it('没有交付权限时不显示启动交付', async () => {
    mockCanProvision = false;

    renderPanel('security_approved');

    expect(await screen.findByText('审批已全部通过')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /启动交付/ })).not.toBeInTheDocument();
  });

  it('到达重试上限后重试按钮消失，人工判定写入判定依据', async () => {
    mockList.mockResolvedValue([task()]);

    renderPanel('failed');

    expect(await screen.findByText('quota exceeded')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '重试' })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '人工判定' }));
    fireEvent.change(
      screen.getByPlaceholderText(
        '例如：资源已在控制台手工创建 i-abc123，凭据吊销无法走自动交付'
      ),
      { target: { value: '资源已在控制台手工创建' } }
    );
    fireEvent.click(screen.getByRole('button', { name: '提交判定' }));

    await waitFor(() =>
      expect(mockResolve).toHaveBeenCalledWith(11, 'delivered', '资源已在控制台手工创建')
    );
  });

  it('没有交付权限时不提供执行与判定入口', async () => {
    mockCanProvision = false;
    mockList.mockResolvedValue([task()]);

    renderPanel('failed');

    expect(await screen.findByText('quota exceeded')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '重试' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '人工判定' })).not.toBeInTheDocument();
  });

  it('非交付阶段不加载任务', () => {
    renderPanel('submitted');

    expect(mockList).not.toHaveBeenCalled();
  });
});
