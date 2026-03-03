// 新建工作流模态框
// Workflow New Modal - 模板选择

'use client';

import React from 'react';
import { Modal, Input, Form, Button, Typography } from 'antd';
import { FileText } from 'lucide-react';
import { WORKFLOW_TEMPLATES, TEMPLATE_CATEGORIES } from '@/lib/workflow-templates';
import type { WorkflowDefinition } from './WorkflowTypes';

const { Text } = Typography;

interface WorkflowNewModalProps {
  visible: boolean;
  onClose: () => void;
  onSelectTemplate: (workflow: WorkflowDefinition) => void;
  onCreateCustom: (values: {
    name: string;
    description?: string;
    sla_response?: number;
    sla_resolution?: number;
  }) => void;
}

export default function WorkflowNewModal({
  visible,
  onClose,
  onSelectTemplate,
  onCreateCustom,
}: WorkflowNewModalProps) {
  const [form] = Form.useForm();

  const handleSelectTemplate = (template: (typeof WORKFLOW_TEMPLATES)[0]) => {
    const newWorkflow: WorkflowDefinition = {
      id: 'new',
      name: template.name,
      description: template.description,
      version: '1.0.0',
      category: template.category,
      status: 'draft',
      xml: template.bpmn_xml,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      created_by: '当前用户',
      tags: [],
      approval_config: {
        require_approval: template.approval_config.require_approval,
        approval_type: template.approval_config.approval_type,
        approvers: template.approval_config.approvers,
        auto_approve_roles: [],
        escalation_rules: [],
      },
      variables: [],
      sla_config: {
        response_time_hours: 24,
        resolution_time_hours: 72,
        business_hours_only: true,
        exclude_weekends: true,
        exclude_holidays: true,
      },
    };
    onSelectTemplate(newWorkflow);
  };

  const handleSubmit = (values: any) => {
    onCreateCustom(values);
  };

  return (
    <Modal
      title="选择工作流模板"
      open={visible}
      onCancel={onClose}
      footer={null}
      width={900}
      destroyOnClose
    >
      <div className="mb-4">
        <Input.Search
          placeholder="搜索模板..."
          style={{ width: 300 }}
          onSearch={value => {
            // 可以添加搜索功能
          }}
        />
      </div>

      <div
        className="grid grid-cols-2 md:grid-cols-3 gap-4"
        style={{ gridTemplateColumns: 'repeat(3, 1fr)' }}
      >
        {WORKFLOW_TEMPLATES.map(template => (
          <div
            key={template.id}
            className="border rounded-lg p-4 cursor-pointer hover:border-blue-500 hover:bg-blue-50 transition-all"
            style={{
              border: '1px solid #d9d9d9',
              borderRadius: '8px',
              padding: '16px',
              cursor: 'pointer',
              transition: 'all 0.2s',
            }}
            onClick={() => handleSelectTemplate(template)}
          >
            <div className="flex items-center mb-2">
              <div className="w-10 h-10 rounded-lg bg-blue-100 flex items-center justify-center mr-3">
                <FileText className="w-5 h-5 text-blue-600" />
              </div>
              <div>
                <div className="font-medium">{template.name}</div>
                <div className="text-xs text-gray-500">
                  {TEMPLATE_CATEGORIES.find(c => c.key === template.category)?.name ||
                    template.category}
                </div>
              </div>
            </div>
            <div className="text-xs text-gray-500 mt-2">{template.description}</div>
          </div>
        ))}
      </div>

      <div className="mt-6 border-t pt-4">
        <div className="text-sm text-gray-500 mb-3">或者自定义创建</div>
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <div className="flex gap-4">
            <Form.Item
              label="工作流名称"
              name="name"
              rules={[{ required: true, message: '请输入工作流名称' }]}
              style={{ flex: 1 }}
            >
              <Input placeholder="自定义流程名称" />
            </Form.Item>
            <Form.Item style={{ marginTop: '32px' }}>
              <Button type="primary" htmlType="submit">
                创建空白流程
              </Button>
            </Form.Item>
          </div>
        </Form>
      </div>
    </Modal>
  );
}
