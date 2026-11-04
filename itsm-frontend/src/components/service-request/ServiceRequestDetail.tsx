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
      message.error('加载详情失败');
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
      message.error('拒绝操作必须填写原因');
      return;
    }

    setSubmitting(true);
    try {
      await ServiceRequestApi.applyApproval(
        requestId,
        currentAction === ApprovalAction.APPROVE ? 'approve' : 'reject',
        comment
      );
      message.success('操作成功');
      setActionModalVisible(false);
      loadDetail(); // 刷新数据
    } catch (error: unknown) {
      // console.error(error);
      message.error(error instanceof Error ? error.message : '操作失败');
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
      message.success('服务请求已取消');
      setCancelModalOpen(false);
      setCancelReason('');
      await loadDetail();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '取消失败');
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
            <p>提交申请</p>
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
            <Text strong>{`${app.level}. ${app.step.toUpperCase()} 审批`}</Text>
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
  if (!request) return <div>未找到请求</div>;

  return (
    <div style={{ padding: '24px' }}>
      <Space orientation="vertical" size="large" style={{ width: '100%' }}>
        {/* 头部信息 */}
        <Card>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start' }}>
            <div>
              <Title level={3}>{request.title || '服务请求'}</Title>
              <Space size="middle">
                <Tag>{request.catalog?.category}</Tag>
                <Text type="secondary">ID: {request.id}</Text>
                <Text type="secondary">
                  提交于: {dayjs(request.createdAt).format('YYYY-MM-DD HH:mm')}
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
                  取消请求
                </Button>
              )}
            </div>
          </div>
        </Card>

        <div style={{ display: 'flex', gap: '24px' }}>
          {/* 左侧：详情 */}
          <div style={{ flex: 2 }}>
            <Card title="请求详情">
              <Descriptions column={1} bordered>
                <Descriptions.Item label="服务名称">{request.catalog?.name}</Descriptions.Item>
                <Descriptions.Item label="申请原因">{request.reason}</Descriptions.Item>
                <Descriptions.Item label="成本中心">{request.costCenter || '-'}</Descriptions.Item>
                <Descriptions.Item label="数据分类">
                  {request.dataClassification || 'Public'}
                </Descriptions.Item>
                <Descriptions.Item label="需要公网IP">
                  {request.needsPublicIp ? '是' : '否'}
                </Descriptions.Item>
                {request.formData && (
                  <Descriptions.Item label="表单数据">
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
            <Card title="审批流程">
              {renderApprovalTimeline()}

              {/* 审批操作区 */}
              {canApprove &&
                request.status !== ServiceRequestStatus.REJECTED &&
                request.status !== ServiceRequestStatus.CANCELLED && (
                  <>
                    <Divider />
                    <Title level={5}>审批操作</Title>
                    <Space style={{ width: '100%', justifyContent: 'center' }}>
                      <Button
                        type="primary"
                        icon={<CheckCircle />}
                        onClick={() => openActionModal(ApprovalAction.APPROVE)}
                      >
                        通过
                      </Button>
                      <Button
                        danger
                        icon={<XCircle />}
                        onClick={() => openActionModal(ApprovalAction.REJECT)}
                      >
                        拒绝
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
        title={currentAction === ApprovalAction.APPROVE ? '确认批准' : '确认拒绝'}
        open={actionModalVisible}
        onOk={handleSubmitApproval}
        onCancel={() => setActionModalVisible(false)}
        confirmLoading={submitting}
        okText="提交"
        cancelText="取消"
        okButtonProps={{ danger: currentAction === ApprovalAction.REJECT }}
      >
        <p>确定要{currentAction === ApprovalAction.APPROVE ? '批准' : '拒绝'}此请求吗？</p>
        <TextArea
          rows={4}
          value={comment}
          onChange={e => setComment(e.target.value)}
          placeholder={
            currentAction === ApprovalAction.REJECT ? '请填写拒绝原因 (必填)' : '审批意见 (选填)'
          }
        />
      </Modal>

      {/* 取消确认弹窗 */}
      <Modal
        title="取消服务请求"
        open={cancelModalOpen}
        onOk={handleCancelRequest}
        onCancel={() => setCancelModalOpen(false)}
        confirmLoading={cancelling}
        okText="确认取消"
        cancelText="返回"
        okButtonProps={{ danger: true }}
        destroyOnHidden
      >
        <p>取消后待审批记录会作废、关联流程实例会终止，且不可恢复。</p>
        <TextArea
          rows={4}
          value={cancelReason}
          onChange={e => setCancelReason(e.target.value)}
          placeholder="取消原因（选填）"
          maxLength={500}
          showCount
        />
      </Modal>
    </div>
  );
};

export default ServiceRequestDetail;
