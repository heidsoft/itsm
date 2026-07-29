// 工作流元数据编辑模态框
// Workflow Metadata Modal

'use client';

import React from 'react';
import { Modal, Form, Input, Select } from 'antd';
import type { FormInstance } from 'antd';
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
  const handleOk = () => {
    form
      .validateFields()
      .then(onSave)
      .catch(() => {});
  };

  return (
    <Modal
      title="编辑工作流信息"
      open={visible}
      onOk={handleOk}
      onCancel={onClose}
      okText="保存"
      cancelText="取消"
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
          <Select placeholder="请选择分类" options={[{ value: "general", label: "通用" }, { value: "approval", label: "审批流程" }, { value: "ticket", label: "工单流程" }, { value: "incident", label: "事件流程" }, { value: "change", label: "变更流程" }]} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
