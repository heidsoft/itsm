/**
 * WorkflowNodeInspector.test.tsx
 *
 * Covers the new "审批语义" (approval semantics) panel added in
 * WorkflowNodeInspector.tsx. The panel is rendered when a UserTask node is
 * selected on the BPMN canvas. Switching `taskPurpose` from "work" to
 * "approval" reveals the approval sub-panel with mode/reject/timeout/
 * delegate/addApprover/commentRequired controls. Changing `approvalMode`
 * to "threshold" reveals the `approvalThreshold` numeric input.
 *
 * Each control must call onUpdateProperties with the right moddle key so
 * the value persists into the BPMN XML.
 */
import React from 'react';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import type { BpmnNodeSelection } from '@/components/workflow/BPMNDesigner';

// Mock the network APIs before importing the component, otherwise the
// component will try to make real HTTP calls in useEffect().
jest.mock('@/lib/api/user-api', () => ({
  UserApi: {
    getUsers: jest.fn().mockResolvedValue({ users: [] }),
  },
}));

jest.mock('@/lib/api/group-api', () => ({
  GroupAPI: {
    getGroups: jest.fn().mockResolvedValue({ groups: [] }),
  },
}));

jest.mock('@/lib/api/role-api', () => ({
  RoleAPI: {
    getRoles: jest.fn().mockResolvedValue({ roles: [] }),
  },
}));

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    getTenantId: () => 1,
  },
}));

// Imports happen after mocks so the component picks them up.
import WorkflowNodeInspector from '@/components/workflow/designer/WorkflowNodeInspector';
import { UserApi } from '@/lib/api/user-api';

function buildUserTaskSelection(overrides?: Record<string, unknown>): BpmnNodeSelection {
  return {
    id: 'Task_Approve',
    type: 'bpmn:UserTask',
    name: 'Manager Approval',
    businessObject: {
      name: 'Manager Approval',
      taskPurpose: 'work',
      ...overrides,
    },
  };
}

function buildApprovalUserTaskSelection(
  overrides?: Record<string, unknown>
): BpmnNodeSelection {
  return buildUserTaskSelection({
    taskPurpose: 'approval',
    approvalMode: 'single',
    approvalThreshold: 1,
    rejectStrategy: 'terminate',
    timeoutAction: 'notify',
    allowDelegate: false,
    allowAddApprover: false,
    commentRequiredOnReject: true,
    ...overrides,
  });
}

describe('WorkflowNodeInspector — 审批语义 panel', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (UserApi.getUsers as jest.Mock).mockResolvedValue({ users: [] });
  });

  it('renders the 审批语义 panel for UserTask nodes with default taskPurpose=work', async () => {
    const onUpdateProperties = jest.fn().mockReturnValue(true);
    render(
      <WorkflowNodeInspector
        selection={buildUserTaskSelection()}
        onUpdateProperties={onUpdateProperties}
      />
    );

    // 审批语义 panel header is always visible for UserTask.
    expect(screen.getByText('审批语义')).toBeInTheDocument();

    // Default taskPurpose is "work" → 普通人工任务 is rendered as the select label.
    await waitFor(() => {
      expect(screen.getAllByText('普通人工任务').length).toBeGreaterThanOrEqual(1);
    });

    // When taskPurpose !== "approval" the approval sub-panel must be hidden.
    expect(screen.queryByText('单人审批')).not.toBeInTheDocument();
    expect(screen.queryByText('委托')).not.toBeInTheDocument();
    expect(screen.queryByText('拒绝意见必填')).not.toBeInTheDocument();
  });

  it('shows the approval sub-panel when taskPurpose is approval', async () => {
    const onUpdateProperties = jest.fn().mockReturnValue(true);
    render(
      <WorkflowNodeInspector
        selection={buildApprovalUserTaskSelection()}
        onUpdateProperties={onUpdateProperties}
      />
    );

    // All approval controls must be visible.
    // 三个 Select 默认值分别为 single/terminate/notify，因此断言的是
    // “当前选中项标签”，不是下拉展开后的选项。
    // 超时动作尚未就绪，标签带后缀说明，故用前缀匹配。
    await waitFor(() => {
      expect(screen.getByText('单人审批')).toBeInTheDocument();
      expect(screen.getByText(/^仅提醒/)).toBeInTheDocument();
      expect(screen.getByText('终止流程')).toBeInTheDocument();
      expect(screen.getByText('委托（未就绪）')).toBeInTheDocument();
      expect(screen.getByText('加签（未就绪）')).toBeInTheDocument();
      expect(screen.getByText('拒绝意见必填')).toBeInTheDocument();
    });
  });

  it('reveals the approvalThreshold input when approvalMode is threshold', async () => {
    const onUpdateProperties = jest.fn().mockReturnValue(true);
    render(
      <WorkflowNodeInspector
        selection={buildApprovalUserTaskSelection({ approvalMode: 'threshold', approvalThreshold: 3 })}
        onUpdateProperties={onUpdateProperties}
      />
    );

    // The 通过人数 label only appears in the threshold sub-panel.
    await waitFor(() => {
      expect(screen.getByText('通过人数')).toBeInTheDocument();
    });

    // Threshold value must be initialized from the moddle attribute.
    const thresholdInput = screen.getByDisplayValue('3') as HTMLInputElement;
    expect(thresholdInput).toBeInTheDocument();
  });

  it('hides the approvalThreshold input for non-threshold approval modes', async () => {
    const onUpdateProperties = jest.fn().mockReturnValue(true);
    render(
      <WorkflowNodeInspector
        selection={buildApprovalUserTaskSelection({ approvalMode: 'single' })}
        onUpdateProperties={onUpdateProperties}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('单人审批')).toBeInTheDocument();
    });
    // 通过人数 label is the threshold input's addonBefore — should not appear.
    expect(screen.queryByText('通过人数')).not.toBeInTheDocument();
  });

  it('writes the right moddle key for every approval control', async () => {
    const onUpdateProperties = jest.fn().mockReturnValue(true);
    render(
      <WorkflowNodeInspector
        selection={buildApprovalUserTaskSelection()}
        onUpdateProperties={onUpdateProperties}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('拒绝意见必填')).toBeInTheDocument();
    });

    // The 拒绝意见必填 switch is wired to commentRequiredOnReject.
    const switches = screen.getAllByRole('switch');
    const commentSwitch = switches[switches.length - 1]; // 拒绝意见必填 is the last switch
    expect(commentSwitch).toBeChecked();
    fireEvent.click(commentSwitch);
    expect(onUpdateProperties).toHaveBeenCalledWith('Task_Approve', { commentRequiredOnReject: false });

    // 未接线能力必须禁用，不能生成看似有效、运行时却被忽略的配置。
    onUpdateProperties.mockClear();
    const delegateSwitch = switches[0];
    expect(delegateSwitch).not.toBeChecked();
    expect(delegateSwitch).toBeDisabled();
    fireEvent.click(delegateSwitch);
    expect(onUpdateProperties).not.toHaveBeenCalled();

    const addApproverSwitch = switches[1];
    expect(addApproverSwitch).not.toBeChecked();
    expect(addApproverSwitch).toBeDisabled();
    fireEvent.click(addApproverSwitch);
    expect(onUpdateProperties).not.toHaveBeenCalled();
  });

  it('does not render the 审批语义 panel for non-UserTask nodes', () => {
    const onUpdateProperties = jest.fn().mockReturnValue(true);
    const selection: BpmnNodeSelection = {
      id: 'Gateway_1',
      type: 'bpmn:ExclusiveGateway',
      name: 'Decision',
      businessObject: { name: 'Decision' },
    };
    render(
      <WorkflowNodeInspector
        selection={selection}
        onUpdateProperties={onUpdateProperties}
      />
    );

    expect(screen.queryByText('审批语义')).not.toBeInTheDocument();
  });

  it('renders empty-state when no selection is provided', () => {
    const onUpdateProperties = jest.fn().mockReturnValue(true);
    render(<WorkflowNodeInspector selection={null} onUpdateProperties={onUpdateProperties} />);

    // Empty hint copy in Chinese.
    expect(screen.getByText(/点击画布上的节点查看\/编辑属性/)).toBeInTheDocument();
  });
});

describe('WorkflowNodeInspector — moddle 值回读', () => {
  const onUpdateProperties = jest.fn().mockReturnValue(true);

  beforeEach(() => {
    onUpdateProperties.mockClear();
  });

  it('把 bpmn:Documentation 列表显示为文本而不是 [object Object]', () => {
    render(
      <WorkflowNodeInspector
        selection={{
          id: 'UT1',
          type: 'bpmn:UserTask',
          businessObject: {
            name: '审批',
            documentation: [{ $type: 'bpmn:Documentation', text: '节点说明' }],
          },
        }}
        onUpdateProperties={onUpdateProperties}
      />
    );

    expect(screen.getByDisplayValue('节点说明')).toBeInTheDocument();
    expect(screen.queryByDisplayValue(/\[object/)).not.toBeInTheDocument();
  });

  it('空 documentation 列表显示为空', () => {
    render(
      <WorkflowNodeInspector
        selection={{
          id: 'UT2',
          type: 'bpmn:UserTask',
          businessObject: { name: 'x', documentation: [] },
        }}
        onUpdateProperties={onUpdateProperties}
      />
    );

    const textarea = screen.getByPlaceholderText('输入节点功能描述（可选）');
    expect(textarea).toHaveValue('');
  });

  it('网关默认分支显示连线 ID，而不是被解析后的引用对象', () => {
    render(
      <WorkflowNodeInspector
        selection={{
          id: 'GW1',
          type: 'bpmn:ExclusiveGateway',
          businessObject: {
            name: '判断',
            default: { $type: 'bpmn:SequenceFlow', id: 'Flow_ok' },
          },
        }}
        onUpdateProperties={onUpdateProperties}
      />
    );

    expect(screen.getByDisplayValue('Flow_ok')).toBeInTheDocument();
    expect(screen.queryByDisplayValue(/\[object Object\]/)).not.toBeInTheDocument();
  });

  it('ccNotify="false" 时发送通知开关是关闭的', () => {
    render(
      <WorkflowNodeInspector
        selection={{
          id: 'ST1',
          type: 'bpmn:ServiceTask',
          businessObject: {
            name: '抄送',
            implementation: 'cc_handler',
            ccType: 'role',
            ccRoleIds: '3,4',
            ccNotify: 'false',
            notifyChannels: 'in_app,email',
          },
        }}
        onUpdateProperties={onUpdateProperties}
      />
    );

    const switches = screen.getAllByRole('switch');
    expect(switches[0]).not.toBeChecked();
    // 抄送类型必须能回读，否则重开流程看起来像配置丢失
    expect(document.querySelector('.ant-select-content-has-value')?.textContent).toContain('角色');
  });

  it('operationRef 输入已移除，未接线的拒绝分支选项被禁用', async () => {
    render(
      <WorkflowNodeInspector
        selection={{
          id: 'UT3',
          type: 'bpmn:UserTask',
          businessObject: {
            name: '审批',
            taskPurpose: 'approval',
            approvalMode: 'single',
            rejectStrategy: 'terminate',
          },
        }}
        onUpdateProperties={onUpdateProperties}
      />
    );

    expect(screen.queryByText('操作引用 (operationRef)')).not.toBeInTheDocument();

    // 下拉顺序：taskPurpose / approvalMode / rejectStrategy / timeoutAction
    const comboboxes = screen.getAllByRole('combobox');
    fireEvent.mouseDown(comboboxes[2]);

    const option = await screen.findByText('进入拒绝分支（运行时未实现）');
    expect(option.closest('.ant-select-item')).toHaveClass('ant-select-item-option-disabled');
    expect(onUpdateProperties).not.toHaveBeenCalled();
  });
});
