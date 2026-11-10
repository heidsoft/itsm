'use client';

/**
 * 服务请求详情组件
 */

import React, { useCallback, useState, useEffect } from 'react';
import {
  App,
  Card,
  Descriptions,
  Tag,
  Timeline,
  Button,
  Typography,
  Space,
  Modal,
  Input,
  Spin,
  Divider,
} from 'antd';
import type { TimelineItemProps } from 'antd';
import { CheckCircle, XCircle, Timer } from 'lucide-react';
import { useParams } from 'next/navigation';
import dayjs from 'dayjs';

import { serviceCategoryLabel } from '@/constants/service-catalog';
import { ServiceRequestApi } from '@/lib/api/';
import { ServiceRequestStatus, ApprovalStatus, ApprovalAction, serviceRequestStatusLabel } from '@/constants/service-request';
import type { ServiceRequest, ServiceRequestApproval } from '@/lib/api/service-request-api';
import { useI18n } from '@/lib/i18n/useI18n';
import ServiceRequestProvisioningPanel from './ServiceRequestProvisioningPanel';

const { Title, Text } = Typography;
const { TextArea } = Input;

// 审批状态颜色映射
const approvalStatusColors: Record<string, string> = {
  [ApprovalStatus.PENDING]: 'orange',
  [ApprovalStatus.APPROVED]: 'green',
  [ApprovalStatus.REJECTED]: 'red',
};

// 审批状态 i18n key 映射（与 src/lib/i18n/translations.ts 对齐）
const approvalStatusLabelKeys: Record<string, string> = {
  [ApprovalStatus.PENDING]: 'detailTabs.approvalStatusPending',
  [ApprovalStatus.APPROVED]: 'detailTabs.approvalStatusApproved',
  [ApprovalStatus.REJECTED]: 'detailTabs.approvalStatusRejected',
};

// 状态机允许直接取消的来源状态（后端 serviceRequestStatusTransitions）。
const CANCELLABLE_STATUSES = new Set<string>([
  ServiceRequestStatus.SUBMITTED,
  ServiceRequestStatus.MANAGER_APPROVED,
  ServiceRequestStatus.IT_APPROVED,
  ServiceRequestStatus.SECURITY_APPROVED,
  ServiceRequestStatus.FAILED,
]);

const ServiceRequestDetail: React.FC = () => {
  const { message } = App.useApp();
  const { id } = useParams() as { id: string };
  const { t } = useI18n();
  // /service-requests/new 这类静态段会落进 [id]，Number("new") 是 NaN，
  // 直接拿它请求就是 GET /service-requests/NaN。非正整数一律按未找到处理。
  const requestId = Number(id);
  const hasValidId = Number.isInteger(requestId) && requestId > 0;
  const [loading, setLoading] = useState(false);
  const [request, setRequest] = useState<ServiceRequest | null>(null);
  const [approvals, setApprovals] = useState<ServiceRequestApproval[]>([]);

  // 审批动作弹窗状态
  const [actionModalVisible, setActionModalVisible] = useState(false);
  const [currentAction, setCurrentAction] = useState<ApprovalAction | null>(null);
  const [comment, setComment] = useState('');
  const [submitting, setSubmitting] = useState(false);

  // 取消弹窗状态
  const [cancelModalOpen, setCancelModalOpen] = useState(false);
  const [cancelReason, setCancelReason] = useState('');
  const [cancelling, setCancelling] = useState(false);

  const loadDetail = useCallback(async () => {
    if (!hasValidId) return;
    setLoading(true);
    try {
      const data = await ServiceRequestApi.getServiceRequest(requestId);
      setRequest(data);
      setApprovals(data.approvals ?? []);
    } catch {
      message.error('Failed to load details');
    } finally {
      setLoading(false);
    }
  }, [hasValidId, requestId, message]);

  useEffect(() => {
    loadDetail();
  }, [loadDetail]);

  // 提交审批
  const handleSubmitApproval = async () => {
    if (!hasValidId || !currentAction) return;

    if (currentAction === ApprovalAction.REJECT && !comment.trim()) {
      message.error('The reason for rejecting the operation must be filled in');
      return;
    }

    setSubmitting(true);
    try {
      await ServiceRequestApi.applyApproval(
        requestId,
        currentAction === ApprovalAction.APPROVE ? 'approve' : 'reject',
        comment
      );
      message.success('Operation succeeded');
      setActionModalVisible(false);
      loadDetail(); // 刷新数据
    } catch (error: unknown) {
      // console.error(error);
      message.error(error instanceof Error ? error.message : 'Operation failed');
    } finally {
      setSubmitting(false);
    }
  };

  const openActionModal = (action: ApprovalAction) => {
    setCurrentAction(action);
    setComment('');
    setActionModalVisible(true);
  };

  const handleCancelRequest = async () => {
    if (!hasValidId) return;
    setCancelling(true);
    try {
      await ServiceRequestApi.updateServiceRequestStatus(
        requestId,
        ServiceRequestStatus.CANCELLED,
        cancelReason.trim() || undefined
      );
      message.success('Service request canceled');
      setCancelModalOpen(false);
      setCancelReason('');
      await loadDetail();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Cancellation failed');
    } finally {
      setCancelling(false);
    }
  };

  const canCancel = request ? CANCELLABLE_STATUSES.has(request.status) : false;

  // 渲染审批时间轴
  const renderApprovalTimeline = () => {
    const items: TimelineItemProps[] = [
      {
        color: 'green',
        content: (
          <>
            <p>Submitted</p>
            <small>{dayjs(request?.createdAt).format('YYYY-MM-DD HH:mm')}</small>
          </>
        ),
      },
      ...approvals.map(app => ({
        key: app.id,
        color: approvalStatusColors[app.status] || 'gray',
        icon:
          app.status === ApprovalStatus.APPROVED ? (
            <CheckCircle />
          ) : app.status === ApprovalStatus.REJECTED ? (
            <XCircle />
          ) : (
            <Timer />
          ),
        content: (
          <Space orientation="vertical" size={2}>
            <Text strong>{`${app.level}. ${app.step.toUpperCase()} Approval`}</Text>
            <div>
              <Tag color={approvalStatusColors[app.status]}>
                {t(approvalStatusLabelKeys[app.status] || app.status)}
              </Tag>
              {app.approverName && <Text type="secondary">by {app.approverName}</Text>}
            </div>
            {app.comment && (
              <Text type="secondary" italic>
                &quot;{app.comment}&quot;
              </Text>
            )}
            {app.processedAt && (
              <div style={{ fontSize: '12px', color: '#999' }}>
                {dayjs(app.processedAt).format('YYYY-MM-DD HH:mm')}
              </div>
            )}
          </Space>
        ),
      })),
    ];

    return <Timeline items={items} />;
  };

  // 审批入口的资格判定：必须以后端下发的 canApprove 为准。
  //
  // 2026-11-04 prod 实测缺陷：此前只判断「当前级是否有 pending 行」，
  // 等于「只要流程没走完，任何打开详情页的人都能看到通过/拒绝」——
  // 注释里写明"依赖后端拦截"，但后端拦截是 500「操作失败」（真实原因被吞），
  // 用户点下去只看到一个无信息量的报错。
  //
  // canApprove 由后端按「当前用户是否落在本级 approver_ids 集合内」判定，
  // 缺字段（旧响应）一律按无资格处理：宁可少给入口，也不给必然失败的入口。
  const canApprove =
    request?.canApprove === true &&
    approvals.some(a => a.status === ApprovalStatus.PENDING && a.level === request?.currentLevel);

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '50px auto' }} />;
  if (!request) return <div>Request not found</div>;

  return (
    <div style={{ padding: '24px' }}>
      <Space orientation="vertical" size="large" style={{ width: '100%' }}>
        {/* 头部信息 */}
        <Card>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start' }}>
            <div>
              <Title level={3}>{request.title || 'Service Request'}</Title>
              <Space size="middle">
                <Tag>{serviceCategoryLabel(request.catalog?.category)}</Tag>
                <Text type="secondary">ID: {request.id}</Text>
                <Text type="secondary">

                  Filed under: {dayjs(request.createdAt).format('YYYY-MM-DD HH:mm')}
                </Text>
              </Space>
            </div>
            <div style={{ textAlign: 'right' }}>
              <Title level={4} style={{ margin: 0 }}>
                <Tag color={request.status === ServiceRequestStatus.DELIVERED ? 'green' : 'blue'}>
                  {serviceRequestStatusLabel(request.status)}
                </Tag>
              </Title>
              {canCancel && (
                <Button
                  danger
                  icon={<XCircle />}
                  style={{ marginTop: 12 }}
                  onClick={() => setCancelModalOpen(true)}
                >

                  Cancel request
                </Button>
              )}
            </div>
          </div>
        </Card>

        <div style={{ display: 'flex', gap: '24px' }}>
          {/* 左侧：详情 */}
          <div style={{ flex: 2 }}>
            <Card title="Request Detail">
              <Descriptions column={1} bordered>
                <Descriptions.Item label="Service Name">{request.catalog?.name}</Descriptions.Item>
                <Descriptions.Item label="Reason">{request.reason}</Descriptions.Item>
                <Descriptions.Item label="Cost Center">{request.costCenter || '-'}</Descriptions.Item>
                <Descriptions.Item label="Data Classification">
                  {request.dataClassification || 'Public'}
                </Descriptions.Item>
                <Descriptions.Item label="Public IP required">
                  {request.needsPublicIp ? 'Yes' : 'No'}
                </Descriptions.Item>
                {request.formData && (
                  <Descriptions.Item label="Form data">
                    <pre style={{ margin: 0, fontSize: '12px' }}>
                      {JSON.stringify(request.formData, null, 2)}
                    </pre>
                  </Descriptions.Item>
                )}
              </Descriptions>
            </Card>
          </div>

          {/* 右侧：审批流 & 操作 */}
          <div style={{ flex: 1 }}>
            <Card title="Workflow">
              {renderApprovalTimeline()}

              {/* 审批操作区 */}
              {canApprove &&
                request.status !== ServiceRequestStatus.REJECTED &&
                request.status !== ServiceRequestStatus.CANCELLED && (
                  <>
                    <Divider />
                    <Title level={5}>Approval operation</Title>
                    <Space style={{ width: '100%', justifyContent: 'center' }}>
                      <Button
                        type="primary"
                        icon={<CheckCircle />}
                        onClick={() => openActionModal(ApprovalAction.APPROVE)}
                      >

                        Approve
                      </Button>
                      <Button
                        danger
                        icon={<XCircle />}
                        onClick={() => openActionModal(ApprovalAction.REJECT)}
                      >

                        Rejected
                      </Button>
                    </Space>
                  </>
                )}
            </Card>
          </div>
        </div>

        {['security_approved', 'provisioning', 'delivered', 'failed'].includes(request.status) && (
          <Card>
            <ServiceRequestProvisioningPanel
              requestId={requestId}
              status={request.status}
              onChanged={loadDetail}
            />
          </Card>
        )}
      </Space>

      {/* 审批确认弹窗 */}
      <Modal
        title={currentAction === ApprovalAction.APPROVE ? 'Confirm approval' : 'Confirm Rejection'}
        open={actionModalVisible}
        onOk={handleSubmitApproval}
        onCancel={() => setActionModalVisible(false)}
        confirmLoading={submitting}
        okText="Submit"
        cancelText="Cancel"
        okButtonProps={{ danger: currentAction === ApprovalAction.REJECT }}
      >
        <p>OK{currentAction === ApprovalAction.APPROVE ? 'Approve' : 'Rejected'}This request?</p>
        <TextArea
          rows={4}
          value={comment}
          onChange={e => setComment(e.target.value)}
          placeholder={
            currentAction === ApprovalAction.REJECT ? 'Please fill in the reason for rejection (required)' : 'Approval comments (optional)'
          }
        />
      </Modal>

      {/* 取消确认弹窗 */}
      <Modal
        title="Cancel service request"
        open={cancelModalOpen}
        onOk={handleCancelRequest}
        onCancel={() => setCancelModalOpen(false)}
        confirmLoading={cancelling}
        okText="Confirm cancellation"
        cancelText="Back"
        okButtonProps={{ danger: true }}
        destroyOnHidden
      >
        <p>After cancellation, the record to be approved will be invalidated, the associated process instance will be terminated, and it cannot be recovered.</p>
        <TextArea
          rows={4}
          value={cancelReason}
          onChange={e => setCancelReason(e.target.value)}
          placeholder="Reason for cancellation (optional)"
          maxLength={500}
          showCount
        />
      </Modal>
    </div>
  );
};

export default ServiceRequestDetail;
