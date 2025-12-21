'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import {
  Card,
  Typography,
  Space,
  Button,
  Descriptions,
  Tag,
  Divider,
  Input,
  message,
  Table,
  Alert,
} from 'antd';
import { ArrowLeftOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { serviceRequestAPI, ProvisioningTask } from '@/lib/api/service-request-api';

const { Title, Text } = Typography;

function statusTag(status: string) {
  const map: Record<string, { color: string; label: string }> = {
    submitted: { color: 'gold', label: '已提交' },
    manager_approved: { color: 'blue', label: '主管已批' },
    it_approved: { color: 'geekblue', label: 'IT已批' },
    security_approved: { color: 'green', label: '安全已批' },
    provisioning: { color: 'cyan', label: '交付中' },
    delivered: { color: 'green', label: '已交付' },
    failed: { color: 'red', label: '交付失败' },
    rejected: { color: 'red', label: '已拒绝' },
    cancelled: { color: 'default', label: '已取消' },
  };
  const cfg = map[status] || { color: 'default', label: status || '-' };
  return <Tag color={cfg.color}>{cfg.label}</Tag>;
}

export default function MyRequestDetailPage() {
  const params = useParams();
  const router = useRouter();
  const requestId = Number(params.requestId);

  const [loading, setLoading] = useState(true);
  const [detail, setDetail] = useState<any>(null);
  const [comment, setComment] = useState('');
  const [provisioningTasks, setProvisioningTasks] = useState<ProvisioningTask[]>([]);
  const [loadingTasks, setLoadingTasks] = useState(false);
  const [executing, setExecuting] = useState(false);
  const approvals = useMemo(() => (detail?.approvals || []) as any[], [detail]);

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
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
      <Space direction='vertical' size={16} style={{ width: '100%' }}>
        <Card>
          <Space align='center'>
            <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/my-requests')}>
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
              {detail?.current_level ?? '-'} / {detail?.total_levels ?? '-'}
            </Descriptions.Item>
            <Descriptions.Item label='标题' span={2}>
              {detail?.title || detail?.catalog?.name || '-'}
            </Descriptions.Item>
            <Descriptions.Item label='原因' span={2}>
              {detail?.reason || '-'}
            </Descriptions.Item>
            <Descriptions.Item label='数据分级'>
              {detail?.data_classification || '-'}
            </Descriptions.Item>
            <Descriptions.Item label='成本中心'>{detail?.cost_center || '-'}</Descriptions.Item>
          </Descriptions>

          <Divider />

          <Title level={4}>审批记录</Title>
          {approvals.length === 0 ? (
            <Text type='secondary'>暂无审批记录</Text>
          ) : (
            <Descriptions bordered column={1} size='small'>
              {approvals.map((a: any) => (
                <Descriptions.Item key={a.id} label={`L${a.level} · ${a.step}`}>
                  <Space wrap>
                    {statusTag(String(a.status))}
                    <Text type='secondary'>审批人：{a.approver_name || '-'}</Text>
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
              <Space direction='vertical' style={{ width: '100%' }}>
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
                message='审批已全部通过'
                description='可以启动交付任务，系统将自动创建云资源'
                type='success'
                showIcon
                style={{ marginBottom: 16 }}
              />
              <Button
                type='primary'
                icon={<PlayCircleOutlined />}
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
                  icon={<ReloadOutlined />}
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
                      dataIndex: 'resource_type',
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
                      title: '错误信息',
                      dataIndex: 'error_message',
                      ellipsis: true,
                      render: (text: string) => text || '-',
                    },
                    {
                      title: '创建时间',
                      dataIndex: 'created_at',
                      width: 180,
                      render: (v: string) => (v ? new Date(v).toLocaleString() : '-'),
                    },
                    {
                      title: '操作',
                      key: 'actions',
                      width: 120,
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
                      icon={<PlayCircleOutlined />}
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
      </Space>
    </div>
  );
}
