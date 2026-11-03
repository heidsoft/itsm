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
  Alert,
  Modal,
} from 'antd';
import { ArrowLeft, XCircle } from 'lucide-react';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import ServiceRequestProvisioningPanel from '@/components/service-request/ServiceRequestProvisioningPanel';

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
  ciId?: number | null;
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

// 状态机允许直接取消的来源状态（后端 serviceRequestStatusTransitions）。
// 交付中的请求不在此列：资源已经在下单，取消只能靠交付失败后的人工判定。
const CANCELLABLE_STATUSES = new Set([
  'submitted',
  'manager_approved',
  'it_approved',
  'security_approved',
  'failed',
]);

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
  const [cancelOpen, setCancelOpen] = useState(false);
  const [cancelReason, setCancelReason] = useState('');
  const [cancelling, setCancelling] = useState(false);
  const approvals = useMemo(() => detail?.approvals ?? [], [detail]);
  const canCancel = detail ? CANCELLABLE_STATUSES.has(detail.status) : false;

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

  useEffect(() => {
    if (Number.isFinite(requestId) && requestId > 0) {
      load();
    }
  }, [requestId]);

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

  const doCancel = async () => {
    try {
      setCancelling(true);
      await ServiceCatalogApi.cancelServiceRequest(requestId, cancelReason.trim() || undefined);
      message.success('服务请求已取消');
      setCancelOpen(false);
      setCancelReason('');
      await load();
    } catch (e) {
      console.error(e);
      message.error(e instanceof Error ? e.message : '取消失败');
    } finally {
      setCancelling(false);
    }
  };

  return (
    <div className='max-w-4xl mx-auto p-6'>
      <Space orientation='vertical' size={16} style={{ width: '100%' }}>
        <Card>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              gap: 16,
              flexWrap: 'wrap',
            }}
          >
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
                <Text type='secondary'>查看审批记录与交付进度</Text>
              </div>
            </Space>
            {canCancel && (
              <Button danger icon={<XCircle />} onClick={() => setCancelOpen(true)}>
                取消请求
              </Button>
            )}
          </div>
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

          {/* 交付操作与任务列表 - 审批通过后显示 */}
          {detail &&
            ['security_approved', 'provisioning', 'delivered', 'failed'].includes(
              detail.status
            ) && (
              <>
                <Divider />
                <ServiceRequestProvisioningPanel
                  requestId={requestId}
                  status={detail.status}
                  onChanged={load}
                />
              </>
            )}
        </Card>

        <Modal
          open={cancelOpen}
          title='取消服务请求'
          onCancel={() => setCancelOpen(false)}
          onOk={doCancel}
          okText='确认取消'
          cancelText='返回'
          okButtonProps={{ danger: true }}
          confirmLoading={cancelling}
          destroyOnHidden
        >
          <Space orientation='vertical' size={12} style={{ width: '100%' }}>
            <Alert
              type='warning'
              showIcon
              title='取消后不可恢复'
              description='待审批记录会作废，关联流程实例会终止。如需继续，请重新提交服务请求。'
            />
            <div>
              <Text type='secondary'>取消原因（选填）</Text>
              <Input.TextArea
                rows={3}
                value={cancelReason}
                onChange={e => setCancelReason(e.target.value)}
                placeholder='例如：需求变更，改为申请更大的规格'
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
