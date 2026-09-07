'use client';

/**
 * 工作流实例详情视图
 *
 * 供以下两个入口共享：
 *   1. `/workflow/instances` 列表页中的"详情"弹窗
 *   2. `/workflow/instances/[id]` 深链接详情页
 *
 * 组件内部按 instanceId 主动拉取实例、节点任务、执行历史三个数据源，
 * 接受可选 `initialInstance` 用于列表场景下立即渲染基本信息。
 */

import React, { useEffect, useMemo, useState } from 'react';
import { App, Button, Descriptions, Space, Table, Tabs, Tag, Timeline, Badge } from 'antd';
import { ArrowLeft, Clock, MessageSquare, RefreshCw, User } from 'lucide-react';

import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { WorkflowApi } from '@/lib/api/workflow-api';
import BPMNDashboardApi, { type ProcessAuditLog } from '@/lib/api/bpmn-dashboard-api';
import type { NodeInstance, WorkflowInstance } from '@/types/workflow';

const formatDateTime = (value?: string | Date | null): string => {
  if (!value) return '-';
  const date = typeof value === 'string' ? new Date(value) : value;
  if (Number.isNaN(date.getTime()) || date.getTime() < 0) {
    return '-';
  }
  return date.toLocaleString('zh-CN');
};

const formatDuration = (ms: number): string => {
  if (!Number.isFinite(ms) || ms < 0) return '-';
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(0)}s`;
  if (ms < 3600000) return `${(ms / 60000).toFixed(0)}m`;
  return `${(ms / 3600000).toFixed(1)}h`;
};

const statusColorMap: Record<string, string> = {
  running: 'green',
  completed: 'blue',
  suspended: 'orange',
  terminated: 'red',
  failed: 'red',
};

const statusTextMap: Record<string, string> = {
  running: '运行中',
  completed: '已完成',
  suspended: '已暂停',
  terminated: '已终止',
  failed: '失败',
};

const taskStatusColorMap: Record<string, string> = {
  pending: 'gold',
  inProgress: 'blue',
  completed: 'green',
  failed: 'red',
  cancelled: 'gray',
  skipped: 'gray',
};

const auditActionColorMap: Record<string, string> = {
  PROCESS_STARTED: 'green',
  PROCESS_COMPLETED: 'blue',
  PROCESS_SUSPENDED: 'orange',
  PROCESS_RESUMED: 'green',
  PROCESS_TERMINATED: 'red',
  TASK_CREATED: 'blue',
  TASK_ASSIGNED: 'cyan',
  TASK_COMPLETED: 'green',
  TASK_FAILED: 'red',
  TASK_SKIPPED: 'gray',
  VARIABLE_UPDATED: 'purple',
  GATEWAY_PASSED: 'orange',
  SEQUENCE_FLOW_TAKEN: 'cyan',
};

interface InstanceSummary {
  id: string;
  processDefinitionKey: string;
  businessKey: string;
  status: string;
  startTime?: string;
  endTime?: string;
}

export interface WorkflowInstanceDetailProps {
  instanceId: string;
  /**
   * 列表页可传入已渲染好的实例行，避免重复请求；详情路由场景可不传，由组件自行 fetch。
   */
  initialInstance?: InstanceSummary;
  /**
   * 深链接详情页用：点击"返回列表"时调用。
   * 不传则不渲染返回按钮（Modal 场景）。
   */
  onBack?: () => void;
  /**
   * 是否允许在内部刷新（用于 Modal 场景的"打开即拉一次"）。
   */
  autoRefresh?: boolean;
}

const toSummary = (instance: WorkflowInstance): InstanceSummary => ({
  id: instance.id,
  processDefinitionKey: instance.workflowId || '-',
  businessKey: (instance as unknown as { businessKey?: string }).businessKey || '-',
  status: String(instance.status),
  startTime: instance.startTime ? new Date(instance.startTime).toISOString() : undefined,
  endTime: instance.endTime ? new Date(instance.endTime).toISOString() : undefined,
});

export const WorkflowInstanceDetail: React.FC<WorkflowInstanceDetailProps> = ({
  instanceId,
  initialInstance,
  onBack,
  autoRefresh = true,
}) => {
  const { message } = App.useApp();
  const [activeTab, setActiveTab] = useState('basic');
  const [loading, setLoading] = useState(false);
  const [summary, setSummary] = useState<InstanceSummary | undefined>(initialInstance);
  const [tasks, setTasks] = useState<NodeInstance[]>([]);
  const [auditLogs, setAuditLogs] = useState<ProcessAuditLog[]>([]);

  const loadDetail = async () => {
    if (!instanceId) return;
    try {
      setLoading(true);
      const [instance, fetchedTasks, timeline] = await Promise.all([
        summary
          ? Promise.resolve(null)
          : WorkflowApi.getInstance(instanceId).catch(() => null),
        WorkflowApi.getNodeInstances(instanceId).catch(() => [] as NodeInstance[]),
        BPMNDashboardApi.getProcessTimeline(instanceId).catch(() => [] as ProcessAuditLog[]),
      ]);

      if (instance) {
        setSummary(toSummary(instance));
      }
      setTasks(fetchedTasks || []);
      setAuditLogs(timeline || []);
    } catch (error) {
      console.error('Failed to load instance detail:', error);
      message.error('加载实例详情失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (autoRefresh) {
      void loadDetail();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [instanceId]);

  // taskColumns 只引用模块级常量（taskStatusColorMap / formatDateTime 等），
  // 这些值在组件生命周期内稳定。刻意省略依赖列表以避免每次 loading 切换时
  // 重复构造数组引用，配合下方 tabItems 的 useMemo 依赖数组避免无效重算。
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const taskColumns = useMemo(
    () => [
      {
        title: '任务ID',
        dataIndex: 'id',
        key: 'id',
        width: 120,
        render: (value: string) => <span className="font-mono text-xs">{value}</span>,
      },
      {
        title: '节点名称',
        dataIndex: 'nodeName',
        key: 'nodeName',
        width: 150,
      },
      {
        title: '节点类型',
        dataIndex: 'nodeType',
        key: 'nodeType',
        width: 100,
        render: (value: string) => (value ? value.replace('_', ' ') : '-'),
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (value: string) => <Tag color={taskStatusColorMap[value] || 'default'}>{value}</Tag>,
      },
      {
        title: '处理人',
        dataIndex: 'assigneeName',
        key: 'assigneeName',
        width: 120,
        render: (value: string, record: NodeInstance) => value || record.assignee || '-',
      },
      {
        title: '创建时间',
        dataIndex: 'createdAt',
        key: 'createdAt',
        width: 180,
        render: (value: string) => formatDateTime(value),
      },
      {
        title: '截止时间',
        dataIndex: 'dueDate',
        key: 'dueDate',
        width: 180,
        render: (value: string) => (value ? formatDateTime(value) : '-'),
      },
    ],
    []
  );

  const tabItems = useMemo(
    () => [
      {
        key: 'basic',
        label: '基本信息',
        children: summary ? (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="实例 ID">
              <span className="font-mono text-xs">{summary.id}</span>
            </Descriptions.Item>
            <Descriptions.Item label="流程 Key">{summary.processDefinitionKey}</Descriptions.Item>
            <Descriptions.Item label="业务键">{summary.businessKey}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={statusColorMap[summary.status] || 'default'}>
                {statusTextMap[summary.status] || summary.status}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="启动时间">{formatDateTime(summary.startTime)}</Descriptions.Item>
            <Descriptions.Item label="结束时间">{formatDateTime(summary.endTime)}</Descriptions.Item>
            {summary.startTime && (
              <Descriptions.Item label="持续时间">
                {summary.endTime
                  ? formatDuration(
                      new Date(summary.endTime).getTime() - new Date(summary.startTime).getTime()
                    )
                  : formatDuration(Date.now() - new Date(summary.startTime).getTime())}
              </Descriptions.Item>
            )}
          </Descriptions>
        ) : (
          <LoadingEmptyError
            state="loading"
            loadingText="正在加载基本信息..."
          />
        ),
      },
      {
        key: 'tasks',
        label: `任务列表 (${tasks.length})`,
        children: (
          <LoadingEmptyError
            state={loading ? 'loading' : tasks.length === 0 ? 'empty' : 'success'}
            loadingText="正在加载任务列表..."
            empty={{
              title: '暂无任务数据',
              description: '该流程实例还没有产生任何任务',
            }}
          >
            <Table
              columns={taskColumns}
              dataSource={tasks}
              rowKey="id"
              pagination={{ pageSize: 10 }}
              size="small"
            />
          </LoadingEmptyError>
        ),
      },
      {
        key: 'history',
        label: `执行历史 (${auditLogs.length})`,
        children: (
          <LoadingEmptyError
            state={loading ? 'loading' : auditLogs.length === 0 ? 'empty' : 'success'}
            loadingText="正在加载执行历史..."
            empty={{
              title: '暂无执行历史',
              description: '该流程实例还没有执行记录',
            }}
          >
            <div className="max-h-[500px] overflow-y-auto pr-2">
              <Timeline
                items={auditLogs.map((log, index) => {
                  const actionColor = auditActionColorMap[log.action] || 'gray';
                  return {
                    color: actionColor,
                    children: (
                      <div className="mb-3" key={log.id ?? index}>
                        <div className="flex justify-between items-center mb-1">
                          <Space>
                            <Badge color={actionColor} />
                            <span className="font-medium text-sm">
                              {(log.action || '').replace(/_/g, ' ')}
                            </span>
                            {log.activityName && <Tag>{log.activityName}</Tag>}
                          </Space>
                          <span className="text-xs text-gray-500">
                            {formatDateTime(log.timestamp)}
                          </span>
                        </div>

                        <div className="text-xs text-gray-600 mb-1 pl-5">
                          {log.userName && (
                            <span className="mr-3">
                              <User className="w-3 h-3 inline mr-1" />
                              {log.userName}
                            </span>
                          )}
                          {log.assigneeName && (
                            <span className="mr-3">
                              <User className="w-3 h-3 inline mr-1" />
                              处理人: {log.assigneeName}
                            </span>
                          )}
                          {log.durationMs ? (
                            <span>
                              <Clock className="w-3 h-3 inline mr-1" />
                              耗时: {formatDuration(log.durationMs)}
                            </span>
                          ) : null}
                        </div>

                        {log.comment && (
                          <div className="text-xs bg-gray-50 p-2 rounded ml-5 mb-1">
                            <MessageSquare className="w-3 h-3 inline mr-1 text-gray-400" />
                            {log.comment}
                          </div>
                        )}

                        {log.variablesAfter && Object.keys(log.variablesAfter).length > 0 && (
                          <div className="text-xs ml-5 mt-1">
                            <details className="cursor-pointer">
                              <summary className="text-blue-500">变量变更</summary>
                              <pre className="mt-1 p-2 bg-gray-50 rounded overflow-x-auto text-[10px]">
                                {JSON.stringify(log.variablesAfter, null, 2)}
                              </pre>
                            </details>
                          </div>
                        )}
                      </div>
                    ),
                  };
                })}
              />
            </div>
          </LoadingEmptyError>
        ),
      },
    ],
    [loading, summary, tasks, auditLogs, taskColumns]
  );

  const header = onBack ? (
    <div className="mb-4 flex items-center justify-between">
      <Space>
        <Button icon={<ArrowLeft className="h-4 w-4" />} onClick={onBack}>
          返回列表
        </Button>
        <span className="text-base font-semibold">
          {summary ? `实例详情 · ${summary.id}` : `实例详情 · ${instanceId}`}
        </span>
      </Space>
      <Button icon={<RefreshCw className="h-4 w-4" />} onClick={loadDetail} loading={loading}>
        刷新
      </Button>
    </div>
  ) : null;

  return (
    <div>
      {header}
      <Tabs activeKey={activeTab} onChange={setActiveTab} items={tabItems} />
    </div>
  );
};

export default WorkflowInstanceDetail;