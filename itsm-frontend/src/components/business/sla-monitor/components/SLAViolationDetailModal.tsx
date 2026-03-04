/**
 * SLA 违规详情模态框（简化版）
 */

import React from 'react';
import { Modal,Descriptions, Tag, Space, Button, message } from 'antd';
import type { SLAViolation } from '../types';
import { SLAApi } from '@/lib/api/sla-api';

interface SLAViolationDetailModalProps {
  violation: SLAViolation | null;
  visible: boolean;
  onClose: () => void;
  onResolve: () => void;
  onAcknowledge: () => void;
}

export const SLAViolationDetailModal: React.FC<SLAViolationDetailModalProps> = ({
  violation,
  visible,
  onClose,
  onResolve,
  onAcknowledge,
}) => {
  const handleResolve = async () => {
    if (!violation) return;
    try {
      await SLAApi.updateSLAViolationStatus(violation.id, true, '手动解决');
      message.success('已标记为已解决');
      onResolve();
    } catch (error) {
      message.error('操作失败');
    }
  };

  const handleAcknowledge = async () => {
    if (!violation) return;
    try {
      await SLAApi.updateSLAViolationStatus(violation.id, false, '已确认违规');
      message.success('已确认违规');
      onAcknowledge();
    } catch (error) {
      message.error('操作失败');
    }
  };

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
          {violation.status === 'open' && (
            <>
              <Button onClick={handleAcknowledge}>确认</Button>
              <Button type="primary" danger onClick={handleResolve}>
                解决
              </Button>
            </>
          )}
          <Button onClick={onClose}>关闭</Button>
        </Space>
      }
      width={700}
    >
      <Descriptions bordered column={2}>
        <Descriptions.Item label="ID">{violation.id}</Descriptions.Item>
        <Descriptions.Item label="工单ID">{violation.ticket_id}</Descriptions.Item>
        <Descriptions.Item label="SLA定义ID">{violation.sla_def_id}</Descriptions.Item>
        <Descriptions.Item label="违规类型">{violation.violation_type}</Descriptions.Item>
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
          {new Date(violation.expected_time).toLocaleString()}
        </Descriptions.Item>
        <Descriptions.Item label="实际时间">
          {new Date(violation.actual_time).toLocaleString()}
        </Descriptions.Item>
        <Descriptions.Item label="延迟分钟数">{violation.delay_minutes} 分钟</Descriptions.Item>
        <Descriptions.Item label="描述" span={2}>
          {violation.description || '-'}
        </Descriptions.Item>
        <Descriptions.Item label="创建时间">
          {new Date(violation.created_at).toLocaleString()}
        </Descriptions.Item>
        <Descriptions.Item label="更新时间">
          {new Date(violation.updated_at).toLocaleString()}
        </Descriptions.Item>
      </Descriptions>
    </Modal>
  );
};

SLAViolationDetailModal.displayName = 'SLAViolationDetailModal';
