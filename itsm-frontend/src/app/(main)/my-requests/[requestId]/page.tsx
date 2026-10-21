'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import {
  App,
  Card,
  Typography,
  Space,
  Button,
  Descriptions,
  Tag,
  Divider,
  Input,
  Table,
  Alert,
  Modal,
  Radio,
} from 'antd';
import { ArrowLeft, RotateCcw, PlayCircle } from 'lucide-react';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import type { ProvisioningTask } from '@/lib/api/service-request-api';
import { serviceRequestAPI } from '@/lib/api/service-request-api';

interface ApprovalRecord {
  id: number;
  level: number;
  step: string;
  status: string;
  approverName?: string;
  comment?: string;
}

interface ServiceRequestDetail {
  status: string;
  currentLevel?: number;
  totalLevels?: number;
  ciId?: number;
  title?: string;
  reason?: string;
  dataClassification?: string;
  costCenter?: string;
  catalog?: { name?: string };
  approvals?: ApprovalRecord[];
}

const { Title, Text } = Typography;

// 服务请求状态（snake_case，对应 backend enum）
// 兼容旧 camelCase 数据，避免历史脏数据回退到英文原文。
const REQUEST_STATUS_LABEL: Record<string, { color: string; label: string }> = {
  submitted: { color: 'gold', label: '已提交' },
  manager_approved: { color: 'blue', label: '主管已批' },
  managerApproved: { color: 'blue', label: '主管已批' },
  it_approved: { color: 'geekblue', label: 'IT已批' },
  itApproved: { color: 'geekblue', label: 'IT已批' },
  security_approved: { color: 'green', label: '安全已批' },
  securityApproved: { color: 'green', label: '安全已批' },
  provisioning: { color: 'cyan', label: '交付中' },
  delivered: { color: 'green', label: '已交付' },
  failed: { color: 'red', label: '交付失败' },
  rejected: { color: 'red', label: '已拒绝' },
  cancelled: { color: 'default', label: '已取消' },
};

// 审批步骤英文 → 中文（manager / it / security）
const APPROVAL_STEP_LABEL: Record<string, string> = {
  manager: '主管',
  it: 'IT',
  security: '安全',
};

// 审批记录状态英文 → 中文（pending / approved / rejected）
const APPROVAL_RECORD_STATUS_LABEL: Record<string, string> = {
  pending: '待审批',
  approved: '已通过',
  rejected: '已拒绝',
  delegated: '已委派',
  timeout: '已超时',
};

function statusTag(status: string) {
  const cfg = REQUEST_STATUS_LABEL[status] || { color: 'default', label: status || '-' };
  return <Tag color={cfg.color}>{cfg.label}</Tag>;
}

export default function MyRequestDetailPage() {
  const { message } = App.useApp();
  const params = useParams();
  const router = useRouter();
  const requestId = Number(params.requestId);

  const [loading, setLoading] = useState(true);
  const [detail, setDetail] = useState<ServiceRequestDetail | null>(null);
  const [comment, setComment] = useState('');
  const [provisioningTasks, setProvisioningTasks] = useState<ProvisioningTask[]>([]);
  const [loadingTasks, setLoadingTasks] = useState(false);
  const [executing, setExecuting] = useState(false);
  // 交付失败恢复：重试（重新排队）与人工判定（自动重试走不通时的出口）。
  const [retryingTaskId, setRetryingTaskId] = useState<number | null>(null);
  const [resolveTask, setResolveTask] = useState<ProvisioningTask | null>(null);
  const [resolveOutcome, setResolveOutcome] = useState<'delivered' | 'failed'>('delivered');
  const [resolveNote, setResolveNote] = useState('');
  const [resolving, setResolving] = useState(false);
  const approvals = useMemo(() => detail?.approvals ?? [], [detail]);

  const load = async () => {
    try {
      setLoading(true);
      const data = await ServiceCatalogApi.getServiceRequest(requestId);
      setDetail(data);
    } catch (e) {
      console.error(e);
      message.error(e instanceof Error ? e.message : '加载失败');
    } finally {
      setLoading(false);
    }
  };

  const loadProvisioningTasks = async () => {
    try {
      setLoadingTasks(true);
      const tasks = await serviceRequestAPI.listProvisioningTasks(requestId);
      setProvisioningTasks(tasks || []);
    } catch (e) {
      console.error(e);
      // 如果还没有任务，不显示错误
      if (
        detail?.status === 'security_approved' ||
        detail?.status === 'provisioning' ||
        detail?.status === 'delivered'
      ) {
        setProvisioningTasks([]);
      }
    } finally {
      setLoadingTasks(false);
    }
  };

  useEffect(() => {
    if (Number.isFinite(requestId) && requestId > 0) {
      load();
    }
  }, [requestId]);

  useEffect(() => {
    if (
      detail &&
      (detail.status === 'security_approved' ||
        detail.status === 'provisioning' ||
        detail.status === 'delivered' ||
        detail.status === 'failed')
    ) {
      loadProvisioningTasks();
    }
  }, [detail]);

  const doApprove = async () => {
    try {
      await ServiceCatalogApi.approveServiceRequest(requestId, comment || undefined);
      message.success('已提交审批动作（通过）');
      setComment('');
      await load();
    } catch (e) {
      console.error(e);
      message.error(e instanceof Error ? e.message : '操作失败');
    }
  };

  const doReject = async () => {
    try {
      if (!comment.trim()) {
        message.warning('请输入拒绝原因/审批意见');
        return;
      }
      await ServiceCatalogApi.rejectServiceRequest(requestId, comment.trim());
      message.success('已提交审批动作（拒绝）');
      setComment('');
      await load();
    } catch (e) {
      console.error(e);
      message.error(e instanceof Error ? e.message : '操作失败');
    }
  };

  const handleStartProvisioning = async () => {
    try {
      setExecuting(true);
      await serviceRequestAPI.startProvisioning(requestId);
      message.success('已启动交付任务');
      await load();
      await loadProvisioningTasks();
    } catch (e) {
      console.error(e);
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
      await loadProvisioningTasks();
      await load();
    } catch (e) {
      console.error(e);
      message.error(e instanceof Error ? e.message : '执行交付任务失败');
    } finally {
      setExecuting(false);
    }
  };

  // 重试 = 重新排队，不是立刻执行：返回后任务为 pending，由队列 worker 领取。
  const handleRetryTask = async (taskId: number) => {
    try {
      setRetryingTaskId(taskId);
      const task = await serviceRequestAPI.retryProvisioningTask(taskId, '运维在详情页发起重试');
      message.success(
        task?.status === 'pending' ? '已重新排队，等待执行' : '已提交重试'
      );
      await loadProvisioningTasks();
      await load();
    } catch (e) {
      console.error(e);
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
      await loadProvisioningTasks();
      await load();
    } catch (e) {
      console.error(e);
      message.error(e instanceof Error ? e.message : '人工判定失败');
    } finally {
      setResolving(false);
    }
  };

  const taskStatusTag = (status: string) => {
    const map: Record<string, { color: string; label: string }> = {
      pending: { color: 'default', label: '待执行' },
      running: { color: 'processing', label: '执行中' },
      succeeded: { color: 'success', label: '成功' },
      failed: { color: 'error', label: '失败' },
    };
    const cfg = map[status] || { color: 'default', label: status || '-' };
    return <Tag color={cfg.color}>{cfg.label}</Tag>;
  };

  return (
    <div className='max-w-4xl mx-auto p-6'>
      <Space orientation='vertical' size={16} style={{ width: '100%' }}>
        <Card>
          <Space align='center'>
            <Button
              icon={<ArrowLeft />}
              onClick={() => {
                try {
                  router.push('/my-requests');
                } catch (navErr) {
                  console.error('[nav]', navErr);
                  message.error('导航失败，请稍后重试');
                }
              }}
            >
              返回
            </Button>
            <div>
              <Title level={2} style={{ marginBottom: 0 }}>
                请求详情 #{requestId}
              </Title>
              <Text type='secondary'>最小可用：展示状态与审批记录，并支持 approve/reject</Text>
            </div>
          </Space>
        </Card>

        <Card loading={loading}>
          <Descriptions bordered column={2} size='middle'>
            <Descriptions.Item label='状态'>
              {statusTag(String(detail?.status || ''))}
            </Descriptions.Item>
            <Descriptions.Item label='当前级别'>
              {detail?.currentLevel ?? '-'} / {detail?.totalLevels ?? '-'}
            </Descriptions.Item>
            <Descriptions.Item label='关联CI'>
              {detail?.ciId ? (
                <Button
                  type='link'
                  onClick={() => {
                    try {
                      router.push(`/cmdb/cis/${detail.ciId}`);
                    } catch (navErr) {
                      console.error('[nav]', navErr);
                      message.error('导航失败，请稍后重试');
                    }
                  }}
                >
                  CI #{detail.ciId}
                </Button>
              ) : (
                '-'
              )}
            </Descriptions.Item>
            <Descriptions.Item label='标题' span={2}>
              {detail?.title || detail?.catalog?.name || '-'}
            </Descriptions.Item>
            <Descriptions.Item label='原因' span={2}>
              {detail?.reason || '-'}
            </Descriptions.Item>
            <Descriptions.Item label='数据分级'>
              {detail?.dataClassification || '-'}
            </Descriptions.Item>
            <Descriptions.Item label='成本中心'>{detail?.costCenter || '-'}</Descriptions.Item>
          </Descriptions>

          <Divider />

          <Title level={4}>审批记录</Title>
          {approvals.length === 0 ? (
            <Text type='secondary'>暂无审批记录</Text>
          ) : (
            <Descriptions bordered column={1} size='small'>
              {approvals.map((a) => (
                <Descriptions.Item
                  key={a.id}
                  label={`L${a.level} · ${APPROVAL_STEP_LABEL[a.step] || a.step}`}
                >
                  <Space wrap>
                    {a.status === 'pending' ||
                    a.status === 'approved' ||
                    a.status === 'rejected' ? (
                      <Tag
                        color={
                          a.status === 'approved'
                            ? 'green'
                            : a.status === 'rejected'
                              ? 'red'
                              : 'orange'
                        }
                      >
                        {APPROVAL_RECORD_STATUS_LABEL[a.status] || a.status}
                      </Tag>
                    ) : (
                      statusTag(String(a.status))
                    )}
                    <Text type='secondary'>审批人：{a.approverName || '-'}</Text>
                    <Text type='secondary'>意见：{a.comment || '-'}</Text>
                  </Space>
                </Descriptions.Item>
              ))}
            </Descriptions>
          )}

          <Divider />

          {/* 审批操作 - 仅在待审批状态显示 */}
          {(detail?.status === 'submitted' ||
            detail?.status === 'manager_approved' ||
            detail?.status === 'it_approved') && (
            <>
              <Title level={4}>审批操作</Title>
              <Space orientation='vertical' style={{ width: '100%' }}>
                <Input.TextArea
                  value={comment}
                  onChange={e => setComment(e.target.value)}
                  rows={3}
                  placeholder='审批意见（拒绝时必填）'
                />
                <Space>
                  <Button type='primary' onClick={doApprove}>
                    通过
                  </Button>
                  <Button danger onClick={doReject}>
                    拒绝
                  </Button>
                </Space>
              </Space>
              <Divider />
            </>
          )}

          {/* 交付操作 - 安全审批通过后显示 */}
          {detail?.status === 'security_approved' && (
            <>
              <Title level={4}>交付操作</Title>
              <Alert
                title='审批已全部通过'
                description='可以启动交付任务，系统将自动创建云资源'
                type='success'
                showIcon
                style={{ marginBottom: 16 }}
              />
              <Button
                type='primary'
                icon={<PlayCircle />}
                onClick={handleStartProvisioning}
                loading={executing}
                size='large'
              >
                启动交付
              </Button>
              <Divider />
            </>
          )}

          {/* 交付任务列表 */}
          {(detail?.status === 'provisioning' ||
            detail?.status === 'delivered' ||
            detail?.status === 'failed') && (
            <>
              <Title level={4}>
                交付任务
                <Button
                  type='link'
                  icon={<RotateCcw />}
                  onClick={loadProvisioningTasks}
                  loading={loadingTasks}
                  size='small'
                >
                  刷新
                </Button>
              </Title>
              {provisioningTasks.length === 0 ? (
                <Text type='secondary'>暂无交付任务</Text>
              ) : (
                <Table<ProvisioningTask>
                  rowKey='id'
                  loading={loadingTasks}
                  dataSource={provisioningTasks}
                  pagination={false}
                  size='small'
                  columns={[
                    {
                      title: 'ID',
                      dataIndex: 'id',
                      width: 80,
                    },
                    {
                      title: '资源类型',
                      dataIndex: 'resourceType',
                      width: 120,
                    },
                    {
                      title: '状态',
                      dataIndex: 'status',
                      width: 100,
                      render: (status: string) => taskStatusTag(status),
                    },
                    {
                      title: '提供商',
                      dataIndex: 'provider',
                      width: 100,
                    },
                    {
                      // 尝试次数直接决定「还能不能重试」：前端不猜，按服务端下发的
                      // maxAttempts 判断，避免按钮点了才报 409。
                      title: '尝试',
                      key: 'attempts',
                      width: 90,
                      render: (_, task) =>
                        `${task.attemptCount ?? 0}/${task.maxAttempts ?? 5}`,
                    },
                    {
                      title: '错误信息',
                      dataIndex: 'errorMessage',
                      ellipsis: true,
                      render: (text: string, task) =>
                        text || (task.resolutionNote ? `人工判定：${task.resolutionNote}` : '-'),
                    },
                    {
                      title: '创建时间',
                      dataIndex: 'createdAt',
                      width: 180,
                      render: (v: string) => (v ? new Date(v).toLocaleString() : '-'),
                    },
                    {
                      title: '操作',
                      key: 'actions',
                      width: 180,
                      render: (_, task) => (
                        <Space>
                          {task.status === 'pending' && (
                            <Button
                              type='link'
                              size='small'
                              onClick={() => handleExecuteTask(task.id)}
                              loading={executing}
                            >
                              执行
                            </Button>
                          )}
                          {task.status === 'failed' && (
                            <>
                              {/* 达到上限后重试按钮消失，只留人工判定——继续显示
                                  一个注定 409 的按钮比不显示更糟。 */}
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
                              <Button
                                type='link'
                                size='small'
                                onClick={() => openResolveModal(task)}
                              >
                                人工判定
                              </Button>
                            </>
                          )}
                          {task.status === 'succeeded' && task.result && (
                            <Button
                              type='link'
                              size='small'
                              onClick={() => {
                                message.info(JSON.stringify(task.result, null, 2));
                              }}
                            >
                              查看结果
                            </Button>
                          )}
                        </Space>
                      ),
                    },
                  ]}
                />
              )}
              {detail?.status === 'provisioning' &&
                provisioningTasks.some(t => t.status === 'pending') && (
                  <div style={{ marginTop: 16 }}>
                    <Button
                      type='primary'
                      icon={<PlayCircle />}
                      onClick={() => {
                        const pendingTask = provisioningTasks.find(t => t.status === 'pending');
                        if (pendingTask) {
                          handleExecuteTask(pendingTask.id);
                        }
                      }}
                      loading={executing}
                    >
                      执行待处理任务
                    </Button>
                  </div>
                )}
            </>
          )}
        </Card>

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
              message='仅在自动重试无法成功时使用'
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
      </Space>
    </div>
  );
}
