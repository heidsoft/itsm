/**
 * 工单流转进度卡片（WorkflowProgressCard）的回归锁。
 *
 * 这个组件此前**一个测试都没有**：TicketDetail 的 5 个测试全部用
 * `jest.mock('...WorkflowProgressCard', () => ({ WorkflowProgressCard: () => null }))`
 * 把它整个抹掉，于是它是「真实用户最高频接触、但完全无人守护」的面。
 *
 * 本文件锁住本轮修掉的三件事：
 *   1) 按钮文案必须说实话：原按钮写「查看完整流程图」，实际跳的是实例**列表页**
 *      （带 businessKey 过滤 + instanceId 弹窗），用户会以为功能坏了；
 *   2) 内联展开：卡片里直接画流程图，默认折叠、首次展开才拉定义（不下载 bpmn-js
 *      整包给只看「卡在哪」的人）；
 *   3) 高亮入参：当前节点用 currentActivityId，已走过节点用 history 的 **activityId**
 *      （不是 activityName —— 展示名与 XML 元素 ID 对不上，拿去打标记一个都打不中）。
 */

import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';

import { WorkflowProgressCard } from '@/components/business/WorkflowProgressCard';
import { TicketWorkflowStateApi } from '@/lib/api/ticket-workflow-state-api';
import { WorkflowApi } from '@/lib/api/workflow-api';
import type { BpmnProcessState } from '@/types/ticket-workflow-state';

jest.mock('@/lib/api/ticket-workflow-state-api', () => ({
  TicketWorkflowStateApi: { tryGetStateV2: jest.fn() },
}));

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowApi: { getWorkflow: jest.fn() },
}));

/**
 * 内联预览在组件里是 next/dynamic 按需加载的，这里给它一个轻量替身。
 *
 * 双重必要性：一是 bpmn-js 是 ESM-only，jest 的 transformIgnorePatterns
 * 不转换 node_modules，真实模块会让整个套件加载失败（不是断言失败）；
 * 二是本轮要断言的是「传了什么入参」，替身里把入参摊平成 data-* 属性最直接。
 */
jest.mock('@/components/workflow/BPMNDiagramPreview', () => ({
  __esModule: true,
  default: ({
    xml,
    activeElementId,
    activeElementLabel,
    visitedElementIds,
  }: {
    xml: string;
    activeElementId?: string;
    activeElementLabel?: string;
    visitedElementIds?: string[];
  }) => (
    <div
      data-testid="bpmn-preview"
      data-xml-length={xml.length}
      data-active={activeElementId ?? ''}
      data-active-label={activeElementLabel ?? ''}
      data-visited={(visitedElementIds ?? []).join('|')}
    />
  ),
}));

const mockTryGetStateV2 = TicketWorkflowStateApi.tryGetStateV2 as jest.MockedFunction<
  typeof TicketWorkflowStateApi.tryGetStateV2
>;
const mockGetWorkflow = WorkflowApi.getWorkflow as jest.MockedFunction<typeof WorkflowApi.getWorkflow>;

const XML = '<bpmn:definitions><bpmn:process id="ticket_general_flow" /></bpmn:definitions>';

/** 一个「运行中」的最小状态：卡在 任务分配，之前走过 创建工单。 */
const runningState = (overrides: Partial<BpmnProcessState> = {}): BpmnProcessState =>
  ({
    processInstanceId: 'inst-1',
    processDefinitionKey: 'ticket_general_flow',
    processDefinitionName: '通用工单流程',
    bpmnStatus: 'running',
    currentActivityId: 'Activity_Assign',
    currentActivityName: '任务分配',
    currentActivityType: 'userTask',
    nextActivities: [
      {
        activityId: 'Activity_Resolve',
        activityName: '处理完成',
        activityType: 'userTask',
        isGateway: false,
      },
    ],
    // activityName 与 activityId 刻意不同：断言必须证明高亮取的是 activityId。
    history: [
      {
        activityId: 'Activity_Create',
        activityName: '创建工单',
        activityType: 'userTask',
        startTime: '2026-11-15T00:00:00Z',
      },
    ],
    ...overrides,
  }) as BpmnProcessState;

describe('WorkflowProgressCard', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockTryGetStateV2.mockResolvedValue(runningState());
    mockGetWorkflow.mockResolvedValue({ bpmnXml: XML } as never);
  });

  it('hidden 时直接返回 null，且不发请求（终态工单不该有待办外的任何流量）', () => {
    const { container } = render(<WorkflowProgressCard ticketId={1} hidden />);
    expect(container.firstChild).toBeNull();
    expect(mockTryGetStateV2).not.toHaveBeenCalled();
  });

  it('加载中先渲染骨架，不渲染任何节点信息', () => {
    mockTryGetStateV2.mockReturnValue(new Promise(() => {}) as never);
    render(<WorkflowProgressCard ticketId={1} />);
    expect(screen.getByTestId('workflow-progress-card-skeleton')).toBeInTheDocument();
  });

  it('后端不可用（tryGetStateV2 返回 null）时静默渲染 null，不破坏工单详情页骨架', async () => {
    mockTryGetStateV2.mockResolvedValue(null);
    const { container } = render(<WorkflowProgressCard ticketId={1} />);
    await waitFor(() => expect(mockTryGetStateV2).toHaveBeenCalledWith(1));
    expect(container.firstChild).toBeNull();
  });

  it('未启动时给出降级提示，不画图也不请求定义', async () => {
    mockTryGetStateV2.mockResolvedValue(
      runningState({ bpmnStatus: 'not_started', currentActivityId: undefined, history: [] })
    );
    render(<WorkflowProgressCard ticketId={1} />);
    expect(
      await screen.findByText('该工单尚未绑定 BPMN 流程，使用简化审批模式。')
    ).toBeInTheDocument();
    expect(mockGetWorkflow).not.toHaveBeenCalled();
  });

  it('已终止时标注「已终止」并提示不会继续流转', async () => {
    mockTryGetStateV2.mockResolvedValue(
      runningState({ bpmnStatus: 'terminated', currentActivityName: undefined })
    );
    render(<WorkflowProgressCard ticketId={1} />);
    expect(await screen.findByText('已终止')).toBeInTheDocument();
    expect(screen.getByText('流程已终止，不会继续流转。')).toBeInTheDocument();
  });

  describe('运行中：文字进度', () => {
    it('展示当前节点、下一步与进度计数', async () => {
      render(<WorkflowProgressCard ticketId={1} />);
      expect(await screen.findByText('任务分配')).toBeInTheDocument();
      expect(screen.getByText('处理完成')).toBeInTheDocument();
      // history 1 项 + 当前 1 + 下一步 1 = 3；已走 1
      expect(screen.getByText('进度 · 1/3')).toBeInTheDocument();
    });

    it('默认折叠流程图，且不预拉定义（只看「卡在哪」的人不该下载 bpmn-js 整包）', async () => {
      render(<WorkflowProgressCard ticketId={1} />);
      await screen.findByTestId('workflow-progress-card');
      expect(screen.queryByTestId('workflow-progress-diagram')).not.toBeInTheDocument();
      expect(screen.queryByTestId('bpmn-preview')).not.toBeInTheDocument();
      expect(mockGetWorkflow).not.toHaveBeenCalled();
    });
  });

  describe('内联展开流程图', () => {
    it('展开后才拉定义，并把当前节点与已走过节点按元素 ID 透传', async () => {
      render(<WorkflowProgressCard ticketId={1} />);
      fireEvent.click(await screen.findByText('展开流程图'));

      const preview = await screen.findByTestId('bpmn-preview');
      expect(mockGetWorkflow).toHaveBeenCalledWith('ticket_general_flow');
      expect(preview).toHaveAttribute('data-active', 'Activity_Assign');
      expect(preview).toHaveAttribute('data-active-label', '任务分配');
      // 关键：已走过节点必须是 activityId（Activity_Create），
      // 若误用 activityName（创建工单）则在 BPMN 图上必然打不中。
      expect(preview).toHaveAttribute('data-visited', 'Activity_Create');
      expect(preview.getAttribute('data-visited')).not.toContain('创建工单');
      expect(preview).toHaveAttribute('data-xml-length', String(XML.length));
    });

    it('history 里同一节点因回退/重入重复出现时去重，只打一次标记', async () => {
      mockTryGetStateV2.mockResolvedValue(
        runningState({
          history: [
            { activityId: 'Activity_Create', activityName: '创建工单', activityType: 'userTask', startTime: '2026-11-15T00:00:00Z' },
            { activityId: 'Activity_Review', activityName: '审核', activityType: 'userTask', startTime: '2026-11-15T01:00:00Z' },
            { activityId: 'Activity_Create', activityName: '创建工单', activityType: 'userTask', startTime: '2026-11-15T02:00:00Z' },
          ],
        })
      );
      render(<WorkflowProgressCard ticketId={1} />);
      fireEvent.click(await screen.findByText('展开流程图'));
      const preview = await screen.findByTestId('bpmn-preview');
      expect(preview).toHaveAttribute('data-visited', 'Activity_Create|Activity_Review');
    });

    it('再次收起再展开不重复请求定义（已拿到 XML 就复用）', async () => {
      render(<WorkflowProgressCard ticketId={1} />);
      fireEvent.click(await screen.findByText('展开流程图'));
      await screen.findByTestId('bpmn-preview');
      expect(mockGetWorkflow).toHaveBeenCalledTimes(1);

      fireEvent.click(screen.getByText('收起流程图'));
      await waitFor(() =>
        expect(screen.queryByTestId('workflow-progress-diagram')).not.toBeInTheDocument()
      );

      fireEvent.click(screen.getByText('展开流程图'));
      await screen.findByTestId('bpmn-preview');
      expect(mockGetWorkflow).toHaveBeenCalledTimes(1);
    });

    it('定义加载失败时给出可重试的告警，且文字进度仍然可见（图只是补充）', async () => {
      mockGetWorkflow.mockRejectedValueOnce(new Error('network down'));
      render(<WorkflowProgressCard ticketId={1} />);
      fireEvent.click(await screen.findByText('展开流程图'));

      expect(await screen.findByText('流程图加载失败')).toBeInTheDocument();
      expect(screen.getByText('加载流程定义失败')).toBeInTheDocument();
      // 图挂了不代表流转信息挂了：当前节点与进度必须还在。
      expect(screen.getByText('任务分配')).toBeInTheDocument();
      expect(screen.getByText('进度 · 1/3')).toBeInTheDocument();
      expect(screen.queryByTestId('bpmn-preview')).not.toBeInTheDocument();

      // 重试成功后出图。
      // 用 role + 正则而不是 getByText('重试')：antd 的 auto-insert-space 会
      // 在两个汉字的按钮文案中间插一个空格（DOM 里是「重 试」），按字面文本匹配
      // 会假红。这不是本组件的缺陷，是 antd 的既定行为。
      fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }));
      expect(await screen.findByTestId('bpmn-preview')).toBeInTheDocument();
      expect(mockGetWorkflow).toHaveBeenCalledTimes(2);
    });

    it('定义存在但没有可渲染图形时给出明确原因的提示（不是笼统的「加载失败」）', async () => {
      mockGetWorkflow.mockResolvedValue({ bpmnXml: '   ' } as never);
      render(<WorkflowProgressCard ticketId={1} />);
      fireEvent.click(await screen.findByText('展开流程图'));
      expect(await screen.findByText('该流程定义没有可渲染的 BPMN 图形')).toBeInTheDocument();
    });

    it('缺失 processDefinitionKey 时不发请求，直接告知原因', async () => {
      mockTryGetStateV2.mockResolvedValue(
        runningState({ processDefinitionKey: '' })
      );
      render(<WorkflowProgressCard ticketId={1} />);
      fireEvent.click(await screen.findByText('展开流程图'));
      expect(await screen.findByText('缺少流程定义 Key，无法渲染流程图')).toBeInTheDocument();
      expect(mockGetWorkflow).not.toHaveBeenCalled();
    });
  });

  describe('按钮文案与行为一致', () => {
    it('不再出现「查看完整流程图」这种与行为不符的文案', async () => {
      render(<WorkflowProgressCard ticketId={1} />);
      await screen.findByTestId('workflow-progress-card');
      expect(screen.queryByText('查看完整流程图')).not.toBeInTheDocument();
      expect(screen.getByText('展开流程图')).toBeInTheDocument();
      expect(screen.getByText('打开流程实例')).toBeInTheDocument();
    });

    it('「打开流程实例」带 businessKey 过滤与 instanceId 打开实例列表，并触发上层刷新', async () => {
      const openSpy = jest.spyOn(window, 'open').mockImplementation(() => null);
      const onRefresh = jest.fn();
      render(<WorkflowProgressCard ticketId={7} onRefresh={onRefresh} />);

      fireEvent.click(await screen.findByText('打开流程实例'));

      expect(openSpy).toHaveBeenCalledTimes(1);
      const url = openSpy.mock.calls[0][0] as string;
      expect(url).toContain('/workflow/instances');
      // businessKey 的拼法必须与后端 8 处构造点一致（`%s:%d`，见
      // bpmn_approval_bridge_service.go 等）：冒号在 query value 里是合法字符
      // （RFC 3986 pchar），不编码也能被后端正确解析，故断言未编码的形态。
      expect(url).toContain('businessKey=ticket:7');
      expect(url).toContain('instanceId=inst-1');
      expect(onRefresh).toHaveBeenCalled();
      openSpy.mockRestore();
    });
  });
});
