'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { App, Button, Card, Descriptions, Form, Input, Modal, Select, Space, Table, Tag, Tabs, Timeline, Empty, Badge } from 'antd';
import { Eye, PauseCircle, PlayCircle, RefreshCw, StopCircle, Clock, User, FileText, MessageSquare, Rocket } from 'lucide-react';

import { FilterToolbarCard } from '@/components/ui/FilterToolbarCard';
import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { ManagementNotice, ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview } from '@/components/ui/StatsOverview';
import { WorkflowApi } from '@/lib/api/workflow-api';
import BPMNDashboardApi from '@/lib/api/bpmn-dashboard-api';
import type { NodeInstance, WorkflowDefinition } from '@/types/workflow';
import type { ProcessAuditLog } from '@/lib/api/bpmn-dashboard-api';

type InstanceRow = {
  id: string;
  businessKey: string;
  processDefinitionKey: string;
  status: string;
  startTime?: string;
  endTime?: string;
};

const formatDateTime = (value?: string | Date): string => {
  if (!value) return '-';
  const date = typeof value === 'string' ? new Date(value) : value;
  return date.toLocaleString('zh-CN');
};

const formatDuration = (ms: number): string => {
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

// 工作流实例状态文本映射
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
  'PROCESS_STARTED': 'green',
  'PROCESS_COMPLETED': 'blue',
  'PROCESS_SUSPENDED': 'orange',
  'PROCESS_RESUMED': 'green',
  'PROCESS_TERMINATED': 'red',
  'TASK_CREATED': 'blue',
  'TASK_ASSIGNED': 'cyan',
  'TASK_COMPLETED': 'green',
  'TASK_FAILED': 'red',
  'TASK_SKIPPED': 'gray',
  'VARIABLE_UPDATED': 'purple',
  'GATEWAY_PASSED': 'orange',
  'SEQUENCE_FLOW_TAKEN': 'cyan',
};

export default function WorkflowInstancesPage() {
  const { message, modal } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [instances, setInstances] = useState<InstanceRow[]>([]);
  const [stats, setStats] = useState({
    total: 0,
    running: 0,
    completed: 0,
    suspended: 0,
    terminated: 0,
  });
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState<string | undefined>();
  
  // 详情弹窗状态
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedInstance, setSelectedInstance] = useState<InstanceRow | null>(null);
  const [tasks, setTasks] = useState<NodeInstance[]>([]);
  const [auditLogs, setAuditLogs] = useState<ProcessAuditLog[]>([]);
  const [detailLoading, setDetailLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('basic');

  // 发起流程弹窗状态
  const [startModalVisible, setStartModalVisible] = useState(false);
  const [startSubmitting, setStartSubmitting] = useState(false);
  const [definitions, setDefinitions] = useState<WorkflowDefinition[]>([]);
  const [definitionsLoading, setDefinitionsLoading] = useState(false);
  const [startForm] = Form.useForm();

  const loadData = async () => {
    try {
      setLoading(true);
      const [instanceResponse, statsResponse] = await Promise.all([
        WorkflowApi.getInstances({
          workflowId: keyword || undefined,
          status,
          page: 1,
          pageSize: 50,
        }),
        WorkflowApi.getInstanceStats({
          processDefinitionKey: keyword || undefined,
          status,
        }),
      ]);

      const rows = (instanceResponse.instances || []).map(instance => ({
        id: instance.id,
        businessKey:
          (instance as unknown as Record<string, string>).businessKey ||
          instance.workflowId ||
          '-',
        processDefinitionKey: instance.workflowId || '-',
        status: String(instance.status),
        startTime: instance.startTime?.toISOString(),
        endTime: instance.endTime?.toISOString(),
      }));

      setInstances(rows);
      setStats(statsResponse);
    } catch (error) {
      console.error('Failed to load workflow instances:', error);
      message.error('加载工作流实例失败');
      setInstances([]);
    } finally {
      setLoading(false);
    }
  };

  const loadInstanceDetail = async (instanceId: string) => {
    try {
      setDetailLoading(true);
      const [tasksRes, timelineRes] = await Promise.all([
        WorkflowApi.getNodeInstances(instanceId),
        BPMNDashboardApi.getProcessTimeline(instanceId).catch(() => []),
      ]);
      
      setTasks(tasksRes || []);
      setAuditLogs(timelineRes || []);
    } catch (error) {
      console.error('Failed to load instance detail:', error);
      message.error('加载实例详情失败');
      setTasks([]);
      setAuditLogs([]);
    } finally {
      setDetailLoading(false);
    }
  };

  const handleViewDetail = (record: InstanceRow) => {
    setSelectedInstance(record);
    setDetailModalVisible(true);
    setActiveTab('basic');
    loadInstanceDetail(record.id);
  };

  // 打开发起流程弹窗：加载已激活的流程定义供选择
  const openStartModal = async () => {
    setStartModalVisible(true);
    setDefinitionsLoading(true);
    try {
      const { workflows } = await WorkflowApi.getWorkflows({ page: 1, pageSize: 100 });
      setDefinitions(workflows.filter(w => String(w.status) === 'active'));
    } catch {
      message.error('加载流程定义失败');
      setDefinitions([]);
    } finally {
      setDefinitionsLoading(false);
    }
  };

  // 发起流程实例：选择已激活的流程定义，填写业务键与变量后调用 startWorkflow
  const handleStartSubmit = async () => {
    try {
      const values = await startForm.validateFields();
      let variables: Record<string, unknown> | undefined;
      if (values.variables) {
        try {
          variables = JSON.parse(values.variables);
        } catch {
          message.error('流程变量必须是合法的 JSON 对象');
          return;
        }
      }
      setStartSubmitting(true);
      const instance = await WorkflowApi.startWorkflow({
        workflowId: values.processDefinitionKey,
        businessKey: values.businessKey || undefined,
        variables: variables as Record<string, unknown> | undefined,
      });
      message.success(`流程实例已启动：${instance.id}`);
      setStartModalVisible(false);
      startForm.resetFields();
      loadData();
    } catch (error) {
      // validateFields 失败时不提示；API 失败提示错误
      if (error instanceof Error) {
        message.error(`启动流程失败：${error.message}`);
      }
    } finally {
      setStartSubmitting(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [keyword, status]);

  const columns = useMemo(
    () => [
      {
        title: '实例 ID',
        dataIndex: 'id',
        key: 'id',
        render: (value: string) => <span className="font-mono text-xs">{value}</span>,
        width: 180,
      },
      {
        title: '流程 Key',
        dataIndex: 'processDefinitionKey',
        key: 'processDefinitionKey',
        width: 150,
      },
      {
        title: '业务键',
        dataIndex: 'businessKey',
        key: 'businessKey',
        width: 150,
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (value: string) => <Tag color={statusColorMap[value] || 'default'}>{statusTextMap[value] || value}</Tag>,
      },
      {
        title: '启动时间',
        dataIndex: 'startTime',
        key: 'startTime',
        width: 180,
        render: (value: string) => formatDateTime(value),
      },
      {
        title: '结束时间',
        dataIndex: 'endTime',
        key: 'endTime',
        width: 180,
        render: (value: string) => formatDateTime(value),
      },
      {
        title: '操作',
        key: 'actions',
        width: 200,
        render: (_: unknown, record: InstanceRow) => (
          <Space size="small">
            <Button 
              type="text" 
              icon={<Eye className="h-4 w-4" />} 
              onClick={() => handleViewDetail(record)}
              size="small"
            >
              详情
            </Button>
            {record.status === 'running' && (
              <>
                <Button
                  type="text"
                  icon={<PauseCircle className="h-4 w-4" />}
                  onClick={() => {
                    modal.confirm({
                      title: '暂停流程实例',
                      content: `确定要暂停实例 ${record.id} 吗？暂停后待办任务将不可处理，可随时恢复。`,
                      okText: '暂停',
                      cancelText: '取消',
                      onOk: async () => {
                        try {
                          await WorkflowApi.suspendWorkflow(record.id);
                          message.success('实例已暂停');
                          loadData();
                        } catch {
                          message.error('暂停实例失败');
                        }
                      },
                    });
                  }}
                  size="small"
                >
                  暂停
                </Button>
                <Button
                  type="text"
                  danger
                  icon={<StopCircle className="h-4 w-4" />}
                  onClick={() => {
                    modal.confirm({
                      title: '终止流程实例',
                      content: `确定要终止实例 ${record.id} 吗？终止后流程不可恢复，未完成的审批链将被中断。`,
                      okText: '终止',
                      okType: 'danger',
                      cancelText: '取消',
                      onOk: async () => {
                        try {
                          await WorkflowApi.terminateWorkflow(record.id, '前端终止');
                          message.success('实例已终止');
                          loadData();
                        } catch {
                          message.error('终止实例失败');
                        }
                      },
                    });
                  }}
                  size="small"
                >
                  终止
                </Button>
              </>
            )}
            {record.status === 'suspended' && (
              <Button
                type="text"
                icon={<PlayCircle className="h-4 w-4" />}
                onClick={() => {
                  modal.confirm({
                    title: '恢复流程实例',
                    content: `确定要恢复实例 ${record.id} 吗？恢复后流程将继续执行。`,
                    okText: '恢复',
                    cancelText: '取消',
                    onOk: async () => {
                      try {
                        await WorkflowApi.resumeWorkflow(record.id);
                        message.success('实例已恢复');
                        loadData();
                      } catch {
                        message.error('恢复实例失败');
                      }
                    },
                  });
                }}
                size="small"
              >
                恢复
              </Button>
            )}
          </Space>
        ),
      },
    ],
    [message, modal]
  );

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
        render: (value: string) => value.replace('_', ' '),
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
        render: (value: string) => value ? formatDateTime(value) : '-',
      },
    ],
    []
  );

  const tabItems = [
    {
      key: 'basic',
      label: '基本信息',
      children: selectedInstance && (
        <Descriptions column={1} bordered size="small">
          <Descriptions.Item label="实例 ID">
            <span className="font-mono text-xs">{selectedInstance.id}</span>
          </Descriptions.Item>
          <Descriptions.Item label="流程 Key">{selectedInstance.processDefinitionKey}</Descriptions.Item>
          <Descriptions.Item label="业务键">{selectedInstance.businessKey}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <Tag color={statusColorMap[selectedInstance.status] || 'default'}>
              {statusTextMap[selectedInstance.status] || selectedInstance.status}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="启动时间">{formatDateTime(selectedInstance.startTime)}</Descriptions.Item>
          <Descriptions.Item label="结束时间">{formatDateTime(selectedInstance.endTime)}</Descriptions.Item>
          {selectedInstance.startTime && (
            <Descriptions.Item label="持续时间">
              {selectedInstance.endTime 
                ? formatDuration(new Date(selectedInstance.endTime).getTime() - new Date(selectedInstance.startTime).getTime())
                : formatDuration(Date.now() - new Date(selectedInstance.startTime).getTime())
              }
            </Descriptions.Item>
          )}
        </Descriptions>
      ),
    },
    {
      key: 'tasks',
      label: `任务列表 (${tasks.length})`,
      children: (
        <LoadingEmptyError
          state={detailLoading ? 'loading' : tasks.length === 0 ? 'empty' : 'success'}
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
          state={detailLoading ? 'loading' : auditLogs.length === 0 ? 'empty' : 'success'}
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
                    <div className="mb-3">
                      <div className="flex justify-between items-center mb-1">
                        <Space>
                          <Badge color={actionColor} />
                          <span className="font-medium text-sm">
                            {log.action.replace('_', ' ')}
                          </span>
                          {log.activityName && (
                            <Tag>{log.activityName}</Tag>
                          )}
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
                        {log.durationMs && (
                          <span>
                            <Clock className="w-3 h-3 inline mr-1" />
                            耗时: {formatDuration(log.durationMs)}
                          </span>
                        )}
                      </div>

                      {log.comment && (
                        <div className="text-xs bg-gray-50 p-2 rounded ml-5 mb-1">
                          <MessageSquare className="w-3 h-3 inline mr-1 text-gray-400" />
                          {log.comment}
                        </div>
                      )}

                      {(log.variablesAfter && Object.keys(log.variablesAfter).length > 0) && (
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
  ];

  return (
    <div className="space-y-6">
      <ManagementPageHeader
        title="工作流实例"
        description="统一查看流程实例状态、生命周期和关键时间点，先保证实例页可用和可筛选。"
        notice={
          <ManagementNotice
            message="实例页已恢复"
            description="这一版先收口列表、状态操作和详情查看，后续会继续补充任务列表与会签视图。"
          />
        }
      />

      <StatsOverview
        items={[
          { key: 'total', title: '总实例', value: stats.total, accentColor: '#1677ff' },
          { key: 'running', title: '运行中', value: stats.running, accentColor: '#52c41a' },
          { key: 'completed', title: '已完成', value: stats.completed, accentColor: '#1677ff' },
          { key: 'suspended', title: '已暂停', value: stats.suspended, accentColor: '#faad14' },
        ]}
      />

      <FilterToolbarCard
        filters={
          <>
            <Input
              placeholder="按流程 Key 搜索"
              value={keyword}
              allowClear
              onChange={event => setKeyword(event.target.value)}
              style={{ width: 250 }}
            />
            <Select
              placeholder="状态筛选"
              allowClear
              value={status}
              onChange={setStatus}
              style={{ width: 180 }}
              options={[
                { label: '运行中', value: 'running' },
                { label: '已完成', value: 'completed' },
                { label: '已暂停', value: 'suspended' },
                { label: '已终止', value: 'terminated' },
                { label: '失败', value: 'failed' },
              ]}
            />
          </>
        }
        actions={
          <Space>
            <Button type="primary" icon={<Rocket className="h-4 w-4" />} onClick={openStartModal}>
              发起流程
            </Button>
            <Button icon={<RefreshCw className="h-4 w-4" />} onClick={loadData}>
              刷新
            </Button>
          </Space>
        }
      />

      <Card className="rounded-xl shadow-sm">
        <LoadingEmptyError
          state={loading ? 'loading' : instances.length === 0 ? 'empty' : 'success'}
          loadingText="正在加载工作流实例..."
          empty={{
            title: '暂无工作流实例',
            description: '当前筛选条件下没有实例数据。',
            actionText: '重新加载',
            onAction: loadData,
          }}
        >
          <Table 
            columns={columns} 
            dataSource={instances} 
            rowKey="id" 
            pagination={{ pageSize: 10 }}
            scroll={{ x: 1140 }}
          />
        </LoadingEmptyError>
      </Card>

      <Modal
        title="发起流程实例"
        open={startModalVisible}
        onCancel={() => {
          setStartModalVisible(false);
          startForm.resetFields();
        }}
        onOk={handleStartSubmit}
        confirmLoading={startSubmitting}
        okText="启动"
        cancelText="取消"
        destroyOnHidden
      >
        <Form form={startForm} layout="vertical">
          <Form.Item
            name="processDefinitionKey"
            label="流程定义"
            rules={[{ required: true, message: '请选择要发起的流程定义' }]}
          >
            <Select
              placeholder="选择已激活的流程定义"
              showSearch
              optionFilterProp="label"
              loading={definitionsLoading}
              options={definitions.map(w => ({ label: `${w.name} (${w.code})`, value: w.code }))}
              notFoundContent="没有已激活的流程定义，请先部署/激活"
            />
          </Form.Item>
          <Form.Item
            name="businessKey"
            label="业务键（可选）"
            tooltip="关联的业务单据标识，如工单号、变更号；留空将自动生成"
          >
            <Input placeholder="例如 TICKET-2026-0001" />
          </Form.Item>
          <Form.Item
            name="variables"
            label="流程变量（可选，JSON）"
            tooltip='以 JSON 对象形式传入启动变量，例如 {"priority": "high"}'
          >
            <Input.TextArea rows={4} placeholder='{"priority": "high"}' />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={selectedInstance ? `实例详情 · ${selectedInstance.id}` : '实例详情'}
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={null}
        destroyOnHidden
        width={900}
      >
        <Tabs 
          activeKey={activeTab} 
          onChange={setActiveTab}
          items={tabItems}
        />
      </Modal>
    </div>
  );
}
