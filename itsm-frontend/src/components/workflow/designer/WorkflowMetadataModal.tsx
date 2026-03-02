// 工作流元数据编辑模态框
// Workflow Metadata Modal

'use client';

import React from 'react';
import { Modal, Form, Input, Select } from 'antd';
import type { FormInstance } from 'antd';
import { useI18n } from '@/lib/i18n';

const { Option } = Select;
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
    form.validateFields().then(onSave).catch(() => {});
  };

  return (
    <Modal
      title="编辑工作流信息"
      open={visible}
      onOk={handleOk}
      onCancel={onClose}
      okText={t('common.save')}
      cancelText={t('common.cancel')}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          label="工作流名称"
          name="name"
          rules={[{ required: true, message: '请输入工作流名称' }]}
        >
          <Input placeholder="请输入工作流名称" />
        </Form.Item>
        <Form.Item label="描述" name="description">
          <TextArea rows={3} placeholder="请输入工作流描述" />
        </Form.Item>
        <Form.Item label="分类" name="category">
          <Select placeholder="请选择分类">
            <Option value="general">通用</Option>
            <Option value="approval">审批流程</Option>
            <Option value="ticket">工单流程</Option>
            <Option value="incident">事件流程</Option>
            <Option value="change">变更流程</Option>
          </Select>
        </Form.Item>
      </Form>
    </Modal>
  );
}
