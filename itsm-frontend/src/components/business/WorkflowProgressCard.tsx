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

import React, { useEffect, useState } from 'react';
import { Card, Tag, Space, Typography, Alert, Progress, Tooltip, Button, Skeleton } from "antd";
import {
  Workflow as WorkflowIcon,
  ArrowRight,
  User as UserIcon,
  ExternalLink,
  CheckCircle,
  CircleAlert,
  CircleSlash,
} from 'lucide-react';
import { TicketWorkflowStateApi } from '@/lib/api/ticket-workflow-state-api';
import type { BpmnProcessState } from '@/types/ticket-workflow-state';

const { Text } = Typography;

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

  const openDiagram = (instanceId: string) => {
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
          message="该工单尚未绑定 BPMN 流程，使用简化审批模式。"
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
            onClick={() => openDiagram(state.processInstanceId)}
          >
            查看完整流程图
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
              message="流程已终止，不会继续流转。"
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
        <Alert type="warning" showIcon message="流程已被挂起，恢复后才能继续流转。" className="text-xs" />
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
        <Button
          type="link"
          size="small"
          icon={<ExternalLink size={12} />}
          onClick={() => {
            openDiagram(state.processInstanceId);
            onRefresh?.();
          }}
        >
          查看完整流程图
        </Button>
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
                {state.currentActivityType || 'userTask'}
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
                    {nx.activityType}
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
      </Space>
    </Card>
  );
};

export default WorkflowProgressCard;
