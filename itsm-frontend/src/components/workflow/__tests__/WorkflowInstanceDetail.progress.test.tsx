/**
 * WorkflowInstanceDetail 可用性增强的回归锁（独立文件）。
 *
 * 与同目录 WorkflowInstanceDetail.test.tsx 分开的原因：那个文件当前有并行会话
 * 在途改动（审计动作中文化），两边同时编辑同一文件会互相覆盖。本文件只锁本轮
 * 新增的三件事：
 *   1) 基本信息里的「当前节点」行（此前该字段在 toSummary 里被丢弃）；
 *   2) 业务键可跳回业务单据（此前只当文本渲染）；
 *   3) 「流程进度」Tab 懒加载定义 XML 并按 currentActivityId 高亮。
 */

import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';

import WorkflowInstanceDetail from '@/components/workflow/WorkflowInstanceDetail';
import { WorkflowApi } from '@/lib/api/workflow-api';
import BPMNDashboardApi from '@/lib/api/bpmn-dashboard-api';

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowApi: {
    getInstance: jest.fn(),
    getNodeInstances: jest.fn(),
    getWorkflow: jest.fn(),
  },
}));

jest.mock('@/lib/api/bpmn-dashboard-api', () => ({
  __esModule: true,
  default: { getProcessTimeline: jest.fn() },
}));

jest.mock('next/navigation', () => ({ useRouter: jest.fn() }));

// BPMNDiagramPreview 是 dynamic 加载的（避免 bpmn-js 的 ESM 拖垮整个测试套件），
// 这里直接给它一个轻量替身，断言高亮参数被正确透传。
jest.mock('@/components/workflow/BPMNDiagramPreview', () => ({
  __esModule: true,
  default: ({ xml, activeElementId }: { xml: string; activeElementId?: string }) => (
    <div data-testid="bpmn-preview" data-xml-length={xml.length} data-active={activeElementId ?? ''} />
  ),
}));

const mockGetInstance = WorkflowApi.getInstance as jest.MockedFunction<typeof WorkflowApi.getInstance>;
const mockGetNodeInstances = WorkflowApi.getNodeInstances as jest.MockedFunction<
  typeof WorkflowApi.getNodeInstances
>;
const mockGetWorkflow = WorkflowApi.getWorkflow as jest.MockedFunction<typeof WorkflowApi.getWorkflow>;
const mockGetProcessTimeline = BPMNDashboardApi.getProcessTimeline as jest.MockedFunction<
  typeof BPMNDashboardApi.getProcessTimeline
>;

const baseInstance = {
  id: 'inst-1',
  processDefinitionKey: 'incident_emergency_flow',
  businessKey: 'incident:2',
  status: 'running',
  startTime: '2026-11-15T00:00:00Z',
} as never;

describe('WorkflowInstanceDetail 工作流可用性增强', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetInstance.mockResolvedValue({} as never);
    mockGetNodeInstances.mockResolvedValue([] as never);
    mockGetProcessTimeline.mockResolvedValue([] as never);
    mockGetWorkflow.mockResolvedValue({ bpmnXml: '<definitions />' } as never);
  });

  it('基本信息展示当前节点（字段从实例透传，不再靠任务表反推）', async () => {
    render(
      <WorkflowInstanceDetail
        instanceId="inst-1"
        initialInstance={{
          ...(baseInstance as object),
          currentActivityId: 'Activity_AutoAssign',
          currentActivityName: '自动分配',
        } as never}
      />
    );

    expect(screen.getByText('当前节点')).toBeInTheDocument();
    expect(screen.getByText('自动分配')).toBeInTheDocument();
    expect(screen.getByText('进行中')).toBeInTheDocument();
  });

  it('节点无 name 时回退 ID 并显式标注「无节点名称」', async () => {
    render(
      <WorkflowInstanceDetail
        instanceId="inst-2"
        initialInstance={{
          ...(baseInstance as object),
          currentActivityId: 'Activity_AutoAssign',
          currentActivityName: 'Activity_AutoAssign',
        } as never}
      />
    );

    expect(screen.getByText('Activity_AutoAssign')).toBeInTheDocument();
    // 用户必须能分辨「这是 ID 还是业务名」，否则会以为是乱码
    expect(screen.getByText('无节点名称')).toBeInTheDocument();
  });

  it('已结束实例没有当前节点时显示明确占位而不是空白', async () => {
    render(
      <WorkflowInstanceDetail
        instanceId="inst-3"
        initialInstance={{
          ...(baseInstance as object),
          status: 'completed',
          currentActivityId: undefined,
          currentActivityName: undefined,
        } as never}
      />
    );

    expect(screen.getByText('-（已结束或未进入任何节点）')).toBeInTheDocument();
  });

  it('不传 initialInstance 时靠 getInstance 填充当前节点（toSummary 不得丢字段）', async () => {
    // 这是 toSummary 唯一被走到的路径：列表弹窗传 initialInstance，
    // 深链接详情页不传。若 toSummary 把 currentActivity* 丢掉，
    // 深链接场景就会丢「当前节点」而列表弹窗场景看不出来。
    mockGetInstance.mockResolvedValueOnce({
      id: 'inst-deeplink',
      workflowId: 'incident_emergency_flow',
      status: 'running',
      businessKey: 'incident:2',
      currentActivityId: 'Activity_AutoAssign',
      currentActivityName: '自动分配',
      startTime: new Date('2026-11-15T00:00:00Z'),
    } as never);

    render(<WorkflowInstanceDetail instanceId="inst-deeplink" />);

    await waitFor(() => expect(mockGetInstance).toHaveBeenCalledWith('inst-deeplink'));
    expect(await screen.findByText('自动分配')).toBeInTheDocument();
  });

  it('业务键渲染成可点击的业务单据链接', async () => {
    render(<WorkflowInstanceDetail instanceId="inst-4" initialInstance={baseInstance} />);

    expect(screen.getByTestId('business-key-link')).toHaveAttribute('href', '/incidents/2');
    expect(screen.getByText('事件 #2')).toBeInTheDocument();
  });

  it('业务键不可解析时不渲染链接（宁可点不动也不误导）', async () => {
    render(
      <WorkflowInstanceDetail
        instanceId="inst-5"
        initialInstance={{ ...(baseInstance as object), businessKey: 'custom-key' } as never}
      />
    );

    expect(screen.queryByTestId('business-key-link')).not.toBeInTheDocument();
    expect(screen.getByText('custom-key')).toBeInTheDocument();
  });

  it('「流程进度」Tab 懒加载定义并以 currentActivityId 高亮', async () => {
    render(
      <WorkflowInstanceDetail
        instanceId="inst-6"
        initialInstance={{
          ...(baseInstance as object),
          currentActivityId: 'Activity_AutoAssign',
          currentActivityName: '自动分配',
        } as never}
      />
    );

    // 懒加载契约：进详情页不拉定义（否则每个用户都白下载 11KB 定义体）
    expect(mockGetWorkflow).not.toHaveBeenCalled();

    fireEvent.click(screen.getByText('流程进度'));

    await waitFor(() =>
      expect(mockGetWorkflow).toHaveBeenCalledWith('incident_emergency_flow')
    );
    const preview = await screen.findByTestId('bpmn-preview');
    expect(preview).toHaveAttribute('data-active', 'Activity_AutoAssign');
  });

  it('定义加载失败时给出可重试的错误态，而不是空白画布', async () => {
    mockGetWorkflow.mockRejectedValueOnce(new Error('boom'));

    render(<WorkflowInstanceDetail instanceId="inst-7" initialInstance={baseInstance} />);
    fireEvent.click(screen.getByText('流程进度'));

    expect(await screen.findByText('流程图加载失败')).toBeInTheDocument();
    expect(screen.queryByTestId('bpmn-preview')).not.toBeInTheDocument();
  });

  it('定义无 XML 时明确说明没有可渲染图形', async () => {
    mockGetWorkflow.mockResolvedValueOnce({ bpmnXml: '' } as never);

    render(<WorkflowInstanceDetail instanceId="inst-8" initialInstance={baseInstance} />);
    fireEvent.click(screen.getByText('流程进度'));

    await waitFor(() =>
      expect(screen.getByText('该流程定义没有可渲染的 BPMN 图形')).toBeInTheDocument()
    );
  });

  it('重复切回「流程进度」不重复请求定义（已有缓存）', async () => {
    render(<WorkflowInstanceDetail instanceId="inst-9" initialInstance={baseInstance} />);

    fireEvent.click(screen.getByText('流程进度'));
    await waitFor(() => expect(screen.getByTestId('bpmn-preview')).toBeInTheDocument());

    fireEvent.click(screen.getByText('基本信息'));
    fireEvent.click(screen.getByText('流程进度'));
    await waitFor(() => expect(screen.getByTestId('bpmn-preview')).toBeInTheDocument());

    expect(mockGetWorkflow).toHaveBeenCalledTimes(1);
  });
});
