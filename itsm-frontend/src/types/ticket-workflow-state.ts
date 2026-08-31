/**
 * 工单 BPMN 流程状态类型定义
 *
 * 与后端 dto/ticket_workflow_dto.go 中的 BpmnProcessState / NextActivityInfo /
 * BpmnHistoryItem 严格 1:1 对齐（camelCase JSON 字段）。
 *
 * 工单三件套改造：详情页 / 流转卡片通过 GET /api/v1/tickets/:id/workflow/state-v2
 * 拉取真实 BPMN 节点状态，用于"当前节点 / 下一节点 / 已走节点 / 终态"展示。
 */

import type { TicketUser } from './ticket';

/**
 * BPMN 流程状态机取值。
 *
 * - not_started：工单存在但尚未关联任何运行中的 BPMN 流程实例（降级 UI）。
 * - running：流程进行中；currentActivity* 有值。
 * - suspended：流程被挂起（暂未触发生产路径）。
 * - completed / terminated：终态；currentActivity* 省略，nextActivities 为空。
 */
export type BpmnStatus =
  | 'not_started'
  | 'running'
  | 'suspended'
  | 'completed'
  | 'terminated';

/**
 * BPMN 节点类型，与后端 enrich 阶段对 XML 元素的归一化结果一致。
 */
export type BpmnActivityType =
  | 'startEvent'
  | 'endEvent'
  | 'userTask'
  | 'serviceTask'
  | 'exclusiveGateway'
  | 'parallelGateway'
  | 'inclusiveGateway'
  | string;

/**
 * 流程当前节点的处理人列表。
 * 与后端 WorkflowUserInfo 字段一一对应；前端可直接复用 TicketUser。
 */
export type WorkflowAssignee = TicketUser;

/**
 * 单个"下一步候选活动"。
 * IsGateway=true 时表示该活动为网关节点（exclusive / parallel / inclusive），
 * 调用方应据此决定是否展开分支说明（例如"如审批通过则进入 X，否则进入 Y"）。
 */
export interface NextActivityInfo {
  activityId: string;
  activityName: string;
  activityType: BpmnActivityType;
  assignees?: WorkflowAssignee[];
  isGateway: boolean;
}

/**
 * 流程历史快照中的一项。
 *
 * - Outcome 取自 process_tasks 的 status + 业务变量（approvalResult / approvalAction），
 *   例如 approved / rejected / completed / skipped。
 * - Assignee 可能为 null（系统任务、网关等无处理人）。
 */
export interface BpmnHistoryItem {
  activityId: string;
  activityName: string;
  activityType: BpmnActivityType;
  startTime: string; // ISO8601
  endTime?: string | null;
  assignee?: WorkflowAssignee | null;
  outcome?: string;
}

/**
 * 工单对应的 BPMN 流程状态聚合。
 *
 * 单结构体覆盖全部状态：调用方按 bpmnStatus 分支：
 * - 'not_started' → 显示降级 Alert
 * - 'running'     → 渲染 current / next / history 三段
 * - 'completed' / 'terminated' → 显示终态 Tag，仅展示 history
 * - 'suspended'   → 保留扩展（暂未在生产触发）
 */
export interface BpmnProcessState {
  processInstanceId: string;
  processDefinitionKey: string;
  processDefinitionName: string;
  bpmnStatus: BpmnStatus;
  currentActivityId?: string;
  currentActivityName?: string;
  currentActivityType?: BpmnActivityType;
  currentAssignees?: WorkflowAssignee[];
  nextActivities?: NextActivityInfo[];
  history?: BpmnHistoryItem[];
  startedAt?: string | null;
  endedAt?: string | null;
}

/**
 * 进度卡片 / 当前节点渲染所需的精简视图。
 *
 * ApprovalWorkflowPanel 与 WorkflowProgressCard 共用，便于上层避免重复解析
 * NextActivities / History（按需计算即可）。
 */
export interface WorkflowProgressView {
  state: BpmnProcessState;
  totalActivities: number;     // 已走节点 + 当前节点 + 下一节点的去重数（估算）
  completedActivities: number; // 历史节点数
}
