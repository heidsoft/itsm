// 工作流元数据编辑模态框
// Workflow Metadata Modal

'use client';

import React from 'react';
import { Modal, Form, Input, Select } from 'antd';
import type { FormInstance } from 'antd';
import { useI18n } from '@/lib/i18n/useI18n';
const { TextArea } = Input;

interface WorkflowMetadataModalProps {
  visible: boolean;
  onClose: () => void;
  onSave: (values: { name: string; description: string; category: string }) => void;
  form: FormInstance;
}

export default function WorkflowMetadataModal({
  visible,
  onClose,
  onSave,
  form,
}: WorkflowMetadataModalProps) {
  const { t } = useI18n();

  const handleOk = () => {
    form
      .validateFields()
      .then(onSave)
      .catch(() => {});
  };

  return (
    <Modal
      title={t('workflow.metadataModal.title')}
      open={visible}
      onOk={handleOk}
      onCancel={onClose}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          label={t('workflow.metadataModal.name')}
          name="name"
          rules={[{ required: true, message: t('workflow.metadataModal.nameRequired') }]}
        >
          <Input placeholder={t('workflow.metadataModal.namePlaceholder')} />
        </Form.Item>
        <Form.Item label={t('workflow.metadataModal.description')} name="description">
          <TextArea rows={3} placeholder={t('workflow.metadataModal.descriptionPlaceholder')} />
        </Form.Item>
        <Form.Item label={t('workflow.metadataModal.category')} name="category">
          <Select placeholder={t('workflow.metadataModal.categoryPlaceholder')} options={[
            { value: 'general', label: t('workflow.metadataModal.categoryGeneral') },
            { value: 'approval', label: t('workflow.metadataModal.categoryApproval') },
            { value: 'ticket', label: t('workflow.metadataModal.categoryTicket') },
            { value: 'incident', label: t('workflow.metadataModal.categoryIncident') },
            { value: 'change', label: t('workflow.metadataModal.categoryChange') },
          ]} />
        </Form.Item>
      </Form>
    </Modal>
  );
}