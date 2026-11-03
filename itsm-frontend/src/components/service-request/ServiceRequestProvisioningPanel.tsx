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
  pending: { color: 'default', label: '待执行' },
  running: { color: 'processing', label: '执行中' },
  succeeded: { color: 'success', label: '成功' },
  failed: { color: 'error', label: '失败' },
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
      message.error(e instanceof Error ? e.message : '加载交付任务失败');
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
      message.success('已启动交付任务');
      await refresh();
    } catch (e) {
      message.error(e instanceof Error ? e.message : '启动交付失败');
    } finally {
      setExecuting(false);
    }
  };

  const handleExecuteTask = async (taskId: number) => {
    try {
      setExecuting(true);
      await serviceRequestAPI.executeProvisioningTask(taskId);
      message.success('交付任务执行中');
      await refresh();
    } catch (e) {
      message.error(e instanceof Error ? e.message : '执行交付任务失败');
    } finally {
      setExecuting(false);
    }
  };

  // 重试只是重新排队，返回后任务为 pending，由队列 worker 领取执行。
  const handleRetryTask = async (taskId: number) => {
    try {
      setRetryingTaskId(taskId);
      const task = await serviceRequestAPI.retryProvisioningTask(taskId, '运维在详情页发起重试');
      message.success(task?.status === 'pending' ? '已重新排队，等待执行' : '已提交重试');
      await refresh();
    } catch (e) {
      message.error(e instanceof Error ? e.message : '重试交付任务失败');
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
      message.warning('请填写判定依据（工单号、资源 ID 或失败原因分析）');
      return;
    }
    try {
      setResolving(true);
      await serviceRequestAPI.resolveProvisioningTask(
        resolveTask.id,
        resolveOutcome,
        resolveNote.trim()
      );
      message.success('已提交人工判定');
      setResolveTask(null);
      setResolveNote('');
      await refresh();
    } catch (e) {
      message.error(e instanceof Error ? e.message : '人工判定失败');
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
          执行
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
              重试
            </Button>
          )}
          <Button type='link' size='small' onClick={() => openResolveModal(task)}>
            人工判定
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
          查看结果
        </Button>
      );
    }
    return '-';
  };

  return (
    <>
      {status === 'security_approved' && (
        <>
          <Title level={4}>交付操作</Title>
          <Alert
            title='审批已全部通过'
            description='可以启动交付任务，系统将自动创建云资源'
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
              启动交付
            </Button>
          )}
        </>
      )}

      {(status === 'provisioning' || status === 'delivered' || status === 'failed') && (
        <>
          <Title level={4}>
            交付任务
            <Button
              type='link'
              icon={<RotateCcw />}
              onClick={loadTasks}
              loading={loadingTasks}
              size='small'
            >
              刷新
            </Button>
          </Title>
          {tasks.length === 0 ? (
            <Text type='secondary'>暂无交付任务</Text>
          ) : (
            <Table<ProvisioningTask>
              rowKey='id'
              loading={loadingTasks}
              dataSource={tasks}
              pagination={false}
              size='small'
              columns={[
                { title: 'ID', dataIndex: 'id', width: 80 },
                { title: '资源类型', dataIndex: 'resourceType', width: 120 },
                {
                  title: '状态',
                  dataIndex: 'status',
                  width: 100,
                  render: (value: string) => taskStatusTag(value),
                },
                { title: '提供商', dataIndex: 'provider', width: 100 },
                {
                  // 尝试次数决定「还能不能重试」，按服务端下发的 maxAttempts 判断。
                  title: '尝试',
                  key: 'attempts',
                  width: 90,
                  render: (_, task) => `${task.attemptCount ?? 0}/${task.maxAttempts ?? 5}`,
                },
                {
                  title: '错误信息',
                  dataIndex: 'errorMessage',
                  ellipsis: true,
                  render: (value: string, task) =>
                    value || (task.resolutionNote ? `人工判定：${task.resolutionNote}` : '-'),
                },
                {
                  title: '创建时间',
                  dataIndex: 'createdAt',
                  width: 180,
                  render: (value: string) => formatDateTime(value) || '-',
                },
                {
                  title: '操作',
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
                  执行待处理任务
                </Button>
              </div>
            )}
        </>
      )}

      <Modal
        open={resolveTask !== null}
        title='人工判定交付结果'
        onCancel={() => setResolveTask(null)}
        onOk={handleResolveTask}
        okText='提交判定'
        cancelText='取消'
        confirmLoading={resolving}
        destroyOnHidden
      >
        <Space orientation='vertical' size={12} style={{ width: '100%' }}>
          <Alert
            type='warning'
            showIcon
            title='仅在自动重试无法成功时使用'
            description='凭据吊销、配额封禁或资源已在平台外建好时，请在此给出终态判定。判定会写入服务请求状态，并记录判定人与依据。'
          />
          <div>
            <Text type='secondary'>判定结果</Text>
            <div style={{ marginTop: 8 }}>
              <Radio.Group
                value={resolveOutcome}
                onChange={e => setResolveOutcome(e.target.value)}
              >
                <Radio.Button value='delivered'>已交付</Radio.Button>
                <Radio.Button value='failed'>交付失败</Radio.Button>
              </Radio.Group>
            </div>
          </div>
          <div>
            <Text type='secondary'>判定依据（必填）</Text>
            <Input.TextArea
              rows={3}
              value={resolveNote}
              onChange={e => setResolveNote(e.target.value)}
              placeholder='例如：资源已在控制台手工创建 i-abc123，凭据吊销无法走自动交付'
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
