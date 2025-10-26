'use client';

import React from 'react';
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
}> = React.memo(({ form, onSubmit, loading = false }) => {
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
            label='Title'
            name='title'
            rules={[{ required: true, message: 'Please enter ticket title' }]}
          >
            <Input placeholder='Please enter ticket title' size='large' />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            label='Type'
            name='type'
            rules={[{ required: true, message: 'Please select ticket type' }]}
          >
            <Select placeholder='Please select ticket type' size='large'>
              <Option value={TicketType.INCIDENT}>Incident</Option>
              <Option value={TicketType.SERVICE_REQUEST}>Service Request</Option>
              <Option value={TicketType.PROBLEM}>Problem</Option>
              <Option value={TicketType.CHANGE}>Change</Option>
            </Select>
          </Form.Item>
        </Col>
      </Row>

      <Row gutter={24}>
        <Col span={12}>
          <Form.Item
            label='Category'
            name='category'
            rules={[{ required: true, message: 'Please select ticket category' }]}
          >
            <Select placeholder='Please select ticket category' size='large'>
              <Option value='System Access'>System Access</Option>
              <Option value='Hardware Equipment'>Hardware Equipment</Option>
              <Option value='Software Services'>Software Services</Option>
              <Option value='Network Services'>Network Services</Option>
            </Select>
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            label='Priority'
            name='priority'
            rules={[{ required: true, message: 'Please select priority' }]}
          >
            <Select placeholder='Please select priority' size='large'>
              <Option value={TicketPriority.LOW}>Low</Option>
              <Option value={TicketPriority.MEDIUM}>Medium</Option>
              <Option value={TicketPriority.HIGH}>High</Option>
              <Option value={TicketPriority.URGENT}>Urgent</Option>
            </Select>
          </Form.Item>
        </Col>
      </Row>

      <Row gutter={24}>
        <Col span={12}>
          <Form.Item label='Assignee' name='assignee_id'>
            <Select placeholder='Please select assignee' allowClear size='large'>
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
          <Form.Item label='Estimated Time' name='estimated_time'>
            <DatePicker
              showTime
              placeholder='Please select estimated time'
              size='large'
              style={{ width: '100%' }}
            />
          </Form.Item>
        </Col>
      </Row>

      <Form.Item
        label='Description'
        name='description'
        rules={[{ required: true, message: 'Please input ticket description' }]}
      >
        <Input.TextArea
          rows={6}
          placeholder='Please detailedly describe the content of the ticket, the problem, the expected result, etc...'
        />
      </Form.Item>

      <Form.Item style={{ marginBottom: 0 }}>
        <div className='flex justify-end gap-3 pt-6 border-t border-gray-100'>
          <Button
            onClick={() => form.resetFields()}
            size='large'
            className='rounded-lg border-gray-200 hover:border-gray-300 transition-colors duration-200'
          >
            Reset
          </Button>
          <Button
            type='primary'
            htmlType='submit'
            size='large'
            loading={loading}
            className='rounded-lg bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 border-0 shadow-md hover:shadow-lg transition-all duration-200'
            icon={<Plus size={16} />}
          >
            Create Ticket
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
                {editingTicket ? 'Edit Ticket' : 'Create Ticket'}
              </h3>
              <p className='text-sm text-gray-500'>
                {editingTicket ? 'Modify ticket information' : 'Fill in ticket details'}
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
        <TicketForm form={form} onSubmit={handleSubmit} loading={loading} />
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
              <h3 className='text-lg font-semibold text-gray-900'>Ticket Template Management</h3>
              <p className='text-sm text-gray-500'>
                Manage and configure ticket templates to improve work efficiency
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
                Manage all available ticket templates to improve ticket creation efficiency
              </p>
            </div>
            <Button type='primary' icon={<Plus size={16} />}>
              New Template
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
                      <span>Type: {template.type}</span>
                      <span>Category: {template.category}</span>
                      <span>Priority: {template.priority}</span>
                      <span>SLA: {template.sla}</span>
                    </div>
                  </div>
                  <div className='flex gap-2'>
                    <Button size='small' icon={<Edit size={14} />}>
                      Edit
                    </Button>
                    <Button size='small' danger>
                      Delete
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
