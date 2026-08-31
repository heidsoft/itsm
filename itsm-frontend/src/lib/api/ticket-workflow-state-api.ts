/**
 * 工单 BPMN 流程状态 API
 *
 * 用于工单详情页 / 流转卡片拉取真实 BPMN 节点状态。
 * 与后端 controller/ticket_workflow_controller.go GetTicketWorkflowStateV2 对齐：
 *
 *   GET /api/v1/tickets/:id/workflow/state-v2
 *
 * 返回的 BpmnProcessState 涵盖 not_started / running / completed / terminated 等
 * 全部状态，由前端按 bpmnStatus 字段分支渲染。
 *
 * 设计要点：
 *   - 不引入 axios / fetch wrapper，直接复用 httpClient（统一租户 / CSRF / 重试）。
 *   - 失败时上层可选择降级到原 TicketApi.getWorkflowState（V1 不含 BPMN 节点）。
 */

import { httpClient } from './http-client';
import type { BpmnProcessState } from '@/types/ticket-workflow-state';
import type { TicketWorkflowState } from '@/types/ticket-workflow';

export const TicketWorkflowStateApi = {
  /**
   * 拉取工单的 BPMN 流程状态聚合（V2）。
   *
   * @param ticketId 工单主键 ID
   * @returns BpmnProcessState；后端对未关联流程的工单返回 bpmnStatus='not_started'，
   *          对终态省略 currentActivity* 字段；调用方应做空值兜底。
   */
  getStateV2: async (ticketId: number): Promise<BpmnProcessState> => {
    const workflowState = await httpClient.get<TicketWorkflowState>(
      `/api/v1/tickets/${ticketId}/workflow/state-v2`
    );
    if (!workflowState.bpmnProcessState) {
      throw new Error('BPMN process state is unavailable');
    }
    return workflowState.bpmnProcessState;
  },

  /**
   * 安全包装：失败返回 null 而非抛出，便于在 render 路径中并发调用而不污染
   * 其它子请求的错误状态。
   *
   * @param ticketId 工单主键 ID
   */
  tryGetStateV2: async (ticketId: number): Promise<BpmnProcessState | null> => {
    try {
      return await TicketWorkflowStateApi.getStateV2(ticketId);
    } catch {
      return null;
    }
  },
};
