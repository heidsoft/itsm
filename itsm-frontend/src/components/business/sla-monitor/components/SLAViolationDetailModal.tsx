/**
 * SLA 违规详情模态框（简化版）
 */

'use client';

import React from 'react';
import { Modal, Descriptions, Tag, Space, Button } from 'antd';
import type { SLAViolation } from '../types';
import { useI18n } from '@/lib/i18n/useI18n';

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
  const { t } = useI18n();

  if (!violation) return null;

  const severityColors: Record<string, string> = {
    critical: 'red',
    high: 'orange',
    medium: 'gold',
    low: 'blue',
  };

  const severityLabels: Record<string, string> = {
    critical: t('sla.violation.severityCritical'),
    high: t('sla.violation.severityHigh'),
    medium: t('sla.violation.severityMedium'),
    low: t('sla.violation.severityLow'),
  };

  return (
    <Modal
      title={t('sla.violation.detailTitle')}
      open={visible}
      onCancel={onClose}
      footer={
        <Space>
          {canManage && !violation.isResolved && (
            <>
              <Button loading={actionLoading} onClick={() => void onAcknowledge(violation)}>
                {t('sla.violation.acknowledge')}
              </Button>
              <Button type="primary" danger loading={actionLoading} onClick={() => void onResolve(violation)}>
                {t('sla.violation.resolve')}
              </Button>
            </>
          )}
          <Button onClick={onClose}>{t('common.close')}</Button>
        </Space>
      }
      width={700}
    >
      <Descriptions bordered column={1}>
        <Descriptions.Item label={t('sla.violation.fieldId')}>{violation.id}</Descriptions.Item>
        <Descriptions.Item label={t('sla.violation.fieldTicketId')}>{violation.ticketId}</Descriptions.Item>
        <Descriptions.Item label={t('sla.violation.fieldTicketNumber')}>
          {violation.ticketNumber || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('sla.violation.fieldTicketTitle')}>
          {violation.ticketTitle || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('sla.violation.fieldSlaDefId')}>{violation.slaDefinitionId}</Descriptions.Item>
        <Descriptions.Item label={t('sla.violation.fieldSlaName')}>
          {violation.slaName || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('sla.violation.fieldViolationType')}>{violation.violationType}</Descriptions.Item>
        <Descriptions.Item label={t('sla.violation.fieldSeverity')}>
          <Tag color={severityColors[violation.severity] || 'default'}>
            {severityLabels[violation.severity] || violation.severity}
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label={t('sla.violation.fieldStatus')}>
          <Tag color={violation.isResolved ? 'green' : 'red'}>
            {violation.isResolved ? t('sla.violation.statusResolved') : t('sla.violation.statusOpen')}
          </Tag>
        </Descriptions.Item>
        <Descriptions.Item label={t('sla.violation.fieldViolationTime')}>
          {violation.violationTime ? new Date(violation.violationTime).toLocaleString() : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('sla.violation.fieldResolvedAt')}>
          {violation.resolvedAt ? new Date(violation.resolvedAt).toLocaleString() : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('sla.violation.fieldResolutionNotes')}>
          {violation.resolutionNotes || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('sla.violation.fieldDescription')}>
          {violation.description || '-'}
        </Descriptions.Item>
      </Descriptions>
    </Modal>
  );
};

SLAViolationDetailModal.displayName = 'SLAViolationDetailModal';
