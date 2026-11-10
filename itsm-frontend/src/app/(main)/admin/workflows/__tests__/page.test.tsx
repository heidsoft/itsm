import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { App } from 'antd';
import WorkflowManagementPage from '../page';
import { WorkflowAPI } from '@/lib/api/workflow-api';

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), replace: jest.fn(), prefetch: jest.fn() }),
}));

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowAPI: {
    getWorkflows: jest.fn(),
    getWorkflow: jest.fn(),
    activateWorkflow: jest.fn(),
    deactivateWorkflow: jest.fn(),
    createWorkflow: jest.fn(),
    updateWorkflow: jest.fn(),
    deleteWorkflow: jest.fn(),
  },
  WorkflowApi: undefined,
}));

// 预览子组件自带 bpmn-js，页面测试只关心它收到的是哪一份 XML
jest.mock('@/components/workflow/BPMNDiagramPreview', () => ({
  __esModule: true,
  default: ({ xml, label }: { xml: string; label?: string }) => (
    <div data-testid="bpmn-preview" data-xml={xml}>
      {label ?? ''}
    </div>
  ),
}));

jest.mock('@/components/workflow/WorkflowTemplateCatalog', () => ({
  __esModule: true,
  default: () => <div data-testid="template-catalog" />,
}));

jest.mock('@/components/common/UsageGuideCard', () => ({
  UsageGuideCard: () => <div data-testid="usage-guide" />,
}));

// 整页含统计卡片、多列表格与 Dropdown/Modal 传送门，并行 worker 抢 CPU 时单例实测 3–6s，
// 全局 10s testTimeout 下会假红（与 approvals 页同形态）。
jest.setTimeout(30000);

// WorkflowAPI.getWorkflows 已经把后端 DTO 映射成前端 WorkflowDefinition 视图：
// createdAt/updatedAt 是 Date，version 是后端 semver 字符串。测试直接返回该视图，
// 页面拿到的 shape 必须和真实调用一致。
const listWorkflow = (id: string) => ({
  id,
  code: id,
  name: `${id} 流程`,
  description: `描述 ${id}`,
  category: 'ticket',
  status: 'active',
  version: '1.0.0',
  createdByName: '系统',
  createdAt: new Date('2024-01-01T10:00:00Z'),
  updatedAt: new Date('2024-01-02T10:00:00Z'),
});

const deferred = <T,>() => {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
};

/** 点开列表行「更多操作」下拉里的「查看详情」；重复打开下拉时取最新一项 */
const clickDetailMenuItem = async (trigger: HTMLElement) => {
  fireEvent.click(trigger);
  const items = await screen.findAllByText("View Details");
  fireEvent.click(items[items.length - 1]);
};

const openDetail = async (rowIndex: number) => {
  const triggers = await screen.findAllByLabelText("More actions");
  await clickDetailMenuItem(triggers[rowIndex]);
  return screen.findByText("Process preview");
};

const renderPage = () =>
  render(
    <App>
      <WorkflowManagementPage />
    </App>
  );

describe('流程管理页详情预览', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (WorkflowAPI.getWorkflows as jest.Mock).mockResolvedValue({
      workflows: [listWorkflow('flow_a'), listWorkflow('flow_b')],
      total: 2,
    });
  });

  it('列表接口不返回 XML，打开详情时按流程标识取当前定义', async () => {
    (WorkflowAPI.getWorkflow as jest.Mock).mockResolvedValue({
      id: 'flow_a',
      bpmnXml: '<definitions>flow_a_current</definitions>',
    });

    renderPage();
    await openDetail(0);

    expect(WorkflowAPI.getWorkflow).toHaveBeenCalledWith('flow_a');
    await waitFor(() =>
      expect(screen.getByTestId('bpmn-preview')).toHaveAttribute(
        'data-xml',
        '<definitions>flow_a_current</definitions>'
      )
    );
  });

  it('加载中显示加载态，而不是空预览', async () => {
    const pending = deferred<{ id: string; bpmnXml: string }>();
    (WorkflowAPI.getWorkflow as jest.Mock).mockReturnValue(pending.promise);

    renderPage();
    await openDetail(0);

    expect(screen.getByText("Loading process definition...")).toBeInTheDocument();
    expect(screen.queryByTestId('bpmn-preview')).not.toBeInTheDocument();

    pending.resolve({ id: 'flow_a', bpmnXml: '<definitions>ok</definitions>' });
    await waitFor(() => expect(screen.getByTestId('bpmn-preview')).toBeInTheDocument());
    expect(screen.queryByText("Loading process definition...")).not.toBeInTheDocument();
  });

  it('快速连续打开两个详情时只接受最后一次请求的结果', async () => {
    const first = deferred<{ id: string; bpmnXml: string }>();
    const second = deferred<{ id: string; bpmnXml: string }>();
    (WorkflowAPI.getWorkflow as jest.Mock).mockImplementation((id: string) =>
      id === 'flow_a' ? first.promise : second.promise
    );

    renderPage();
    const triggers = await screen.findAllByLabelText("More actions");
    await clickDetailMenuItem(triggers[0]);
    await clickDetailMenuItem(triggers[1]);
    await screen.findByText("Process preview");

    second.resolve({ id: 'flow_b', bpmnXml: '<definitions>flow_b</definitions>' });
    await waitFor(() =>
      expect(screen.getByTestId('bpmn-preview')).toHaveAttribute(
        'data-xml',
        '<definitions>flow_b</definitions>'
      )
    );

    // 先发起的请求最后返回，不能把预览覆盖回 flow_a
    first.resolve({ id: 'flow_a', bpmnXml: '<definitions>flow_a</definitions>' });
    await new Promise(yieldToMicrotasks => setTimeout(yieldToMicrotasks, 0));
    expect(screen.getByTestId('bpmn-preview')).toHaveAttribute(
      'data-xml',
      '<definitions>flow_b</definitions>'
    );
  });

  it('流程定义拉取失败时如实提示，不伪装成空流程图', async () => {
    (WorkflowAPI.getWorkflow as jest.Mock).mockRejectedValue(new Error('5001 internal error'));

    renderPage();
    await openDetail(0);

    await waitFor(() =>
      expect(screen.getByText("Failed to load process definition and cannot preview")).toBeInTheDocument()
    );
    expect(screen.queryByTestId('bpmn-preview')).not.toBeInTheDocument();
  });
});
