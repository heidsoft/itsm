/**
 * SLA 违规详情模态框（简化版）
 */

import React from 'react';
import { Modal, Descriptions, Tag, Space, Button } from 'antd';
import type { SLAViolation } from '../types';

interface SLAViolationDetailModalProps {
  violation: SLAViolation | null;
  visible: boolean;
  onClose: () => void;
  canManage?: boolean;
  actionLoading?: boolean;
  onResolve: (violation: SLAViolation) => void | Promise<void>;
  onAcknowledge: (violation: SLAViolation) => void | Promise<void>;
}

export const SLAViolationDetailModal: React.FC<SLAViolationDetailModalProps> = ({
  violation,
  visible,
  onClose,
  onResolve,
  onAcknowledge,
  canManage = false,
  actionLoading = false,
}) => {
  if (!violation) return null;

  const severityColors: Record<string, string> = {
    critical: 'red',
    high: 'orange',
    medium: 'gold',
    low: 'blue',
  };

  return (
    <Modal
      title="SLA 违规详情"
      open={visible}
      onCancel={onClose}
      footer={
        <Space>
          {canManage && violation.status === 'open' && (
            <>
              <Button loading={actionLoading} onClick={() => void onAcknowledge(violation)}>确认</Button>
              <Button type="primary" danger loading={actionLoading} onClick={() => void onResolve(violation)}>
                解决
              </Button>
            </>
          )}
          <Button onClick={onClose}>关闭</Button>
        </Space>
      }
      width={700}
    >
      <Descriptions bordered column={1}>
        <Descriptions.Item label="ID">{violation.id}</Descriptions.Item>
        <Descriptions.Item label="工单ID">{violation.ticketId}</Descriptions.Item>
        <Descriptions.Item label="SLA定义ID">{violation.slaDefId}</Descriptions.Item>
        <Descriptions.Item label="违规类型">{violation.violationType}</Descriptions.Item>
        <Descriptions.Item label="严重程度">
          <Tag color={severityColors[violation.severity] || 'default'}>
            {violation.severity}
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label="状态">
          <Tag color={violation.status === 'resolved' ? 'green' : 'red'}>
            {violation.status}
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label="期望时间">
          {new Date(violation.expectedTime).toLocaleString()}
        </Descriptions.Item>
        <Descriptions.Item label="实际时间">
          {new Date(violation.actualTime).toLocaleString()}
        </Descriptions.Item>
        <Descriptions.Item label="延迟分钟数">{violation.delayMinutes} 分钟</Descriptions.Item>
        <Descriptions.Item label="描述">
          {violation.description || '-'}
        </Descriptions.Item>
        <Descriptions.Item label="创建时间">
          {violation.createdAt ? new Date(violation.createdAt).toLocaleString() : '-'}
        </Descriptions.Item>
        <Descriptions.Item label="更新时间">
          {violation.updatedAt ? new Date(violation.updatedAt).toLocaleString() : '-'}
        </Descriptions.Item>
      </Descriptions>
    </Modal>
  );
};

SLAViolationDetailModal.displayName = 'SLAViolationDetailModal';
