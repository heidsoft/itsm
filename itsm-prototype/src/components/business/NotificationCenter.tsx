'use client';

import React, { useState, useEffect } from 'react';
import {
  Drawer,
  List,
  Badge,
  Button,
  Space,
  Typography,
  Tabs,
  Card,
  Form,
  Input,
  Select,
  Switch,
  DatePicker,
  Table,
  Tag,
  Modal,
  message,
  Tooltip,
  Popconfirm,
  Divider,
  Row,
  Col,
  Statistic,
  Progress,
  Alert,
  Empty,
} from 'antd';
import {
  Bell,
  Mail,
  MessageSquare,
  Smartphone,
  Settings,
  FileText,
  History,
  Edit,
  Trash2,
  Plus,
  Send,
  CheckCircle,
  Clock,
  AlertCircle,
  Filter,
  Download,
  Eye,
  Copy,
} from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;
const { TabPane } = Tabs;
const { TextArea } = Input;

interface Notification {
  id: number;
  title: string;
  message: string;
  type: 'info' | 'success' | 'warning' | 'error';
  channel: 'in_app' | 'email' | 'sms' | 'webhook';
  status: 'pending' | 'sent' | 'failed' | 'read';
  recipient: string;
  sent_at?: string;
  read_at?: string;
  created_at: string;
  template_id?: number;
  metadata?: Record<string, any>;
}

interface NotificationTemplate {
  id: number;
  name: string;
  description: string;
  type: 'info' | 'success' | 'warning' | 'error';
  channels: string[];
  subject?: string;
  content: string;
  variables: string[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

interface NotificationChannel {
  id: string;
  name: string;
  type: 'email' | 'sms' | 'webhook' | 'in_app';
  config: Record<string, any>;
  is_active: boolean;
  status: 'connected' | 'disconnected' | 'error';
  last_used?: string;
}

interface NotificationStats {
  total: number;
  unread: number;
  sent_today: number;
  failed_today: number;
  delivery_rate: number;
  avg_response_time: number;
}

const NotificationCenter: React.FC<{
  open: boolean;
  onClose: () => void;
}> = ({ open, onClose }) => {
  const [activeTab, setActiveTab] = useState('notifications');
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [templates, setTemplates] = useState<NotificationTemplate[]>([]);
  const [channels, setChannels] = useState<NotificationChannel[]>([]);
  const [stats, setStats] = useState<NotificationStats>({
    total: 0,
    unread: 0,
    sent_today: 0,
    failed_today: 0,
    delivery_rate: 0,
    avg_response_time: 0,
  });
  const [loading, setLoading] = useState(false);
  const [showTemplateModal, setShowTemplateModal] = useState(false);
  const [showChannelModal, setShowChannelModal] = useState(false);
  const [selectedTemplate, setSelectedTemplate] = useState<NotificationTemplate | null>(null);
  const [selectedChannel, setSelectedChannel] = useState<NotificationChannel | null>(null);
  const [form] = Form.useForm();
  const [channelForm] = Form.useForm();

  // 模拟数据
  useEffect(() => {
    if (open) {
      loadData();
    }
  }, [open]);

  const loadData = async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));

      // 模拟通知数据
      const mockNotifications: Notification[] = [
        {
          id: 1,
          title: '工单分配通知',
          message: '您有一个新的工单被分配：网络连接异常',
          type: 'info',
          channel: 'in_app',
          status: 'read',
          recipient: '张三',
          sent_at: '2024-01-15T10:30:00Z',
          read_at: '2024-01-15T10:35:00Z',
          created_at: '2024-01-15T10:30:00Z',
          template_id: 1,
        },
        {
          id: 2,
          title: 'SLA预警通知',
          message: '工单T-2024-001即将超时，请及时处理',
          type: 'warning',
          channel: 'email',
          status: 'sent',
          recipient: '李四',
          sent_at: '2024-01-15T09:00:00Z',
          created_at: '2024-01-15T09:00:00Z',
          template_id: 2,
        },
        {
          id: 3,
          title: '系统维护通知',
          message: '系统将于今晚22:00-24:00进行维护',
          type: 'info',
          channel: 'sms',
          status: 'sent',
          recipient: '王五',
          sent_at: '2024-01-15T08:00:00Z',
          created_at: '2024-01-15T08:00:00Z',
          template_id: 3,
        },
      ];

      const mockTemplates: NotificationTemplate[] = [
        {
          id: 1,
          name: '工单分配通知',
          description: '当工单被分配给用户时发送的通知',
          type: 'info',
          channels: ['in_app', 'email'],
          subject: '工单分配通知',
          content: '您有一个新的工单被分配：{{ticket_title}}，优先级：{{priority}}',
          variables: ['ticket_title', 'priority'],
          is_active: true,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
        {
          id: 2,
          name: 'SLA预警通知',
          description: '当工单即将超时时发送的预警通知',
          type: 'warning',
          channels: ['email', 'sms'],
          subject: 'SLA预警通知',
          content: '工单{{ticket_id}}即将超时，剩余时间：{{remaining_time}}',
          variables: ['ticket_id', 'remaining_time'],
          is_active: true,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ];

      const mockChannels: NotificationChannel[] = [
        {
          id: 'email',
          name: '邮件通知',
          type: 'email',
          config: {
            smtp_server: 'smtp.company.com',
            smtp_port: 587,
            username: 'notifications@company.com',
          },
          is_active: true,
          status: 'connected',
          last_used: '2024-01-15T10:30:00Z',
        },
        {
          id: 'sms',
          name: '短信通知',
          type: 'sms',
          config: {
            provider: '阿里云',
            api_key: '***',
            template_id: 'SMS_123456',
          },
          is_active: true,
          status: 'connected',
          last_used: '2024-01-15T09:00:00Z',
        },
        {
          id: 'webhook',
          name: 'Webhook通知',
          type: 'webhook',
          config: {
            url: 'https://api.company.com/webhook',
            method: 'POST',
            headers: { Authorization: 'Bearer ***' },
          },
          is_active: false,
          status: 'disconnected',
        },
      ];

      setNotifications(mockNotifications);
      setTemplates(mockTemplates);
      setChannels(mockChannels);

      // 计算统计数据
      const total = mockNotifications.length;
      const unread = mockNotifications.filter(n => n.status === 'pending').length;
      const sentToday = mockNotifications.filter(
        n => n.sent_at && new Date(n.sent_at).toDateString() === new Date().toDateString()
      ).length;
      const failedToday = mockNotifications.filter(n => n.status === 'failed').length;

      setStats({
        total,
        unread,
        sent_today: sentToday,
        failed_today: failedToday,
        delivery_rate: 95.5,
        avg_response_time: 2.3,
      });
    } catch (error) {
      console.error('加载通知数据失败:', error);
      message.error('加载数据失败');
    } finally {
      setLoading(false);
    }
  };

  const handleMarkRead = async (id: number) => {
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 200));

      setNotifications(prev =>
        prev.map(n =>
          n.id === id ? { ...n, status: 'read' as const, read_at: new Date().toISOString() } : n
        )
      );

      message.success('已标记为已读');
    } catch (error) {
      message.error('操作失败');
    }
  };

  const handleMarkAllRead = async () => {
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 200));

      setNotifications(prev =>
        prev.map(n => ({ ...n, status: 'read' as const, read_at: new Date().toISOString() }))
      );

      message.success('已全部标记为已读');
    } catch (error) {
      message.error('操作失败');
    }
  };

  const handleDeleteNotification = async (id: number) => {
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 200));

      setNotifications(prev => prev.filter(n => n.id !== id));
      message.success('删除成功');
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleSaveTemplate = async () => {
    try {
      const values = await form.validateFields();

      if (selectedTemplate) {
        // 更新模板
        setTemplates(prev =>
          prev.map(t =>
            t.id === selectedTemplate.id
              ? { ...t, ...values, updated_at: new Date().toISOString() }
              : t
          )
        );
        message.success('模板更新成功');
      } else {
        // 创建新模板
        const newTemplate: NotificationTemplate = {
          id: Date.now(),
          ...values,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        };
        setTemplates(prev => [...prev, newTemplate]);
        message.success('模板创建成功');
      }

      setShowTemplateModal(false);
      setSelectedTemplate(null);
      form.resetFields();
    } catch (error) {
      console.error('保存模板失败:', error);
    }
  };

  const handleSaveChannel = async () => {
    try {
      const values = await channelForm.validateFields();

      if (selectedChannel) {
        // 更新通道
        setChannels(prev => prev.map(c => (c.id === selectedChannel.id ? { ...c, ...values } : c)));
        message.success('通道更新成功');
      } else {
        // 创建新通道
        const newChannel: NotificationChannel = {
          id: values.type + '_' + Date.now(),
          ...values,
          status: 'disconnected',
        };
        setChannels(prev => [...prev, newChannel]);
        message.success('通道创建成功');
      }

      setShowChannelModal(false);
      setSelectedChannel(null);
      channelForm.resetFields();
    } catch (error) {
      console.error('保存通道失败:', error);
    }
  };

  const handleTestChannel = async (channelId: string) => {
    try {
      // 模拟测试通道
      await new Promise(resolve => setTimeout(resolve, 1000));
      message.success('通道测试成功');
    } catch (error) {
      message.error('通道测试失败');
    }
  };

  const getChannelIcon = (type: string) => {
    switch (type) {
      case 'email':
        return <Mail className='w-4 h-4' />;
      case 'sms':
        return <Smartphone className='w-4 h-4' />;
      case 'webhook':
        return <MessageSquare className='w-4 h-4' />;
      case 'in_app':
        return <Bell className='w-4 h-4' />;
      default:
        return <Bell className='w-4 h-4' />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'sent':
        return 'success';
      case 'pending':
        return 'processing';
      case 'failed':
        return 'error';
      case 'read':
        return 'default';
      default:
        return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'sent':
        return <CheckCircle className='w-4 h-4' />;
      case 'pending':
        return <Clock className='w-4 h-4' />;
      case 'failed':
        return <AlertCircle className='w-4 h-4' />;
      case 'read':
        return <CheckCircle className='w-4 h-4' />;
      default:
        return <Clock className='w-4 h-4' />;
    }
  };

  const notificationColumns: ColumnsType<Notification> = [
    {
      title: '通知内容',
      key: 'content',
      render: (_, record) => (
        <div className='space-y-1'>
          <div className='font-medium'>{record.title}</div>
          <div className='text-sm text-gray-600'>{record.message}</div>
          <div className='flex items-center gap-2 text-xs text-gray-500'>
            {getChannelIcon(record.channel)}
            <span>{record.recipient}</span>
            <span>•</span>
            <span>{new Date(record.created_at).toLocaleString()}</span>
          </div>
        </div>
      ),
    },
    {
      title: '类型',
      key: 'type',
      width: 100,
      render: (_, record) => (
        <Tag
          color={
            record.type === 'error'
              ? 'red'
              : record.type === 'warning'
              ? 'orange'
              : record.type === 'success'
              ? 'green'
              : 'blue'
          }
        >
          {record.type === 'error'
            ? '错误'
            : record.type === 'warning'
            ? '警告'
            : record.type === 'success'
            ? '成功'
            : '信息'}
        </Tag>
      ),
    },
    {
      title: '状态',
      key: 'status',
      width: 100,
      render: (_, record) => (
        <Tag color={getStatusColor(record.status)}>
          {getStatusIcon(record.status)}
          {record.status === 'sent'
            ? '已发送'
            : record.status === 'pending'
            ? '待发送'
            : record.status === 'failed'
            ? '发送失败'
            : '已读'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      render: (_, record) => (
        <Space size='small'>
          {record.status === 'pending' && (
            <Button size='small' onClick={() => handleMarkRead(record.id)}>
              标记已读
            </Button>
          )}
          <Popconfirm
            title='确认删除'
            description='确定要删除这条通知吗？'
            onConfirm={() => handleDeleteNotification(record.id)}
            okText='确认'
            cancelText='取消'
          >
            <Button size='small' danger icon={<Trash2 className='w-3 h-3' />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const templateColumns: ColumnsType<NotificationTemplate> = [
    {
      title: '模板名称',
      key: 'name',
      render: (_, record) => (
        <div className='space-y-1'>
          <div className='font-medium'>{record.name}</div>
          <div className='text-sm text-gray-600'>{record.description}</div>
        </div>
      ),
    },
    {
      title: '类型',
      key: 'type',
      width: 100,
      render: (_, record) => (
        <Tag
          color={
            record.type === 'error'
              ? 'red'
              : record.type === 'warning'
              ? 'orange'
              : record.type === 'success'
              ? 'green'
              : 'blue'
          }
        >
          {record.type === 'error'
            ? '错误'
            : record.type === 'warning'
            ? '警告'
            : record.type === 'success'
            ? '成功'
            : '信息'}
        </Tag>
      ),
    },
    {
      title: '通道',
      key: 'channels',
      width: 150,
      render: (_, record) => (
        <div className='flex gap-1'>
          {record.channels.map(channel => (
            <Tag key={channel}>
              {getChannelIcon(channel)}
              {channel === 'email'
                ? '邮件'
                : channel === 'sms'
                ? '短信'
                : channel === 'webhook'
                ? 'Webhook'
                : '站内信'}
            </Tag>
          ))}
        </div>
      ),
    },
    {
      title: '状态',
      key: 'is_active',
      width: 80,
      render: (_, record) => (
        <Tag color={record.is_active ? 'success' : 'default'}>
          {record.is_active ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      render: (_, record) => (
        <Space size='small'>
          <Button
            size='small'
            icon={<Edit className='w-3 h-3' />}
            onClick={() => {
              setSelectedTemplate(record);
              form.setFieldsValue(record);
              setShowTemplateModal(true);
            }}
          >
            编辑
          </Button>
          <Button
            size='small'
            icon={<Copy className='w-3 h-3' />}
            onClick={() => {
              const newTemplate = { ...record, id: Date.now(), name: `${record.name}_副本` };
              setTemplates(prev => [...prev, newTemplate]);
              message.success('模板复制成功');
            }}
          >
            复制
          </Button>
        </Space>
      ),
    },
  ];

  const channelColumns: ColumnsType<NotificationChannel> = [
    {
      title: '通道名称',
      key: 'name',
      render: (_, record) => (
        <div className='space-y-1'>
          <div className='font-medium'>{record.name}</div>
          <div className='text-sm text-gray-600'>
            {record.type === 'email'
              ? 'SMTP邮件服务'
              : record.type === 'sms'
              ? '短信服务'
              : record.type === 'webhook'
              ? 'Webhook回调'
              : '站内通知'}
          </div>
        </div>
      ),
    },
    {
      title: '状态',
      key: 'status',
      width: 120,
      render: (_, record) => (
        <div className='space-y-1'>
          <Tag color={record.is_active ? 'success' : 'default'}>
            {record.is_active ? '启用' : '禁用'}
          </Tag>
          <Tag
            color={
              record.status === 'connected'
                ? 'success'
                : record.status === 'error'
                ? 'error'
                : 'default'
            }
          >
            {record.status === 'connected'
              ? '已连接'
              : record.status === 'disconnected'
              ? '未连接'
              : '连接错误'}
          </Tag>
        </div>
      ),
    },
    {
      title: '最后使用',
      key: 'last_used',
      width: 150,
      render: (_, record) => (
        <div className='text-sm text-gray-500'>
          {record.last_used ? new Date(record.last_used).toLocaleString() : '从未使用'}
        </div>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_, record) => (
        <Space size='small'>
          <Button size='small' onClick={() => handleTestChannel(record.id)}>
            测试
          </Button>
          <Button
            size='small'
            icon={<Edit className='w-3 h-3' />}
            onClick={() => {
              setSelectedChannel(record);
              channelForm.setFieldsValue(record);
              setShowChannelModal(true);
            }}
          >
            配置
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Drawer
      title={
        <div className='flex items-center gap-2'>
          <Bell className='w-5 h-5 text-blue-500' />
          <span>通知中心</span>
          {stats.unread > 0 && <Badge count={stats.unread} size='small' />}
        </div>
      }
      placement='right'
      width={800}
      open={open}
      onClose={onClose}
      styles={{
        header: {
          borderBottom: '1px solid #f3f4f6',
          paddingBottom: '16px',
        },
      }}
    >
      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab='通知列表' key='notifications'>
          <div className='space-y-4'>
            {/* 统计卡片 */}
            <Row gutter={[16, 16]}>
              <Col span={6}>
                <Card size='small'>
                  <Statistic
                    title='总通知'
                    value={stats.total}
                    prefix={<Bell className='w-4 h-4' />}
                  />
                </Card>
              </Col>
              <Col span={6}>
                <Card size='small'>
                  <Statistic
                    title='未读'
                    value={stats.unread}
                    valueStyle={{ color: '#1890ff' }}
                    prefix={<AlertCircle className='w-4 h-4' />}
                  />
                </Card>
              </Col>
              <Col span={6}>
                <Card size='small'>
                  <Statistic
                    title='今日发送'
                    value={stats.sent_today}
                    valueStyle={{ color: '#52c41a' }}
                    prefix={<CheckCircle className='w-4 h-4' />}
                  />
                </Card>
              </Col>
              <Col span={6}>
                <Card size='small'>
                  <Statistic
                    title='发送成功率'
                    value={stats.delivery_rate}
                    suffix='%'
                    valueStyle={{ color: '#52c41a' }}
                    prefix={<Progress type='circle' size={24} percent={stats.delivery_rate} />}
                  />
                </Card>
              </Col>
            </Row>

            {/* 操作按钮 */}
            <div className='flex justify-between items-center'>
              <Space>
                <Button icon={<Filter className='w-4 h-4' />} onClick={() => {}}>
                  筛选
                </Button>
                <Button icon={<Download className='w-4 h-4' />} onClick={() => {}}>
                  导出
                </Button>
              </Space>
              <Space>
                <Button onClick={handleMarkAllRead} disabled={stats.unread === 0}>
                  全部标记已读
                </Button>
              </Space>
            </div>

            {/* 通知列表 */}
            <Table
              columns={notificationColumns}
              dataSource={notifications}
              rowKey='id'
              loading={loading}
              pagination={{
                pageSize: 10,
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: total => `共 ${total} 条记录`,
              }}
              size='small'
            />
          </div>
        </TabPane>

        <TabPane tab='通知模板' key='templates'>
          <div className='space-y-4'>
            <div className='flex justify-between items-center'>
              <Title level={5}>通知模板管理</Title>
              <Button
                type='primary'
                icon={<Plus className='w-4 h-4' />}
                onClick={() => {
                  setSelectedTemplate(null);
                  form.resetFields();
                  setShowTemplateModal(true);
                }}
              >
                新建模板
              </Button>
            </div>

            <Table columns={templateColumns} dataSource={templates} rowKey='id' size='small' />
          </div>
        </TabPane>

        <TabPane tab='通知通道' key='channels'>
          <div className='space-y-4'>
            <div className='flex justify-between items-center'>
              <Title level={5}>通知通道配置</Title>
              <Button
                type='primary'
                icon={<Plus className='w-4 h-4' />}
                onClick={() => {
                  setSelectedChannel(null);
                  channelForm.resetFields();
                  setShowChannelModal(true);
                }}
              >
                添加通道
              </Button>
            </div>

            <Table columns={channelColumns} dataSource={channels} rowKey='id' size='small' />
          </div>
        </TabPane>
      </Tabs>

      {/* 模板编辑模态框 */}
      <Modal
        title={selectedTemplate ? '编辑模板' : '新建模板'}
        open={showTemplateModal}
        onOk={handleSaveTemplate}
        onCancel={() => {
          setShowTemplateModal(false);
          setSelectedTemplate(null);
          form.resetFields();
        }}
        width={600}
        okText='保存'
        cancelText='取消'
      >
        <Form form={form} layout='vertical'>
          <Form.Item
            label='模板名称'
            name='name'
            rules={[{ required: true, message: '请输入模板名称' }]}
          >
            <Input placeholder='请输入模板名称' />
          </Form.Item>

          <Form.Item label='描述' name='description'>
            <Input placeholder='请输入模板描述' />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label='通知类型'
                name='type'
                rules={[{ required: true, message: '请选择通知类型' }]}
              >
                <Select placeholder='请选择通知类型'>
                  <Option value='info'>信息</Option>
                  <Option value='success'>成功</Option>
                  <Option value='warning'>警告</Option>
                  <Option value='error'>错误</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label='通知通道'
                name='channels'
                rules={[{ required: true, message: '请选择通知通道' }]}
              >
                <Select mode='multiple' placeholder='请选择通知通道'>
                  <Option value='in_app'>站内信</Option>
                  <Option value='email'>邮件</Option>
                  <Option value='sms'>短信</Option>
                  <Option value='webhook'>Webhook</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item label='邮件主题' name='subject'>
            <Input placeholder='请输入邮件主题' />
          </Form.Item>

          <Form.Item
            label='模板内容'
            name='content'
            rules={[{ required: true, message: '请输入模板内容' }]}
          >
            <TextArea rows={4} placeholder='请输入模板内容，支持变量：{{variable_name}}' />
          </Form.Item>

          <Form.Item label='可用变量' name='variables'>
            <Select mode='tags' placeholder='输入变量名后按回车'>
              <Option value='user_name'>用户名</Option>
              <Option value='ticket_id'>工单ID</Option>
              <Option value='ticket_title'>工单标题</Option>
              <Option value='priority'>优先级</Option>
            </Select>
          </Form.Item>

          <Form.Item label='启用状态' name='is_active' valuePropName='checked'>
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      {/* 通道配置模态框 */}
      <Modal
        title={selectedChannel ? '配置通道' : '添加通道'}
        open={showChannelModal}
        onOk={handleSaveChannel}
        onCancel={() => {
          setShowChannelModal(false);
          setSelectedChannel(null);
          channelForm.resetFields();
        }}
        width={600}
        okText='保存'
        cancelText='取消'
      >
        <Form form={channelForm} layout='vertical'>
          <Form.Item
            label='通道名称'
            name='name'
            rules={[{ required: true, message: '请输入通道名称' }]}
          >
            <Input placeholder='请输入通道名称' />
          </Form.Item>

          <Form.Item
            label='通道类型'
            name='type'
            rules={[{ required: true, message: '请选择通道类型' }]}
          >
            <Select placeholder='请选择通道类型'>
              <Option value='email'>邮件</Option>
              <Option value='sms'>短信</Option>
              <Option value='webhook'>Webhook</Option>
            </Select>
          </Form.Item>

          <Form.Item label='启用状态' name='is_active' valuePropName='checked'>
            <Switch />
          </Form.Item>

          <Divider>配置信息</Divider>

          <Form.Item
            label='SMTP服务器'
            name={['config', 'smtp_server']}
            rules={[{ required: true, message: '请输入SMTP服务器' }]}
          >
            <Input placeholder='请输入SMTP服务器地址' />
          </Form.Item>

          <Form.Item
            label='SMTP端口'
            name={['config', 'smtp_port']}
            rules={[{ required: true, message: '请输入SMTP端口' }]}
          >
            <Input type='number' placeholder='请输入SMTP端口' />
          </Form.Item>

          <Form.Item
            label='用户名'
            name={['config', 'username']}
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input placeholder='请输入用户名' />
          </Form.Item>

          <Form.Item
            label='密码'
            name={['config', 'password']}
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password placeholder='请输入密码' />
          </Form.Item>
        </Form>
      </Modal>
    </Drawer>
  );
};

export default NotificationCenter;
