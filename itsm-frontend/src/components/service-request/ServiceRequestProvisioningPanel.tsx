'use client';

// 我的请求与管理端详情共用一份交付入口，避免两套 UI 各自漂移。

import React, { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  App,
  Button,
  Input,
  Modal,
  Radio,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import { PlayCircle, RotateCcw } from 'lucide-react';

import type { ProvisioningTask } from '@/lib/api/service-request-api';
import { serviceRequestAPI } from '@/lib/api/service-request-api';
import { usePermissions } from '@/lib/hooks/use-permissions';
import { formatDateTime } from '@/lib/formatters';

const { Title, Text } = Typography;

const DELIVERY_STATUSES = new Set(['security_approved', 'provisioning', 'delivered', 'failed']);

const TASK_STATUS: Record<string, { color: string; label: string }> = {
  pending: { color: 'default', label: 'Pending execution' },
  running: { color: 'processing', label: 'Running' },
  succeeded: { color: 'success', label: 'Success' },
  failed: { color: 'error', label: 'Failed' },
};

function taskStatusTag(status: string) {
  const cfg = TASK_STATUS[status] || { color: 'default', label: status || '-' };
  return <Tag color={cfg.color}>{cfg.label}</Tag>;
}

interface ServiceRequestProvisioningPanelProps {
  requestId: number;
  status: string;
  /** 动作完成后回调，让详情页重新读取服务请求状态。 */
  onChanged?: () => void | Promise<void>;
}

const ServiceRequestProvisioningPanel: React.FC<ServiceRequestProvisioningPanelProps> = ({
  requestId,
  status,
  onChanged,
}) => {
  const { message } = App.useApp();
  const { hasPermission } = usePermissions();
  // 启动/执行/重试/判定都会真实创建云资源，权限口径与后端路由一致。
  const canProvision = hasPermission('service_request', 'provision');

  const [tasks, setTasks] = useState<ProvisioningTask[]>([]);
  const [loadingTasks, setLoadingTasks] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [retryingTaskId, setRetryingTaskId] = useState<number | null>(null);
  const [resolveTask, setResolveTask] = useState<ProvisioningTask | null>(null);
  const [resolveOutcome, setResolveOutcome] = useState<'delivered' | 'failed'>('delivered');
  const [resolveNote, setResolveNote] = useState('');
  const [resolving, setResolving] = useState(false);

  const visible = DELIVERY_STATUSES.has(status);

  const loadTasks = useCallback(async () => {
    try {
      setLoadingTasks(true);
      setTasks((await serviceRequestAPI.listProvisioningTasks(requestId)) || []);
    } catch (e) {
      setTasks([]);
      message.error(e instanceof Error ? e.message : 'Failed to load delivery task');
    } finally {
      setLoadingTasks(false);
    }
  }, [requestId, message]);

  const refresh = useCallback(async () => {
    await loadTasks();
    await onChanged?.();
  }, [loadTasks, onChanged]);

  useEffect(() => {
    if (!visible) return;
    loadTasks();
  }, [visible, status, loadTasks]);

  const handleStartProvisioning = async () => {
    try {
      setExecuting(true);
      await serviceRequestAPI.startProvisioning(requestId);
      message.success('Delivery task started');
      await refresh();
    } catch (e) {
      message.error(e instanceof Error ? e.message : 'Failed to initiate delivery');
    } finally {
      setExecuting(false);
    }
  };

  const handleExecuteTask = async (taskId: number) => {
    try {
      setExecuting(true);
      await serviceRequestAPI.executeProvisioningTask(taskId);
      message.success('Delivery task in progress');
      await refresh();
    } catch (e) {
      message.error(e instanceof Error ? e.message : 'Failed to execute delivery task');
    } finally {
      setExecuting(false);
    }
  };

  // 重试只是重新排队，返回后任务为 pending，由队列 worker 领取执行。
  const handleRetryTask = async (taskId: number) => {
    try {
      setRetryingTaskId(taskId);
      const task = await serviceRequestAPI.retryProvisioningTask(taskId, 'Operation and maintenance initiates a retry on the details page');
      message.success(task?.status === 'pending' ? 'Requeued, waiting for execution' : 'Submitted for retry');
      await refresh();
    } catch (e) {
      message.error(e instanceof Error ? e.message : 'Retry delivery task failed');
    } finally {
      setRetryingTaskId(null);
    }
  };

  const openResolveModal = (task: ProvisioningTask) => {
    setResolveTask(task);
    setResolveOutcome('delivered');
    setResolveNote('');
  };

  const handleResolveTask = async () => {
    if (!resolveTask) return;
    if (!resolveNote.trim()) {
      message.warning('Please fill in the basis for judgment (work order number, resource ID or failure reason analysis)');
      return;
    }
    try {
      setResolving(true);
      await serviceRequestAPI.resolveProvisioningTask(
        resolveTask.id,
        resolveOutcome,
        resolveNote.trim()
      );
      message.success('Manual determination submitted');
      setResolveTask(null);
      setResolveNote('');
      await refresh();
    } catch (e) {
      message.error(e instanceof Error ? e.message : 'Manual judgment failed');
    } finally {
      setResolving(false);
    }
  };

  if (!visible) return null;

  const renderActions = (task: ProvisioningTask) => {
    if (!canProvision) return '-';
    if (task.status === 'pending') {
      return (
        <Button
          type='link'
          size='small'
          onClick={() => handleExecuteTask(task.id)}
          loading={executing}
        >

          Execute
        </Button>
      );
    }
    if (task.status === 'failed') {
      return (
        <Space>
          {/* 达到上限后重试按钮消失，只留人工判定——继续显示一个注定 409 的按钮更糟。 */}
          {(task.attemptCount ?? 0) < (task.maxAttempts ?? 5) && (
            <Button
              type='link'
              size='small'
              onClick={() => handleRetryTask(task.id)}
              loading={retryingTaskId === task.id}
            >

              Retry
            </Button>
          )}
          <Button type='link' size='small' onClick={() => openResolveModal(task)}>

            Manual judgment
          </Button>
        </Space>
      );
    }
    if (task.status === 'succeeded' && task.result) {
      return (
        <Button
          type='link'
          size='small'
          onClick={() => message.info(JSON.stringify(task.result, null, 2))}
        >

          View results
        </Button>
      );
    }
    return '-';
  };

  return (
    <>
      {status === 'security_approved' && (
        <>
          <Title level={4}>Delivery operation</Title>
          <Alert
            title="All approvals passed"
            description="The delivery task can be started and the system will automatically create cloud resources"
            type='success'
            showIcon
            style={{ marginBottom: 16 }}
          />
          {canProvision && (
            <Button
              type='primary'
              icon={<PlayCircle />}
              onClick={handleStartProvisioning}
              loading={executing}
              size='large'
            >

              Start delivery
            </Button>
          )}
        </>
      )}

      {(status === 'provisioning' || status === 'delivered' || status === 'failed') && (
        <>
          <Title level={4}>

            Delivery task
            <Button
              type='link'
              icon={<RotateCcw />}
              onClick={loadTasks}
              loading={loadingTasks}
              size='small'
            >

              Refresh
            </Button>
          </Title>
          {tasks.length === 0 ? (
            <Text type='secondary'>No delivery tasks yet</Text>
          ) : (
            <Table<ProvisioningTask>
              rowKey='id'
              loading={loadingTasks}
              dataSource={tasks}
              pagination={false}
              size='small'
              columns={[
                { title: 'ID', dataIndex: 'id', width: 80 },
                { title: 'Resource Type', dataIndex: 'resourceType', width: 120 },
                {
                  title: 'Status',
                  dataIndex: 'status',
                  width: 100,
                  render: (value: string) => taskStatusTag(value),
                },
                { title: 'Provider', dataIndex: 'provider', width: 100 },
                {
                  // 尝试次数决定「还能不能重试」，按服务端下发的 maxAttempts 判断。
                  title: 'Try',
                  key: 'attempts',
                  width: 90,
                  render: (_, task) => `${task.attemptCount ?? 0}/${task.maxAttempts ?? 5}`,
                },
                {
                  title: 'Error message',
                  dataIndex: 'errorMessage',
                  ellipsis: true,
                  render: (value: string, task) =>
                    value || (task.resolutionNote ? `Manual judgment:${task.resolutionNote}` : '-'),
                },
                {
                  title: 'Created',
                  dataIndex: 'createdAt',
                  width: 180,
                  render: (value: string) => formatDateTime(value) || '-',
                },
                {
                  title: 'Actions',
                  key: 'actions',
                  width: 180,
                  render: (_, task) => renderActions(task),
                },
              ]}
            />
          )}
          {status === 'provisioning' &&
            canProvision &&
            tasks.some(task => task.status === 'pending') && (
              <div style={{ marginTop: 16 }}>
                <Button
                  type='primary'
                  icon={<PlayCircle />}
                  onClick={() => {
                    const pendingTask = tasks.find(task => task.status === 'pending');
                    if (pendingTask) handleExecuteTask(pendingTask.id);
                  }}
                  loading={executing}
                >

                  Execute pending tasks
                </Button>
              </div>
            )}
        </>
      )}

      <Modal
        open={resolveTask !== null}
        title="Manual judgment of delivery results"
        onCancel={() => setResolveTask(null)}
        onOk={handleResolveTask}
        okText="Submit decision"
        cancelText="Cancel"
        confirmLoading={resolving}
        destroyOnHidden
      >
        <Space orientation='vertical' size={12} style={{ width: '100%' }}>
          <Alert
            type='warning'
            showIcon
            title="Only used if automatic retry fails"
            description="When credentials are revoked, quotas are banned, or resources have been built outside the platform, please give the final status here. The decision will be written into the service request status, and the person and basis for the decision will be recorded."
          />
          <div>
            <Text type='secondary'>Judgment result</Text>
            <div style={{ marginTop: 8 }}>
              <Radio.Group
                value={resolveOutcome}
                onChange={e => setResolveOutcome(e.target.value)}
              >
                <Radio.Button value='delivered'>Delivered</Radio.Button>
                <Radio.Button value='failed'>Failed</Radio.Button>
              </Radio.Group>
            </div>
          </div>
          <div>
            <Text type='secondary'>Judgment basis (required)</Text>
            <Input.TextArea
              rows={3}
              value={resolveNote}
              onChange={e => setResolveNote(e.target.value)}
              placeholder="For example: the resource i-abc123 has been manually created in the console, and automatic delivery cannot be performed if the credentials are revoked."
              maxLength={500}
              showCount
              style={{ marginTop: 8 }}
            />
          </div>
        </Space>
      </Modal>
    </>
  );
};

export default ServiceRequestProvisioningPanel;
