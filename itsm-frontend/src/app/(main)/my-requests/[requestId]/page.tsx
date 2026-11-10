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
  /**
   * 后端判定的「当前用户是否具备当前审批节点的审批资格」。
   *
   * 不能只按请求状态渲染审批按钮：审批人集合在审批行的 node.approver_ids 里，
   * 前端拿不到，此前只按状态渲染，导致发起人自己、以及不在本级审批人集合内的
   * 管理员都会看到「通过/拒绝」，点下去必然失败（403/500）。
   */
  canApprove?: boolean;
}

const { Title, Text } = Typography;

// 服务请求状态（snake_case，对应 backend enum）
// 兼容旧 camelCase 数据，避免历史脏数据回退到英文原文。
const REQUEST_STATUS_LABEL: Record<string, { color: string; label: string }> = {
  submitted: { color: 'gold', label: 'Submitted' },
  manager_approved: { color: 'blue', label: 'Manager Approved' },
  managerApproved: { color: 'blue', label: 'Manager Approved' },
  it_approved: { color: 'geekblue', label: 'IT Approved' },
  itApproved: { color: 'geekblue', label: 'IT Approved' },
  security_approved: { color: 'green', label: 'Security Approved' },
  securityApproved: { color: 'green', label: 'Security Approved' },
  provisioning: { color: 'cyan', label: 'Provisioning' },
  delivered: { color: 'green', label: 'Delivered' },
  failed: { color: 'red', label: 'Failed' },
  rejected: { color: 'red', label: 'Rejected' },
  cancelled: { color: 'default', label: 'Cancelled' },
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
  manager: 'Supervisor',
  it: 'IT',
  security: 'Safe',
};

// 审批记录状态英文 → 中文（pending / approved / rejected）
const APPROVAL_RECORD_STATUS_LABEL: Record<string, string> = {
  pending: 'Pending Approval',
  approved: 'Approved',
  rejected: 'Rejected',
  delegated: 'Delegated',
  timeout: 'Timed out',
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
      message.error(e instanceof Error ? e.message : 'LoadingFailed');
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
      message.success('Approval action submitted (passed)');
      setComment('');
      await load();
    } catch (e) {
      console.error(e);
      message.error(e instanceof Error ? e.message : 'Operation failed');
    }
  };

  const doReject = async () => {
    try {
      if (!comment.trim()) {
        message.warning('Please enter the reason for rejection/approval opinion');
        return;
      }
      await ServiceCatalogApi.rejectServiceRequest(requestId, comment.trim());
      message.success('Approval action submitted (rejected)');
      setComment('');
      await load();
    } catch (e) {
      console.error(e);
      message.error(e instanceof Error ? e.message : 'Operation failed');
    }
  };

  const doCancel = async () => {
    try {
      setCancelling(true);
      await ServiceCatalogApi.cancelServiceRequest(requestId, cancelReason.trim() || undefined);
      message.success('Service request canceled');
      setCancelOpen(false);
      setCancelReason('');
      await load();
    } catch (e) {
      console.error(e);
      message.error(e instanceof Error ? e.message : 'Cancellation failed');
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
                    message.error('Navigation failed, please try again later');
                  }
                }}
              >

                Back
              </Button>
              <div>
                <Title level={2} style={{ marginBottom: 0 }}>

                  Request details #{requestId}
                </Title>
                <Text type='secondary'>View approval records and delivery progress</Text>
              </div>
            </Space>
            {canCancel && (
              <Button danger icon={<XCircle />} onClick={() => setCancelOpen(true)}>

                Cancel request
              </Button>
            )}
          </div>
        </Card>

        <Card loading={loading}>
          <Descriptions bordered column={2} size='middle'>
            <Descriptions.Item label="Status">
              {statusTag(String(detail?.status || ''))}
            </Descriptions.Item>
            <Descriptions.Item label="Current level">
              {detail?.currentLevel ?? '-'} / {detail?.totalLevels ?? '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Associated CI">
              {detail?.ciId ? (
                <Button
                  type='link'
                  onClick={() => {
                    try {
                      router.push(`/cmdb/cis/${detail.ciId}`);
                    } catch (navErr) {
                      console.error('[nav]', navErr);
                      message.error('Navigation failed, please try again later');
                    }
                  }}
                >
                  CI #{detail.ciId}
                </Button>
              ) : (
                '-'
              )}
            </Descriptions.Item>
            <Descriptions.Item label="Title" span={2}>
              {detail?.title || detail?.catalog?.name || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Reason" span={2}>
              {detail?.reason || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Data classification">
              {detail?.dataClassification || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Cost Center">{detail?.costCenter || '-'}</Descriptions.Item>
          </Descriptions>

          <Divider />

          <Title level={4}>Approval Records</Title>
          {approvals.length === 0 ? (
            <Text type='secondary'>No approval records</Text>
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
                    <Text type='secondary'>Approver:{a.approverName || '-'}</Text>
                    <Text type='secondary'>Comments:{a.comment || '-'}</Text>
                  </Space>
                </Descriptions.Item>
              ))}
            </Descriptions>
          )}

          <Divider />

          {/* 审批操作 - 仅在待审批状态且当前用户具备本级审批资格时显示 */}
          {detail?.canApprove === true &&
            (detail?.status === 'submitted' ||
              detail?.status === 'manager_approved' ||
              detail?.status === 'it_approved') && (
            <>
              <Title level={4}>Approval operation</Title>
              <Space orientation='vertical' style={{ width: '100%' }}>
                <Input.TextArea
                  value={comment}
                  onChange={e => setComment(e.target.value)}
                  rows={3}
                  placeholder="Approval opinion (required when rejecting)"
                />
                <Space>
                  <Button type='primary' onClick={doApprove}>

                    Approve
                  </Button>
                  <Button danger onClick={doReject}>

                    Rejected
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
          title="Cancel service request"
          onCancel={() => setCancelOpen(false)}
          onOk={doCancel}
          okText="Confirm cancellation"
          cancelText="Back"
          okButtonProps={{ danger: true }}
          confirmLoading={cancelling}
          destroyOnHidden
        >
          <Space orientation='vertical' size={12} style={{ width: '100%' }}>
            <Alert
              type='warning'
              showIcon
              title="Cannot be restored after cancellation"
              description="The record to be approved will be invalidated and the associated process instance will be terminated. To continue, please resubmit your service request."
            />
            <div>
              <Text type='secondary'>Reason for cancellation (optional)</Text>
              <Input.TextArea
                rows={3}
                value={cancelReason}
                onChange={e => setCancelReason(e.target.value)}
                placeholder="For example: requirements change, apply for larger specifications instead"
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
