'use client';

import React, { useEffect, useRef, useState } from 'react';
import { Alert, Table, Tag, Button, Card, App, Space, Modal, Input } from 'antd';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { useI18n } from '@/lib/i18n';
import { usePermissions } from '@/lib/hooks/use-permissions';
import { PERM } from '@/lib/permissions';
import { ServiceRequestStatus } from '@/types/service-catalog';

interface ServiceRequestRecord {
  id: number;
  serviceName: string;
  requesterName: string;
  createdAt: string;
  reason?: string;
}

export default function ServiceApprovalsPage() {
  const { message } = App.useApp();
  const { t } = useI18n();
  // 审批动作由后端 POST /service-requests/:id/approval 上的 service_request:approve
  // 把关，菜单准入用的却是 service_request:read。只按 read 放人进来，会出现
  // 「能看到待审列表、点审批必 403」——service_catalog_admin / technician /
  // end_user / audit_admin 都只持 read。这里按真实动作码收敛按钮。
  const { hasPermission } = usePermissions();
  const canApprove = hasPermission(PERM.service_request.approve);
  const [requests, setRequests] = useState<ServiceRequestRecord[]>([]);
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染空表格。
  const [listLoading, setListLoading] = useState(true);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [rejectModalVisible, setRejectModalVisible] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState<ServiceRequestRecord | null>(null);
  const [rejectReason, setRejectReason] = useState('');
  // 写操作防重复提交：行级 in-flight 集合拦截审批按钮快速连点，
  // actionLoadingId 驱动按钮 loading；驳回弹窗用 confirmLoading + 同步 guard。
  const inFlightRef = useRef<Set<number>>(new Set());
  const [actionLoadingId, setActionLoadingId] = useState<number | null>(null);
  const [rejectSubmitting, setRejectSubmitting] = useState(false);

  useEffect(() => {
    loadApprovals();
  }, []);

  const loadApprovals = async () => {
    setListLoading(true);
    try {
      const data = await ServiceCatalogApi.getServiceRequests({
        status: ServiceRequestStatus.PENDING_APPROVAL,
      });
      setRequests(data.items as ServiceRequestRecord[]);
    } catch (error) {
      message.error(t('common.getFailed'));
    } finally {
      setListLoading(false);
    }
  };

  const handleApprove = async (id: number) => {
    if (inFlightRef.current.has(id)) return;
    inFlightRef.current.add(id);
    setActionLoadingId(id);
    try {
      await ServiceCatalogApi.approveServiceRequest(id);
      message.success(t('service.approveSuccess'));
      loadApprovals();
    } catch (error) {
      message.error(t('service.approveFailed'));
    } finally {
      inFlightRef.current.delete(id);
      setActionLoadingId(null);
    }
  };

  const handleReject = async (id: number) => {
    if (!rejectReason.trim()) {
      message.error(t('service.rejectReasonRequired'));
      return;
    }
    if (rejectSubmitting) return;
    setRejectSubmitting(true);
    try {
      await ServiceCatalogApi.rejectServiceRequest(id, rejectReason);
      message.success(t('service.rejectSuccess'));
      setRejectModalVisible(false);
      loadApprovals();
    } catch (error) {
      message.error(t('service.rejectFailed'));
    } finally {
      setRejectSubmitting(false);
    }
  };

  const columns = [
    { title: t('service.requestId'), dataIndex: 'id' },
    { title: t('service.serviceName'), dataIndex: 'serviceName' },
    { title: t('service.requester'), dataIndex: 'requesterName' },
    { title: t('service.createdAt'), dataIndex: 'createdAt' },
    {
      title: t('ticketDetail.labelStatus'),
      render: (_: unknown, record: ServiceRequestRecord) => (
        <Tag color="orange">{t('service.pending')}</Tag>
      ),
    },
    {
      title: t('common.actions'),
      render: (_: unknown, record: ServiceRequestRecord) => (
        <Space>
          {canApprove && (
            <Button
              size="small"
              type="primary"
              loading={actionLoadingId === record.id}
              onClick={() => handleApprove(record.id)}
            >
              {t('service.approve')}
            </Button>
          )}
          <Button
            size="small"
            onClick={() => {
              setSelectedRequest(record);
              setDetailModalVisible(true);
            }}
          >
            {t('common.view')}
          </Button>
          {canApprove && (
            <Button
              size="small"
              danger
              onClick={() => {
                setSelectedRequest(record);
                setRejectModalVisible(true);
              }}
            >
              {t('service.reject')}
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      <Card title={t('service.pendingApprovals')}>
        {!canApprove && (
          <Alert
            type="info"
            showIcon
            title="当前账号可查看待审列表，但没有审批权限"
            description="审批与驳回需要 service_request:approve。这里只提供查看，需要处理时请联系审批人。"
            style={{ marginBottom: 16 }}
          />
        )}
        <Table columns={columns} dataSource={requests} loading={listLoading} rowKey="id" />
      </Card>

      {/* Detail Modal */}
      <Modal
        title={t('service.requestDetail')}
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={null}
      >
        {selectedRequest && (
          <div>
            <p>
              <strong>{t('service.serviceName')}:</strong> {selectedRequest.serviceName}
            </p>
            <p>
              <strong>{t('service.requester')}:</strong> {selectedRequest.requesterName}
            </p>
            <p>
              <strong>{t('service.reason')}:</strong> {selectedRequest.reason}
            </p>
          </div>
        )}
      </Modal>

      {/* Reject Modal */}
      <Modal
        title={t('service.rejectRequest')}
        open={rejectModalVisible}
        onCancel={() => setRejectModalVisible(false)}
        confirmLoading={rejectSubmitting}
        onOk={() => {
          if (selectedRequest?.id) handleReject(selectedRequest.id);
        }}
      >
        <p>{t('service.rejectReason')}</p>
        <Input.TextArea
          value={rejectReason}
          onChange={e => setRejectReason(e.target.value)}
          rows={4}
        />
      </Modal>
    </div>
  );
}
