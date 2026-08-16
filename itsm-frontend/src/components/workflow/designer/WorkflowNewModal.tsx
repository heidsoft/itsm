// 新建工作流模态框
// Workflow New Modal - 模板选择

'use client';

import React from 'react';
import { Modal, Input, Form, Button } from 'antd';
import { FileText } from 'lucide-react';
import { WORKFLOW_TEMPLATES, TEMPLATE_CATEGORIES } from '@/lib/workflow-templates';
import type { WorkflowDefinition } from './WorkflowTypes';
import { useI18n } from '@/lib/i18n/useI18n';

interface WorkflowNewModalProps {
  visible: boolean;
  onClose: () => void;
  onSelectTemplate: (workflow: WorkflowDefinition) => void;
  onCreateCustom: (values: {
    name: string;
    description?: string;
    slaResponse?: number;
    slaResolution?: number;
  }) => void;
}

export default function WorkflowNewModal({
  visible,
  onClose,
  onSelectTemplate,
  onCreateCustom,
}: WorkflowNewModalProps) {
  const { t } = useI18n();
  const [form] = Form.useForm();

  const handleSelectTemplate = (template: (typeof WORKFLOW_TEMPLATES)[0]) => {
    const newWorkflow: WorkflowDefinition = {
      id: 'new',
      name: template.name,
      description: template.description,
      version: '1.0.0',
      category: template.category,
      status: 'draft',
      xml: template.bpmnXml,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      createdBy: t('workflow.newModal.currentUser'),
      tags: [],
      approvalConfig: {
        requireApproval: template.approvalConfig.requireApproval,
        approvalType: template.approvalConfig.approvalType,
        approvers: template.approvalConfig.approvers,
        autoApproveRoles: [],
        escalationRules: [],
      },
      variables: [],
      slaConfig: {
        responseTimeHours: 24,
        resolutionTimeHours: 72,
        businessHoursOnly: true,
        excludeWeekends: true,
        excludeHolidays: true,
      },
    };
    onSelectTemplate(newWorkflow);
  };

  const handleSubmit = (values: { name: string; description?: string; slaResponse?: number; slaResolution?: number }) => {
    onCreateCustom(values);
  };

  return (
    <Modal
      title={t('workflow.newModal.title')}
      open={visible}
      onCancel={onClose}
      footer={null}
      width={900}
    >
      <div className="mb-4">
        <Input.Search
          placeholder={t('workflow.newModal.searchPlaceholder')}
          style={{ width: 300 }}
          onSearch={() => {
            // can add search functionality here
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
        <div className="text-sm text-gray-500 mb-3">{t('workflow.newModal.orCustomCreate')}</div>
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <div className="flex gap-4">
            <Form.Item
              label={t('workflow.newModal.workflowName')}
              name="name"
              rules={[{ required: true, message: t('workflow.newModal.workflowNameRequired') }]}
              style={{ flex: 1 }}
            >
              <Input placeholder={t('workflow.newModal.workflowNamePlaceholder')} />
            </Form.Item>
            <Form.Item style={{ marginTop: '32px' }}>
              <Button type="primary" htmlType="submit">
                {t('workflow.newModal.createBlank')}
              </Button>
            </Form.Item>
          </div>
        </Form>
      </div>
    </Modal>
  );
}