'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { useParams } from 'next/navigation';
import { TicketApi } from '@/lib/api/ticket-api';
import { UserApi } from '@/lib/api/user-api';
import type { Ticket } from '@/app/lib/api-config';
import type { User } from '@/lib/api/user-api';
import { ArrowLeft, AlertCircle, XCircle, UserCheck, Edit, Save, X } from 'lucide-react';
import Link from 'next/link';
import { Button, Card, Typography, App, Badge, Tag, Descriptions, Space, Modal, Form, Select, Input } from 'antd';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { Option } = Select;

const TicketDetailPage: React.FC = () => {
  const params = useParams();
  const { message: antMessage } = App.useApp();
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [assignModalVisible, setAssignModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [users, setUsers] = useState<User[]>([]);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [assignForm] = Form.useForm();
  const [editForm] = Form.useForm();

  const ticketId = parseInt(params.ticketId as string);

  // Get ticket details
  const fetchTicket = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await TicketApi.getTicket(ticketId);
      setTicket(data as Ticket);
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Network error');
    } finally {
      setLoading(false);
    }
  }, [ticketId]);

  // Get users for assignment
  const fetchUsers = useCallback(async () => {
    try {
      setLoadingUsers(true);
      const data = await UserApi.getUsers({ page_size: 1000 });
      setUsers(data.users || []);
    } catch (error) {
      antMessage.error('获取用户列表失败');
    } finally {
      setLoadingUsers(false);
    }
  }, [antMessage]);

  useEffect(() => {
    if (ticketId) {
      fetchTicket();
    }
  }, [ticketId, fetchTicket]);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  // Handle approval (轻量版：仅改状态)
  const handleApprove = async () => {
    try {
      await TicketApi.updateTicketStatus(ticketId, 'approved');
      antMessage.success('批准成功');
      fetchTicket();
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : '网络错误');
    }
  };

  // Handle rejection (轻量版：仅改状态)
  const handleReject = async () => {
    try {
      await TicketApi.updateTicketStatus(ticketId, 'rejected');
      antMessage.success('已拒绝');
      fetchTicket();
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : '网络错误');
    }
  };

  // Handle assignment
  const handleAssign = () => {
    setAssignModalVisible(true);
  };

  // Handle assignment submit
  const handleAssignSubmit = async (values: { assignee_id: number; comment?: string }) => {
    try {
      await TicketApi.assignTicket(ticketId, values);
      antMessage.success('工单分配成功');
      setAssignModalVisible(false);
      assignForm.resetFields();
      fetchTicket();
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : '分配失败');
    }
  };

  // Handle edit
  const handleUpdate = () => {
    if (ticket) {
      editForm.setFieldsValue({
        title: ticket.title,
        description: ticket.description,
        priority: ticket.priority,
        status: ticket.status,
      });
      setEditModalVisible(true);
    }
  };

  // Handle edit submit
  const handleEditSubmit = async (values: Partial<Ticket>) => {
    try {
      await TicketApi.updateTicket(ticketId, values);
      antMessage.success('工单更新成功');
      setEditModalVisible(false);
      fetchTicket();
    } catch (error) {
      antMessage.error(error instanceof Error ? error.message : '更新失败');
    }
  };

  if (loading) {
    return (
      <div className='p-6'>
        <Card>
          <div className='text-center py-8'>
            <div className='animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto'></div>
            <Text className='mt-4 block'>加载中...</Text>
          </div>
        </Card>
      </div>
    );
  }

  if (error) {
    return (
      <div className='p-6'>
        <Card>
          <div className='text-center py-8'>
            <AlertCircle className='w-12 h-12 text-red-500 mx-auto mb-4' />
            <Title level={4} className='text-red-600 mb-2'>
              加载失败
            </Title>
            <Text type='secondary'>{error}</Text>
            <div className='mt-4'>
              <Button type='primary' onClick={fetchTicket}>
                重试
              </Button>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  if (!ticket) {
    return (
      <div className='p-6'>
        <Card>
          <div className='text-center py-8'>
            <XCircle className='w-12 h-12 text-gray-400 mx-auto mb-4' />
            <Title level={4} className='text-gray-600 mb-2'>
              未找到工单
            </Title>
            <Text type='secondary'>未找到指定的工单</Text>
            <div className='mt-4'>
              <Link href='/tickets'>
                <Button type='primary'>返回工单列表</Button>
              </Link>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className='p-6'>
      {/* Page header */}
      <div className='mb-6'>
        <div className='flex items-center justify-between mb-4'>
          <div className='flex items-center space-x-4'>
            <Link href='/tickets'>
              <Button icon={<ArrowLeft />} type='text'>
                返回列表
              </Button>
            </Link>
            <div>
              <Title level={2} className='!mb-1 !text-gray-900'>
                工单详情 #{ticket.id}
              </Title>
              <Text type='secondary'>{ticket.title}</Text>
            </div>
          </div>
          <div className='flex items-center space-x-2'>
            <Badge
              status={
                ticket.status === 'open'
                  ? 'processing'
                  : ticket.status === 'closed'
                  ? 'success'
                  : 'warning'
              }
              text={
                ticket.status === 'open'
                  ? '处理中'
                  : ticket.status === 'closed'
                  ? '已关闭'
                  : '待处理'
              }
            />
            <Tag
              color={
                ticket.priority === 'high'
                  ? 'red'
                  : ticket.priority === 'medium'
                  ? 'orange'
                  : 'green'
              }
            >
              {ticket.priority === 'high'
                ? '高优先级'
                : ticket.priority === 'medium'
                ? '中优先级'
                : '低优先级'}
            </Tag>
          </div>
        </div>
      </div>

      <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
        <Space orientation='vertical' size={16} style={{ width: '100%' }}>
          <Descriptions column={2} bordered size='middle'>
            <Descriptions.Item label='标题'>{ticket.title}</Descriptions.Item>
            <Descriptions.Item label='编号'>{ticket.ticket_number}</Descriptions.Item>
            <Descriptions.Item label='状态'>{ticket.status}</Descriptions.Item>
            <Descriptions.Item label='优先级'>{ticket.priority}</Descriptions.Item>
            <Descriptions.Item label='创建时间'>{ticket.created_at}</Descriptions.Item>
            <Descriptions.Item label='更新时间'>{ticket.updated_at}</Descriptions.Item>
            <Descriptions.Item label='描述' span={2}>
              {ticket.description}
            </Descriptions.Item>
          </Descriptions>

          <Space>
            <Button type='primary' onClick={handleApprove}>
              批准
            </Button>
            <Button danger onClick={handleReject}>
              拒绝
            </Button>
            <Button icon={<UserCheck size={16} />} onClick={handleAssign} loading={loadingUsers}>
              分配
            </Button>
            <Button icon={<Edit size={16} />} onClick={handleUpdate}>
              编辑
            </Button>
          </Space>

          <Text type='secondary'>
            工单详情页功能已恢复 - 支持分配、编辑等基本操作。
          </Text>
        </Space>

        {/* Assignment Modal */}
        <Modal
          title={
            <Space>
              <UserCheck className="w-5 h-5 text-blue-600" />
              分配工单
            </Space>
          }
          open={assignModalVisible}
          onCancel={() => {
            setAssignModalVisible(false);
            assignForm.resetFields();
          }}
          footer={null}
          width={500}
        >
          <Form
            form={assignForm}
            layout="vertical"
            onFinish={handleAssignSubmit}
          >
            <Form.Item
              label="分配给"
              name="assignee_id"
              rules={[{ required: true, message: '请选择处理人' }]}
            >
              <Select
                placeholder="请选择处理人"
                loading={loadingUsers}
                showSearch
                filterOption={(input, option) =>
                  (option?.children as unknown as string)?.toLowerCase().includes(input.toLowerCase())
                }
              >
                {users.map(user => (
                  <Option key={user.id} value={user.id}>
                    <Space>
                      <span>{user.name}</span>
                      <Text type="secondary" className="text-xs">
                        ({user.username})
                      </Text>
                      {user.department && (
                        <Tag color="blue">
                          {user.department}
                        </Tag>
                      )}
                    </Space>
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item
              label="备注"
              name="comment"
            >
              <TextArea
                rows={3}
                placeholder="请输入分配备注（可选）"
                maxLength={500}
                showCount
              />
            </Form.Item>
            <Form.Item className="mb-0">
              <Space className="w-full justify-end">
                <Button
                  icon={<X />}
                  onClick={() => {
                    setAssignModalVisible(false);
                    assignForm.resetFields();
                  }}
                >
                  取消
                </Button>
                <Button
                  type="primary"
                  htmlType="submit"
                  icon={<Save />}
                >
                  确认分配
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Modal>

        {/* Edit Modal */}
        <Modal
          title={
            <Space>
              <Edit className="w-5 h-5 text-green-600" />
              编辑工单
            </Space>
          }
          open={editModalVisible}
          onCancel={() => {
            setEditModalVisible(false);
            editForm.resetFields();
          }}
          footer={null}
          width={600}
        >
          <Form
            form={editForm}
            layout="vertical"
            onFinish={handleEditSubmit}
          >
            <Form.Item
              label="工单标题"
              name="title"
              rules={[
                { required: true, message: '请输入工单标题' },
                { max: 100, message: '标题不能超过100个字符' }
              ]}
            >
              <Input placeholder="请输入工单标题" />
            </Form.Item>
            <Form.Item
              label="工单描述"
              name="description"
              rules={[
                { required: true, message: '请输入工单描述' },
                { max: 2000, message: '描述不能超过2000个字符' }
              ]}
            >
              <TextArea
                rows={6}
                placeholder="请输入工单描述"
                showCount
                maxLength={2000}
              />
            </Form.Item>
            <div className="grid grid-cols-2 gap-4">
              <Form.Item
                label="优先级"
                name="priority"
                rules={[{ required: true, message: '请选择优先级' }]}
              >
                <Select placeholder="请选择优先级">
                  <Option value="low">
                    <Tag color="green">低优先级</Tag>
                  </Option>
                  <Option value="medium">
                    <Tag color="orange">中优先级</Tag>
                  </Option>
                  <Option value="high">
                    <Tag color="red">高优先级</Tag>
                  </Option>
                </Select>
              </Form.Item>
              <Form.Item
                label="状态"
                name="status"
                rules={[{ required: true, message: '请选择状态' }]}
              >
                <Select placeholder="请选择状态">
                  <Option value="new">待处理</Option>
                  <Option value="open">处理中</Option>
                  <Option value="pending">暂停</Option>
                  <Option value="resolved">已解决</Option>
                  <Option value="closed">已关闭</Option>
                </Select>
              </Form.Item>
            </div>
            <Form.Item className="mb-0">
              <Space className="w-full justify-end">
                <Button
                  icon={<X />}
                  onClick={() => {
                    setEditModalVisible(false);
                    editForm.resetFields();
                  }}
                >
                  取消
                </Button>
                <Button
                  type="primary"
                  htmlType="submit"
                  icon={<Save />}
                >
                  保存修改
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Modal>
      </Card>
    </div>
  );
};

export default TicketDetailPage;
