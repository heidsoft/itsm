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

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import dynamic from 'next/dynamic';
import { App, Button, Descriptions, Space, Spin, Table, Tabs, Tag, Timeline, Badge, Typography } from 'antd';
import { ArrowLeft, Clock, ExternalLink, MessageSquare, RefreshCw, User } from 'lucide-react';

import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { WorkflowApi } from '@/lib/api/workflow-api';
import BPMNDashboardApi, { type ProcessAuditLog } from '@/lib/api/bpmn-dashboard-api';
import { formatDateTime } from '@/lib/formatters';
import { parseBusinessKey } from '@/lib/workflow/business-key';
import type { NodeInstance, WorkflowInstance } from '@/types/workflow';

/**
 * BPMN 预览按需加载。
 *
 * 两个理由都必须写在代码里，否则下一个人会顺手改回静态 import：
 * 1) bpmn-js 是 ESM-only 且体积大（含 diagram-js + bpmn-moddle）。详情页的
 *    基本信息/任务列表根本不需要它，静态 import 会让每个打开实例详情的人都
 *    下载整包；
 * 2) jest 的 transformIgnorePatterns 不转换 node_modules，静态 import 会把
 *    bpmn-js 的 `import inherits from 'inherits-browser'` 直接抛进测试运行时，
 *    让所有引用本组件的测试全部无法启动（不是断言失败，是测试套件加载失败）。
 * 用 dynamic 后本组件在测试里可正常渲染，预览面独立测。
 */
const BPMNDiagramPreview = dynamic(() => import('@/components/workflow/BPMNDiagramPreview'), {
  ssr: false,
  loading: () => <Spin />,
});

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
  created: 'default',
  assigned: 'cyan',
  pending: 'gold',
  inProgress: 'blue',
  in_progress: 'blue',
  completed: 'green',
  failed: 'red',
  cancelled: 'gray',
  skipped: 'gray',
};

const taskStatusTextMap: Record<string, string> = {
  created: '已创建',
  pending: '待处理',
  assigned: '已分配',
  claimed: '已领取',
  inprogress: '处理中',
  completed: '已完成',
  failed: '已失败',
  cancelled: '已取消',
  skipped: '已跳过',
};

// 节点类型在任务表里可能是 user_task（后端）或 userTask（BPMN），统一归一化后映射中文。
const nodeTypeTextMap: Record<string, string> = {
  startevent: '开始事件',
  endevent: '结束事件',
  usertask: '用户任务',
  servicetask: '服务任务',
  scripttask: '脚本任务',
  manualtask: '人工任务',
  exclusivegateway: '排他网关',
  parallelgateway: '并行网关',
  inclusivegateway: '包容网关',
  eventbasedgateway: '事件网关',
  subprocess: '子流程',
  callactivity: '调用活动',
};

const normalizeType = (type?: string | null) => (type || '').replace(/[_-]/g, '').toLowerCase();

const getNodeTypeText = (type?: string | null) => {
  const normalized = normalizeType(type);
  return nodeTypeTextMap[normalized] ?? (type || '-');
};

// 审计动作来自后端枚举（PROCESS_STARTED 等），直接 replace 下划线仍是英文，
// 运维面板需要中文；未收录的动作保留原文（下划线转空格）作为确定性兜底。
const auditActionTextMap: Record<string, string> = {
  PROCESS_STARTED: '流程已启动',
  PROCESS_COMPLETED: '流程已完成',
  PROCESS_SUSPENDED: '流程已挂起',
  PROCESS_RESUMED: '流程已恢复',
  PROCESS_TERMINATED: '流程已终止',
  TASK_CREATED: '任务已创建',
  TASK_ASSIGNED: '任务已分配',
  TASK_COMPLETED: '任务已完成',
  TASK_FAILED: '任务失败',
  TASK_SKIPPED: '任务已跳过',
  VARIABLE_UPDATED: '变量已更新',
  GATEWAY_PASSED: '网关已通过',
  SEQUENCE_FLOW_TAKEN: '顺序流已执行',
};

const getAuditActionText = (action?: string | null) => {
  if (!action) return '-';
  return auditActionTextMap[action.toUpperCase()] ?? action.replace(/_/g, ' ');
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
  /**
   * 当前停留节点。
   *
   * 这两个字段是「我的单子走到哪了」唯一的权威来源（后端 ProcessInstance 的
   * currentActivityId / currentActivityName）。此前 toSummary 把它们丢弃，
   * 于是详情页只能靠任务表反推当前节点——任务表是历史，不是现状。
   */
  currentActivityId?: string;
  currentActivityName?: string;
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
  currentActivityId: instance.currentActivityId,
  currentActivityName: instance.currentActivityName,
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
  // 流程定义 XML 只在「流程进度」Tab 首次打开时拉取：定义体量大（prod 实测
  // ticket_general_flow 11.2KB），进详情页就加载会拖慢基本信息与任务表的首屏。
  const [definitionXml, setDefinitionXml] = useState<string | undefined>(undefined);
  const [definitionLoading, setDefinitionLoading] = useState(false);
  const [definitionError, setDefinitionError] = useState<string | undefined>(undefined);
  const definitionRequestedRef = useRef(false);

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

  /**
   * 懒加载流程定义 XML（供「流程进度」Tab 渲染 BPMN 图）。
   *
   * 用 getWorkflow(key) 而不是 getInstance 里的字段：实例接口不下发 XML。
   * 注意取的是「当前最新版本」的定义 —— 与实例实际运行时用的版本可能不同
   * （prod 实测实例 version=18 是实例自增序号，定义版本是 1.1.0，两套口径）。
   * 因此高亮靠节点 ID 匹配而非版本对齐：节点 ID 在定义里不变，改版后旧实例
   * 若引用了已删除的节点，BPMNDiagramPreview 会判掉不画高亮而不是崩图。
   */
  const loadDefinition = useCallback(async () => {
    if (definitionRequestedRef.current) return;
    const key = summary?.processDefinitionKey;
    if (!key || key === '-') {
      setDefinitionError('缺少流程定义 Key，无法渲染流程图');
      return;
    }
    definitionRequestedRef.current = true;
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
    } catch (error) {
      console.error('Failed to load process definition:', error);
      // 失败必须可重试：把 requested 复位，否则用户点了重试也不会再发请求。
      definitionRequestedRef.current = false;
      setDefinitionError('加载流程定义失败');
    } finally {
      setDefinitionLoading(false);
    }
  }, [summary?.processDefinitionKey]);

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
        render: (value: string) => getNodeTypeText(value),
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (value: string) => (
          <Tag color={taskStatusColorMap[value] || 'default'}>
            {taskStatusTextMap[normalizeType(value)] ?? value}
          </Tag>
        ),
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
        render: (value: string) => formatDateTime(value) || '-',
      },
      {
        title: '截止时间',
        dataIndex: 'dueDate',
        key: 'dueDate',
        width: 180,
        render: (value: string) => formatDateTime(value) || '-',
      },
    ],
    []
  );

  const businessKeyParsed = useMemo(
    () => parseBusinessKey(summary?.businessKey),
    [summary?.businessKey]
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
            <Descriptions.Item label="业务键">
              {/* 业务键是实例与业务单据之间唯一的关联锚点（后端写死 `type:id` 格式）。
                  只渲染文本的话，用户看完流程还得回列表页自己翻那张单，这里直接给一跳。 */}
              {businessKeyParsed.href ? (
                <a
                  href={businessKeyParsed.href}
                  className="inline-flex items-center gap-1"
                  data-testid="business-key-link"
                >
                  {businessKeyParsed.label}
                  <ExternalLink className="h-3 w-3" />
                </a>
              ) : (
                <span className={businessKeyParsed.rawType ? 'font-mono text-xs' : ''}>
                  {businessKeyParsed.label}
                </span>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={statusColorMap[summary.status] || 'default'}>
                {statusTextMap[summary.status] || summary.status}
              </Tag>
            </Descriptions.Item>
            {/* 当前节点排在状态之后：这两行合起来才完整回答「是否在跑 + 卡在哪」。
                终态实例后端不再下发该字段，此时显式显示 '-' 而不是留空。 */}
            <Descriptions.Item label="当前节点">
              {summary.currentActivityName || summary.currentActivityId ? (
                <Space size={4}>
                  <Tag color="orange">进行中</Tag>
                  <span>{summary.currentActivityName || summary.currentActivityId}</span>
                  {summary.currentActivityName &&
                    summary.currentActivityId &&
                    summary.currentActivityName === summary.currentActivityId && (
                      // 后端在节点无 name 时回退成原始 ID。透出 ID 便于运维去 XML 里
                      // 定位配置缺失，但用等宽弱化样式区分「这是标识符不是业务名称」。
                      <Typography.Text type="secondary" className="font-mono text-xs">
                        无节点名称
                      </Typography.Text>
                    )}
                </Space>
              ) : (
                <Typography.Text type="secondary">-（已结束或未进入任何节点）</Typography.Text>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="启动时间">{formatDateTime(summary.startTime) || '-'}</Descriptions.Item>
            <Descriptions.Item label="结束时间">{formatDateTime(summary.endTime) || '-'}</Descriptions.Item>
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
        key: 'progress',
        label: '流程进度',
        children: (
          <div>
            <div className="mb-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm">
              <span>
                <Typography.Text type="secondary">当前节点：</Typography.Text>
                <span className="font-medium">{summary?.currentActivityName || '-'}</span>
              </span>
              {tasks.length > 0 && (
                <span>
                  <Typography.Text type="secondary">已产生任务：</Typography.Text>
                  <span className="font-medium">{tasks.length}</span>
                </span>
              )}
              {auditLogs.length > 0 && (
                <span>
                  <Typography.Text type="secondary">执行记录：</Typography.Text>
                  <span className="font-medium">{auditLogs.length}</span>
                </span>
              )}
            </div>
            <LoadingEmptyError
              state={definitionLoading ? 'loading' : definitionError ? 'error' : 'success'}
              loadingText="正在加载流程定义..."
              error={{
                title: '流程图加载失败',
                description: definitionError || '未知错误',
                actionText: '重新加载',
                onAction: () => {
                  definitionRequestedRef.current = false;
                  void loadDefinition();
                },
              }}
            >
              {definitionXml ? (
                <BPMNDiagramPreview
                  xml={definitionXml}
                  height={420}
                  activeElementId={summary?.currentActivityId}
                  activeElementLabel={summary?.currentActivityName || summary?.currentActivityId}
                />
              ) : null}
            </LoadingEmptyError>
          </div>
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
                  const actionColor = auditActionColorMap[(log.action || '').toUpperCase()] || 'gray';
                  return {
                    color: actionColor,
                    children: (
                      <div className="mb-3" key={log.id ?? index}>
                        <div className="flex justify-between items-center mb-1">
                          <Space>
                            <Badge color={actionColor} />
                            <span className="font-medium text-sm">
                              {getAuditActionText(log.action)}
                            </span>
                            {log.activityName && <Tag>{log.activityName}</Tag>}
                          </Space>
                          <span className="text-xs text-gray-500">
                            {formatDateTime(log.timestamp) || '-'}
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
    [
      loading,
      summary,
      tasks,
      auditLogs,
      taskColumns,
      businessKeyParsed,
      definitionXml,
      definitionLoading,
      definitionError,
      loadDefinition,
    ]
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
      {/* 切到「流程进度」时才拉定义 XML（懒加载），避免进详情页就拖 11KB 定义体。 */}
      <Tabs
        activeKey={activeTab}
        onChange={next => {
          setActiveTab(next);
          if (next === 'progress') void loadDefinition();
        }}
        items={tabItems}
      />
    </div>
  );
};

export default WorkflowInstanceDetail;