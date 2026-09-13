import React from 'react';
import { fireEvent, render, screen, waitFor } from '@/lib/test-utils';
import userEvent from '@testing-library/user-event';
import { TicketTypeFormModal } from '../TicketTypeFormModal';
import type { TicketTypeDefinition } from '@/types/ticket-type';
import { TicketTypeStatus } from '@/types/ticket-type';

const BOUND_WORKFLOW = { id: 7, key: 'incident_bug_flow', name: '故障审批流程' };

// 只解析 userTask，serviceTask 不得计入审批级次
const BPMN_XML = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:itsm="https://github.com/heidsoft/itsm/schema/bpmn">
  <bpmn:process id="incident_bug_flow" name="故障审批流程">
    <bpmn:userTask id="Activity_Level1" name="直属主管审批"
                   itsm:taskPurpose="approval" itsm:approvalMode="any"
                   itsm:candidateGroups="team_lead" />
    <bpmn:userTask id="Activity_Level2" name="技术总监审批"
                   itsm:taskPurpose="approval" itsm:approvalMode="all"
                   itsm:assignee="42" />
    <bpmn:serviceTask id="Activity_Notify" name="通知申请人" />
  </bpmn:process>
</bpmn:definitions>`;

jest.mock('@/lib/api/sla-api', () => ({
  SLAApi: { getSLADefinitions: jest.fn().mockResolvedValue({ items: [] }) },
}));

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowApi: {
    getProcessDefinition: jest.fn().mockResolvedValue({ bpmnXml: BPMN_XML }),
  },
}));

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: jest.fn().mockImplementation((url: string) => {
      if (url.includes('ticket-categories')) return Promise.resolve({ categories: [] });
      if (url.includes('process-definitions')) return Promise.resolve({ definitions: [BOUND_WORKFLOW] });
      if (url.includes('assignment-rules')) return Promise.resolve({ rules: [] });
      return Promise.resolve({});
    }),
  },
}));

const renderModal = (onSubmit: jest.Mock) =>
  render(
    <TicketTypeFormModal visible editingType={null} onCancel={jest.fn()} onSubmit={onSubmit} />,
  );

const renderModalWithBoundWorkflow = () => {
  const editingType: TicketTypeDefinition = {
    id: 1,
    code: 'incident_bug',
    name: '故障工单',
    status: TicketTypeStatus.ACTIVE,
    defaultPriority: 'high',
    sortOrder: 0,
    workflowDefinitionKey: BOUND_WORKFLOW.key,
    customFields: [],
    approvalEnabled: false,
    slaEnabled: false,
    autoAssignEnabled: false,
    createdBy: 1,
    createdByName: 'admin',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    tenantId: 1,
  };

  return render(
    <TicketTypeFormModal
      visible
      editingType={editingType}
      onCancel={jest.fn()}
      onSubmit={jest.fn().mockResolvedValue(undefined)}
    />,
  );
};

const fillBasicInfo = () => {
  fireEvent.change(screen.getByPlaceholderText('例如：incident_bug'), {
    target: { value: 'incident_bug' },
  });
  fireEvent.change(screen.getByPlaceholderText('例如：故障工单'), {
    target: { value: '故障工单' },
  });
};

describe('TicketTypeFormModal', () => {
  // Ant Design Tabs rendering is heavy in jsdom; individual tests routinely take 3-9s.
  jest.setTimeout(30000);

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

    expect(screen.getByText('审批级次在流程里配置')).toBeInTheDocument();
    expect(screen.getByText('绑定 BPMN 工作流')).toBeInTheDocument();
  });

  // 回归：审批链是运行时零消费的写入型配置（resolveApprovalWorkflow 只读 ProcessBinding
  // 与 legacy ApprovalWorkflow），强制用户填写等于"配了却不生效"。可编辑链必须彻底移除。
  it('no longer offers an editable approval chain', async () => {
    const user = userEvent.setup();
    renderModal(jest.fn().mockResolvedValue(undefined));

    await user.click(screen.getByRole('tab', { name: /审批设置/ }));

    expect(screen.queryByText('启用审批流程')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /新增审批级/ })).not.toBeInTheDocument();
    expect(screen.getByText('还没绑定工作流，这类工单不会进入审批')).toBeInTheDocument();
  });

  it('previews approval nodes of the bound workflow', async () => {
    const user = userEvent.setup();
    renderModalWithBoundWorkflow();

    await user.click(screen.getByRole('tab', { name: /审批设置/ }));

    expect(await screen.findByText('直属主管审批')).toBeInTheDocument();
    expect(screen.getByText('技术总监审批')).toBeInTheDocument();
    // serviceTask 不计入，所以是 2 而不是 3
    expect(screen.getByText('当前流程包含 2 个人工节点')).toBeInTheDocument();
    expect(screen.getByText('任一人通过即可')).toBeInTheDocument();
    expect(screen.getByText('会签，需要全部通过')).toBeInTheDocument();
    expect(screen.getByText('team_lead')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /打开流程设计器/ })).toBeInTheDocument();
  });

  it('surfaces a warning when the bound workflow cannot be loaded', async () => {
    const { WorkflowApi } = await import('@/lib/api/workflow-api');
    const spy = jest.mocked(WorkflowApi.getProcessDefinition);
    spy.mockRejectedValueOnce(new Error('network error'));

    renderModalWithBoundWorkflow();

    fireEvent.click(screen.getByRole('tab', { name: /审批设置/ }));

    expect(
      await screen.findByText('绑定流程的审批节点加载失败，请到流程设计器里确认'),
    ).toBeInTheDocument();
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
    expect(Array.isArray(payload.assignmentRules)).toBe(true);
    expect(payload).not.toHaveProperty('approvalChain');
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
