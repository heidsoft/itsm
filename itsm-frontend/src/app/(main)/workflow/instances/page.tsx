'use client';

import React, { Suspense, useEffect, useMemo, useState } from 'react';
import { App, Button, Card, Form, Input, Modal, Select, Space, Table, Tag } from 'antd';
import { Eye, PauseCircle, PlayCircle, RefreshCw, StopCircle, Rocket } from 'lucide-react';
import { useSearchParams } from 'next/navigation';

import { FilterToolbarCard } from '@/components/ui/FilterToolbarCard';
import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { ManagementNotice, ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview } from '@/components/ui/StatsOverview';
import { WorkflowApi } from '@/lib/api/workflow-api';
import { WorkflowInstanceDetail } from '@/components/workflow/WorkflowInstanceDetail';
import type { WorkflowDefinition } from '@/types/workflow';
import { formatDateTime } from '@/lib/formatters';

type InstanceRow = {
  id: string;
  businessKey: string;
  processDefinitionKey: string;
  status: string;
  startTime?: string;
  endTime?: string;
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
  running: 'Running',
  completed: 'Completed',
  suspended: 'Paused',
  terminated: 'Terminated',
  failed: 'Failed',
};

export default function WorkflowInstancesPage() {
  return (
    <Suspense fallback={<div className="p-6 text-center text-gray-500">Loading…</div>}>
      <WorkflowInstancesContent />
    </Suspense>
  );
}

function WorkflowInstancesContent() {
  const { message, modal } = App.useApp();
  const searchParams = useSearchParams();
  const urlWorkflowId = searchParams.get('workflowId') || searchParams.get('processDefinitionKey') || '';
  // 「查看流程实例」从业务详情跳转时会带 businessKey（如 ticket:42）+ instanceId，
  // 后端 GET /bpmn/process-instances 支持 businessKey 精确过滤，用它收敛到该单据的实例。
  const urlBusinessKey = searchParams.get('businessKey') || '';
  const urlInstanceId = searchParams.get('instanceId') || '';
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染空列表。
  const [listLoading, setListLoading] = useState(true);
  const [instances, setInstances] = useState<InstanceRow[]>([]);
  const [stats, setStats] = useState({
    total: 0,
    running: 0,
    completed: 0,
    suspended: 0,
    terminated: 0,
  });
  const [keyword, setKeyword] = useState(urlWorkflowId);
  const [status, setStatus] = useState<string | undefined>();

  // URL 参数变化时（例如从流程设计器跳转过来）同步 keyword 并重新查询
  useEffect(() => {
    if (urlWorkflowId && urlWorkflowId !== keyword) {
      setKeyword(urlWorkflowId);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [urlWorkflowId]);

  // 详情弹窗状态
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedInstanceId, setSelectedInstanceId] = useState<string | null>(null);
  const [selectedInstanceSummary, setSelectedInstanceSummary] = useState<InstanceRow | null>(null);

  // 发起流程弹窗状态
  const [startModalVisible, setStartModalVisible] = useState(false);
  const [startSubmitting, setStartSubmitting] = useState(false);
  const [definitions, setDefinitions] = useState<WorkflowDefinition[]>([]);
  const [definitionsLoading, setDefinitionsLoading] = useState(false);
  const [startForm] = Form.useForm();

  const loadData = async () => {
    try {
      setListLoading(true);
      const [instanceResponse, statsResponse] = await Promise.all([
        WorkflowApi.getInstances({
          workflowId: keyword || undefined,
          businessKey: urlBusinessKey || undefined,
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
        // 业务键缺失时明确显示 '-',不得用流程 Key 冒充业务键。
        businessKey: instance.businessKey || '-',
        processDefinitionKey: instance.workflowId || '-',
        status: String(instance.status),
        startTime: instance.startTime?.toISOString(),
        endTime: instance.endTime?.toISOString(),
      }));

      setInstances(rows);
      setStats(statsResponse);
    } catch (error) {
      console.error('Failed to load workflow instances:', error);
      message.error('Failed to load workflow instance');
      setInstances([]);
    } finally {
      setListLoading(false);
    }
  };

  const handleViewDetail = (record: InstanceRow) => {
    setSelectedInstanceId(record.id);
    setSelectedInstanceSummary(record);
    setDetailModalVisible(true);
  };

  // 打开发起流程弹窗：加载已激活的流程定义供选择
  const openStartModal = async () => {
    setStartModalVisible(true);
    setDefinitionsLoading(true);
    try {
      const { workflows } = await WorkflowApi.getWorkflows({ page: 1, pageSize: 100 });
      setDefinitions(workflows.filter(w => String(w.status) === 'active'));
    } catch {
      message.error('Failed to load process definition');
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
          message.error('Process variables must be valid JSON objects');
          return;
        }
      }
      setStartSubmitting(true);
      const instance = await WorkflowApi.startWorkflow({
        workflowId: values.processDefinitionKey,
        businessKey: values.businessKey || undefined,
        variables: variables as Record<string, unknown> | undefined,
      });
      message.success(`Process instance started:${instance.id}`);
      setStartModalVisible(false);
      startForm.resetFields();
      loadData();
    } catch (error) {
      // validateFields 失败时不提示；API 失败提示错误
      if (error instanceof Error) {
        message.error(`Failed to start process:${error.message}`);
      }
    } finally {
      setStartSubmitting(false);
    }
  };

  useEffect(() => {
    loadData();
  }, [keyword, status, urlBusinessKey]);

  // 从业务详情带 instanceId 跳转时直接打开该实例详情，而不是只落在实例列表。
  useEffect(() => {
    if (!urlInstanceId) return;
    setSelectedInstanceId(urlInstanceId);
    setSelectedInstanceSummary(null);
    setDetailModalVisible(true);
  }, [urlInstanceId]);

  const columns = useMemo(
    () => [
      {
        title: 'Instance ID',
        dataIndex: 'id',
        key: 'id',
        render: (value: string) => <span className="font-mono text-xs">{value}</span>,
        width: 180,
      },
      {
        title: 'Process Key',
        dataIndex: 'processDefinitionKey',
        key: 'processDefinitionKey',
        width: 150,
      },
      {
        title: 'Business key',
        dataIndex: 'businessKey',
        key: 'businessKey',
        width: 150,
      },
      {
        title: 'Status',
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (value: string) => <Tag color={statusColorMap[value] || 'default'}>{statusTextMap[value] || value}</Tag>,
      },
      {
        title: 'Startup time',
        dataIndex: 'startTime',
        key: 'startTime',
        width: 180,
        render: (value: string) => formatDateTime(value) || '-',
      },
      {
        title: 'End Time',
        dataIndex: 'endTime',
        key: 'endTime',
        width: 180,
        render: (value: string) => formatDateTime(value) || '-',
      },
      {
        title: 'Actions',
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

              Details
            </Button>
            {record.status === 'running' && (
              <>
                <Button
                  type="text"
                  icon={<PauseCircle className="h-4 w-4" />}
                  onClick={() => {
                    modal.confirm({
                      title: 'Pause process instance',
                      content: `Are you sure you want to pause the instance? ${record.id} ? To-do tasks cannot be processed after being paused and can be resumed at any time.`,
                      okText: 'On Hold',
                      cancelText: 'Cancel',
                      onOk: async () => {
                        try {
                          await WorkflowApi.suspendWorkflow(record.id);
                          message.success('Instance paused');
                          loadData();
                        } catch (error) {
                          // 后端会给出可行动的原因（如「当前状态为 completed，不能暂停」），
                          // 统一文案会把它吞掉，操作员只能反复点同一个按钮。
                          message.error(`Failed to suspend instance: ${error instanceof Error ? error.message : 'Please try again later'}`);
                        }
                      },
                    });
                  }}
                  size="small"
                >

                  On Hold
                </Button>
                <Button
                  type="text"
                  danger
                  icon={<StopCircle className="h-4 w-4" />}
                  onClick={() => {
                    modal.confirm({
                      title: 'Terminating process instance',
                      content: `Are you sure you want to terminate the instance? ${record.id} ? After termination, the process cannot be resumed and the unfinished approval chain will be interrupted.`,
                      okText: 'Termination',
                      okType: 'danger',
                      cancelText: 'Cancel',
                      onOk: async () => {
                        try {
                          await WorkflowApi.terminateWorkflow(record.id, 'Frontend terminated');
                          message.success('Instance terminated');
                          loadData();
                        } catch (error) {
                          message.error(`Failed to terminate instance: ${error instanceof Error ? error.message : 'Please try again later'}`);
                        }
                      },
                    });
                  }}
                  size="small"
                >

                  Termination
                </Button>
              </>
            )}
            {record.status === 'suspended' && (
              <Button
                type="text"
                icon={<PlayCircle className="h-4 w-4" />}
                onClick={() => {
                  modal.confirm({
                    title: 'Recovery process instance',
                    content: `Are you sure you want to restore the instance? ${record.id} ? The process will continue after recovery.`,
                    okText: 'Restore',
                    cancelText: 'Cancel',
                    onOk: async () => {
                      try {
                        await WorkflowApi.resumeWorkflow(record.id);
                        message.success('Instance restored');
                        loadData();
                      } catch (error) {
                        message.error(`Failed to restore instance: ${error instanceof Error ? error.message : 'Please try again later'}`);
                      }
                    },
                  });
                }}
                size="small"
              >

                Restore
              </Button>
            )}
          </Space>
        ),
      },
    ],
    [message, modal]
  );

  return (
    <div className="space-y-6">
      <ManagementPageHeader
        title="Workflow Instances"
        description="View the running status, start and end time, and duration of a process instance, and support filtering by status and process."
        notice={
          <ManagementNotice
            message="This version"
            description="Supports instance lists, status actions (suspend/resume/terminate), and details including node tasks and execution history. Countersign views will be added in a future release."
          />
        }
      />

      <StatsOverview
        items={[
          { key: 'total', title: 'Total instances', value: stats.total, accentColor: '#1677ff' },
          { key: 'running', title: 'Running', value: stats.running, accentColor: '#52c41a' },
          { key: 'completed', title: 'Completed', value: stats.completed, accentColor: '#1677ff' },
          { key: 'suspended', title: 'Paused', value: stats.suspended, accentColor: '#faad14' },
        ]}
      />

      <FilterToolbarCard
        filters={
          <>
            <Input
              placeholder="Search by process Key"
              value={keyword}
              allowClear
              onChange={event => setKeyword(event.target.value)}
              style={{ width: 250 }}
            />
            <Select
              placeholder="Status"
              allowClear
              value={status}
              onChange={setStatus}
              style={{ width: 180 }}
              options={[
                { label: 'Running', value: 'running' },
                { label: 'Completed', value: 'completed' },
                { label: 'Suspended', value: 'suspended' },
                { label: 'Terminated', value: 'terminated' },
              ]}
            />
          </>
        }
        actions={
          <Space>
            <Button type="primary" icon={<Rocket className="h-4 w-4" />} onClick={openStartModal}>

              Initiate process
            </Button>
            <Button icon={<RefreshCw className="h-4 w-4" />} onClick={loadData}>

              Refresh
            </Button>
          </Space>
        }
      />

      <Card className="rounded-xl shadow-sm">
        <LoadingEmptyError
          state={listLoading ? 'loading' : instances.length === 0 ? 'empty' : 'success'}
          loadingText="Loading workflow instances..."
          empty={{
            title: 'No workflow instance yet',
            description: 'There is no instance data under the current filter criteria.',
            actionText: 'Reload',
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
        title="Initiate process instance"
        open={startModalVisible}
        onCancel={() => {
          setStartModalVisible(false);
          startForm.resetFields();
        }}
        onOk={handleStartSubmit}
        confirmLoading={startSubmitting}
        okText="Start"
        cancelText="Cancel"
        destroyOnHidden
      >
        <Form form={startForm} layout="vertical">
          <Form.Item
            name="processDefinitionKey"
            label="Process Definitions"
            rules={[{ required: true, message: 'Please select the process definition to initiate' }]}
          >
            <Select
              placeholder="Select an activated process definition"
              showSearch
              optionFilterProp="label"
              loading={definitionsLoading}
              options={definitions.map(w => ({ label: `${w.name} (${w.code})`, value: w.code }))}
              notFoundContent="No activated process definition, please deploy/activate first"
            />
          </Form.Item>
          <Form.Item
            name="businessKey"
            label="Business key (optional)"
            tooltip="associated business document identification, such as work order number, change number; leaving blank will be automatically generated"
          >
            <Input placeholder="For example TICKET-2026-0001" />
          </Form.Item>
          <Form.Item
            name="variables"
            label="Process variables (optional, JSON)"
            tooltip="Pass startup variables as JSON objects, for example {&quot;priority&quot;: &quot;high&quot;}"
          >
            <Input.TextArea rows={4} placeholder='{"priority": "high"}' />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={selectedInstanceSummary ? `Instance details · ${selectedInstanceSummary.id}` : 'Instance details'}
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={null}
        destroyOnHidden
        width={900}
      >
        {selectedInstanceId && (
          <WorkflowInstanceDetail
            instanceId={selectedInstanceId}
            initialInstance={selectedInstanceSummary ? {
              id: selectedInstanceSummary.id,
              processDefinitionKey: selectedInstanceSummary.processDefinitionKey,
              businessKey: selectedInstanceSummary.businessKey,
              status: selectedInstanceSummary.status,
              startTime: selectedInstanceSummary.startTime,
              endTime: selectedInstanceSummary.endTime,
            } : undefined}
          />
        )}
      </Modal>
    </div>
  );
}