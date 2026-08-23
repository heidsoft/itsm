import React from 'react';
import { fireEvent, render, screen, waitFor } from '@/lib/test-utils';
import userEvent from '@testing-library/user-event';
import { TicketTypeFormModal } from '../TicketTypeFormModal';

jest.mock('@/lib/api/sla-api', () => ({
  SLAApi: { getSLADefinitions: jest.fn().mockResolvedValue({ items: [] }) },
}));

jest.mock('@/lib/api/user-api', () => ({
  UserApi: { getUsers: jest.fn().mockResolvedValue({ users: [] }) },
}));

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn().mockImplementation((url: string) => {
      if (url.includes('ticket-categories')) return Promise.resolve({ categories: [] });
      if (url.includes('process-definitions')) return Promise.resolve({ definitions: [] });
      if (url.includes('assignment-rules')) return Promise.resolve({ rules: [] });
      return Promise.resolve({});
    }),
  },
}));

const renderModal = (onSubmit: jest.Mock) =>
  render(
    <TicketTypeFormModal visible editingType={null} onCancel={jest.fn()} onSubmit={onSubmit} />,
  );

const fillBasicInfo = () => {
  fireEvent.change(screen.getByPlaceholderText('例如：incident_bug'), {
    target: { value: 'incident_bug' },
  });
  fireEvent.change(screen.getByPlaceholderText('例如：故障工单'), {
    target: { value: '故障工单' },
  });
};

describe('TicketTypeFormModal', () => {
  it('renders all configuration tabs in create mode', () => {
    renderModal(jest.fn().mockResolvedValue(undefined));

    expect(screen.getByRole('tab', { name: /基本信息/ })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /自定义字段/ })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /审批设置/ })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /SLA 设置/ })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /自动分派/ })).toBeInTheDocument();
  });

  // 核心回归：审批设置页签包含 Form.Item（原 bug 是 Form 只包裹了基本信息，
  // 其余页签的 Form.Item 游离在外导致开关状态丢失）
  it('renders approval configuration form items inside the approval tab', async () => {
    const user = userEvent.setup();
    renderModal(jest.fn().mockResolvedValue(undefined));

    await user.click(screen.getByRole('tab', { name: /审批设置/ }));

    expect(screen.getByText('启用审批流程')).toBeInTheDocument();
    expect(screen.getByText('绑定 BPMN 工作流')).toBeInTheDocument();
  });

  it('renders SLA and auto-assignment config inside their tabs', async () => {
    const user = userEvent.setup();
    renderModal(jest.fn().mockResolvedValue(undefined));

    await user.click(screen.getByRole('tab', { name: /SLA 设置/ }));
    expect(screen.getByText('启用 SLA')).toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: /自动分派/ }));
    expect(screen.getByText('启用自动分派')).toBeInTheDocument();
  });

  it('collects form values on submit', async () => {
    const onSubmit = jest.fn().mockResolvedValue(undefined);
    renderModal(onSubmit);

    fillBasicInfo();
    fireEvent.click(screen.getByRole('button', { name: /创\s*建/ }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    const payload = onSubmit.mock.calls[0][0] as Record<string, unknown>;

    expect(payload.code).toBe('incident_bug');
    expect(payload.name).toBe('故障工单');
    expect(Array.isArray(payload.customFields)).toBe(true);
    expect(Array.isArray(payload.approvalChain)).toBe(true);
    expect(Array.isArray(payload.assignmentRules)).toBe(true);
  });

  it('blocks submit when a custom field is missing required values', async () => {
    const onSubmit = jest.fn().mockResolvedValue(undefined);
    renderModal(onSubmit);

    fillBasicInfo();

    fireEvent.click(screen.getByRole('tab', { name: /自定义字段/ }));
    fireEvent.click(screen.getByRole('button', { name: /新增字段/ }));

    fireEvent.click(screen.getByRole('button', { name: /创\s*建/ }));

    expect(await screen.findByText(/缺少显示名称/)).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('keeps the modal open when submit fails so users can retry', async () => {
    const onSubmit = jest.fn().mockRejectedValue(new Error('network error'));
    renderModal(onSubmit);

    fillBasicInfo();
    fireEvent.click(screen.getByRole('button', { name: /创\s*建/ }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(screen.getByRole('button', { name: /创\s*建/ })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /创\s*建/ }));
    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(2));
  });
});
