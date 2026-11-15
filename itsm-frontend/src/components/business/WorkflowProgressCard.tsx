'use client';

/**
 * 工单流转进度卡片
 *
 * 工单三件套改造核心组件：在详情页 header 下方、详情 Tabs 上方展示当前 BPMN
 * 节点的"轻量概览"（区别于 ApprovalWorkflowPanel 的详细 Steps）。
 *
 * 渲染规则：
 * - 加载中：Skeleton
 * - 未启动：Alert 提示 + "使用简化审批模式"
 * - 运行中：当前节点卡片 + 处理人头像 + 下一节点预览 + 进度条
 * - 已完成 / 已终止：终态 Tag
 * - 异常 / 加载失败：静默渲染 null（由 ApprovalWorkflowPanel 兜底）
 */

import React, { useCallback, useMemo, useState, useEffect } from 'react';
import { Card, Tag, Space, Typography, Alert, Progress, Tooltip, Button, Skeleton } from "antd";
import dynamic from 'next/dynamic';
import {
  Workflow as WorkflowIcon,
  ArrowRight,
  User as UserIcon,
  ExternalLink,
  CheckCircle,
  CircleAlert,
  CircleSlash,
  ChevronDown,
  ChevronUp,
} from 'lucide-react';
import { TicketWorkflowStateApi } from '@/lib/api/ticket-workflow-state-api';
import { WorkflowApi } from '@/lib/api/workflow-api';
import type { BpmnProcessState } from '@/types/ticket-workflow-state';

/**
 * 内联流程图按需加载。
 *
 * 三条理由都必须写下来，否则下一个人会顺手改回静态 import：
 * 1) 工单详情页是最高频页面，绝大多数用户只看「卡在哪个节点」，不需要
 *    立即下载 bpmn-js（diagram-js + bpmn-moddle 整包）；
 * 2) jest 的 transformIgnorePatterns 不转换 node_modules，静态 import 会让
 *    bpmn-js 的 `import inherits from 'inherits-browser'` 抛进测试运行时，
 *    引用本卡片的测试套件整体加载失败（不是断言失败）；
 * 3) 卡片在 hidden 状态下直接返回 null，此时连代码都不该被请求。
 */
const BPMNDiagramPreview = dynamic(() => import('@/components/workflow/BPMNDiagramPreview'), {
  ssr: false,
  loading: () => <Skeleton active paragraph={{ rows: 4 }} />,
});

const { Text } = Typography;

// BPMN 活动类型在后端可能是 camelCase（userTask）或 snake_case（user_task），
// 统一归一化为中文标签，不在卡片里直出英文枚举。
const ACTIVITY_TYPE_LABELS: Record<string, string> = {
  startevent: '开始事件',
  endevent: '结束事件',
  usertask: '用户任务',
  servicetask: '服务任务',
  scripttask: '脚本任务',
  manualtask: '人工任务',
  businessruletask: '规则任务',
  sendtask: '发送任务',
  receivetask: '接收任务',
  exclusivegateway: '排他网关',
  parallelgateway: '并行网关',
  inclusivegateway: '包容网关',
  eventbasedgateway: '事件网关',
  subprocess: '子流程',
  callactivity: '调用活动',
};

function getActivityTypeLabel(type?: string | null): string {
  const key = (type || '').replace(/[_-]/g, '').toLowerCase();
  return ACTIVITY_TYPE_LABELS[key] ?? (type || '用户任务');
}

export interface WorkflowProgressCardProps {
  ticketId: number;
  /**
   * 是否折叠 / 隐藏；true 时直接返回 null（用于 ticketDetail 在 isTicketFinal 时折叠）。
   */
  hidden?: boolean;
  /**
   * 数据刷新回调；卡片自身有 polling 的话会触发该回调同步上层。
   */
  onRefresh?: () => void;
}

interface ProgressView {
  /** 已走节点数（来自 history.length） */
  completed: number;
  /** 总节点估算 = completed + current(1, 仅 running) + nextActivities(可能多) */
  total: number;
}

function deriveProgress(state: BpmnProcessState): ProgressView {
  const completed = state.history?.length ?? 0;
  if (state.bpmnStatus !== 'running') {
    return { completed, total: completed };
  }
  // running 状态：当前节点 +1，下一节点数计入总进度
  const nextCount = state.nextActivities?.length ?? 0;
  const total = Math.max(completed + 1 + nextCount, completed + 1);
  return { completed, total };
}

export const WorkflowProgressCard: React.FC<WorkflowProgressCardProps> = ({
  ticketId,
  hidden,
  onRefresh,
}) => {
  const [loading, setLoading] = useState(true);
  const [state, setState] = useState<BpmnProcessState | null>(null);
  // 内联流程图默认折叠：卡片定位是「一眼看卡在哪」，展开才是「看全貌」。
  const [diagramOpen, setDiagramOpen] = useState(false);
  const [definitionXml, setDefinitionXml] = useState<string | undefined>(undefined);
  const [definitionLoading, setDefinitionLoading] = useState(false);
  const [definitionError, setDefinitionError] = useState<string | undefined>(undefined);

  useEffect(() => {
    if (hidden) return;
    let cancelled = false;
    setLoading(true);
    TicketWorkflowStateApi.tryGetStateV2(ticketId)
      .then((s) => {
        if (!cancelled) setState(s);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [ticketId, hidden]);

  /**
   * 拉取流程定义 XML（仅在展开流程图时触发）。
   *
   * state-v2 不下发 bpmnXml（工单域的 DTO 只带节点状态，定义体属流程定义域），
   * 因此必须按 processDefinitionKey 单取。失败要可重试，不能把卡片打成错误态——
   * 文字版进度本身仍然有效，图只是补充。
   */
  const loadDefinition = useCallback(async () => {
    const key = state?.processDefinitionKey;
    if (!key) {
      setDefinitionError('缺少流程定义 Key，无法渲染流程图');
      return;
    }
    setDefinitionLoading(true);
    setDefinitionError(undefined);
    try {
      const definition = await WorkflowApi.getWorkflow(key);
      const xml = definition.bpmnXml;
      if (!xml || !xml.trim()) {
        setDefinitionError('该流程定义没有可渲染的 BPMN 图形');
        return;
      }
      setDefinitionXml(xml);
    } catch {
      setDefinitionError('加载流程定义失败');
    } finally {
      setDefinitionLoading(false);
    }
  }, [state?.processDefinitionKey]);

  const toggleDiagram = useCallback(() => {
    setDiagramOpen(prev => {
      const next = !prev;
      // 首次展开才拉定义；已成功加载过不重复请求
      if (next && !definitionXml && !definitionLoading) void loadDefinition();
      return next;
    });
  }, [definitionXml, definitionLoading, loadDefinition]);

  /**
   * 已走过节点的 BPMN 元素 ID（供图上打标记）。
   *
   * 注意 history 的 activityId 才是结构 ID；activityName 是展示名，与 XML 对不上。
   */
  const visitedActivityIds = useMemo(() => {
    const ids = (state?.history ?? []).map(item => item.activityId).filter(Boolean);
    return Array.from(new Set(ids));
  }, [state?.history]);

  if (hidden) return null;

  if (loading) {
    return (
      <Card size="small" className="shadow-sm" data-testid="workflow-progress-card-skeleton">
        <Skeleton active paragraph={{ rows: 2 }} />
      </Card>
    );
  }

  if (!state) {
    // 后端不可用或网络错误：静默渲染，不破坏工单详情页骨架。
    return null;
  }

  const { completed, total } = deriveProgress(state);
  const percent = total > 0 ? Math.min(100, Math.round((completed / total) * 100)) : 0;

  const openInstanceList = (instanceId: string) => {
    if (typeof window === 'undefined') return;
    const url = `/workflow/instances?businessKey=ticket:${ticketId}&instanceId=${encodeURIComponent(instanceId)}`;
    window.open(url, '_blank', 'noopener,noreferrer');
  };

  // ---- not_started：降级提示 ----
  if (state.bpmnStatus === 'not_started') {
    return (
      <Card
        size="small"
        className="shadow-sm"
        title={
          <Space size={6}>
            <WorkflowIcon size={14} />
            <Text strong>流转进度</Text>
            <Tag color="default">未启动</Tag>
          </Space>
        }
      >
        <Alert
          type="info"
          showIcon
          title="该工单尚未绑定 BPMN 流程，使用简化审批模式。"
          className="text-xs"
        />
      </Card>
    );
  }

  // ---- 终态 ----
  if (state.bpmnStatus === 'completed' || state.bpmnStatus === 'terminated') {
    const isTerminated = state.bpmnStatus === 'terminated';
    return (
      <Card
        size="small"
        className="shadow-sm"
        title={
          <Space size={6}>
            <WorkflowIcon size={14} />
            <Text strong>流转进度</Text>
            <Tag color={isTerminated ? 'red' : 'success'} icon={isTerminated ? <CircleSlash size={12} /> : <CheckCircle size={12} />}>
              {isTerminated ? '已终止' : '已完成'}
            </Tag>
          </Space>
        }
        extra={
          <Button
            type="link"
            size="small"
            icon={<ExternalLink size={12} />}
            onClick={() => openInstanceList(state.processInstanceId)}
          >
            查看流程实例
          </Button>
        }
      >
        <Space orientation="vertical" size={4} className="w-full">
          <Text type="secondary" className="text-xs">
            {state.processDefinitionName || state.processDefinitionKey}
          </Text>
          {isTerminated && (
            <Alert
              type="error"
              showIcon
              icon={<CircleAlert size={14} />}
              title="流程已终止，不会继续流转。"
              className="text-xs"
            />
          )}
        </Space>
      </Card>
    );
  }

  // ---- suspended ----
  if (state.bpmnStatus === 'suspended') {
    return (
      <Card
        size="small"
        className="shadow-sm"
        title={
          <Space size={6}>
            <WorkflowIcon size={14} />
            <Text strong>流转进度</Text>
            <Tag color="warning">已挂起</Tag>
          </Space>
        }
      >
        <Alert type="warning" showIcon title="流程已被挂起，恢复后才能继续流转。" className="text-xs" />
      </Card>
    );
  }

  // ---- running ----
  return (
    <Card
      size="small"
      className="shadow-sm"
      data-testid="workflow-progress-card"
      title={
        <Space size={6}>
          <WorkflowIcon size={14} />
          <Text strong>流转进度</Text>
          <Tag color="processing">进行中</Tag>
        </Space>
      }
      extra={
        <Space size={4}>
          <Button
            type="link"
            size="small"
            icon={diagramOpen ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
            onClick={toggleDiagram}
          >
            {diagramOpen ? '收起流程图' : '展开流程图'}
          </Button>
          <Button
            type="link"
            size="small"
            icon={<ExternalLink size={12} />}
            onClick={() => {
              // 这里打开的是实例列表（带 businessKey 过滤 + 指定 instanceId 弹窗），
              // 不是流程图本身。文案必须说实话——按钮写着「查看完整流程图」却跳到
              // 一个列表页，用户会以为功能坏了。看图请用上面的「展开流程图」。
              openInstanceList(state.processInstanceId);
              onRefresh?.();
            }}
          >
            打开流程实例
          </Button>
        </Space>
      }
    >
      <Space orientation="vertical" size={10} className="w-full">
        {/* 当前节点 */}
        {state.currentActivityName && (
          <div>
            <Text type="secondary" className="text-xs">
              当前节点
            </Text>
            <div className="mt-1 flex items-center gap-2">
              <Tag color="blue" className="mr-0">
                {getActivityTypeLabel(state.currentActivityType)}
              </Tag>
              <Text strong>{state.currentActivityName}</Text>
            </div>
            {state.currentAssignees && state.currentAssignees.length > 0 && (
              <Space size={4} className="mt-2" wrap>
                <Text type="secondary" className="text-xs">
                  处理人
                </Text>
                {state.currentAssignees.map((u) => (
                  <Tooltip key={u.id} title={u.fullName || u.username}>
                    <Tag color="geekblue" icon={<UserIcon size={10} />}>
                      {u.fullName || u.username}
                    </Tag>
                  </Tooltip>
                ))}
              </Space>
            )}
          </div>
        )}

        {/* 下一节点 */}
        {state.nextActivities && state.nextActivities.length > 0 && (
          <div>
            <Text type="secondary" className="text-xs">
              下一步
            </Text>
            <Space orientation="vertical" size={2} className="mt-1 w-full">
              {state.nextActivities.map((nx) => (
                <Space key={nx.activityId} size={6} className="text-xs">
                  <ArrowRight size={12} className="text-gray-400" />
                  <Tag color={nx.isGateway ? 'orange' : 'blue'} className="mr-0">
                    {getActivityTypeLabel(nx.activityType)}
                  </Tag>
                  <Text>{nx.activityName}</Text>
                  {nx.isGateway && (
                    <Tag color="orange" className="ml-1">
                      网关分支
                    </Tag>
                  )}
                </Space>
              ))}
            </Space>
          </div>
        )}

        {/* 进度条 */}
        <div>
          <Text type="secondary" className="text-xs">
            进度 · {completed}/{total}
          </Text>
          <Progress percent={percent} size="small" showInfo={false} />
        </div>

        {/* 内联流程图（默认折叠）。展开时画出整张流程并标出「已走过」与「当前节点」，
            这样用户在同一页就能回答「我从哪来、现在在哪、下一步去哪」，
            不必开新窗口、也不必在实例列表里再找一次。 */}
        {diagramOpen && (
          <div data-testid="workflow-progress-diagram">
            {definitionLoading && <Skeleton active paragraph={{ rows: 4 }} />}
            {!definitionLoading && definitionError && (
              <Alert
                type="warning"
                showIcon
                title="流程图加载失败"
                description={definitionError}
                className="text-xs"
                action={
                  <Button size="small" onClick={() => void loadDefinition()}>
                    重试
                  </Button>
                }
              />
            )}
            {!definitionLoading && !definitionError && definitionXml && (
              <BPMNDiagramPreview
                xml={definitionXml}
                height={320}
                activeElementId={state.currentActivityId}
                activeElementLabel={state.currentActivityName}
                visitedElementIds={visitedActivityIds}
              />
            )}
          </div>
        )}
      </Space>
    </Card>
  );
};

export default WorkflowProgressCard;
