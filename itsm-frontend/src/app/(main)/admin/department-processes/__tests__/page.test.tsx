import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { App } from 'antd';
import DepartmentProcessPage from '../page';
import { ProcessBindingApi } from '@/lib/api/process-binding-api';
import { WorkflowApi } from '@/lib/api/workflow-api';
import { departmentService } from '@/lib/services/department-service';

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), replace: jest.fn(), prefetch: jest.fn() }),
}));

jest.mock('@/lib/api/process-binding-api', () => ({
  ProcessBindingApi: {
    list: jest.fn(),
    listDepartmentProcesses: jest.fn(),
    create: jest.fn(),
    update: jest.fn(),
    delete: jest.fn(),
    initDepartmentProcesses: jest.fn(),
  },
}));

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowApi: {
    getWorkflows: jest.fn(),
  },
}));

jest.mock('@/lib/services/department-service', () => ({
  departmentService: {
    getDepartmentTree: jest.fn(),
  },
}));

const mockedBindingApi = ProcessBindingApi as jest.Mocked<typeof ProcessBindingApi>;
const mockedWorkflowApi = WorkflowApi as jest.Mocked<typeof WorkflowApi>;
const mockedDeptService = departmentService as jest.Mocked<typeof departmentService>;

const deptTree = [{
  id: 10,
  name: '运维部',
  code: 'OPS',
  description: '运维',
  parentId: 0,
  children: [],
}];

const binding = {
  id: 77,
  businessType: 'change',
  processDefinitionKey: 'change_standard_flow',
  priority: 100,
  isActive: true,
  departmentId: 10,
  scenario: 'change_release',
};

const renderPage = () =>
  render(
    <App>
      <DepartmentProcessPage />
    </App>,
  );

/** 选中左侧部门树节点，触发右侧列表加载 */
const selectDepartment = async () => {
  const treeNode = await screen.findByText('运维部 (OPS)');
  fireEvent.click(treeNode);
  await waitFor(() =>
    expect(mockedBindingApi.listDepartmentProcesses).toHaveBeenCalledWith(10),
  );
};

describe('部门流程配置页 CRUD', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedDeptService.getDepartmentTree.mockResolvedValue(deptTree as never);
    mockedBindingApi.listDepartmentProcesses.mockResolvedValue([binding as never]);
    mockedWorkflowApi.getWorkflows.mockResolvedValue({
      workflows: [{ code: 'change_standard_flow', name: '标准变更流程' }],
    } as never);
  });

  it('选中部门后展示绑定列表与操作列（编辑/删除/启停）', async () => {
    renderPage();
    await selectDepartment();

    // 流程定义列中文名优先、Key 兜底（英文工程标识用户看不懂）；Key 保留在 title 里可追溯
    expect(await screen.findByText('标准变更流程')).toBeInTheDocument();
    expect(screen.getByTitle('change_standard_flow')).toBeInTheDocument();
    expect(await screen.findByLabelText('编辑绑定')).toBeInTheDocument();
    expect(await screen.findByLabelText('删除绑定')).toBeInTheDocument();
    expect(await screen.findByLabelText('停用绑定')).toBeInTheDocument();
  });

  it('新增绑定预填并锁定当前选中部门，提交携带 departmentId', async () => {
    renderPage();
    await selectDepartment();

    fireEvent.click(await screen.findByText('新增绑定'));
    const modal = await screen.findByRole('dialog');
    // 部门字段锁定为当前选中部门（disabled input 的值经 value 属性暴露）
    expect(modal.querySelector('input[disabled]')).toHaveValue('运维部 (OPS)');

    // 通过 antd Form.Item 的 id 约定定位业务类型/流程定义 select（businessType / processDefinitionKey）
    const openSelectByFormId = async (formId: string) => {
      const selector = document.querySelector(`#basic .ant-select-selector, #${formId} .ant-select-selector, .ant-form-item #${formId}`);
      // Form.Item 无 name 时 antd id 是字段名本身；直接按 id 找 select 容器
      const node = document.querySelector(`[id="${formId}"]`);
      const target = node ?? selector;
      expect(target).not.toBeNull();
      fireEvent.mouseDown(target!);
    };
    // 无 name 的 Form.Item 包裹 Select：antd 会给 form id=字段名的元素
    const selectOption = async (formId: string, optionText: string) => {
      await openSelectByFormId(formId);
      const option = await screen.findByText(optionText, { selector: '.ant-select-item-option-content *, .ant-select-item-option-content' });
      fireEvent.click(option);
    };
    await selectOption('businessType', '变更');
    await selectOption('processDefinitionKey', '标准变更流程 (change_standard_flow)');

    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));
    await waitFor(() =>
      expect(mockedBindingApi.create).toHaveBeenCalledWith(
        expect.objectContaining({ departmentId: 10, businessType: 'change' }),
      ),
    );
  });

  it('编辑回填现有字段，更新按全量覆盖契约回传（不丢 departmentId/priority）', async () => {
    renderPage();
    await selectDepartment();

    fireEvent.click(await screen.findByLabelText('编辑绑定'));
    await screen.findByRole('dialog');

    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }));
    await waitFor(() =>
      expect(mockedBindingApi.update).toHaveBeenCalledWith(
        77,
        expect.objectContaining({
          departmentId: 10,
          priority: 100,
          isActive: true,
          businessType: 'change',
          processDefinitionKey: 'change_standard_flow',
        }),
      ),
    );
  });

  it('行内启停点击调用 update 并取反 isActive，同时回传全部字段防清零', async () => {
    renderPage();
    await selectDepartment();

    fireEvent.click(await screen.findByLabelText('停用绑定'));
    await waitFor(() =>
      expect(mockedBindingApi.update).toHaveBeenCalledWith(
        77,
        expect.objectContaining({
          isActive: false,
          departmentId: 10,
          priority: 100,
          scenario: 'change_release',
        }),
      ),
    );
  });

  /** Popconfirm 确认按钮藏在 .ant-popover 内（antd v6 okType=danger → .ant-btn-dangerous） */
  const confirmPopconfirm = async () => {
    await screen.findByText('确定删除这条流程绑定吗？');
    const popover = document.querySelector('.ant-popover');
    expect(popover).not.toBeNull();
    const okBtn = popover!.querySelector('button.ant-btn-dangerous');
    expect(okBtn).not.toBeNull();
    fireEvent.click(okBtn as HTMLElement);
  };

  it('删除需二次确认，确认后才调用删除接口', async () => {
    renderPage();
    await selectDepartment();

    fireEvent.click(await screen.findByLabelText('删除绑定'));
    await screen.findByText('确定删除这条流程绑定吗？');
    expect(mockedBindingApi.delete).not.toHaveBeenCalled();

    await confirmPopconfirm();
    await waitFor(() => expect(mockedBindingApi.delete).toHaveBeenCalledWith(77));
  });

  it('删除失败时显式报错，不伪装成功', async () => {
    renderPage();
    await selectDepartment();

    mockedBindingApi.delete.mockRejectedValue(new Error('该绑定正被流程实例引用'));
    fireEvent.click(await screen.findByLabelText('删除绑定'));
    await confirmPopconfirm();

    await waitFor(() =>
      expect(screen.getByText(/该绑定正被流程实例引用/)).toBeInTheDocument(),
    );
  });
});
