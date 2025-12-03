'use client';

import React, { useEffect } from 'react';
import { Modal, Form, Input, Select, Button, Row, Col, DatePicker, Avatar } from 'antd';
import { Plus, Edit, BookOpen } from 'lucide-react';
import {
  Ticket,
  TicketStatus,
  TicketPriority,
  TicketType,
} from '../../lib/services/ticket-service';

const { Option } = Select;

interface TicketModalProps {
  visible: boolean;
  editingTicket: Ticket | null;
  onCancel: () => void;
  onSubmit: (values: any) => void;
  loading?: boolean;
}

interface TicketTemplateModalProps {
  visible: boolean;
  onCancel: () => void;
}

const TicketForm: React.FC<{
  form: any;
  onSubmit: (values: any) => void;
  loading?: boolean;
  isEditing?: boolean;
}> = React.memo(({ form, onSubmit, loading = false, isEditing = false }) => {
  const userList = [
    { id: 1, name: 'Alice', role: 'IT Support', avatar: 'A' },
    { id: 2, name: 'Bob', role: 'Network Admin', avatar: 'B' },
    { id: 3, name: 'Charlie', role: 'Service Desk', avatar: 'C' },
  ];

  return (
    <Form layout='vertical' form={form} onFinish={onSubmit} style={{ marginTop: 20 }}>
      <Row gutter={24}>
        <Col span={12}>
          <Form.Item
            label='标题'
            name='title'
            rules={[{ required: true, message: '请输入工单标题' }]}
          >
            <Input placeholder='请输入工单标题' size='large' />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            label='类型'
            name='type'
            rules={[{ required: true, message: '请选择工单类型' }]}
          >
            <Select placeholder='请选择工单类型' size='large'>
              <Option value={TicketType.INCIDENT}>事件</Option>
              <Option value={TicketType.SERVICE_REQUEST}>服务请求</Option>
              <Option value={TicketType.PROBLEM}>问题</Option>
              <Option value={TicketType.CHANGE}>变更</Option>
            </Select>
          </Form.Item>
        </Col>
      </Row>

      <Row gutter={24}>
        <Col span={12}>
          <Form.Item
            label='分类'
            name='category'
            rules={[{ required: true, message: '请选择工单分类' }]}
          >
            <Select placeholder='请选择工单分类' size='large'>
              <Option value='System Access'>系统访问</Option>
              <Option value='Hardware Equipment'>硬件设备</Option>
              <Option value='Software Services'>软件服务</Option>
              <Option value='Network Services'>网络服务</Option>
            </Select>
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            label='优先级'
            name='priority'
            rules={[{ required: true, message: '请选择优先级' }]}
          >
            <Select placeholder='请选择优先级' size='large'>
              <Option value={TicketPriority.LOW}>低</Option>
              <Option value={TicketPriority.MEDIUM}>中</Option>
              <Option value={TicketPriority.HIGH}>高</Option>
              <Option value={TicketPriority.URGENT}>紧急</Option>
            </Select>
          </Form.Item>
        </Col>
      </Row>

      <Row gutter={24}>
        <Col span={12}>
          <Form.Item label='处理人' name='assignee_id'>
            <Select placeholder='请选择处理人' allowClear size='large'>
              {userList.map(user => (
                <Option key={user.id} value={user.id}>
                  <div style={{ display: 'flex', alignItems: 'center' }}>
                    <Avatar size='small' style={{ backgroundColor: '#1890ff', marginRight: 8 }}>
                      {user.avatar}
                    </Avatar>
                    <span>{user.name}</span>
                    <span style={{ color: '#666', marginLeft: 8 }}>({user.role})</span>
                  </div>
                </Option>
              ))}
            </Select>
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item label='预计完成时间' name='estimated_time'>
            <DatePicker
              showTime
              placeholder='请选择预计完成时间'
              size='large'
              style={{ width: '100%' }}
            />
          </Form.Item>
        </Col>
      </Row>

      <Form.Item
        label='描述'
        name='description'
        rules={[{ required: true, message: '请输入工单描述' }]}
      >
        <Input.TextArea
          rows={6}
          placeholder='请详细描述工单内容、问题、预期结果等...'
        />
      </Form.Item>

      <Form.Item style={{ marginBottom: 0 }}>
        <div className='flex justify-end gap-3 pt-6 border-t border-gray-100'>
          <Button
            onClick={() => form.resetFields()}
            size='large'
            className='rounded-lg border-gray-200 hover:border-gray-300 transition-colors duration-200'
          >
            重置
          </Button>
          <Button
            type='primary'
            htmlType='submit'
            size='large'
            loading={loading}
            className='rounded-lg bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 border-0 shadow-md hover:shadow-lg transition-all duration-200'
            icon={<Plus size={16} />}
          >
            {isEditing ? '保存修改' : '创建工单'}
          </Button>
        </div>
      </Form.Item>
    </Form>
  );
});

TicketForm.displayName = 'TicketForm';

export const TicketModal: React.FC<TicketModalProps> = React.memo(
  ({ visible, editingTicket, onCancel, onSubmit, loading = false }) => {
    const [form] = Form.useForm();

    // 当编辑工单时，初始化表单数据
    useEffect(() => {
      if (visible) {
        if (editingTicket) {
          form.setFieldsValue({
            title: editingTicket.title,
            type: editingTicket.type,
            category: editingTicket.category,
            priority: editingTicket.priority,
            assignee_id: editingTicket.assignee_id,
            description: editingTicket.description,
            // 如果有预计完成时间，需要转换为 dayjs 对象
            // estimated_time: editingTicket.estimated_time ? dayjs(editingTicket.estimated_time) : undefined,
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
          <div className='flex items-center gap-3 pb-4 border-b border-gray-100'>
            <div className='w-10 h-10 bg-gradient-to-r from-blue-500 to-blue-600 rounded-lg flex items-center justify-center shadow-md'>
              {editingTicket ? (
                <Edit size={20} className='text-white' />
              ) : (
                <Plus size={20} className='text-white' />
              )}
            </div>
            <div>
              <h3 className='text-lg font-semibold text-gray-900'>
                {editingTicket ? '编辑工单' : '创建工单'}
              </h3>
              <p className='text-sm text-gray-500'>
                {editingTicket ? '修改工单信息' : '填写工单详细信息'}
              </p>
            </div>
          </div>
        }
        open={visible}
        onCancel={handleCancel}
        footer={null}
        width={900}
        className='[&_.ant-modal-content]:rounded-xl [&_.ant-modal-content]:shadow-2xl [&_.ant-modal-header]:border-0 [&_.ant-modal-header]:pb-0 [&_.ant-modal-body]:pt-6'
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
    const ticketTemplates = [
      {
        id: 1,
        name: 'System Login Issue',
        type: TicketType.INCIDENT,
        category: 'System Access',
        priority: TicketPriority.MEDIUM,
        description: 'User unable to login to system, technical support needed',
        estimatedTime: '2 hours',
        sla: '4 hours',
      },
      {
        id: 2,
        name: 'Printer Malfunction',
        type: TicketType.INCIDENT,
        category: 'Hardware Equipment',
        priority: TicketPriority.HIGH,
        description: 'Office printer not working properly',
        estimatedTime: '1 hour',
        sla: '2 hours',
      },
      {
        id: 3,
        name: 'Software Installation Request',
        type: TicketType.SERVICE_REQUEST,
        category: 'Software Services',
        priority: TicketPriority.LOW,
        description: 'Need to install new office software',
        estimatedTime: '30 minutes',
        sla: '4 hours',
      },
    ];

    return (
      <Modal
        title={
          <div className='flex items-center gap-3 pb-4 border-b border-gray-100'>
            <div className='w-10 h-10 bg-gradient-to-r from-green-500 to-green-600 rounded-lg flex items-center justify-center shadow-md'>
              <BookOpen size={20} className='text-white' />
            </div>
            <div>
              <h3 className='text-lg font-semibold text-gray-900'>工单模板管理</h3>
              <p className='text-sm text-gray-500'>
                管理和配置工单模板，提高工作效率
              </p>
            </div>
          </div>
        }
        open={visible}
        onCancel={onCancel}
        footer={null}
        width={1100}
        className='[&_.ant-modal-content]:rounded-xl [&_.ant-modal-content]:shadow-2xl [&_.ant-modal-header]:border-0 [&_.ant-modal-header]:pb-0 [&_.ant-modal-body]:pt-6'
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
              <p style={{ color: '#666' }}>
                管理所有可用的工单模板，提高工单创建效率
              </p>
            </div>
            <Button type='primary' icon={<Plus size={16} />}>
              新建模板
            </Button>
          </div>

          <div className='space-y-4'>
            {ticketTemplates.map(template => (
              <div
                key={template.id}
                className='p-4 border border-gray-200 rounded-lg hover:border-blue-300 transition-colors'
              >
                <div className='flex justify-between items-start'>
                  <div>
                    <h4 className='font-semibold text-gray-900'>{template.name}</h4>
                    <p className='text-sm text-gray-600 mt-1'>{template.description}</p>
                    <div className='flex gap-4 mt-2 text-xs text-gray-500'>
                      <span>类型: {template.type}</span>
                      <span>分类: {template.category}</span>
                      <span>优先级: {template.priority}</span>
                      <span>SLA: {template.sla}</span>
                    </div>
                  </div>
                  <div className='flex gap-2'>
                    <Button size='small' icon={<Edit size={14} />}>
                      编辑
                    </Button>
                    <Button size='small' danger>
                      删除
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </Modal>
    );
  }
);

TicketTemplateModal.displayName = 'TicketTemplateModal';
