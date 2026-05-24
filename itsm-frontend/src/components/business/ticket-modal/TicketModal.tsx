/**
 * 重构后的 TicketModal 组件
 * 简化版：只负责容器和状态管理，业务逻辑已拆分
 */

import React, { useEffect, useState } from 'react';
import { Modal, Form, Button, Input, Select, message, App } from 'antd';
import { Plus, Edit, BookOpen, Save } from 'lucide-react';
import type { Ticket } from '@/lib/services/ticket-service';
import { useI18n } from '@/lib/i18n';

import { TicketForm } from './components/TicketForm';
import { TemplateCard } from './components/TemplateCard';
import type { TicketModalProps, TicketTemplateModalProps, TicketTemplate, User } from './types';
import { MOCK_TICKET_TEMPLATES, MOCK_USER_LIST } from './utils/ticket-form-utils';

// 注意：这里使用模拟数据，实际应该通过 API 或 props 传入

export const TicketModal: React.FC<TicketModalProps> = React.memo(
  ({ visible, editingTicket, onCancel, onSubmit, loading = false }) => {
    const { t } = useI18n();
    const [form] = Form.useForm();
    const { message } = App.useApp();

    // 当编辑工单时，初始化表单数据
    useEffect(() => {
      if (visible) {
        if (editingTicket) {
          form.setFieldsValue({
            title: editingTicket.title,
            type: editingTicket.type,
            category: editingTicket.category,
            priority: editingTicket.priority,
            assignee_id: editingTicket.assigneeId,
            description: editingTicket.description,
          });
        } else {
          form.resetFields();
        }
      }
    }, [visible, editingTicket, form]);

    const handleSubmit = (values: any) => {
      onSubmit(values);
      form.resetFields();
    };

    const handleCancel = () => {
      form.resetFields();
      onCancel();
    };

    return (
      <Modal
        title={
          <div className="flex items-center gap-3 pb-4 border-b border-gray-100">
            <div className="w-10 h-10 bg-gradient-to-r from-blue-500 to-blue-600 rounded-lg flex items-center justify-center shadow-md">
              {editingTicket ? (
                <Edit size={20} className="text-white" />
              ) : (
                <Plus size={20} className="text-white" />
              )}
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-900">
                {editingTicket ? t('tickets.edit') : t('tickets.create')}
              </h3>
              <p className="text-sm text-gray-500">
                {editingTicket
                  ? t('tickets.editDescription') || '修改工单信息'
                  : t('tickets.createDescription') || '填写工单详细信息'}
              </p>
            </div>
          </div>
        }
        open={visible}
        onCancel={handleCancel}
        footer={null}
        width={900}
        className="[&_.ant-modal-content]:rounded-xl [&_.ant-modal-content]:shadow-2xl [&_.ant-modal-header]:border-0 [&_.ant-modal-header]:pb-0 [&_.ant-modal-body]:pt-6"
      >
        <TicketForm
          form={form}
          onSubmit={handleSubmit}
          loading={loading}
          isEditing={!!editingTicket}
        />
      </Modal>
    );
  }
);

TicketModal.displayName = 'TicketModal';

export const TicketTemplateModal: React.FC<TicketTemplateModalProps> = React.memo(
  ({ visible, onCancel }) => {
    const [templates, setTemplates] = useState<TicketTemplate[]>(MOCK_TICKET_TEMPLATES);
    const [editingTemplate, setEditingTemplate] = useState<TicketTemplate | null>(null);
    const [showEditModal, setShowEditModal] = useState(false);
    const [templateForm] = Form.useForm();

    // 编辑模板
    const handleEditTemplate = (template: TicketTemplate) => {
      setEditingTemplate(template);
      templateForm.setFieldsValue({
        name: template.name,
        description: template.description,
        type: template.type,
      });
      setShowEditModal(true);
    };

    // 保存编辑
    const handleSaveTemplate = async () => {
      try {
        const values = await templateForm.validateFields();
        if (editingTemplate) {
          setTemplates(prev =>
            prev.map(t => (t.id === editingTemplate.id ? { ...t, ...values } : t))
          );
          message.success('模板更新成功');
        }
        setShowEditModal(false);
        setEditingTemplate(null);
        templateForm.resetFields();
      } catch {
        // 表单验证失败
      }
    };

    // 删除模板
    const handleDeleteTemplate = (templateId: number) => {
      setTemplates(prev => prev.filter(t => t.id !== templateId));
      message.success('模板删除成功');
    };

    return (
      <>
        <Modal
          title={
            <div className="flex items-center gap-3 pb-4 border-b border-gray-100">
              <div className="w-10 h-10 bg-gradient-to-r from-green-500 to-green-600 rounded-lg flex items-center justify-center shadow-md">
                <BookOpen size={20} className="text-white" />
              </div>
              <div>
                <h3 className="text-lg font-semibold text-gray-900">工单模板管理</h3>
                <p className="text-sm text-gray-500">管理和配置工单模板，提高工作效率</p>
              </div>
            </div>
          }
          open={visible}
          onCancel={onCancel}
          footer={null}
          width={1100}
          className="[&_.ant-modal-content]:rounded-xl [&_.ant-modal-content]:shadow-2xl [&_.ant-modal-header]:border-0 [&_.ant-modal-header]:pb-0 [&_.ant-modal-body]:pt-6"
        >
          <div style={{ padding: '24px 0' }}>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                marginBottom: 24,
              }}
            >
              <div>
                <p style={{ color: '#666' }}>管理所有可用的工单模板，提高工单创建效率</p>
              </div>
              <Button
                type="primary"
                icon={<Plus size={16} />}
                onClick={() => {
                  setEditingTemplate(null);
                  templateForm.resetFields();
                  setShowEditModal(true);
                }}
              >
                新建模板
              </Button>
            </div>

            <div className="space-y-4">
              {templates.map(template => (
                <TemplateCard
                  key={template.id}
                  template={template}
                  onEdit={handleEditTemplate}
                  onDelete={handleDeleteTemplate}
                />
              ))}
            </div>
          </div>
        </Modal>

        {/* 编辑模板弹窗 */}
        <Modal
          title={editingTemplate ? '编辑模板' : '新建模板'}
          open={showEditModal}
          onCancel={() => {
            setShowEditModal(false);
            setEditingTemplate(null);
            templateForm.resetFields();
          }}
          footer={[
            <Button key="cancel" onClick={() => setShowEditModal(false)}>
              取消
            </Button>,
            <Button
              key="save"
              type="primary"
              icon={<Save size={16} />}
              onClick={handleSaveTemplate}
            >
              保存
            </Button>,
          ]}
        >
          <Form form={templateForm} layout="vertical" className="mt-4">
            <Form.Item name="name" label="模板名称" rules={[{ required: true }]}>
              <Input placeholder="请输入模板名称" />
            </Form.Item>
            <Form.Item name="description" label="模板描述">
              <Input.TextArea placeholder="请输入模板描述" rows={3} />
            </Form.Item>
            <Form.Item name="type" label="模板类型" rules={[{ required: true }]}>
              <Select placeholder="请选择模板类型">
                <Select.Option value="incident">故障</Select.Option>
                <Select.Option value="request">请求</Select.Option>
                <Select.Option value="change">变更</Select.Option>
              </Select>
            </Form.Item>
          </Form>
        </Modal>
      </>
    );
  }
);

TicketTemplateModal.displayName = 'TicketTemplateModal';
