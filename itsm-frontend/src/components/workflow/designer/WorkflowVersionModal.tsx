// 版本管理模态框
// Workflow Version Modal

'use client';

import React from 'react';
import { Modal, Alert, Typography, Tag } from 'antd';
import type { WorkflowDefinition } from './WorkflowTypes';
import { useI18n } from '@/lib/i18n/useI18n';

const { Text } = Typography;

interface WorkflowVersionModalProps {
  visible: boolean;
  onClose: () => void;
  onCreate: () => void;
  workflow: WorkflowDefinition | null;
}

export default function WorkflowVersionModal({
  visible,
  onClose,
  onCreate,
  workflow,
}: WorkflowVersionModalProps) {
  const { t } = useI18n();

  return (
    <Modal
      title={t('workflow.versionModal.title')}
      open={visible}
      onOk={onCreate}
      onCancel={onClose}
      okText={t('common.create')}
      cancelText={t('common.cancel')}
    >
      <div className="space-y-4">
        <Alert
          message={t('workflow.versionModal.alertTitle')}
          description={t('workflow.versionModal.alertDesc')}
          type="info"
          showIcon
        />
        <div>
          <Text strong>{t('workflow.versionModal.currentVersion')}</Text>
          <div className="mt-1">
            <Tag color="blue">{workflow?.version}</Tag>
          </div>
        </div>
        <div>
          <Text strong>{t('workflow.versionModal.newVersion')}</Text>
          <div className="mt-1">
            <Tag color="green">
              {workflow ? `${parseFloat(workflow.version) + 0.1}`.slice(0, 3) : '1.1'}
            </Tag>
          </div>
        </div>
      </div>
    </Modal>
  );
}
