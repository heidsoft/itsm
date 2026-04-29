'use client';

import React, { useEffect, useState } from 'react';
import { Table, Tag, Button, Card, App, Space, Modal, Input } from 'antd';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { useI18n } from '@/lib/i18n';
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
  const [requests, setRequests] = useState<ServiceRequestRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [rejectModalVisible, setRejectModalVisible] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState<ServiceRequestRecord | null>(null);
  const [rejectReason, setRejectReason] = useState('');

  useEffect(() => {
    loadApprovals();
  }, []);

  const loadApprovals = async () => {
    setLoading(true);
    try {
      const data = await ServiceCatalogApi.getServiceRequests({
        status: ServiceRequestStatus.PENDING_APPROVAL,
      });
      setRequests((data.requests || []) as ServiceRequestRecord[]);
    } catch (error) {
      message.error(t('common.getFailed'));
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async (id: number) => {
    try {
      await ServiceCatalogApi.approveServiceRequest(id);
      message.success(t('service.approveSuccess'));
      loadApprovals();
    } catch (error) {
      message.error(t('service.approveFailed'));
    }
  };

  const handleReject = async (id: number) => {
    if (!rejectReason.trim()) {
      message.error(t('service.rejectReasonRequired'));
      return;
    }
    try {
      await ServiceCatalogApi.rejectServiceRequest(id, rejectReason);
      message.success(t('service.rejectSuccess'));
      setRejectModalVisible(false);
      loadApprovals();
    } catch (error) {
      message.error(t('service.rejectFailed'));
    }
  };

  const columns = [
    { title: t('service.requestId'), dataIndex: 'id' },
    { title: t('service.serviceName'), dataIndex: 'serviceName' },
    { title: t('service.requester'), dataIndex: 'requesterName' },
    { title: t('service.createdAt'), dataIndex: 'createdAt' },
    {
      title: t('service.status'),
      render: (_: unknown, record: ServiceRequestRecord) => (
        <Tag color="orange">{t('service.pending')}</Tag>
      ),
    },
    {
      title: t('common.actions'),
      render: (_: unknown, record: ServiceRequestRecord) => (
        <Space>
          <Button size="small" type="primary" onClick={() => handleApprove(record.id)}>
            {t('service.approve')}
          </Button>
          <Button
            size="small"
            onClick={() => {
              setSelectedRequest(record);
              setDetailModalVisible(true);
            }}
          >
            {t('common.view')}
          </Button>
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
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      <Card title={t('service.pendingApprovals')}>
        <Table columns={columns} dataSource={requests} loading={loading} rowKey="id" />
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
