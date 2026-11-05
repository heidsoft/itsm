import React from 'react';
import { render, screen, fireEvent, within, waitFor } from '@testing-library/react';
import { App } from 'antd';
import WorkflowVersionsPage from '../page';
import { WorkflowAPI } from '@/lib/api/workflow-api';
import type { BpmnProcessVersion, BpmnVersionComparison } from '@/lib/api/workflow-api';

jest.mock('next/navigation', () => ({
  useSearchParams: () => new URLSearchParams('workflowId=incident_flow'),
}));

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowAPI: {
    getWorkflows: jest.fn(),
    listVersions: jest.fn(),
    compareVersions: jest.fn(),
    activateVersion: jest.fn(),
    rollbackVersion: jest.fn(),
    deleteVersion: jest.fn(),
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

// jest.setup.js 全局 mock 了 dayjs，时间列断言改为可预测的格式化结果，
// 只验证后端时间戳确实落到对应列。
jest.mock('@/lib/formatters', () => {
  const actual = jest.requireActual('@/lib/formatters');
  return {
    ...actual,
    formatDateTime: (value?: string | Date | null) => (value ? `FMT(${String(value)})` : ''),
  };
});

const version = (overrides: Partial<BpmnProcessVersion> = {}): BpmnProcessVersion => ({
  id: '42',
  processDefinitionKey: 'incident_flow',
  version: 2,
  name: '事件流程',
  description: '第二轮审批',
  bpmnXml: '<definitions>v2</definitions>',
  deploymentId: 'dep-2',
  isActive: true,
  createdAt: '2024-01-01T10:00:00Z',
  updatedAt: '2024-01-02T10:00:00Z',
  createdBy: '1',
  tenantId: 7,
  changeLog: '调整审批人',
  compatibilityNotes: '',
  ...overrides,
});

const comparisonOf = (
  overrides: Partial<BpmnVersionComparison> = {}
): BpmnVersionComparison => ({
  baseVersion: version({ version: 1, isActive: false }),
  targetVersion: version({ version: 2 }),
  changes: [
    {
      type: 'modified',
      changeType: 'modified',
      elementType: 'bpmn:UserTask',
      elementId: 'Task_1',
      elementName: '审批',
      description: 'assignee changed',
      impact: 'high',
    },
  ],
  breakingChanges: ['assignee changed'],
  compatibility: 'risky',
  ...overrides,
});

const renderPage = () =>
  render(
    <App>
      <WorkflowVersionsPage />
    </App>
  );

describe('工作流版本页', () => {
  const duplicateKeyErrors: string[] = [];
  let consoleErrorSpy: jest.SpyInstance;

  beforeEach(() => {
    jest.clearAllMocks();
    duplicateKeyErrors.length = 0;
    consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation((...args: unknown[]) => {
      const text = args.map(String).join(' ');
      if (text.includes('same key')) duplicateKeyErrors.push(text);
    });
    (WorkflowAPI.getWorkflows as jest.Mock).mockResolvedValue({
      workflows: [{ code: 'incident_flow', name: '事件流程' }],
      total: 1,
    });
    (WorkflowAPI.listVersions as jest.Mock).mockResolvedValue([
      version(),
      version({ id: '41', version: 1, isActive: false, bpmnXml: '<definitions>v1</definitions>' }),
    ]);
  });

  afterEach(() => {
    consoleErrorSpy.mockRestore();
  });

  it('按 processKey 拉取版本，状态来自 isActive 且行 key 唯一', async () => {
    renderPage();

    const table = await screen.findByRole('table');
    expect(WorkflowAPI.listVersions).toHaveBeenCalledWith('incident_flow');

    expect(within(table).getAllByRole('row')).toHaveLength(3);
    expect(within(table).getAllByText('已激活')).toHaveLength(1);
    expect(within(table).getAllByText('未激活')).toHaveLength(1);
    expect(duplicateKeyErrors).toEqual([]);
  });

  it('创建/更新时间列分别取该版本的后端时间戳', async () => {
    renderPage();

    const table = await screen.findByRole('table');
    const rows = within(table).getAllByRole('row').slice(1);
    const firstRowCells = within(rows[0]).getAllByRole('cell');

    expect(firstRowCells[3]).toHaveTextContent('FMT(2024-01-01T10:00:00Z)');
    expect(firstRowCells[4]).toHaveTextContent('FMT(2024-01-02T10:00:00Z)');
  });

  it('已激活版本不允许再次激活或删除', async () => {
    renderPage();

    await screen.findByRole('table');
    expect(screen.getByLabelText('激活版本 v2')).toBeDisabled();
    expect(screen.getByLabelText('删除版本 v2')).toBeDisabled();
    expect(screen.getByLabelText('激活版本 v1')).not.toBeDisabled();
  });

  it('版本详情弹窗预览该版本的 BPMN 图形', async () => {
    renderPage();

    await screen.findByRole('table');
    fireEvent.click(screen.getByLabelText('查看版本 v1 详情'));

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('流程预览')).toBeInTheDocument();
    expect(within(dialog).getByTestId('bpmn-preview')).toHaveAttribute(
      'data-xml',
      '<definitions>v1</definitions>'
    );
    expect(within(dialog).getByText('事件流程')).toBeInTheDocument();
  });

  it('比较时以较低版本为 base、最新版本为 target', async () => {
    (WorkflowAPI.compareVersions as jest.Mock).mockResolvedValue(comparisonOf());
    renderPage();

    await screen.findByRole('table');
    fireEvent.click(screen.getByLabelText('比较版本 v1 与最新版本'));

    await waitFor(() =>
      expect(WorkflowAPI.compareVersions).toHaveBeenCalledWith('incident_flow', 1, 2)
    );

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('高风险变更')).toBeInTheDocument();
    expect(within(dialog).getByText('存在 1 项破坏性变更，运行中的流程实例可能受影响。')).toBeInTheDocument();
    // 破坏性变更清单与变更明细各出现一次
    expect(within(dialog).getAllByText('assignee changed')).toHaveLength(2);
    expect(within(dialog).getByText(/审批/)).toBeInTheDocument();
    expect(within(dialog).getByText('high')).toBeInTheDocument();
  });

  it('对最新版本自身比较时不发请求', async () => {
    renderPage();

    await screen.findByRole('table');
    fireEvent.click(screen.getByLabelText('比较版本 v2 与最新版本'));

    expect(await screen.findByText('当前版本已经是最新版本，无需比较。')).toBeInTheDocument();
    expect(WorkflowAPI.compareVersions).not.toHaveBeenCalled();
  });

  it('后端返回空比较结果时不弹出零差异结论', async () => {
    (WorkflowAPI.compareVersions as jest.Mock).mockResolvedValue(null);
    renderPage();

    await screen.findByRole('table');
    fireEvent.click(screen.getByLabelText('比较版本 v1 与最新版本'));

    expect(await screen.findByText('版本相同，无需比较。')).toBeInTheDocument();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('兼容结论未知时仍给出可读提示', async () => {
    (WorkflowAPI.compareVersions as jest.Mock).mockResolvedValue(
      comparisonOf({ compatibility: 'unknown', breakingChanges: [], changes: [] })
    );
    renderPage();

    await screen.findByRole('table');
    fireEvent.click(screen.getByLabelText('比较版本 v1 与最新版本'));

    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText('兼容性：unknown')).toBeInTheDocument();
    expect(within(dialog).getByText('未发现破坏性变更。')).toBeInTheDocument();
    expect(within(dialog).getByText('两个版本之间没有检测到结构变更。')).toBeInTheDocument();
  });
});
